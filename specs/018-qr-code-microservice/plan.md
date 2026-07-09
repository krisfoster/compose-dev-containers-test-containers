# Implementation Plan: QR Code Microservice

**Branch**: `018-qr-code-microservice` | **Date**: 2026-07-09 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/018-qr-code-microservice/spec.md`

## Summary

Extract the PNG rendering capability from `app/internal/qrcode/qrcode.go` into a new standalone
`qr-service` microservice. The service exposes a single HTTP endpoint
`GET /qr.png?content=<url>&size=<pixels>` and wraps `github.com/skip2/go-qrcode` — no new
runtime dependencies are introduced. The `app` service is updated to call `qr-service` over HTTP
for both the dynamic QR code (`/qr.png`) and the static repo QR code (`/repo-qr.png`); it
continues to own window-ID lookup (Redis) and ngrok public-host discovery. `BuildPlayURL` stays in
the app module since it encodes app-level routing knowledge. `RenderPNG` is removed from the app
module once the delegation is in place.

## Technical Context

**Language/Version**: Go 1.25 (consistent with app, commits-service, scores-service)

**Primary Dependencies**: `github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e` (same version currently in `app/go.mod`); Go standard library `net/http`, `strconv`

**Storage**: N/A — qr-service is a pure rendering service with no persistent state

**Testing**: Go standard testing (`go test ./...`); `net/http/httptest` for HTTP handler tests; no Testcontainers needed (no external boundary dependencies)

**Target Platform**: Linux container (`dhi.io/static:20260611-alpine3.24` final image, same as commits-service)

**Project Type**: Microservice (HTTP, stateless, single-route)

**Performance Goals**: QR PNG generation is CPU-only and completes in under 5ms per request; no special performance targets beyond "fast enough that the presenter never notices"

**Constraints**: Must not introduce new compose service dependencies (qr-service has no Redis, no ngrok, no external network calls); statically compiled Go binary (`CGO_ENABLED=0`)

**Scale/Scope**: One service, one route, one internal package (`internal/qrcode/`). Total new code: ~100 lines of Go.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Demo-First Delivery | PASS | Presenter UX is unchanged; QR codes still appear on `/host`. No visible regression. |
| II. Compose-Orchestrated Reproducibility | PASS | `qr-service` is added as a service in `docker-compose.yml`; `docker compose up` continues to reach a demoable state. No host-side installs added. |
| III. Testcontainers Over Mocks for Boundary Tests | PASS | qr-service has no external boundary (no Redis, no HTTP client calls out). Handler tests use `httptest`. App-side tests for `handleQRPNG`/`handleRepoQRPNG` will stub qr-service via `httptest.NewServer`. |
| IV. Visible-in-the-Browser Definition of Done | PASS | Feature is done when the presenter's `/host` page displays a scannable QR code against the running compose stack. quickstart.md captures the exact validation path. |
| V. Vendored-Code Hygiene | PASS | `go-qrcode` is already in `app/go.mod` with an existing `ATTRIBUTION.md` entry (if present) or one is added; no new third-party assets beyond a Go module already in use. |
| Technology Stack: Backend Go | PASS | qr-service is a Go module; no new language or runtime introduced. |
| Technology Stack: no new runtime dependency | PASS | `github.com/skip2/go-qrcode` is already used by the app; adding it to the new `qr-service/go.mod` is not a new runtime dependency for the stack. No constitution amendment needed. |

**Post-design re-check**: No violations found. `qr-service` is structurally identical to
`commits-service` (single Go module, single internal package, single HTTP handler, same DHI base
images). The constitution's Technology Stack and Principle II are fully satisfied.

## Project Structure

### Documentation (this feature)

```text
specs/018-qr-code-microservice/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── qr-http-contract.md
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
qr-service/              # NEW — mirrors commits-service/ structure
├── Dockerfile           # dhi.io/golang:1.25-alpine-dev → dhi.io/static
├── go.mod               # module crossywhale/qr-service; requires go-qrcode
├── go.sum
├── main.go              # HTTP server on :8084; one route: GET /qr.png
└── internal/
    └── qrcode/
        ├── handler.go   # http.Handler: parses content+size, calls render, writes PNG
        └── handler_test.go

app/                     # MODIFIED
├── main.go              # handleQRPNG and handleRepoQRPNG call qr-service over HTTP
├── internal/
│   └── qrcode/
│       ├── qrcode.go    # RenderPNG REMOVED; BuildPlayURL stays
│       └── qrcode_test.go  # TestRenderPNG* tests removed

docker-compose.yml       # MODIFIED — new qr-service block added
```

**Structure Decision**: Single new top-level `qr-service/` directory following the established
`commits-service/` and `scores-service/` pattern. Each microservice is an independent Go module
with its own `go.mod`/`go.sum` and a minimal `internal/` package.

## Complexity Tracking

> No constitution violations to justify. Section intentionally empty.
