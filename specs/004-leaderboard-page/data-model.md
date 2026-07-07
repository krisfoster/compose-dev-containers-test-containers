# Data Model: Leaderboard Display Page

Derived from the spec's Key Entities, resolved against the read approach decided in `research.md`.
This feature reads the same underlying data 003-leaderboard-score-submission's `data-model.md`
defines (`Leaderboard Entry`, backed by the `leaderboard:scores` Redis Stream) — it does not
redefine or change that storage. What's new here is the read-time view derived from it.

## Leaderboard Standing (new — derived, not stored)

A ranked row computed at read time from the existing Leaderboard Entry stream. Not persisted
anywhere; recomputed on every call to the read endpoint.

| Field | Type | Notes |
|-------|------|-------|
| `rank` | positive integer | Position within the returned top-N list, 1-based. Computed from sort order, not stored. |
| `name` | string | Copied from the underlying Leaderboard Entry's `name`. |
| `score` | non-negative integer | Copied from the underlying Leaderboard Entry's `score`. |

**Derivation rules**:
- Source entries are read from the same `leaderboard:scores` Stream 003 writes to, via a full
  `XRange` read (see `research.md` §1).
- Sort order: `score` descending; ties broken by the source Stream entry's ID descending (more
  recently written entry ranks first), satisfying the spec's tie-break edge case.
- The returned list is truncated to a bounded top-N (default 10; see `contracts/` for the exact
  request/response shape and limit bounds) — FR-004.
- If the Stream has zero entries, the derived list is empty (FR-008's empty state is a display-layer
  concern, not a data-model one — an empty list is a perfectly valid response).

**Relationships**: Many Leaderboard Standings are derived from one Leaderboard Entry stream on each
read; a Standing has no independent identity or lifecycle of its own — it exists only for the
duration of one read response and is recomputed fresh next time, so it can never go stale in the
data model itself (only the client's last-rendered copy can, which is FR-007's concern).

## State summary

```
Leaderboard page opened
        │
        ▼
Loading state shown ──(GET /api/leaderboard/scores fails)──► loading state retained, retry on
        │                                                      next poll interval
        │ succeeds
        ▼
Ranked standings rendered (rank, name, score — top N)
        │
        │ poll interval elapses
        ▼
GET /api/leaderboard/scores ──(fails)──► last rendered standings retained unchanged (FR-007),
        │                                  retry on next interval
        │ succeeds
        ▼
Standings re-rendered from the fresh response  ──► (loop back to "poll interval elapses")
```

No entity introduced by this feature has create/update/delete operations of its own — it is
strictly a read/derive/display feature over data 003 already owns writing.
