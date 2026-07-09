# Research: QR Code Microservice

**Feature**: 018-qr-code-microservice
**Date**: 2026-07-09

No NEEDS CLARIFICATION markers were present in the spec. This document records the design
decisions that were derived from examining the existing codebase.

---

## Decision 1: Service interface — what parameters does qr-service accept?

**Decision**: `GET /qr.png?content=<url-encoded-string>&size=<integer-pixels>`

**Rationale**:
- The current in-process API is `RenderPNG(content string, size int) ([]byte, error)`.
  Translating this directly to a query-string API requires no design leap.
- `content` is the URL to encode (already built by `BuildPlayURL` in the app).
- `size` is the approximate pixel dimension (current callers use 320).
- Using query parameters rather than a request body keeps the interface cacheable in principle
  (even though the dynamic QR code is always `no-store`) and is consistent with the kind of
  one-shot image-generation APIs common in internal tooling.

**Alternatives considered**:
- POST with JSON body: unnecessary complexity for a side-effect-free, read-only operation.
- Path-based encoding (`/qr/<base64url-content>.png`): overly clever; breaks standard proxy caching
  headers and makes the URL opaque.

---

## Decision 2: Does `BuildPlayURL` move to qr-service or stay in the app?

**Decision**: `BuildPlayURL` stays in the app module (`app/internal/qrcode/qrcode.go` or inlined
into `main.go`).

**Rationale**:
- `BuildPlayURL` encodes the app's routing contract: the `/play?w=<windowID>` URL shape is an
  application-level concern owned by the app service, not by a rendering utility.
- The app service is the one that knows about window IDs and public ngrok hostnames; it is the
  correct place to assemble the URL.
- Moving it to qr-service would couple the rendering service to the app's routing model, making it
  harder to reuse qr-service for arbitrary content (e.g., `handleRepoQRPNG` encodes the GitHub URL,
  not a play URL).

**Alternatives considered**:
- Move `BuildPlayURL` to qr-service and pass `host` + `windowID` as separate params: creates an
  unnecessary coupling between a URL-building convention and the rendering service.

---

## Decision 3: How does the app call qr-service, and what happens on failure?

**Decision**: The app makes a synchronous `GET` to `http://qr-service:8084/qr.png?content=<url>&size=320`,
reads the response body, and proxies the PNG bytes directly to the original caller. On any non-200
response or network error, the app returns HTTP 503 to the browser.

**Rationale**:
- Synchronous HTTP is the simplest client pattern; the QR PNG is needed immediately to serve the
  browser request.
- The app already has an `httpClient` field with a 3-second timeout used for ngrok discovery; the
  same timeout is appropriate for qr-service calls.
- 503 on failure is consistent with how the app already handles Redis unavailability and ngrok
  unreachability in `handleQRPNG`.

**Alternatives considered**:
- Cache the PNG in memory: unnecessary complexity; the PNG is cheap to regenerate and the window
  rotates regularly.
- Return a placeholder image on failure: doesn't help the presenter — they need the real QR code.

---

## Decision 4: Port assignment

**Decision**: Port **8084** for qr-service.

**Rationale**: Follows the ascending sequence: 8082 (commits-service), 8083 (scores-service), 8084 (qr-service).

---

## Decision 5: What happens to `app/internal/qrcode/qrcode.go` after extraction?

**Decision**: Remove `RenderPNG` from the file. If `BuildPlayURL` is the only remaining function,
either keep the file (it still provides a tested, named abstraction) or inline `BuildPlayURL` into
`app/main.go` — that choice is left to the implementor. Either way, the `go-qrcode` import is
removed from `app/go.mod` in the same step.

**Rationale**: The primary goal is to remove the rendering dependency from the app binary; where
`BuildPlayURL` lives is a secondary style concern.

---

## Decision 6: Does nginx need updating?

**Decision**: No nginx changes required for the initial extraction.

**Rationale**: `/qr.png` and `/repo-qr.png` continue to proxy to `app:8080` via nginx. The app
internally calls qr-service over the Compose network (`http://qr-service:8084`). From nginx's
perspective nothing changes. A future improvement could route `/qr.png` directly to qr-service
from nginx, but that is out of scope for this feature.

---

## Decision 7: Testing approach for qr-service

**Decision**: Unit tests in `qr-service/internal/qrcode/handler_test.go` using `httptest.NewRecorder`
and `httptest.NewRequest`. No Testcontainers needed (no external dependencies).
App-side handler tests (`handleQRPNG`, `handleRepoQRPNG`) are updated to stub qr-service with a
local `httptest.NewServer` that returns a fake PNG response.

**Rationale**: Constitution Principle III requires Testcontainers only for tests that cross an
external boundary (Redis, real HTTP client, filesystem beyond a temp dir). qr-service itself has
no such boundary — it is pure Go + CPU. The app-side stub satisfies "no real container needed"
because the app's handler logic is what's under test, not qr-service's rendering.
