# Research: Fix Host QR Rotate Route

## No open unknowns

The specification maps directly to existing patterns in the codebase. No external research was
required. The decisions below are documented for traceability.

---

## Decision 1: Route placement — `ungatedMux` vs. a new mux

**Decision**: Register `/host/rotate` in the existing `ungatedMux()` in `app/main.go`.

**Rationale**: The leaderboard page (which triggers the rotation) is itself served by
`ungatedMux` with no `cw_grant` cookie requirement. Consistent placement avoids a two-mux design
for what is logically a single server. All other host-control routes (`/qr.png`,
`/leaderboard`, `/auth/check`, `/api/ping`) live on this same mux.

**Alternatives considered**:
- A separate authenticated endpoint behind `gate.Middleware`: rejected — the leaderboard page
  already sends the request without credentials. Adding auth here would break the feature in a
  different way, and the spec explicitly notes the endpoint is at the same trust level as the
  leaderboard page itself.

---

## Decision 2: HTTP response code on success — 204 vs. 200

**Decision**: Return `204 No Content`.

**Rationale**: The leaderboard JavaScript already branches on `resp.ok` to refresh the QR image.
204 is the conventional response for a mutation that produces no body. The existing
`handleAuthCheck` and other POST-like operations in the app follow this pattern.

**Alternatives considered**:
- `200 OK` with a JSON body containing the new window ID: unnecessary — the frontend only needs to
  know success/failure, not the window ID itself.

---

## Decision 3: Test double for `WindowStore` — `FakeWindowStore` vs. Testcontainers

**Decision**: Use `gatetest.FakeWindowStore` for the three handler unit tests.

**Rationale**: `handleHostRotate` is pure handler logic above the store boundary. The
`WindowStore` interface contract (behaviour of `Activate` against a real Redis) is already
verified by `app/internal/gate/window_test.go` using Testcontainers. Adding a second set of
Testcontainers tests for the same boundary from the handler level would add container-spin-up cost
without covering any new failure mode. Constitution Principle III specifically permits fakes
"for pure-logic tests" — this handler is exactly that.

**Alternatives considered**:
- Testcontainers test at the handler level: unnecessary and slower. The boundary is already covered.

---

## Decision 4: Error store for the "store failure" test

**Decision**: Reuse the existing `erroringStore` / `appWithErroringStore` helpers already defined
in `main_test.go`.

**Rationale**: `erroringStore` (an `Activate`-always-fails `WindowStore`) is already used for
`TestHandleQRPNGWhenStoreErrors` and is exactly what the handler's error branch needs. No new
test double is required.

**Alternatives considered**:
- A new `activateOnlyFailsStore` that allows `Current` but fails `Activate`: unnecessary, because
  `handleHostRotate` only calls `Activate`, not `Current`. `erroringStore` fails both, but only
  `Activate` is invoked by this handler.
