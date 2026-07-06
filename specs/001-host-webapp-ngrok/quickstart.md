# Quickstart: Host Web App with Public Ngrok Access

Validates the feature end-to-end against the running compose stack, per constitution Principle IV
(Visible-in-the-Browser Definition of Done). Run every scenario in a browser; do not report done
from passing config checks alone.

## Prerequisites

- Docker Desktop (or equivalent Docker Compose v2 runtime) installed — no other host installs.
- A free ngrok account and its authtoken, only needed for the public-access scenarios below.
- Copy `.env.example` to `.env` and set `NGROK_AUTHTOKEN` if testing public access.

## Scenario 1 — Local hosting (US1, FR-001, FR-002, SC-001)

```bash
docker compose up -d
```

1. Within 2 minutes of a cold start, open `http://localhost:8080` in a browser.
2. Confirm the web app (Crossy Road) loads and is playable.
3. `docker compose down && docker compose up -d` — confirm it comes back up the same way with no
   config changes.

## Scenario 2 — Public hosting (US2, FR-003, FR-004, FR-005, SC-002, SC-003)

```bash
docker compose --profile public up -d
```

1. Open `http://localhost:4040` (ngrok's own agent UI) and copy the `https://*.ngrok-free.app` (or
   equivalent) forwarding URL shown there.
2. From a device on a different network (e.g. a phone on cellular data, not the same wifi), open
   that URL.
3. Confirm the same app loads within 5 seconds and behaves identically to the local view.
4. Leave it running; recheck the same URL after a couple of hours to confirm it is still serving
   without any manual step in between.

## Scenario 3 — Resilience when the tunnel is unavailable (US3, FR-006)

```bash
docker compose stop ngrok
```

1. With the `ngrok` container stopped (or `NGROK_AUTHTOKEN` unset before startup), confirm
   `http://localhost:8080` still loads locally.
2. Confirm `http://localhost:4040` is not reachable — this is the "public sharing is currently
   unavailable" signal for the presenter (per Edge Cases in spec.md).
3. `docker compose start ngrok` (or re-run with the profile) and confirm a public URL becomes
   available again without touching the webserver.

## Scenario 4 — Port conflict (Edge Case, FR-008)

```bash
# Occupy the port first to simulate a conflict:
docker run -d --rm -p 8080:80 nginx:alpine
docker compose up
```

1. Confirm `docker compose up` fails fast with Compose's "port is already allocated" message
   rather than starting silently on an unexpected port.
2. Stop the conflicting container, or set a different `WEB_PORT` in `.env`, and retry.

## Cleanup

```bash
docker compose --profile public down
```
