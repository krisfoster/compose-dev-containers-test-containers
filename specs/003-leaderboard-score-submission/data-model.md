# Data Model: Player Name Entry, Game Over Score Display, and Leaderboard Score Submission

Derived from the spec's Key Entities, resolved against the storage decisions in `research.md`.

## Leaderboard Entry

Represents one completed game attempt's result. Backed by one entry in the `leaderboard:scores`
Redis Stream.

| Field | Type | Notes |
|-------|------|-------|
| `id` | Redis Stream entry ID (auto-assigned by `XADD`) | Not player-facing; exists purely as the Stream's own ordering key. |
| `name` | string, 1-32 chars after trim | Player-entered display name (FR-001, FR-002, FR-003). Not unique — multiple entries, including from the same name, coexist independently (FR-008). |
| `score` | non-negative integer | The score achieved in that specific attempt (FR-004, FR-006). |

**Validation rules**:
- `name` MUST be non-empty after trimming leading/trailing whitespace (FR-002) and MUST NOT exceed
  32 characters (FR-003).
- `score` MUST be a non-negative integer; missing, negative, or non-integer values are rejected
  with `400 Bad Request` and no entry is written.
- Every accepted submission produces exactly one new Stream entry via `XADD` (FR-007). No update or
  delete operation on an existing entry is part of this feature — the Stream is append-only.

**Relationships**: None. Entries do not reference each other, and this feature defines no
aggregate/ranked view over them (FR-015) — that is left to a future leaderboard-viewing feature,
which can read this same Stream.

## Score Submission Credential

Represents the shared secret that authorizes a write to the Leaderboard Entry stream. Not stored in
Redis — held as server configuration (an environment variable) and injected into the served game
page for the legitimate client to present back (see `research.md` §4).

| Field | Type | Notes |
|-------|------|-------|
| `value` | opaque string (server-configured) | Compared byte-for-byte (constant-time) against the value presented in each request's credential header. |

**Validation rules**:
- A `POST /api/leaderboard/scores` request MUST present the current credential value in its
  request header (see `contracts/leaderboard-openapi.yaml`) or the request is rejected with
  `401 Unauthorized` and no Leaderboard Entry is written (FR-012).
- An invalid (non-matching) credential is treated identically to a missing one — both are `401`,
  with no distinguishing information returned (consistent with this project's existing pattern of
  not leaking which failure mode occurred, per `002-qr-gated-access`'s gate error responses).

**Relationships**: None — this is a single, global value; this feature does not introduce
per-player or per-session credentials.

## State summary

```
Player loads the game
        │  no name yet
        ▼
Name prompt shown ── empty/whitespace submit ──┐
        │                                       │
        │ non-empty name entered                │ (stays on prompt)
        ▼                                       │
Gameplay in progress  <───────────────────────────┘
        │  player dies
        ▼
Game Over screen shown (own score displayed)
        │  score submission fires automatically
        ▼
Leaderboard Entry appended to `leaderboard:scores`  (best-effort; failure does not block below)
        │
        │ Replay activated
        ▼
Gameplay in progress  (same retained name, no prompt shown again)
```

A Leaderboard Entry, once appended, has no further lifecycle within this feature — there is no
edit, delete, or read-back path defined here (FR-015).
