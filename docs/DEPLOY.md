# Deployment (VPS, build-on-server)

The backend is deployed by GitHub Actions over SSH: after the CI workflow
passes on a push to `main`, the deploy workflow logs into the VPS, checks out
the tested commit, and rebuilds the stack
with `docker compose up -d --build`. The VPS builds the image itself — no
container registry is involved.

See [`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml).

## 1. One-time VPS setup

Install Docker Engine + the Compose plugin, then clone the repo once. The
deploy keeps building from this checkout.

```bash
# As the deploy user (must be able to run docker without sudo):
#   sudo usermod -aG docker "$USER"   # then re-login
git clone https://github.com/Mr-coo/VleeFruit.git /opt/vleefruit
cd /opt/vleefruit
```

Provide the two things that are intentionally **not** in git (both survive the
`git reset --hard` the deploy runs, because they are untracked / ignored):

```bash
# a) The ONNX model the backend serves.
mkdir -p backend/models
#   copy your model to backend/models/model.onnx  (MODEL_PATH default)

# b) Runtime secrets / overrides. Compose reads .env automatically.
cp .env.example .env
#   then edit .env — at minimum set GEMINI_API_KEY=...
```

Verify it comes up by hand before wiring CI:

```bash
docker compose up -d --build
docker compose ps
curl -fsS http://localhost:8080/health   # adjust to the real health route
```

## 2. GitHub repository secrets

Settings → Secrets and variables → Actions → **New repository secret**:

| Secret        | Value                                                        |
|---------------|-------------------------------------------------------------|
| `VPS_HOST`    | server hostname or IP                                       |
| `VPS_USER`    | SSH user from step 1 (in the `docker` group)                |
| `VPS_SSH_KEY` | **private** key whose public half is in the user's `authorized_keys` |
| `VPS_APP_DIR` | absolute checkout path, e.g. `/opt/vleefruit`               |
| `VPS_PORT`    | optional, SSH port (defaults to `22`)                       |

Generate a dedicated deploy key (don't reuse a personal key):

```bash
ssh-keygen -t ed25519 -f deploy_key -N "" -C "github-actions-deploy"
# append deploy_key.pub to the VPS user's ~/.ssh/authorized_keys
# paste the contents of deploy_key (the private file) into VPS_SSH_KEY
```

## 3. How a deploy runs

1. You merge a PR into `main` (or click **Run workflow** on the Deploy action).
2. CI runs. If it fails, nothing is deployed.
3. On CI success, Actions SSHes in, `git reset --hard <tested sha>`, `docker compose up -d --build`.
4. Only changed layers rebuild; the backend container is swapped and MQTT stays up.

## Notes

- **Ports:** the compose publishes `8080` (HTTP API) and `1883` (MQTT). Put a
  reverse proxy (Caddy/nginx) in front for TLS on the API if it's public.
- **Services:** only `backend` + `mqtt` run. Postgres/Redis/MinIO were removed —
  the current app doesn't use them. Re-add them here if that changes.
- **CI gate:** `ci.yml` builds, vets and tests the Go module and verifies the
  Docker image builds on every push/PR. The deploy is triggered by CI
  succeeding on `main` (`workflow_run`), so a red CI blocks the deploy.
