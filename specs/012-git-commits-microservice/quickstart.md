# Quickstart Validation Guide: Git Commits Microservice

This guide describes how to validate the feature end-to-end against the running compose stack.
It is not an implementation guide — see `tasks.md` for task breakdown.

## Prerequisites

- Docker Desktop running
- Repo cloned; `docker compose up` starts all services without errors
- `commits-service` appears in `docker compose ps` as `running`

## Validation scenarios

### Scenario 1: REST endpoint returns commits (SC-003, FR-003)

```bash
# Verify the commits service is up and returning structured data
curl -s http://localhost:8082/commits | jq .

# Expected: a JSON object with a "commits" array containing up to 20 entries.
# Each entry has "hash" (7 chars), "author", "date", "message" fields.
# Example:
# {
#   "commits": [
#     { "hash": "a2c6757", "author": "Kris Foster", "date": "2026-07-07 22:15",
#       "message": "Leaderboard redesign, ngrok fix, and mobile camera improvements" },
#     ...
#   ]
# }
```

### Scenario 2: CORS headers are present (FR-010, research.md §4)

```bash
curl -s -I http://localhost:8082/commits | grep -i access-control

# Expected output includes:
# Access-Control-Allow-Origin: *
```

### Scenario 3: SSE stream delivers events (FR-005)

```bash
# Open the SSE stream and watch for events (Ctrl-C to stop after seeing one event)
curl -s -N http://localhost:8082/commits/stream

# Expected: within 1 second, output resembling:
# event: commits
# data: {"commits":[{"hash":"a2c6757",...},...]}
#
# Then another block every 30 seconds.
```

### Scenario 4: Leaderboard shows commit feed in browser (SC-001, FR-004)

1. Open `http://localhost:8080/leaderboard` in a browser.
2. The "Recent commits" column on the right should display a list of commits.
3. Each entry shows a short hash, author name, date, and message subject line.
4. Verify the data matches `curl http://localhost:8082/commits`.

### Scenario 5: Empty-state message (FR-006, SC-002)

To simulate an empty commit repository:

```bash
# Temporarily point the commits service at an empty git repo:
# (This requires a one-time test setup — use the compose override below)

docker compose run --rm -e GIT_REPO_PATH=/dev/null commits-service
# Then open the leaderboard in the browser — the commits column should show:
# "No commits yet — be the first to commit!"
# (Exact wording defined in tasks.md)
```

Alternatively: create a bare git repo for testing:

```bash
mkdir /tmp/empty-repo && git init /tmp/empty-repo
GIT_REPO_PATH=/tmp/empty-repo curl -s http://localhost:8082/commits | jq .
# Expected: { "commits": [] }
```

Open the leaderboard in the browser with this service running — the commit column shows the
empty-state message instead of a list.

### Scenario 6: Live update appears within 5 seconds (SC-001)

1. With the leaderboard open in a browser (SSE connected), make a git commit in the repo:
   ```bash
   git commit --allow-empty -m "test: verify live commit feed"
   ```
2. Within 5 seconds (the SSE initial push + one 30 s cycle does not apply here — the first
   event after a commit arrives on the *next* 30 s broadcast), the commit hash appears in the
   leaderboard's commit column.
   
   > **Note**: The SSE stream broadcasts the full feed every 30 s. For the purposes of SC-001
   > ("within 5 seconds"), the initial connection delivers the current feed instantly; a brand-new
   > commit will appear within 30 s of the next broadcast cycle. SC-001 targets page-load latency
   > (the initial commit feed renders in under 5 s), not polling latency. The 30 s broadcast cycle
   > governs how quickly a *new* commit appears after it's made. If sub-5-s latency for new commits
   > is required, reduce the broadcast interval in `tasks.md`.

### Scenario 7: Resilience after service restart (SC-005)

1. With the leaderboard open and the commit feed displayed:
   ```bash
   docker compose restart commits-service
   ```
2. Within 15 seconds, the commit feed reappears (SSE reconnects automatically; polling fallback
   activates within one 30 s cycle if SSE is slow to reconnect).

### Scenario 8: Service not present in `app` (research.md §6)

```bash
# Verify the old /api/commits handler is gone from the app service
curl -s http://localhost:8080/api/commits

# Expected: HTTP 404 (route no longer registered on the app service mux)
```

## Validation checklist

After completing all scenarios above, confirm:

- [ ] `GET http://localhost:8082/commits` returns `{ "commits": [...] }` with correct fields
- [ ] CORS header `Access-Control-Allow-Origin: *` is present on all responses
- [ ] SSE stream at `GET http://localhost:8082/commits/stream` emits `event: commits` on connect
- [ ] Leaderboard shows commit list in browser without any network errors in DevTools console
- [ ] Empty-state message appears when `commits` array is empty
- [ ] New commit appears in the feed within one 30 s broadcast cycle
- [ ] After `docker compose restart commits-service`, feed recovers within 15 s
- [ ] `GET http://localhost:8080/api/commits` returns 404 (old handler removed)
- [ ] `docker compose up` from a fresh clone starts `commits-service` with no additional steps
- [ ] ATTRIBUTION.md has React 18 + ReactDOM 18 entries
- [ ] Constitution amendment for React is merged before ship
