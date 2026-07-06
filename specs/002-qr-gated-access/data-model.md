# Data Model: QR-Gated Public Access to Crossy Whale

Derived from the spec's Key Entities, resolved against the storage decisions in `research.md`.

## QR Access Code

Represents the currently valid means of entry. Backed by one Redis key.

| Field | Type | Notes |
|-------|------|-------|
| `window_id` | opaque random string (value of `access:window:current`) | Generated fresh on first activation and on every rotation. Long enough to be unguessable (not sequential, not short). |
| `expires_at` | implicit, via Redis key TTL | Default validity period 15 minutes from creation/rotation; overridable via environment variable. No explicit field is stored — the key's own absence *is* expiry. |
| `status` | derived, not stored | "active" iff `access:window:current` exists and its value equals the token presented; "expired/rotated out" iff the key is absent or holds a different value. There is exactly one active code at a time (per spec Assumptions). |

**Validation rules**:
- A presented token is valid only if it exactly matches the live value of `access:window:current`
  at lookup time (FR-003, FR-009).
- Rotation (FR-007) is a single overwrite of the key with a new `window_id` and a fresh TTL; it
  never leaves the old value valid, even transiently.
- Automatic expiry (FR-006) requires no explicit transition — Redis TTL expiry is the mechanism.

**Relationships**: A QR Access Code, when successfully presented, is the origin of zero or more
Visitor Access Grants (FR-011 "issued_window_id"). It has no relationship back to those grants —
per FR-008, a grant survives its originating code's rotation or expiry.

## Visitor Access Grant

Represents a visitor's device having passed the gate. Not stored server-side; carried entirely in
a signed cookie on the visitor's browser (see `research.md` §4 for why).

| Field | Type | Notes |
|-------|------|-------|
| `grant_id` | UUIDv4 | The unique identifier required by FR-011. Generated once, at grant time; never regenerated for the same visit. This is the value a future leaderboard feature attributes a score to. |
| `issued_window_id` | string | The `window_id` that was valid at issuance. Recorded for traceability only; not re-validated afterward (FR-008). |
| `issued_at` | timestamp | When the grant was minted. Used to enforce the grant's own fixed lifetime, independent of the QR Access Code's TTL. |
| signature | HMAC-SHA256 over the above fields | Prevents a visitor from forging or altering `grant_id`/`issued_at` client-side. Verified on every gated request. |

**Validation rules**:
- A request to a gated route is authorized if it carries a cookie whose signature verifies against
  the server secret and whose `issued_at` is within the grant's fixed lifetime — regardless of
  whether `issued_window_id` is still the current QR Access Code (FR-005, FR-008).
- A request with no valid grant cookie is authorized only if it also carries a `w` query parameter
  matching the current QR Access Code, at which point a new grant is minted (see `research.md` §4).
- Requests with neither a valid grant nor a valid `w` token are rejected (FR-003, FR-009).

**Relationships**: Many Visitor Access Grants may exist concurrently and independently (FR-010);
none of them reference each other. Each references the QR Access Code it was issued from only by
value (`issued_window_id`), not by a live link.

## State summary

```
No QR Access Code exists
        │  presenter opens /host (first time) → generates window
        ▼
QR Access Code active (TTL running)
   │             │
   │ presenter   │ TTL lapses with
   │ rotates     │ no action
   ▼             ▼
New QR Access Code active   No QR Access Code (fail closed, FR-009)
   │                               │
   │ presenter can rotate again ───┘ (regenerates a new one)
```

A Visitor Access Grant, once minted, does not appear in this diagram — it is independent of the QR
Access Code's lifecycle from the moment it is issued (FR-008).
