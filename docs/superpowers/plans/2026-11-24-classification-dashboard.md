# Classification Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist every successful Gemini classification in SQLite and show per-category counts on a public web page.

**Architecture:** New `internal/store` package (SQLite, one row per classification) is passed into the MQTT client and the image controller, which record after a successful `Analyze`. A new stats controller exposes counts at `GET /api/v1/stats`; an embedded static page at `GET /` polls it.

**Tech Stack:** Go 1.26, Gin, `modernc.org/sqlite` (pure Go, `database/sql` driver name `"sqlite"`), `go:embed`.

**Spec:** `docs/superpowers/specs/2026-11-24-classification-dashboard-design.md`

## Global Constraints

- Labels, in this display order: `ripe`, `unripe`, `overripe`, `not fruit`, plus `other` for anything unrecognized.
- Sources: exactly `"mqtt"` and `"http"`.
- Only successful `Analyze` results are recorded; Gemini errors are not.
- A failed DB write is logged (`log.Printf`) and never changes the HTTP response or the MQTT reply.
- `DB_PATH` env var, default `/app/data/vleefruit.db`.
- Stats JSON shape: `{"counts":{"ripe":n,"unripe":n,"overripe":n,"not fruit":n,"other":n},"total":n}` — all five keys always present.
- Page is public, no auth, polls every 5 seconds.
- **No local Go toolchain and the VPS must not compile Go** (2 GB RAM, freezes; other projects share it). Do NOT run `go build`/`go test` here. Verification happens in CI (`gofmt -l`, `go vet`, `go build`, `go test -race`). Write code carefully and gofmt-clean by hand (tabs, import grouping std / blank line / third-party).
- Match existing style: small packages, doc comments on exported identifiers, constructor `NewXxx`, Swagger annotations on handlers.

## Review Focus

1. **Concurrent writes** (MQTT handler goroutines + HTTP requests at once) must not fail with `SQLITE_BUSY` → `db.SetMaxOpenConns(1)`; test `TestRecordConcurrent` in Task 1.
2. **Gemini reply variations** (`"Ripe."`, `" RIPE\n"`, `"Not  Fruit"`, `"not-fruit"`, `"banana"`) land in the right bucket → `TestNormalize` table in Task 1.
3. **Counts survive restart** (redeploy reopens the same file) → `TestCountsPersistAcrossReopen` in Task 1.
4. **Store failed to open** (bad path / unwritable volume): server still starts, uploads still work, stats returns 503 → nil-store tests in Tasks 2 and 3.
5. **Volume not writable by uid 10001** → Dockerfile creates `/app/data` owned by `app` (Task 4); checked live in Task 5.

---

### Task 1: SQLite store

**Files:**
- Create: `backend/internal/store/store.go`
- Test: `backend/internal/store/store_test.go`
- Modify: `backend/go.mod`, `backend/go.sum` (add `modernc.org/sqlite`)

**Interfaces:**
- Produces:
  - `var Labels = []string{"ripe", "unripe", "overripe", "not fruit", "other"}`
  - `func Normalize(raw string) string`
  - `type Store struct` with `func Open(path string) (*Store, error)`, `func (s *Store) Record(ctx context.Context, source, raw string) error`, `func (s *Store) Counts(ctx context.Context) (map[string]int, error)`, `func (s *Store) Close() error`

- [ ] **Step 1: Write the failing tests** in `store_test.go`:
  - `TestNormalize` table: `"ripe"→"ripe"`, `" RIPE\n"→"ripe"`, `"Ripe."→"ripe"`, `"Overripe!"→"overripe"`, `"unripe"→"unripe"`, `"Not  Fruit"→"not fruit"`, `"not-fruit"→"not fruit"`, `"\"ripe\""→"ripe"`, `"banana"→"other"`, `""→"other"`.
  - `TestCountsEmpty`: fresh store (path `filepath.Join(t.TempDir(), "sub", "t.db")` — parent dir must be created by `Open`) → `Counts` returns map with all 5 `Labels` keys, each 0.
  - `TestRecordAndCount`: record `("mqtt","Ripe.")`, `("http","ripe")`, `("mqtt","unripe")`, `("http","???")` → counts `ripe:2, unripe:1, overripe:0, not fruit:0, other:1`.
  - `TestCountsPersistAcrossReopen`: record 1 `ripe`, `Close`, `Open` same path → `ripe == 1`.
  - `TestRecordConcurrent`: 50 goroutines each `Record(ctx,"mqtt","ripe")`, all return nil → `ripe == 50`.

- [ ] **Step 2: Implement `store.go`**
  - `Normalize`: lowercase; replace `-` and `_` with space; trim spaces and `.!?"'` (`strings.Trim` with `" \t\r\n.!?\"'"`); collapse whitespace via `strings.Join(strings.Fields(s), " ")`; return it if in `Labels[:4]`, else `"other"`.
  - `Open`: `os.MkdirAll(filepath.Dir(path), 0o755)`, `sql.Open("sqlite", path)`, `db.SetMaxOpenConns(1)`, then exec:
    ```sql
    PRAGMA journal_mode=WAL;
    CREATE TABLE IF NOT EXISTS classifications (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
      source TEXT NOT NULL,
      label TEXT NOT NULL,
      raw TEXT NOT NULL
    );
    ```
  - `Record`: insert `source, Normalize(raw), raw`.
  - `Counts`: `SELECT label, COUNT(*) FROM classifications GROUP BY label`, starting from a map with every `Labels` key at 0.
  - Add `modernc.org/sqlite` to `go.mod` require block (latest stable v1.x) and its `go.sum` entries. Since `go mod tidy` can't run here, the implementer generates go.mod/go.sum in a throwaway container ONLY for `go mod tidy` (it downloads, doesn't compile): `docker run --rm --memory=400m -v /opt/vleefruit/backend:/src -w /src golang:1.26-bookworm go mod tidy`. If it gets OOM-killed, stop and report.

- [ ] **Step 3: Commit**
  ```bash
  git add backend/internal/store backend/go.mod backend/go.sum
  git commit -m "feat(store): sqlite store for gemini classifications"
  ```

### Task 2: Record results from MQTT and HTTP

**Files:**
- Modify: `backend/internal/mqtt/mqtt.go` (`Client`, `NewClient`, `handleImage`)
- Modify: `backend/internal/controller/image_controller.go`
- Test: `backend/internal/controller/image_controller_test.go`, `backend/internal/mqtt/mqtt_test.go`

**Interfaces:**
- Consumes: `(*store.Store).Record` from Task 1 (via interface, no import of `store` needed in mqtt/controller).
- Produces:
  - In `mqtt`: `type Recorder interface { Record(ctx context.Context, source, raw string) error }`; `func NewClient(cfg Config, analyzer ImageAnalyzer, recorder Recorder) *Client` (recorder may be nil).
  - In `controller`: `type Recorder interface { Record(ctx context.Context, source, raw string) error }`; `func NewImageController(client ImageAnalyzer, recorder Recorder) *ImageController`, where `controller.ImageAnalyzer` is a new interface `Analyze(ctx context.Context, imageData []byte, mimeType string) (string, error)` so tests can fake Gemini (`*llm.Client` satisfies it; keep the `errors.Is(err, llm.ErrNotConfigured)` check).

- [ ] **Step 1: Write failing tests**
  - `mqtt_test.go` `TestRecordImageResult`: extract the post-analyze part of `handleImage` into `func (c *Client) analyzeAndRecord(ctx context.Context, data []byte) string` (returns the payload to publish). Fake analyzer returning `"ripe"` + fake recorder → returns `"ripe"`, recorder got `("mqtt","ripe")`. Fake analyzer erroring → returns `"error: ..."`, recorder not called. Recorder returning an error → still returns `"ripe"`. Nil recorder → no panic.
  - `image_controller_test.go` (gin test mode, `httptest`, multipart with a PNG content-type part): fake analyzer `"Overripe"` → 200, recorder got `("http","Overripe")`; analyzer error → 502 and no record; recorder error → still 200; nil recorder → 200.

- [ ] **Step 2: Implement.** Record with `context.Background()`-derived ctx (not the request ctx, so a client disconnect doesn't cancel the write) using a 5s timeout; on error `log.Printf("store: record failed: %v", err)`. `handleImage` keeps its existing nil/empty checks then publishes `analyzeAndRecord`'s result.

- [ ] **Step 3: Commit** — `feat: record gemini classifications from mqtt and http`

### Task 3: Stats endpoint, dashboard page, wiring

**Files:**
- Create: `backend/internal/controller/stats_controller.go`, test `stats_controller_test.go`
- Create: `backend/internal/web/web.go`, `backend/internal/web/index.html`
- Modify: `controller/controller.go` (add `Stats *StatsController`), `router/router.go`, `config/config.go` (add `DBPath string` = `getenv("DB_PATH", "/app/data/vleefruit.db")`), `cmd/server/main.go`
- Modify: `backend/docs/docs.go`, `swagger.json`, `swagger.yaml` — hand-add `GET /api/v1/stats` (tag `stats`, 200 object, 503 object), matching the existing entries' format (swag CLI can't run here).

**Interfaces:**
- Consumes: `store.Open`, `(*store.Store).Counts`, `store.Labels` (Task 1); new constructors from Task 2.
- Produces: `type Counter interface { Counts(ctx context.Context) (map[string]int, error) }`; `func NewStatsController(counter Counter) *StatsController`; handler `Stats(c *gin.Context)`; `web.Index []byte` (via `//go:embed index.html`).

- [ ] **Step 1: Write failing tests** in `stats_controller_test.go`: fake counter `{ripe:2, unripe:1, overripe:0, "not fruit":3, other:0}` → 200, body decodes to `counts` equal to that map and `total == 6`; counter error → 500 `{"error":...}`; nil counter → 503 `{"error":"stats storage unavailable"}`.

- [ ] **Step 2: Implement controller + router**: `api.GET("/stats", c.Stats.Stats)`; `r.GET("/", func(ctx *gin.Context){ ctx.Data(200, "text/html; charset=utf-8", web.Index) })`.

- [ ] **Step 3: main.go wiring**: `st, err := store.Open(cfg.DBPath)`; on error `log.Printf("store: open %s failed, stats disabled: %v", ...)` and pass **untyped nil** interfaces (careful: never pass a nil `*store.Store` into an interface — branch so the interface value itself is nil). `defer st.Close()` when non-nil.

- [ ] **Step 4: `index.html`**: self-contained (inline CSS/JS, no CDN). Title "VleeFruit". Four tiles in `Labels` order with large numbers, line below with total and other, "updated Ns ago". `fetch('/api/v1/stats')` on load and every 5000 ms; on failure show "stats unavailable" without clearing last numbers. Responsive (tiles wrap at phone width), light/dark via `prefers-color-scheme`.

- [ ] **Step 5: Commit** — `feat: stats endpoint and dashboard page`

### Task 4: Deploy config

**Files:** `backend/Dockerfile`, `docker-compose.yml`, `.env.example`, `docs/DEPLOY.md`

- [ ] **Step 1:** Dockerfile final stage, before `USER app`: `RUN mkdir -p /app/data && chown app:app /app/data`.
- [ ] **Step 2:** compose `backend`: volume `- vleefruit_data:/app/data`, env `DB_PATH: ${DB_PATH:-/app/data/vleefruit.db}`; top-level `volumes:` add `vleefruit_data:`.
- [ ] **Step 3:** `.env.example`: add `DB_PATH=/app/data/vleefruit.db` with a comment; remove the stale Postgres/Redis/MinIO block. `DEPLOY.md` Notes: one bullet on the SQLite volume (`vleefruit_data`) and dashboard at `/`; fix the "only backend + mqtt" bullet.
- [ ] **Step 4: Commit** — `feat(deploy): persist sqlite db in a volume`

### Task 5: Deploy and verify live

- [ ] **Step 1:** User pushes `main` (no push credentials on this host). Wait for CI (gofmt, vet, build, `go test -race`) and the Deploy workflow to succeed; fix CI failures with new commits.
- [ ] **Step 2:** `docker ps` shows `vleefruit-backend-1` on the new commit's image tag; `docker logs vleefruit-backend-1` has no `store: open` error.
- [ ] **Step 3:** `curl -fsS http://localhost:8090/api/v1/stats` → JSON with all five keys. `curl -fsS http://localhost:8090/ | grep -i vleefruit`. Same via `https://vleefruit.marcolinardi.site/`.
- [ ] **Step 4:** Publish a test image as the device user (`mosquitto_pub` inside `vleefruit-mqtt-1`, creds from `.env`, `-f` a small JPEG) to `vleefruit/image/request`; subscribe to `vleefruit/image/response` for the reply; stats total increases by 1 in the replied category. `POST /api/v1/images` with the same image → total increases again.
- [ ] **Step 5:** `docker compose restart backend` → stats unchanged (persistence).
