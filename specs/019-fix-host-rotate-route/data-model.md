# Data Model: Fix Host QR Rotate Route

## No new entities

This feature introduces no new data entities. It reuses the existing **Join Window** entity
managed by `gate.WindowStore`.

---

## Existing entity: Join Window (unchanged)

| Attribute | Description |
|-----------|-------------|
| ID | Opaque, unguessable, URL-safe random token (16 bytes, base64url-encoded) |
| Activation time | Set by `Activate`; implicit — not stored separately |
| TTL | Configured at app startup via `qrWindowTTL`; enforced by the store |
| Cardinality | At most one active window at any time |

**State transitions** (unchanged):

```
[no window] --Activate()--> [active]
[active]    --Activate()--> [active]   (new ID, previous window superseded immediately)
[active]    --TTL lapses--> [no window]
```

`handleHostRotate` triggers the `[active] → [active]` (or `[no window] → [active]`) transition
by calling `store.Activate(ctx, a.qrWindowTTL)`. No other state is read or written by this handler.

## Validation rules (unchanged)

- Only POST is accepted; any other method → 405 Method Not Allowed with `Allow: POST` header.
- Store errors → 500 Internal Server Error; the caller must not refresh the QR image.
