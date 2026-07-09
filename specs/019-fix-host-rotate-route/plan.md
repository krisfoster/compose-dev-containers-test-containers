# Implementation Plan: Fix Host QR Rotate Route

**Branch**: `019-fix-host-rotate-route` | **Date**: 2026-07-09 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/019-fix-host-rotate-route/spec.md`

## Summary

Register a missing `/host/rotate` POST route in the Go app's ungated mux so the leaderboard page
can rotate the active QR join window on demand (manual button) and on a 60-second timer. The route
calls the existing `Activate` operation on the `WindowStore` and returns 204 on success or 500 on
store error. Three unit tests cover success, store error, and method-not-allowed.

## Technical Context

**Language/Version**: Go 1.25

**Primary Dependencies**: `net/http` (stdlib), `crossywhale/app/internal/gate.WindowStore` (existing interface), `crossywhale/app/internal/gate/gatetest.FakeWindowStore` (test double, existing)

**Storage**: Redis via `gate.RedisWindowStore` — no changes to the store layer; the `Activate`
method and the `WindowStore` interface already exist and are production-ready.

**Testing**: Go standard library `testing` + `net/http/httptest`. Unit tests use `gatetest.FakeWindowStore` (in-memory fake) for handler-level tests — consistent with every other handler test in `app/main_test.go`. No Testcontainers test is needed here: the new code path only exercises `WindowStore.Activate`, whose real-Redis contract is already covered by `gate/window_test.go`.

**Target Platform**: Linux container (Docker), served via nginx as front-door proxy.

**Project Type**: Web service (Go HTTP server, one of several microservices behind nginx).

**Performance Goals**: No change to existing request throughput. `/host/rotate` is called at most
once per 60 seconds from the leaderboard page; it is not on any hot path.

**Constraints**: The new route must be added to `ungatedMux()` — not to a separate mux — matching
where `/qr.png` and `/leaderboard` are registered today.

**Scale/Scope**: Two-function change: one `HandleFunc` registration in `ungatedMux` and one new
handler method on `App`. Three new test functions in `main_test.go`.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Assessment | Status |
|-----------|------------|--------|
| I. Demo-First Delivery | Fixes a silently broken leaderboard feature (QR rotation) that presenters rely on. High demo impact. | PASS |
| II. Compose-Orchestrated Reproducibility | No new services, no new compose entries. All changes are inside the existing `app` Go module. | PASS |
| III. Testcontainers for Boundary Tests | Handler tests use `FakeWindowStore` (same pattern as all existing handler tests). No new boundary is crossed — `WindowStore.Activate` boundary is already covered by `gate/window_test.go` with a real Redis container. | PASS |
| IV. Visible-in-the-Browser Definition of Done | quickstart.md documents the browser validation steps required before closing this feature. | PASS |
| V. Vendored-Code Hygiene | No third-party assets introduced. | PASS |
| Technology Stack | Pure Go stdlib change inside an existing module. No new dependency. No constitution amendment required. | PASS |

**Post-design re-check**: No design decision changed the assessment above.

## Project Structure

### Documentation (this feature)

```text
specs/019-fix-host-rotate-route/
├── plan.md              ← this file
├── research.md          ← Phase 0 output
├── data-model.md        ← Phase 1 output
├── quickstart.md        ← Phase 1 output
├── contracts/
│   └── host-rotate.md   ← Phase 1 output
└── tasks.md             ← /speckit-tasks output (not created here)
```

### Source Code (repository root)

```text
app/
├── main.go              ← add HandleFunc registration + handleHostRotate method
└── main_test.go         ← add TestHandleHostRotate* (3 test functions)
```

No other files change.

## Complexity Tracking

> No constitution violations — section omitted.
