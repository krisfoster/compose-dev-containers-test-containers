# Data Model: Leaderboard Scores Microservice

## Overview

The scores-service does not own any persistent data. It reads from the existing `leaderboard:scores` Redis Stream (written by `app`) and subscribes to the `leaderboard:score-updated` Redis pub/sub channel (published by `app` on each score write). All entities below are either derived from that stream or represent the wire shapes exposed by the microservice's API.

---

## Redis Data Sources (owned by `app`, read by `scores-service`)

### ScoreStreamEntry

Stored in the Redis Stream `leaderboard:scores` (append-only, written by `app` on each score submission).

| Field | Type | Constraints |
|-------|------|-------------|
| stream entry ID | Redis auto-generated (timestamp-sequence) | Unique per entry |
| `name` | string | Non-empty; max 32 UTF-8 characters (enforced by `app` on write) |
| `score` | string (integer encoded as string) | Non-negative integer |

Access pattern: `XRANGE leaderboard:scores - +` — full stream read, oldest-to-newest. At booth scale (≤ a few hundred entries per session), this is negligible overhead.

### ScoreChangeNotification

Redis pub/sub channel: `leaderboard:score-updated`

| Field | Value |
|-------|-------|
| Channel | `leaderboard:score-updated` |
| Publisher | `app` service, immediately after successful `XADD leaderboard:scores` |
| Payload | Empty string (presence of the message is the signal; content is unused) |

---

## In-Process Aggregations (computed by `scores-service`)

### PlayerBestScore

Derived by reading the full stream and grouping by `name`, retaining only the maximum `score` per player.

| Field | Type | Derivation |
|-------|------|------------|
| `name` | string | Key from stream entry `name` field |
| `bestScore` | integer | `max(score)` across all stream entries for this player |

**Aggregation rules**:
1. Read all entries from `leaderboard:scores` using `XRANGE - +`.
2. For each entry, parse `name` (string) and `score` (string → integer).
3. Build a map `name → max(score)`.
4. Convert the map to a slice, sort descending by `bestScore`.
5. Truncate to `SCORES_LIMIT` (environment variable, default `10`).
6. Assign 1-based ranks.

---

## API Wire Shapes

### Standing

One entry in the ranked standings response.

| Field | Type | Constraints |
|-------|------|------------|
| `rank` | integer | 1-based; sequential; no gaps |
| `name` | string | Player name as stored in Redis |
| `score` | integer | Player's best score |

### StandingsResponse

Top-level JSON envelope returned by `GET /scores` and carried in each SSE `standings` event.

| Field | Type | Constraints |
|-------|------|------------|
| `standings` | array of Standing | Ordered by `score` descending; may be empty (not null) |

**Empty standings**: when no scores exist in the stream, `standings` is `[]` (empty array). The React component renders the "no scores yet" empty-state message in this case.

---

## State Transitions

```
App writes score
      │
      ▼
Redis Stream (leaderboard:scores) ──────► scores-service reads full stream on each trigger
      │                                         │
      │                                         ▼
      └─► App publishes to                Aggregates best-per-player
          leaderboard:score-updated             │
                │                              ▼
                └─────────────────► scores-service pushes SSE to all
                                    connected clients
```

- The Redis Stream is append-only. There are no updates or deletions.
- Best-per-player aggregation is stateless and recomputed on each pub/sub trigger.
- The scores-service holds no persistent state of its own.

---

## Validation Rules

These rules are enforced by `app` on the write path and are documented here so the scores-service knows what it can safely assume about the data in the stream:

| Rule | Enforced by |
|------|-------------|
| `name` is non-empty after trimming whitespace | `app` serveSubmit validation |
| `name` is at most 32 UTF-8 characters | `app` serveSubmit validation |
| `score` is a non-negative integer | `app` serveSubmit validation |
| `score` is stored as a decimal string in the Redis Stream | `app` store.Write implementation |

The scores-service MUST handle malformed stream entries gracefully (log and skip) rather than returning an error, for resilience against any hypothetical future schema changes.

---
