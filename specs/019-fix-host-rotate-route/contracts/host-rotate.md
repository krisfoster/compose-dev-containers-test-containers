# Contract: POST /host/rotate

## Endpoint

`POST /host/rotate`

Served on the single public port (port 80 via nginx → port 8080 on the `app` container).
No authentication required (same trust level as the leaderboard page).

## Request

| Property | Value |
|----------|-------|
| Method | `POST` |
| Path | `/host/rotate` |
| Body | Empty (ignored) |
| Headers | None required |

## Responses

### 204 No Content — success

The active join window has been replaced with a new one. The caller should refresh the QR code
image to display the new window.

```
HTTP/1.1 204 No Content
```

No response body.

### 500 Internal Server Error — store failure

The window store could not be reached or failed to create a new window. The current window
(if any) is unchanged. The caller must not refresh the QR code image.

```
HTTP/1.1 500 Internal Server Error
Content-Type: text/plain; charset=utf-8

failed to rotate window
```

### 405 Method Not Allowed — wrong HTTP method

Any request method other than POST.

```
HTTP/1.1 405 Method Not Allowed
Allow: POST
```

No response body.

## Caller behaviour (leaderboard page)

The caller (leaderboard JavaScript) checks `resp.ok` after calling this endpoint:
- On `true` (204): reload the `/qr.png` image to display the new window.
- On `false` (500, 405, or any other non-2xx): do not reload the image; the current window remains.
