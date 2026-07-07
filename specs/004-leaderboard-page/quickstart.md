# Quickstart: Leaderboard Display Page

Validates the user stories in `spec.md` end-to-end against the running compose stack. Route
behavior referenced below is defined in `contracts/leaderboard-openapi.yaml`; the derived
standings view is defined in `data-model.md`.

## Prerequisites

- Docker Desktop (or equivalent) running.
- `redis-cli` available locally (or via `docker compose exec redis redis-cli`) to seed/inspect the
  `leaderboard:scores` stream directly, and to simulate an outage for Scenario 3.

## Start the stack

```bash
docker compose up
```

## Scenario 1 — Standings render, ranked highest first (User Story 1)

1. Record a few entries directly (fastest way to seed data without playing multiple rounds):
   ```bash
   curl -s -X POST http://localhost:8080/api/leaderboard/scores \
     -H 'Content-Type: application/json' \
     -H "X-Leaderboard-Token: ${LEADERBOARD_API_SECRET:-dev-only-change-me}" \
     -d '{"name":"kris","score":42}'
   curl -s -X POST http://localhost:8080/api/leaderboard/scores \
     -H 'Content-Type: application/json' \
     -H "X-Leaderboard-Token: ${LEADERBOARD_API_SECRET:-dev-only-change-me}" \
     -d '{"name":"mo","score":17}'
   ```
2. Open `http://localhost:8080/leaderboard`.
3. **Expect**: both entries appear, `kris` (42) ranked above `mo` (17).
4. On a fresh stack with no entries recorded yet, open `/leaderboard` first.
   **Expect**: a clear empty state, not an error or blank page.

## Scenario 2 — Standings refresh automatically, no manual reload (User Story 2)

1. With `/leaderboard` already open from Scenario 1, submit one more score in another terminal:
   ```bash
   curl -s -X POST http://localhost:8080/api/leaderboard/scores \
     -H 'Content-Type: application/json' \
     -H "X-Leaderboard-Token: ${LEADERBOARD_API_SECRET:-dev-only-change-me}" \
     -d '{"name":"whale-fan","score":99}'
   ```
2. **Expect**: without touching the browser, `whale-fan` appears at rank 1 within 10 seconds
   (SC-001).
3. Leave the page open and idle for a few minutes.
   **Expect**: it keeps polling and would reflect any further submissions with no restart needed
   (SC-002).

## Scenario 3 — Display survives a temporary Redis outage (User Story 3)

1. With `/leaderboard` open and showing standings, stop Redis:
   ```bash
   docker compose stop redis
   ```
2. **Expect**: the page keeps showing the last standings it successfully loaded — no error screen,
   no blank list (SC-003).
3. Restart Redis:
   ```bash
   docker compose start redis
   ```
4. **Expect**: on the next poll, the page resumes showing current (accurate) standings
   automatically, no manual reload needed.

## Read endpoint spot checks

```bash
curl -s http://localhost:8080/api/leaderboard/scores | jq
curl -s "http://localhost:8080/api/leaderboard/scores?limit=1" | jq   # only the top entry
curl -s -X POST http://localhost:8080/api/leaderboard/scores          # GET-only creds check:
```

**Expect**: `GET` never requires the `X-Leaderboard-Token` header (FR-013) and always returns a
bounded list (`limit` clamps rather than errors on out-of-range values).

## Automated coverage

Go tests cover the pieces above that don't require a browser: the new Redis `XRange`-based read
method against a real Redis via Testcontainers-go (constitution Principle III), and the `GET`
handler branch's ranking, tie-breaking, and limit-bounding as unit/handler tests against the
extended `leaderboardtest.FakeScoreStore`. The auto-refresh and stale-data-on-failure behavior
(Scenarios 2 and 3) are inherently browser interactions and are validated manually per this
quickstart, per constitution Principle IV.
