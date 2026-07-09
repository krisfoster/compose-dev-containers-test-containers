# Data Model: Git Commits Microservice

Derived from the spec's Key Entities and resolved against the implementation decisions in
`research.md`. The commits service reads from the mounted `.git` directory; it stores nothing
persistently.

## CommitEntry (new — read from git, returned in responses)

A single git commit surfaced by the commits service. Derived at request time from the on-disk
`.git` directory; never stored in Redis or any other database.

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| `hash` | string | 7 characters, hex | Short-form SHA-1 hash of the commit object. |
| `author` | string | Non-empty, at most 64 characters (display only) | The commit author's display name from the git object. |
| `date` | string | ISO 8601 UTC, format `2006-01-02 15:04` | Commit author timestamp, rendered in UTC. |
| `message` | string | Subject line only (first line before any `\n`) | The commit message subject. Multi-line commit messages are truncated at the first newline. |

**Derivation rules**:
- Source: the git log starting from `HEAD`, traversed via `go-git`'s `repo.Log`.
- Limit: at most 20 commits, newest first (same limit as the existing `app` handler).
- Truncation: `author` is truncated to 64 characters to keep the leaderboard column readable
  at typical projected display sizes. `message` is the subject line only.
- If the git repository is unavailable or `HEAD` cannot be resolved, the service returns HTTP 503
  with an empty body (not a JSON error) — the React component treats any non-200 as "unavailable"
  and retains its last known state.

## CommitFeed (REST response shape)

The ordered collection of CommitEntry records returned by `GET /commits`.

| Field | Type | Notes |
|-------|------|-------|
| `commits` | `CommitEntry[]` | Array, newest-first. May be empty (`[]`) when the repo has no commits. |

A top-level wrapper object (rather than a bare array) is used so that future metadata fields
(e.g., `total`, `since`) can be added without a breaking schema change.

## SSE Event (push shape)

Each Server-Sent Event emitted on `GET /commits/stream` carries the full current commit feed as
its `data` field.

| SSE field | Value |
|-----------|-------|
| `event` | `commits` |
| `data` | JSON-encoded `CommitFeed` (same schema as the REST response) |

The commits service broadcasts the current feed on connection open, then re-broadcasts on a
30-second refresh cycle. Each broadcast emits one `event: commits\ndata: {...}\n\n` block. The
React component replaces its entire rendered list on each `commits` event — no partial-update or
diffing logic is needed at the component level.

## State transitions (leaderboard commits component)

```
Component mounts
      │
      ▼
SSE connection opened ──(EventSource not available)──► Polling mode (30 s setInterval)
      │
      │ "commits" event received
      ▼
commits.length > 0 ?
      │ yes                │ no
      ▼                    ▼
Render commit list   Render "No commits yet" message
      │
      │ next "commits" event received
      ▼
Re-render list from new data (full replace, no diff)

SSE error / permanent close ──► Switch to polling fallback
```

The component never shows an error state for transient disconnects — it retains its last-rendered
list until either a new event arrives or polling returns data. The "no commits" empty state only
appears when the server explicitly returns an empty commits array, not on connection failure.
