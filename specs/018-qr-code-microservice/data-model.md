# Data Model: QR Code Microservice

**Feature**: 018-qr-code-microservice
**Date**: 2026-07-09

qr-service is a stateless rendering service. It has no persistent state, no database, and no
shared data store. This document describes the in-flight data that flows through the service.

---

## Entities

### QR Render Request

Represents an inbound request to generate a QR code PNG.

| Field     | Type    | Source            | Constraints                                      |
|-----------|---------|-------------------|--------------------------------------------------|
| `content` | string  | query param       | Required; non-empty; the URL to encode in the QR |
| `size`    | integer | query param       | Optional; default 320; clamped to [64, 1024]     |

**Validation rules**:
- If `content` is absent or empty → HTTP 400 `content parameter required`
- If `size` is absent, zero, or negative → use default of 320
- If `size` exceeds 1024 → clamp to 1024 (avoids excessively large PNG allocations)
- `content` is passed verbatim to the QR library; it is not validated as a URL (the service is
  a rendering primitive, not a URL validator)

### QR Render Response

| Field          | Value                                       |
|----------------|---------------------------------------------|
| HTTP status    | 200 on success; 400 on bad input; 500 on render failure |
| Content-Type   | `image/png`                                 |
| Cache-Control  | Set by the caller (app service), not by qr-service |
| Body           | PNG bytes encoding `content` as a QR code  |

### App → qr-service Call Context

The app service constructs this call for the dynamic QR code (`/qr.png`):

| Field     | Value                                                      |
|-----------|------------------------------------------------------------|
| `content` | `BuildPlayURL(publicHost, windowID)` — e.g. `https://abc123.ngrok-free.app/play?w=<id>` |
| `size`    | 320                                                        |

And for the static repo QR code (`/repo-qr.png`):

| Field     | Value                                                         |
|-----------|---------------------------------------------------------------|
| `content` | `https://github.com/krisfoster/compose-dev-containers-test-containers` |
| `size`    | 320                                                           |

---

## State Transitions

None. qr-service is fully stateless; every request is independent.

---

## Module Boundary Changes

### Removed from `app` module

| Symbol                  | File                              | Reason                          |
|-------------------------|-----------------------------------|---------------------------------|
| `qrcode.RenderPNG`      | `app/internal/qrcode/qrcode.go`  | Moved to qr-service             |
| `go-qrcode` import      | `app/go.mod`                      | No longer used by app           |

### Kept in `app` module

| Symbol                  | File                              | Reason                          |
|-------------------------|-----------------------------------|---------------------------------|
| `qrcode.BuildPlayURL`   | `app/internal/qrcode/qrcode.go`  | App-level URL routing knowledge |

### New in `qr-service` module

| Symbol                          | File                                    | Role                             |
|---------------------------------|-----------------------------------------|----------------------------------|
| `qrcode.Handler`                | `qr-service/internal/qrcode/handler.go` | http.Handler for GET /qr.png    |
| `RenderPNG` (unexported or pkg) | `qr-service/internal/qrcode/handler.go` | Wraps go-qrcode; called by Handler |
