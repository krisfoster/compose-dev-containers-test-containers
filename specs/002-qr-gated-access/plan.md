# Implementation Plan: QR-Gated Public Access to Crossy Whale

**Branch**: `002-qr-gated-access` | **Date**: 2026-07-06 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-qr-gated-access/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add a Go backend (`app/`) that serves `frontend/game/` on two listeners: an ungated one for the
presenter's local access, and a gated one that `ngrok` points at. The gated listener only serves
the game to a visitor holding a valid grant — either an existing signed grant cookie, or a request
carrying the current QR window token, in which case a grant is minted on the spot. Redis holds a
single current-window pointer (with TTL, for automatic expiry, and overwritable, for manual
rotation); grants themselves are stateless signed cookies carrying a unique ID for future
leaderboard attribution. Introducing this Go+Redis backend also retires the interim `webserver`
(nginx) service from `001-host-webapp-ngrok`, per that feature's documented sunset condition.

## Technical Context

**Language/Version**: Go (1.22+; the sandbox/dev toolchain has 1.26 available).

**Primary Dependencies**: Go standard library `net/http` (two listeners, static file serving via
`http.FileServer`); `github.com/redis/go-redis/v9` (Redis client); a small MIT-licensed Go QR
encoding library (e.g. `skip2/go-qrcode`) for `/qr.png`; `github.com/testcontainers/testcontainers-go`
plus its Redis module for boundary tests (constitution Principle III).

**Storage**: Redis — one string key, `access:window:current`, holding the active window ID with a
TTL (see `data-model.md`). No other persistent storage; grants are stateless (signed cookies, not
stored server-side).

**Testing**: Target ≥80% statement coverage across all `app/` Go code, measured via
`go test ./... -cover`. Redis-touching window logic (activate/rotate/expiry) is tested against a
real Redis via Testcontainers-go — no mocks, per constitution Principle III. Everything above that
boundary (grant cookie signing/verification, gate middleware decisions, HTTP handlers for
`/qr.png`, `/host`, `/host/rotate`) is pure logic once Redis access is isolated behind a small
`WindowStore` interface, so it is covered by standard unit/handler tests against an in-memory fake
implementation of that interface — fast, no container needed, and still consistent with Principle
III since the interface's real (Redis) implementation itself is exhaustively covered by
Testcontainers-go tests. The camera-scan flow itself is validated manually per `quickstart.md` and
constitution Principle IV, and is the one piece of this feature coverage tooling cannot reach.

**Target Platform**: Presenter's laptop (macOS/Linux/Windows) running Docker Desktop; Linux
containers only — same target as `001-host-webapp-ngrok`.

**Project Type**: Web service — first Go backend in this repository, added alongside the existing
static frontend it now serves directly.

**Performance Goals**: Scan-to-playable under 10 seconds (SC-001, dominated by network/ngrok, not
gate logic); rotation takes effect for new requests within a few seconds (SC-003 — a single Redis
`SET` is effectively instant).

**Constraints**: No host installs beyond Docker Desktop + git, unchanged from `001`. The app must
not depend on `ngrok` being healthy to serve local traffic (same resilience property `001`
established for the webserver, now owned by `app`'s ungated listener). `redis` and `app` are both
local containers started together, so this doesn't reintroduce a public-network dependency for
local access.

**Scale/Scope**: Booth-demo scale — at most a few dozen concurrent grants, one Redis key for
window state, one Go binary.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Status | Notes |
|-----------|------|--------|-------|
| I. Demo-First Delivery | Change must improve the demo or unblock it | PASS | The QR-scan-to-play flow is the documented core interaction attendees use to join at all (`crossy.md` player flow step 1-2); without a gate the public URL is an open door, which the feature request treats as unacceptable for a live booth. |
| II. Compose-Orchestrated Reproducibility | Every runtime component is a compose service; `docker compose up` reaches a demoable state with no extra host installs | PASS | Adds `app` (Go) and `redis` services to `docker-compose.yml`; `ngrok`'s `command` is repointed at `app`'s gated port instead of the retired `webserver`. Still one command, no new host-side installs. |
| III. Testcontainers Over Mocks (NON-NEGOTIABLE) | Go tests crossing a boundary use Testcontainers | PASS (binding on implementation) | Window activation/rotation/expiry logic reads/writes Redis and MUST be tested against a real Redis via Testcontainers-go; cookie signing/verification is pure logic and may use standard unit tests. Tracked for `/speckit-tasks`. |
| IV. Visible-in-the-Browser Definition of Done | Done = observed in a browser against the compose stack, with a documented repeatable path | PASS | `quickstart.md` documents six scenarios covering scan-to-play, blocked access, rotation, concurrency, and expiry, run against the live compose stack. |
| V. Vendored-Code Hygiene | Vendored code/assets carry attribution | N/A | The QR-encoding library is a standard Go module dependency tracked via `go.mod`/`go.sum`, not copy-pasted code or a third-party asset vendored into the repo — consistent with how `001` treated the `nginx`/`ngrok` base images. No `ATTRIBUTION.md` entry required. |

**Interim Static Hosting carve-out**: This feature's introduction of a Go backend for
`frontend/game/` triggers that carve-out's own sunset clause (constitution v1.1.0,
`TODO(INTERIM-HOSTING-SUNSET)`) — the `webserver` (nginx) service is retired in this same phase, as
required. Once this feature ships, the carve-out itself is moot for this surface; removing it (or
narrowing it, if no other surface uses it) from the constitution is a follow-up PATCH amendment to
raise at ship time via `/speckit-analyze`, not something this plan modifies directly.

No unjustified violations. Complexity Tracking is not needed.

*Re-checked after Phase 1 design (`research.md`, `data-model.md`, `contracts/`, `quickstart.md`):
no additional services, dependencies, or vendored assets beyond those already listed above. Table
holds unchanged.*

## Project Structure

### Documentation (this feature)

```text
specs/002-qr-gated-access/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/
│   └── gate-http-contract.md   # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
docker-compose.yml          # Updated. `webserver` (nginx) removed; adds `app` and `redis`
                             # services; `ngrok`'s command now points at `app`'s gated port.
.env.example                 # Updated. Adds GRANT_COOKIE_SECRET and the window-TTL override;
                             # keeps existing WEB_PORT / NGROK_AUTHTOKEN.

app/                          # New Go module — the first backend service in this repo.
├── go.mod
├── go.sum
├── main.go                   # Wires up both listeners (ungated local, gated public).
├── main_test.go              # Handler tests (/qr.png, /host, /host/rotate) against the fake store.
└── internal/
    ├── gate/
    │   ├── window.go          # WindowStore interface + Redis-backed implementation.
    │   ├── window_test.go     # Testcontainers-go Redis tests (Principle III).
    │   ├── window_fake_test.go # In-memory fake WindowStore, for use by other tests in this package
    │   │                       # and by app/main_test.go.
    │   ├── grant.go            # Grant cookie payload + HMAC sign/verify.
    │   ├── grant_test.go       # Unit tests: sign/verify, tamper detection, expiry.
    │   ├── middleware.go       # Gate decision + reject-path response.
    │   └── middleware_test.go  # Tests against the fake WindowStore (no container needed).
    └── qrcode/                 # Wraps the QR-encoding library to render /qr.png.
        ├── qrcode.go
        └── qrcode_test.go      # Unit test: encoded URL content, valid PNG output.

frontend/game/                # Existing, unchanged content; now served by `app` via
                               # http.FileServer instead of nginx.

webserver/                    # Retired. nginx.conf removed along with the compose service.
```

**Structure Decision**: Single Go module (`app/`) alongside the existing static frontend —
Option 2's `backend/` + `frontend/` split isn't warranted at this scale (one small service, no
separate API consumers yet), and keeping the module at `app/` mirrors this repo's existing
top-level layout (`frontend/`, `webserver/`, now `app/`) rather than introducing a new nesting
convention. `frontend/game/` is consumed read-only, as it already was under `001`.

## Complexity Tracking

*No unjustified Constitution Check violations — table intentionally omitted. The one structural
change (retiring `webserver`) is required by the constitution's own carve-out sunset clause, not a
violation of it; see the Constitution Check section above.*
