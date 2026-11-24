# Classification dashboard — design

**Date:** 2026-11-24 · **Size:** small change

## Intent

A simple public web page showing how many images Gemini has classified into
each category: ripe, unripe, overripe, not fruit. Today no result is stored
anywhere, so counting starts from zero when this ships.

## Goals / non-goals

- **Goal:** persistent per-category counts that survive restarts and redeploys,
  shown on one page that refreshes on its own.
- **Non-goals:** storing images (MinIO idea dropped), image galleries, per-fruit
  counts, YOLO `/analyze` results, charts over time, login.

## Approaches

- **A. SQLite file in a Docker volume, one row per classification** — *chosen.*
  No new service; keeps history so time-based views can be added later.
- **B. JSON file with four totals** — least code, but loses all history.
- In-memory counters — rejected: every push to `main` redeploys and resets them.

## Design

- **Storage** — new `internal/store` package using SQLite via
  `modernc.org/sqlite` (pure Go, no extra system libs). Table
  `classifications(id, created_at, source, label, raw)`; `source` is `mqtt` or
  `http`, `raw` is Gemini's untouched reply. Path from `DB_PATH`, default
  `/app/data/vleefruit.db`. Schema created on startup.
- **Normalization** — lowercase, trim whitespace and trailing punctuation; match
  `ripe` / `unripe` / `overripe` / `not fruit`, else `other`.
- **Recording** — after a *successful* `Analyze` in `mqtt.handleImage` and
  `ImageController.Upload`. Gemini errors aren't recorded. A failed DB write is
  logged and never fails the request or the MQTT reply.
- **API** — `GET /api/v1/stats` →
  `{"counts":{"ripe":n,"unripe":n,"overripe":n,"not fruit":n,"other":n},"total":n}`,
  with Swagger annotations regenerated.
- **Page** — one static HTML file embedded in the binary (`go:embed`), served at
  `GET /`. Four tiles, plus total and "other", polls `/api/v1/stats` every 5s.
  Public, no auth.
- **Deploy** — named volume `vleefruit_data` mounted at `/app/data` in
  `docker-compose.yml`; the Dockerfile creates `/app/data` owned by the `app`
  user (uid 10001) so the volume is writable. `DB_PATH` added to compose and
  `.env.example`.
- **Tests** — normalization table test, store insert/count test (temp file),
  stats handler test.

## Left to the builder

Exact page styling, SQL details, how the store is passed into the MQTT client
and controller (constructor argument, matching existing wiring in `main.go`).
