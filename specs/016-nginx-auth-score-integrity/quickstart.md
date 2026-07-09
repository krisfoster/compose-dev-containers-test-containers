# Quickstart: Validation Guide

**Feature**: 016-nginx-auth-score-integrity  
**Date**: 2026-07-09

This guide describes how to confirm the feature works end-to-end after implementation. Run
each scenario against the compose stack (`docker compose up --build`).

---

## Prerequisites

- `docker compose up --build` completes successfully with no service restarts.
- No `LEADERBOARD_API_SECRET` variable is set in the environment or `.env` file (confirm the
  compose stack starts and operates without it).
- The QR window is active: open `http://localhost/host` in a browser to auto-activate a window.

---

## Scenario 1: Score submission rejected without QR grant

**Goal**: Verify FR-001 — requests without a `cw_grant` cookie are rejected before reaching the
Go app.

```bash
# Send a score submission with no cookies
curl -s -o /dev/null -w "%{http_code}" \
  -X POST http://localhost/api/leaderboard/scores \
  -H "Content-Type: application/json" \
  -d '{"name":"attacker","score":9999}'
```

**Expected**: `401`

**Also verify** (negative test — token header alone is not sufficient):
```bash
curl -s -o /dev/null -w "%{http_code}" \
  -X POST http://localhost/api/leaderboard/scores \
  -H "Content-Type: application/json" \
  -H "X-Leaderboard-Token: dev-only-change-me" \
  -d '{"name":"attacker","score":9999}'
```

**Expected**: `401` (the token header is no longer recognised; only the cookie matters)

---

## Scenario 2: Score submission succeeds with a valid QR grant

**Goal**: Verify FR-002 — requests carrying a valid `cw_grant` cookie are forwarded and accepted.

**Step 1**: Obtain a grant cookie using one of these methods:
- **Local shortcut (fastest)**: visit `http://localhost/play-local` in the browser. The Go handler mints a fresh `cw_grant` cookie over plain HTTP and serves the game page immediately — no QR scan needed.
- **QR path**: visit `/play?w=<active-window-id>` in the browser. The gate middleware validates the window ID and redirects to `/play` with the cookie set.

Copy the `cw_grant` cookie value from the browser's DevTools (Application → Cookies).

**Step 2**:
```bash
# Replace <cookie-value> with the actual cw_grant cookie value from the browser
curl -s -w "\n%{http_code}" \
  -X POST http://localhost/api/leaderboard/scores \
  -H "Content-Type: application/json" \
  -H "Cookie: cw_grant=<cookie-value>" \
  -d '{"name":"ValidPlayer","score":42}'
```

**Expected**: `201` with body `{"recorded":true}`

---

## Scenario 3: No credential in page source

**Goal**: Verify FR-003 — the game page contains no embedded leaderboard secret.

```bash
# Fetch the game page (as a QR-granted visitor would, or via the ungated listener)
curl -s http://localhost/play | grep -i "leaderboard_token\|X-Leaderboard-Token"
```

**Expected**: No output (zero matches).

**Also check globals** — open the game page in a browser (`http://localhost/play` after scanning
the QR code), open DevTools → Console, and run:
```js
window.__LEADERBOARD_TOKEN__
```

**Expected**: `undefined`

---

## Scenario 4: `/auth/check` is not directly accessible

**Goal**: Verify FR-007 — the auth check endpoint is internal-only.

```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost/auth/check
```

**Expected**: `404` (nginx's `internal;` directive causes nginx to return 404 for direct external
requests to an internal location).

---

## Scenario 5: No `LEADERBOARD_API_SECRET` in compose environment

**Goal**: Verify FR-008 — the removed configuration is gone.

```bash
docker compose config | grep -i "LEADERBOARD_API_SECRET"
```

**Expected**: No output.

---

## Scenario 6: App starts on a single port

**Goal**: Verify FR-009 — only port 8080 is in use; no second listener on 8081.

```bash
# Check that only port 8080 is published for the app service
docker compose port app 8080
docker compose port app 8081
```

**Expected**:
- `docker compose port app 8080`: prints `0.0.0.0:8080` (or configured `WEB_PORT`)
- `docker compose port app 8081`: error or empty (port not published)

**Also verify via logs**:
```bash
docker compose logs app | grep "listener starting"
```

**Expected**: One line — `ungated listener starting on :8080`. No second listener line.

---

## Scenario 7: Play path still gates correctly

**Goal**: Verify that `/play` still requires a QR grant (regression test for the gate).

```bash
# Request /play with no cookie — should return forbidden
curl -s -o /dev/null -w "%{http_code}" http://localhost/play
```

**Expected**: `403` (gate middleware rejects the unauthenticated request before serving the page).

---

## Scenario 8: Legitimate game session end-to-end

**Goal**: Constitution Principle IV — observed working in a browser.

1. Open `http://localhost/host` to activate the QR window.
2. Either scan the QR code on a phone, or use the local shortcut: visit `http://localhost/play-local`
   in the browser (mints a grant cookie without a QR scan — convenient for local validation).
3. Verify the game page loads.
4. Play the game and complete a round.
5. Verify the score appears on `http://localhost/leaderboard`.
6. Open the game page's source — confirm `__LEADERBOARD_TOKEN__` does not appear anywhere.

---

## Log Verification

After running scenarios 1–2, check the Go app logs to confirm the auth sub-request reached
`/auth/check`:

```bash
docker compose logs app | grep "auth/check"
```

Go's default `net/http` server does not log requests unless explicitly added. If no log output
appears, that is acceptable — the nginx log is the authoritative record.

Check nginx access log for auth sub-request entries:
```bash
docker compose logs nginx | grep "auth/check"
```

You should see `GET /auth/check` entries with `200` (for the authorised score submission) and
no entries for the rejected ones (nginx never makes the sub-request when the original request
has no cookie — wait, nginx always makes the sub-request and gets 401 back; the 401 is in the
sub-request log).

Correct: for the rejected scenario 1, you will see both:
- `GET /auth/check` → `401` (sub-request)
- `POST /api/leaderboard/scores` → `401` (returned to client)
