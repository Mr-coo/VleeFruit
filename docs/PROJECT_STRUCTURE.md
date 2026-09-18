# VleeFruit Backend — Project Structure & Foundations

## Context

VleeFruit is a **fruit ripeness detection** backend. The repo is greenfield: only
`go.mod`/`go.sum`, a minimal `main.go` (Gin + GORM/Postgres + Redis health check),
and the Docker stack already built (`backend/Dockerfile`, `docker-compose.yml` with
Postgres, Redis, MinIO). This document defines the real project skeleton plus the IoT
ingestion + auth foundation so features have a home.

**Core flow:** an IoT device publishes an image over MQTT → the backend stores it,
runs ONNX ripeness detection → publishes the result back to the device over MQTT.

## Decisions

- **Data stores:** Postgres (relational, GORM) + Redis (cache). No MongoDB.
- **Detection:** ONNX model run **in-process** in Go, behind a pluggable interface
  (a `noop` stub first so everything builds before a real model exists).
- **Image transport:** **MQTT request/response** (`devices/{id}/images` in,
  `devices/{id}/results` out). REST also exists for management/query.
- **Sensor readings:** IoT sensors also publish over MQTT (`devices/{id}/readings`).
- **Device auth:** **hardened per-device API key** — unique key per device, stored
  **hashed** in Postgres, sent over **TLS (MQTTS)**, with **per-device topic ACLs**
  so one leaked key can't touch another device. Designed so mTLS is a later swap.
- **Broker:** **Mosquitto** added to Compose, using the `mosquitto-go-auth` plugin so
  the broker authenticates/authorizes **directly against the Postgres devices table**
  (single source of truth — no hand-edited password files).
- **Storage:** S3-compatible (MinIO local → Cloudflare R2 prod), images
  resized/compressed before upload for free-tier budget.
- **Layout:** standard `cmd/` + `internal/`.

## Target structure

```
backend/
  cmd/
    api/main.go              # entrypoint: config -> deps -> start HTTP + MQTT + shutdown
  internal/
    config/config.go         # env loading (godotenv), typed Config (DB/Redis/S3/MQTT/TLS)
    server/
      server.go              # gin engine, middleware, graceful shutdown
      router.go              # REST route registration
    database/
      postgres.go            # GORM connect (reuse DSN logic from current main.go)
      migrate.go             # AutoMigrate over domain models
    cache/redis.go           # redis client init (reuse from current main.go)
    storage/
      storage.go             # Storage interface: Put, Get, PresignGet, Delete
      minio.go               # MinIO/S3 impl (env: S3_ENDPOINT/S3_ACCESS_KEY/...)
    mqtt/
      client.go              # paho connect over TLS as the privileged backend client
      consumer.go            # subscribe devices/+/images, devices/+/readings
      publisher.go           # publish devices/{id}/results
      handlers.go            # image msg -> ingestion service; reading msg -> repo
    detection/
      detector.go            # Detector interface: Detect(ctx, img) (Result, error)
      onnx.go                # ONNX runtime impl (loads model, pre/post-process)
      noop.go                # stub impl so the app builds/boots before a model exists
    domain/                  # GORM models (structs + enums)
      device.go batch.go image.go detection.go reading.go
    repository/              # data access (Postgres)
      device_repo.go image_repo.go detection_repo.go reading_repo.go
    service/                 # business logic
      ingestion_service.go   # image -> resize -> store -> detect -> persist -> result
      device_service.go      # provision device, generate+hash API key, revoke/rotate
      detection_service.go
    handler/                 # HTTP handlers (gin) — management/query API
      health.go device.go detection.go reading.go
    middleware/
      logger.go recovery.go requestid.go admin_auth.go
  pkg/
    imageutil/resize.go      # decode, resize (x/image/draw), re-encode JPEG q~80
  go.mod go.sum Dockerfile .dockerignore
```

## Domain model (initial)

- **Device** — `id, device_id (unique), name, location, kind (camera|gas|temp|humidity),
  key_hash, status (active|revoked), last_seen_at`. `key_hash` = bcrypt of the per-device
  API key; the raw key is shown **once** at provisioning and never stored.
- **Batch** — fruit lot: `id, fruit_type, batch_code, storage_location, created_at`.
- **Image** — `id, batch_id?, device_id, object_key, bucket, content_type, size_bytes,
  width, height, captured_at`.
- **DetectionResult** — the per-image verdict across all three tasks:
  `id, image_id, ripeness (unripe|ripe|overripe|spoiled), ripeness_confidence,
  defective (bool), defect_confidence, size, size_unit, model_version, created_at`.
- **SensorReading** — `id, device_id, batch_id?, metric (ethylene|temperature|humidity),
  value, unit, recorded_at`.

Ripeness/metric are typed enums. `migrate.go` runs `AutoMigrate` on all models.

## Device auth & MQTT security (the hardening)

- **Transport:** Mosquitto TLS listener on **8883**; devices and backend connect with
  MQTTS. A local self-signed CA + server cert is generated for dev (dev-only).
- **Authentication:** `mosquitto-go-auth` plugin points at Postgres. On connect the
  broker looks up the device by username (= `device_id`), verifies the API key against
  `key_hash`, and rejects `status = 'revoked'`.
- **Authorization (ACLs):** pattern-based, so a device is auto-scoped to its own topics:
  - publish → `devices/%u/images`, `devices/%u/readings`
  - subscribe → `devices/%u/results`

  A stolen key for `dev1` therefore cannot read/publish any other device's topics.
- **Backend** connects as a privileged account allowed to subscribe `devices/+/images`
  and `devices/+/readings` and publish `devices/+/results`.
- **Provisioning:** `device_service` creates a device, generates a high-entropy key,
  stores only its hash, returns the raw key once. Revoke/rotate = update the row (broker
  picks it up on next connect). Migration path to **mTLS** is a swap of the plugin's
  auth backend, not a code rewrite.

## Key flows

- **Image round-trip (MQTT):** device publishes JPEG bytes to `devices/{id}/images`
  → `mqtt.consumer` → `ingestion_service`: `imageutil.resize` → `storage.Put`
  → `detection.Detect` → persist Image + DetectionResult → `mqtt.publisher` sends the
  result JSON to `devices/{id}/results`.
- **Sensor readings (MQTT):** `devices/{id}/readings` → parsed → SensorReading row.
- **Management REST API** (behind a simple admin token for now): register/list devices
  (returns the one-time key), list detection results, list readings, `/healthz`.

## Reuse from existing code

- `backend/main.go` — its GORM DSN builder, Redis client setup, and `/healthz` ping
  logic move into `internal/database`, `internal/cache`, `internal/handler/health.go`.
  Root `main.go` is deleted; `cmd/api/main.go` becomes the entrypoint.
- Compose env vars already defined (`DB_*`, `DATABASE_URL`, `REDIS_ADDR`, `S3_*`) are
  the contract `internal/config` reads. New MQTT/TLS vars get added alongside them.

## docker-compose changes

- Add **mosquitto** service (`iegomez/mosquitto-go-auth` image): config file, TLS
  certs, Postgres-backed auth/ACL, ports `8883` (TLS) and `1883` (local/dev only),
  `depends_on: postgres`.
- Add a `mosquitto/` dir under repo root for `mosquitto.conf`, ACL patterns, dev certs.
- Backend gains `MQTT_BROKER_URL`, `MQTT_USERNAME`/`MQTT_PASSWORD` (privileged account),
  `MQTT_TLS_CA` and `depends_on: mosquitto`.

## Dependencies to add

- **Storage:** `github.com/minio/minio-go/v7` (S3 client; MinIO + R2).
- **Detection:** `github.com/yalue/onnxruntime_go` (loads ONNX Runtime shared lib).
- **Image resize:** `golang.org/x/image/draw`.
- **Password hashing:** `golang.org/x/crypto/bcrypt` (x/crypto already present).
- Run `go mod tidy` (also promotes current `// indirect` deps to direct).

## Dockerfile change for ONNX (flagged, lands with the first real model)

ONNX Runtime's shared lib is glibc-based, so when the real model arrives the **runtime**
base switches `alpine` → `debian:bookworm-slim`, COPY in `libonnxruntime.so`, and set
`ONNXRUNTIME_LIB_PATH` (CGO can stay off — the binding dlopens the lib). Until then
`cmd/api` wires `detection.noop` so the stack builds and boots today.

## Verification

1. `cd backend && go mod tidy && go build ./...` — compiles with the new layout.
2. `docker compose up -d --build` — postgres, redis, minio, **mosquitto**, backend healthy.
3. `curl localhost:8080/healthz` → postgres/redis/storage/mqtt all `up`.
4. Register a device via the admin REST endpoint → receive a one-time API key.
5. Using an MQTT client (e.g. `mosquitto_pub` over TLS) as that device, publish a test
   JPEG to `devices/{id}/images`; confirm: object appears in MinIO console
   (`localhost:9003`, bucket `fruit-images`), an Image + DetectionResult row exist, and a
   result message arrives on `devices/{id}/results`.
6. Publish JSON to `devices/{id}/readings` → confirm a SensorReading row.
7. Negative test: with `dev1`'s key, try to publish to `devices/dev2/images` → broker
   denies (ACL enforcement).

## Out of scope (later)

Full user auth/JWT for the admin API, the real ONNX model + Dockerfile runtime swap,
WebSocket live dashboard (gorilla/websocket is present), rate limiting, and CI.
