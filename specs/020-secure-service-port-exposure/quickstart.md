# Quickstart & Validation Guide: Secure Microservice Port Exposure

**Feature**: 020-secure-service-port-exposure | **Date**: 2026-07-09

## Prerequisites

- Docker Desktop (or Docker Engine) running
- Repo cloned; working directory is repo root
- No services currently running (`docker compose down` if needed)

## Setup

```bash
docker compose up --build -d
```

Wait for all services to report healthy (or use `docker compose ps` to confirm all are `running`).

## Validation Scenarios

### Scenario 1 — Direct port connections refused (P1 & P2)

Confirm that host-to-microservice direct access is blocked:

```bash
# Should fail with "Connection refused" within ~1 second
curl -sv --max-time 3 http://localhost:8083/scores
curl -sv --max-time 3 http://localhost:8082/commits
curl -sv --max-time 3 http://localhost:8084/qr.png
```

**Expected**: All three return `curl: (7) Failed to connect to localhost port XXXX: Connection refused`

**Before the fix**: All three return HTTP responses (200 or similar).

---

### Scenario 2 — Score submission via nginx gate still works (P1)

Submit a score through the auth gate (requires a valid session; use the host/leaderboard page to obtain a `cw_grant` cookie, or test 401 rejection):

```bash
# Without a valid cw_grant cookie — should get 401
curl -sv -X POST http://localhost/api/leaderboard/scores \
  -H "Content-Type: application/json" \
  -d '{"player":"test","score":42}'
```

**Expected**: `HTTP/1.1 401`

```bash
# With a valid cw_grant cookie (obtain by visiting http://localhost and scanning the QR)
curl -sv -X POST http://localhost/api/leaderboard/scores \
  -H "Content-Type: application/json" \
  -b "cw_grant=<your-cookie-value>" \
  -d '{"player":"test","score":42}'
```

**Expected**: `HTTP/1.1 204` and score appears on leaderboard.

---

### Scenario 3 — Leaderboard commit feed works via nginx (P2)

```bash
curl -sv http://localhost/commits
```

**Expected**: JSON array of recent git commits (HTTP 200).

---

### Scenario 4 — QR code renders via nginx (P2)

Open a browser to `http://localhost` (the host leaderboard page). The QR code image should render. Alternatively:

```bash
# The app fetches QR internally; nginx serves /qr.png to browsers
curl -sv -o /dev/null -w "%{http_code}" http://localhost/qr.png
```

**Expected**: `200`

---

### Scenario 5 — Full demo flow (Principle IV)

1. Open `http://localhost` in a browser — leaderboard renders.
2. Scan the QR code with a mobile device — game loads at the ngrok URL.
3. Play a game — score appears on the leaderboard within seconds.
4. Confirm the commit feed updates when a new commit is pushed.

**Expected**: End-to-end demo flow works identically to pre-fix behaviour.

## Pass Criteria

| Check | Expected Result |
|-------|----------------|
| `curl localhost:8083` | Connection refused |
| `curl localhost:8082` | Connection refused |
| `curl localhost:8084` | Connection refused |
| `curl localhost/api/leaderboard/scores` (no cookie) | HTTP 401 |
| `curl localhost/commits` | HTTP 200 + JSON |
| `curl localhost/qr.png` | HTTP 200 + PNG |
| Browser: full demo flow | Leaderboard renders, QR works, scores update |

## Teardown

```bash
docker compose down
```
