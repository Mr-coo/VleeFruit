# VleeFruit — Usage Guide

VleeFruit is a fruit-inspection backend. An IoT device publishes a fruit image
over MQTT; the backend stores it, runs detection (**ripeness**, **defect**,
**fruit size**), and publishes the result back to the device. Sensors also
publish environmental readings (ethylene / temperature / humidity). A REST API
provides device management and read-only queries.

> **Detection is a static stub right now.** No ML model is wired in yet, so every
> image returns the same fixed verdict (`model_version: "dummy-static-v1"`). The
> plumbing is real; only the inference is fake. See [Detection output](#detection-output).

---

## 1. Run the stack

Requires Docker + Docker Compose.

```bash
# 1. Generate dev TLS certs for the MQTT broker (one time)
sh mosquitto/certs/gen-dev-certs.sh

# 2. Copy env defaults (optional — Compose has fallbacks)
cp .env.example .env

# 3. Build and start everything
docker compose up -d --build

# 4. Confirm the backend is healthy
curl http://localhost:8080/healthz
```

`/healthz` returns `200` with `{"status":"ok","postgres":"up","redis":"up","storage":"up"}`
when dependencies are reachable, otherwise `503` with the failing component.

### Ports

| Service | URL / Port | Notes |
|---|---|---|
| Backend REST API | `http://localhost:8080` | Management + query API |
| MQTT (TLS) | `localhost:8883` | Device-facing (MQTTS) |
| MQTT (plain) | `localhost:1883` | Local/dev tooling |
| Adminer (Postgres UI) | `http://localhost:8081` | Inspect the database |
| redis-commander (Redis UI) | `http://localhost:8082` | Inspect Redis |
| MinIO console | `http://localhost:9001` | Object storage (`minioadmin`/`minioadmin`) |

---

## 2. Authentication

- **REST admin API** (`/api/v1/*`) is guarded by a static bearer token
  (`ADMIN_TOKEN`, default `dev-admin-token`):
  `Authorization: Bearer <ADMIN_TOKEN>`
- **Devices** authenticate to the MQTT broker with their `device_id` as username
  and their per-device **API key** as password. The broker enforces per-device
  topic ACLs — a device can only touch its own `devices/<its-id>/*` topics.

`/healthz` is the only public endpoint.

---

## 3. Provision a device (REST)

Creates a device and returns its API key **once** — store it immediately.

```bash
curl -X POST http://localhost:8080/api/v1/devices \
  -H "Authorization: Bearer dev-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"device_id":"dev1","name":"Cam 1","location":"cold room","kind":"camera"}'
```

Request fields: `device_id` (required; letters/digits/`.`/`_`/`-`, ≤64 chars),
`name`, `location`, `kind` (one of `camera`, `gas`, `temperature`, `humidity`).

Response (`201`):

```json
{
  "device": { "id": 1, "device_id": "dev1", "kind": "camera", "status": "active", "...": "..." },
  "api_key": "b3f1...e9"   // shown once — never returned again
}
```

### Other device endpoints

| Method & path | Purpose |
|---|---|
| `GET /api/v1/devices` | List devices (no key material) |
| `POST /api/v1/devices/{deviceID}/revoke` | Disable a device (broker rejects it on next connect) |
| `POST /api/v1/devices/{deviceID}/rotate-key` | Issue a fresh API key (returned once) |

---

## 4. Send data over MQTT

Install a client (no admin needed): `npm install -g mqtt` (MQTT.js CLI). The
broker requires auth on **both** ports, so always pass `-u <device_id> -P <api_key>`.

### Publish an image → get a detection result

Subscribe to the result topic first, then publish the image:

```bash
# terminal A — watch results
mqtt sub -h localhost -p 1883 -u dev1 -P "<api_key>" -t "devices/dev1/results" -v

# terminal B — send a JPEG/PNG
mqtt pub -h localhost -p 1883 -u dev1 -P "<api_key>" -t "devices/dev1/images" -f sample.jpg
```

Over TLS instead, use port 8883 with the dev CA:
`-h localhost -p 8883 --ca-file mosquitto/certs/ca.crt`.

### Publish a sensor reading

```bash
mqtt pub -h localhost -p 1883 -u dev1 -P "<api_key>" -t "devices/dev1/readings" \
  -m '{"metric":"temperature","value":4.5,"unit":"C"}'
```

Reading payload (JSON):

| Field | Type | Notes |
|---|---|---|
| `metric` | string | one of `ethylene`, `temperature`, `humidity` (required) |
| `value` | number | finite number (required) |
| `unit` | string | ≤16 chars |
| `batch_id` | number | optional — associate with a batch |
| `recorded_at` | string | optional RFC3339 timestamp; defaults to now |

### Topic map

| Topic | Direction | Payload |
|---|---|---|
| `devices/{id}/images` | device → backend | raw image bytes (JPEG/PNG) |
| `devices/{id}/readings` | device → backend | JSON reading (above) |
| `devices/{id}/results` | backend → device | JSON detection result (below) |

---

## 5. Detection output

The result published to `devices/{id}/results` (and stored per image):

```json
{
  "id": 1,
  "image_id": 1,
  "ripeness": "ripe",
  "ripeness_confidence": 0.92,
  "defective": false,
  "defect_confidence": 0.97,
  "size": 75.0,
  "size_unit": "mm",
  "model_version": "dummy-static-v1",
  "created_at": "2026-09-18T10:00:00Z"
}
```

- `ripeness` ∈ `unripe`, `ripe`, `overripe`, `spoiled`
- `defective` is a boolean; `size` is a numeric estimate in `size_unit`
- `model_version: "dummy-static-v1"` marks stub output. When the real ONNX model
  is added, only `main.go`'s detector wiring changes.

---

## 6. Query stored data (REST)

All read-only, behind the admin bearer token. `?limit=N` defaults to `50`, capped
at `500`.

```bash
curl http://localhost:8080/api/v1/detections -H "Authorization: Bearer dev-admin-token"
curl http://localhost:8080/api/v1/images?limit=100 -H "Authorization: Bearer dev-admin-token"
curl http://localhost:8080/api/v1/readings -H "Authorization: Bearer dev-admin-token"
```

You can also browse the tables directly in **Adminer** (`http://localhost:8081`,
system PostgreSQL, server `postgres`, user/password/db = `vleefruit`).

---

## 7. Configuration (env vars)

Set in `.env` or the environment. Key ones (see `.env.example` for the full list):

| Variable | Default | Purpose |
|---|---|---|
| `APP_ENV` | `development` | `development` = text logs; `production` = JSON logs + Gin release mode |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `ADMIN_TOKEN` | `dev-admin-token` | Bearer token for `/api/v1/*` |
| `RATE_LIMIT_RPS` | `10` | Per-IP HTTP request rate (`0` disables) |
| `RATE_LIMIT_BURST` | `20` | Burst allowance above the rate |
| `HTTP_MAX_BODY_BYTES` | `1048576` | Max REST request body (1 MiB) |
| `MQTT_MAX_IMAGE_BYTES` | `8388608` | Max image payload over MQTT (8 MiB) |

Requests over the rate limit get `429`; bodies/images over the size limits are
rejected.

---

## 8. Security notes (dev defaults)

The stack ships with **development defaults** — change these before any real
deployment:

- Default credentials everywhere (`ADMIN_TOKEN`, `minioadmin`, Postgres
  `vleefruit`, MQTT `backend-dev-secret`, no Redis password).
- The plaintext MQTT `1883` listener and the Adminer / redis-commander / MinIO
  UIs are exposed on host ports — dev conveniences, not for production.
- The MQTT TLS certs under `mosquitto/certs/` are self-signed and dev-only.
```
