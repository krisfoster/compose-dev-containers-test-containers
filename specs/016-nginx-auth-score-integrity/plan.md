# Implementation Plan: nginx auth_request for Score Integrity

**Branch**: `016-nginx-auth-score-integrity` | **Date**: 2026-07-09 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/016-nginx-auth-score-integrity/spec.md`

## Summary

Replace the browser-visible `LEADERBOARD_API_SECRET` token with nginx `auth_request` that
delegates cookie validation to a new Go `/auth/check` endpoint. Score submissions are blocked at
the nginx layer if the request does not carry a valid `cw_grant` cookie — no secret is sent to
the browser. The Go app is consolidated from two listeners (ungated port 8080 + gated port 8081)
to a single listener on port 8080, with `gate.Middleware` applied per-route for `/play`.

All cleanup (secret removal, dead listener, frontend token removal) ships in the same change.

## Technical Context

**Language/Version**: Go (module `crossywhale/app`)

**Primary Dependencies**:
- nginx (existing compose service — `ngx_http_auth_request_module` is built into standard nginx)
- Go `net/http` stdlib
- `crossywhale/app/internal/gate` package (existing — `Signer.Verify`, `Gate`, `Middleware`)
- `crossywhale/app/internal/leaderboard` package (existing — `Handler`, `ScoreStore`)

**Storage**: Redis (unchanged — no new keys, no schema changes)

**Testing**: Go standard `testing` package; `net/http/httptest` for handler-level tests. No new
Testcontainers tests required (the `/auth/check` endpoint only calls `gate.Signer.Verify()`, which
is a pure HMAC check with no Redis dependency — unit tests with a fake signer suffice. Existing
gate and leaderboard Testcontainers tests cover the Redis-crossing boundaries).

**Target Platform**: Linux container in Docker Compose (unchanged)

**Project Type**: Compose web-service stack

**Performance Goals**: Unchanged (booth demo). Each score submission gains one nginx internal
sub-request (`/auth/check`) before the Go handler runs. This adds negligible latency (<1ms
loopback).

**Constraints**:
- No new runtime dependencies (constitution § Technology Stack)
- All cleanup ships atomically — no half-migrated state where both auth paths co-exist

**Scale/Scope**: Eight files modified; one new handler; one new test file section. No new services.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### PASS — Core principles

| Principle | Assessment |
|-----------|------------|
| I. Demo-First Delivery | Auth improvement is invisible to the presenter; score integrity increases attendee trust. PASS. |
| II. Compose-Orchestrated Reproducibility | No new services. Compose stack starts cleanly without `LEADERBOARD_API_SECRET`. PASS. |
| III. Testcontainers Over Mocks for Boundary Tests | New `/auth/check` handler uses only `gate.Signer.Verify()` — pure HMAC, no Redis — unit testable. Existing gate and leaderboard Testcontainers tests remain unchanged. PASS. |
| IV. Visible-in-the-Browser Definition of Done | Feature must be validated in browser before ship (quickstart.md scenarios 7–8). PASS. |
| V. Vendored-Code Hygiene | No new vendored assets. PASS. |

### VIOLATION — "gated internal port" clause (PATCH amendment required before ship)

**Clause** (constitution § Technology Stack, Permanent Routing Layer carve-out):
> "Gate enforcement MUST remain in the Go app and is reached by proxying to the Go gated internal
> port."

**Status**: VIOLATION. This feature removes the gated internal port (8081). Gate enforcement for
score submission is now reached via nginx `auth_request` sub-request to port 8080 (`/auth/check`).
Gate enforcement for `/play` is now reached via `gate.Middleware` applied per-route in the single
mux on port 8080.

**Justification**: The spirit of the clause — gate enforcement remains in the Go app — is fully
preserved. The specific mechanism ("gated internal port") is superseded by the new model. The
two-listener approach was a structural workaround to let nginx route by port to a gate-enforcing
mux; `auth_request` achieves the same result more directly without the second listener.

**Required**: A PATCH-level constitution amendment updating the "gated internal port" language to
describe the post-016 model. This amendment is a prerequisite for ship, not for task execution.

## Project Structure

### Documentation (this feature)

```text
specs/016-nginx-auth-score-integrity/
├── handoff.md           # Research input (pre-existing)
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── auth-check-contract.md   # Phase 1 output
└── checklists/
    └── requirements.md
```

### Source Code (files modified by this feature)

```text
app/
├── main.go                            # Add handleAuthCheck; remove gatedMux, GatedPort,
│                                      # leaderboardSecret, token injection in handlePlayIndex;
│                                      # add gate.Middleware to /play on single mux
├── main_test.go                       # Remove token/gated-mux tests; add auth-check tests
└── internal/
    └── leaderboard/
        └── handler.go                 # Remove secret field, CredentialHeader,
                                       # validCredential(), credential check in serveSubmit

nginx/
└── nginx.conf                         # Add auth_request for /api/leaderboard/scores;
                                       # add internal /auth/check location;
                                       # route /play to app:8080 (was app:8081);
                                       # remove app:8081 references

docker-compose.yml                     # Remove LEADERBOARD_API_SECRET and APP_GATED_PORT

frontend/game/
├── index.html                         # Remove __LEADERBOARD_TOKEN__ script tag
└── script.js                          # Remove X-Leaderboard-Token header from score fetch
```

**Structure Decision**: Single-project layout, minimal scope. No new files beyond test changes.
The `leaderboardtest/fake_store.go` is unaffected (store/notifier interfaces are unchanged).

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| Constitution "gated internal port" clause | The gated port exists only to route to a gate-enforcing mux. `auth_request` achieves the same gate enforcement without a second port, making the architecture cleaner (single ingress entry point, single app port). | Keeping port 8081 would work but leaves dead infrastructure — the gated mux's only role was score-submission credential checking, now done by `auth_request`. The port itself carries no value once the check moves. |
| `/play` uses gate.Middleware (not auth_request) | nginx `auth_request` is incompatible with the QR grant issuance flow — a fresh QR scan (`?w=<windowID>`) carries no cookie, so `auth_request /auth/check` would return 401 before the grant is issued. | Redesigning grant issuance (separate endpoint, no middleware) would be a materially larger change than the feature intends. Gate middleware applied to `/play` on the single mux achieves identical security with minimal change. |

## Implementation Order

Tasks are ordered so each step leaves the codebase in a working state (no broken build mid-way):

1. **Go: Add `/auth/check` handler** (`app/main.go`)
   - New `handleAuthCheck` method on `*App`
   - Register on the single mux
   - Unit tests for 200/401/405 responses

2. **Go: Consolidate to single mux** (`app/main.go`)
   - Move `/play` with `gate.Middleware` into the previously-ungated mux (rename or refactor)
   - Remove `gatedMux()` method
   - Remove second `http.ListenAndServe` goroutine
   - Remove `GatedPort` from `Config` and `envOr("APP_GATED_PORT", ...)` from `loadConfig`
   - `App.gate` field is **retained** (used by `gate.Middleware` on `/play`)

3. **Go: Remove leaderboard token** (`app/main.go`, `app/internal/leaderboard/handler.go`)
   - Remove `LeaderboardAPISecret` from `Config`, `leaderboardSecret` from `App`
   - Remove `LeaderboardToken` data injection in `handlePlayIndex` (template becomes no-data)
   - Remove `secret` param from `leaderboard.NewHandler`; remove `Handler.secret`
   - Remove `CredentialHeader`, `validCredential()`, credential check in `serveSubmit`

4. **Frontend: Remove token** (`frontend/game/index.html`, `frontend/game/script.js`)
   - Remove `window.__LEADERBOARD_TOKEN__` script tag from `index.html`
   - Remove `X-Leaderboard-Token` header from score fetch in `script.js`

5. **nginx: Add auth_request** (`nginx/nginx.conf`)
   - Add `auth_request /auth/check;` to `location /api/leaderboard/scores`
   - Add `location = /auth/check { internal; proxy_pass ...; }` block
   - Change `location = /play` proxy target from `app:8081` to `app:8080`
   - Remove any remaining `app:8081` references

6. **Compose: Remove dead config** (`docker-compose.yml`)
   - Remove `LEADERBOARD_API_SECRET` env entry
   - Remove `APP_GATED_PORT` env entry

7. **Tests: Update** (`app/main_test.go`)
   - Remove tests that relied on `gatedMux()` or token injection (see research.md §8)
   - Update `newTestApp` and helper constants
   - Add `/auth/check` tests

8. **Constitution amendment** (`.specify/memory/constitution.md`)
   - PATCH amendment updating "gated internal port" clause
   - Version bump 1.4.0 → 1.4.1

### Build verification after each step

After steps 1–3: `cd app && go build ./... && go test ./...`  
After step 4: inspect `frontend/game/index.html` and `script.js` for removed references  
After steps 5–6: `docker compose build && docker compose up` — run quickstart scenarios  
After step 7: `cd app && go test ./...` (all tests pass)  
After step 8: `cat .specify/memory/constitution.md | grep -A3 "gated"` (clause updated)
