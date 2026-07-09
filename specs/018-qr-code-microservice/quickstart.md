# Quickstart: QR Code Microservice

**Feature**: 018-qr-code-microservice
**Date**: 2026-07-09

This guide documents how to validate that the QR code microservice extraction works correctly,
end-to-end, against the running compose stack.

---

## Prerequisites

- Docker Desktop running
- Repo cloned and on branch `018-qr-code-microservice`
- No other process using ports 80, 8080, 8082, 8083, or 8084

---

## Build and Start

```bash
# From repo root
docker compose build
docker compose up
```

All services should start: `redis`, `app`, `commits-service`, `scores-service`, `qr-service`, `nginx`.

---

## Validation Scenarios

### Scenario 1: Dynamic QR code appears on `/host`

**What it validates**: SC-001 and the primary extraction (FR-004).

1. Open `http://localhost/host` in a browser.
2. A QR code image should appear on the page.
3. The QR code is rendered by `qr-service` — you can confirm with:
   ```bash
   docker compose logs qr-service
   # Should show a request log line when /host loads
   ```
4. Scan the QR code with a phone. It should open `https://<ngrok-host>/play?w=<windowID>` — or,
   without ngrok, the browser will show a "no active window" or connection error (expected).

**Expected result**: QR code image visible at `/host`; compose logs show qr-service handled a request.

---

### Scenario 2: Dynamic QR code PNG is a valid image

**What it validates**: FR-001, FR-004.

```bash
# Activate a window first (visit /host once in the browser), then:
curl -s -o /tmp/qr.png http://localhost/qr.png
file /tmp/qr.png
# Expected: /tmp/qr.png: PNG image data, ...
```

**Expected result**: `file` reports a valid PNG.

---

### Scenario 3: Repo QR code is still served

**What it validates**: FR-005, User Story 2.

```bash
curl -s -o /tmp/repo-qr.png http://localhost/repo-qr.png
file /tmp/repo-qr.png
# Expected: PNG image data
```

To decode and verify the URL:
```bash
# If zbarimg is available:
zbarimg /tmp/repo-qr.png
# Expected: QR-Code:https://github.com/krisfoster/compose-dev-containers-test-containers
```

**Expected result**: Valid PNG returned; decodes to the GitHub repo URL.

---

### Scenario 4: qr-service accessible directly (developer access)

**What it validates**: User Story 3 (FR-001, FR-002).

```bash
# Valid request
curl -s -o /tmp/direct.png \
  "http://localhost:8084/qr.png?content=https%3A%2F%2Fexample.com&size=256"
file /tmp/direct.png
# Expected: PNG image data

# Missing content → 400
curl -sv "http://localhost:8084/qr.png" 2>&1 | grep "< HTTP"
# Expected: < HTTP/1.1 400 Bad Request

# Size defaults when absent
curl -s -o /tmp/default-size.png "http://localhost:8084/qr.png?content=https%3A%2F%2Fexample.com"
file /tmp/default-size.png
# Expected: PNG image data (320×320 default)
```

---

### Scenario 5: `app` module no longer imports go-qrcode

**What it validates**: FR-006, SC-003.

```bash
grep "skip2/go-qrcode" app/go.mod
# Expected: no output (import removed)

grep -r "skip2/go-qrcode" app/
# Expected: no output (no remaining imports in app source)
```

---

### Scenario 6: All app tests still pass

**What it validates**: SC-005, FR-004, FR-005.

```bash
cd app && go test ./...
# Expected: all tests pass; no compilation errors
```

The `handleQRPNG` and `handleRepoQRPNG` tests will use a local `httptest.NewServer` stub instead
of the in-process qrcode call — see `app/main_test.go`.

---

### Scenario 7: qr-service tests pass

**What it validates**: FR-009.

```bash
cd qr-service && go test ./...
# Expected: all tests pass
```

---

## Failure Modes

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| `/host` shows broken image | qr-service not running or crashed | `docker compose logs qr-service` |
| `/qr.png` returns 503 | qr-service unreachable from app container | Check `depends_on` in docker-compose.yml |
| `app/go.mod` still references go-qrcode | `go mod tidy` not run after removing import | `cd app && go mod tidy` |
| `qr-service` build fails | Missing go.sum or wrong module path | `cd qr-service && go mod tidy` |

---

## Reference

- Contract: [contracts/qr-http-contract.md](contracts/qr-http-contract.md)
- Data model: [data-model.md](data-model.md)
- Spec: [spec.md](spec.md)
