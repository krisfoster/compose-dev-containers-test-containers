# Quickstart: Validate Fix Host QR Rotate Route

## Prerequisites

- Docker Desktop running
- Repo cloned; working directory is the repo root

## Unit test validation (fast, no containers)

```bash
cd app
go test ./... -run TestHandleHostRotate -v
```

**Expected**: Three tests pass — success (204), store error (500), method not allowed (405).

## Integration validation (compose stack)

Start the full stack:

```bash
docker compose up --build -d
```

Wait for services to be healthy:

```bash
docker compose ps
```

### Scenario 1: Manual rotation via curl

```bash
curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost/host/rotate
```

**Expected output**: `204`

### Scenario 2: Manual rotation via leaderboard UI

1. Open `http://localhost/leaderboard` in a browser.
2. Note the QR code displayed on screen.
3. Click "Refresh QR."
4. **Expected**: the QR code image updates to a new, distinct code within 2 seconds.
   If the QR image is unchanged after clicking, the fix is not working.

### Scenario 3: Auto-rotation timer

1. Keep the leaderboard page open (from Scenario 2).
2. Wait 60 seconds.
3. **Expected**: the QR code updates automatically without any manual action.
4. Wait another 60 seconds.
5. **Expected**: the QR code updates again (second cycle).

### Scenario 4: Wrong method is rejected

```bash
curl -s -o /dev/null -w "%{http_code}" -X GET http://localhost/host/rotate
```

**Expected output**: `405`

## Definition of done (Constitution Principle IV)

The feature is complete when Scenario 2 has been observed in a browser: clicking "Refresh QR"
produces a visibly updated QR code. Screenshot or screen recording confirming this is the
required sign-off evidence.

## Teardown

```bash
docker compose down
```
