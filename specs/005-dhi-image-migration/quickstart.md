# Quickstart: Validate the DHI Migration

Runnable validation that proves the hardened-image migration works end-to-end, with no behavioural
regression. Per constitution Principle IV, "done" is the game observed working **in a browser**
against the migrated compose stack — not just green tests/builds.

## Prerequisites

- Docker Desktop (or Docker Engine + Compose v2) and git — as before.
- **New**: a free Docker account and a login to the DHI registry:
  ```bash
  docker login dhi.io          # use your free Docker ID; pulling DHI Community images is free
  ```
  Without this, image pulls fail with an auth error (see Troubleshooting).
- No `.env` needed for the local (non-public) path.

## Scenario 1 — App image builds & the game plays in a browser (US1, FR-002/003/006, SC-003)

```bash
docker compose build app     # builds from dhi.io/golang:1.25-alpine-dev → dhi.io/static:*-alpine
docker compose up -d          # starts app + redis (both hardened)
docker compose ps             # app and redis are Up
```

Then in a browser:
1. Open <http://localhost:8080/> → the getting-started landing page renders.
2. Click **Play the game** (`/play`) → the Crossy Whale game loads and is playable.
3. Confirm the app process runs as **nonroot** in the hardened runtime:
   ```bash
   docker compose exec app id 2>/dev/null || \
     docker inspect --format '{{.Config.User}}' "$(docker compose ps -q app)"
   ```
   Expected: uid 65532 / `nonroot` (note: the `exec` shell may be absent — the `static` runtime has
   no shell; the `docker inspect` fallback confirms the configured user).

**Expected**: game visibly playable; app served by a nonroot, shell-less hardened image. No code or
behaviour change vs the pre-migration app.

## Scenario 2 — Redis on the hardened 8.x image, state intact (US2, FR-004, SC-003)

With the stack up from Scenario 1:
1. In the browser, enter a player name, play until Game Over → a score is recorded.
2. Open <http://localhost:8080/leaderboard> → the score appears and the board auto-refreshes.
3. Open <http://localhost:8080/host> → a QR code renders (exercises the Redis-backed gate window).

**Expected**: gameplay, leaderboard, and QR/gate all work against `dhi.io/redis:8-alpine` with no
visible difference from `redis:7-alpine`.

## Scenario 3 — Boundary tests pass against hardened Redis (US2, FR-005, SC-004)

```bash
cd app
go test ./...                 # Testcontainers spins up dhi.io/redis:8-alpine for boundary tests
```

**Expected**: all tests pass, including `internal/leaderboard` and `internal/gate` boundary tests,
which now launch the hardened Redis image via Testcontainers. No test is skipped, mocked, or
weakened. (If the module's readiness wait misbehaves, add `wait.ForLog("Ready to accept
connections")` to the test helper — see research.md Decision 4.)

## Scenario 4 — CVE reduction (US1/US2, SC-005)

Scan the migrated images and compare against the public originals:

```bash
docker scout cves "$(docker compose images -q app)"    # or: docker scout cves dhi.io/redis:8-alpine
docker scout cves redis:7-alpine                         # baseline for comparison
```

**Expected**: the migrated app and Redis images report **fewer** known CVEs than the public images
they replace (DHI tags were 0/0/0/0 at plan time).

## Scenario 5 — Inventory completeness & docs (US3, FR-008/011/012, SC-002/006/007)

```bash
# Every image reference in the repo is accounted for in the inventory:
grep -rnE 'image:|^FROM |tcredis\.Run' docker-compose.yml app/Dockerfile app/**/*_test.go
```

Cross-check each hit against [`contracts/image-inventory.md`](./contracts/image-inventory.md):
- 3 images migrated to `dhi.io/...`, 1 (ngrok) exempt with rationale.
- `project.md` image table matches the inventory.
- `README.md` documents the `docker login dhi.io` prerequisite and that ngrok is a demo-only
  exception; a new contributor following only the README reaches the playable demo (SC-007).

## Scenario 6 — Public profile still works, ngrok unchanged (exemption sanity)

```bash
cp .env.example .env && $EDITOR .env    # set NGROK_AUTHTOKEN=<free ngrok token>
docker compose --profile public up -d
open http://localhost:8080/host          # QR code renders; scanning reaches the gated game
```

**Expected**: the optional public tunnel still runs on the unchanged `ngrok/ngrok:3` image — the
exemption does not break the public demo.

## Teardown

```bash
docker compose --profile public down     # or: docker compose down
```
