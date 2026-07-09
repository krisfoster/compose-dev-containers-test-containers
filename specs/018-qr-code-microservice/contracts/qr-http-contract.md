# HTTP Contract: qr-service

**Version**: 1.0
**Feature**: 018-qr-code-microservice
**Service**: `qr-service` (internal Compose network: `qr-service:8084`)
**Published port** (direct developer access): `localhost:8084`

---

## Endpoints

### `GET /qr.png`

Renders a QR code encoding `content` as a PNG image.

#### Request

| Component   | Details                                                   |
|-------------|-----------------------------------------------------------|
| Method      | `GET`                                                     |
| Path        | `/qr.png`                                                 |
| Query params | See below                                                |
| Headers     | None required                                             |
| Body        | None                                                      |

**Query parameters**:

| Parameter | Type    | Required | Default | Constraints       | Description                       |
|-----------|---------|----------|---------|-------------------|-----------------------------------|
| `content` | string  | Yes      | —       | Non-empty         | URL or text to encode in the QR   |
| `size`    | integer | No       | `320`   | Clamped [64,1024] | Approximate PNG dimension (pixels) |

#### Response: Success

```
HTTP/1.1 200 OK
Content-Type: image/png

<PNG bytes>
```

The PNG encodes `content` using QR error-correction level Medium. The actual image dimensions
may vary slightly from `size` due to QR module alignment; `size` is the requested dimension
passed to the library.

#### Response: Bad Request (missing content)

```
HTTP/1.1 400 Bad Request
Content-Type: text/plain; charset=utf-8

content parameter required
```

Returned when `content` query parameter is absent or empty.

#### Response: Internal Error (render failure)

```
HTTP/1.1 500 Internal Server Error
Content-Type: text/plain; charset=utf-8

failed to render QR code
```

Returned if the QR library returns an error (e.g., content too long for the chosen size).

---

## Caller Contracts

### `app` → `qr-service` (dynamic QR code)

Called from `handleQRPNG` when a window is active and the public host is known.

```
GET http://qr-service:8084/qr.png?content=https%3A%2F%2F<ngrok-host>%2Fplay%3Fw%3D<windowID>&size=320
```

- `content` is the output of `qrcode.BuildPlayURL(publicHost, windowID)`, URL-encoded.
- The app sets `Cache-Control: no-store` on its own response; qr-service does not set cache headers.
- On any non-200 response or connection error, app returns HTTP 503 to the browser.

### `app` → `qr-service` (static repo QR code)

Called from `handleRepoQRPNG` on every request (repo URL never changes).

```
GET http://qr-service:8084/qr.png?content=https%3A%2F%2Fgithub.com%2Fkrisfoster%2Fcompose-dev-containers-test-containers&size=320
```

- The app sets `Cache-Control: public, max-age=86400` on its own response.
- On any non-200 response or connection error, app returns HTTP 503 to the browser.

---

## Health / Readiness

qr-service exposes no `/health` endpoint. Docker Compose uses `service_started` (not a
healthcheck) as the dependency condition from `app`. A first-request failure will surface
as HTTP 503 from the app handler, which already handles the case gracefully.

---

## Notes

- qr-service has no authentication; it is reachable only on the internal Compose network
  (not published behind nginx). The published port 8084 is for developer access only.
- nginx does not proxy to qr-service directly; all browser traffic for `/qr.png` and
  `/repo-qr.png` continues to route through `app:8080`, which internally calls qr-service.
