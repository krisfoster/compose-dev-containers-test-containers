# Quickstart: Player Name Entry, Game Over Score Display, and Leaderboard Score Submission

Validates the user stories in `spec.md` end-to-end against the running compose stack. Route
behavior referenced below is defined in `contracts/leaderboard-openapi.yaml`; storage details are
in `data-model.md`.

## Prerequisites

- Docker Desktop (or equivalent) running.
- `redis-cli` available locally (or run it inside the `redis` container via
  `docker compose exec redis redis-cli`) to inspect the `leaderboard:scores` stream directly.

## Start the stack

```bash
docker compose up
```

(The `public`/ngrok profile is not required to validate this feature — the credential check applies
identically on both listeners, per `research.md` §2.)

## Scenario 1 — Name prompt gates the start of play (User Story 1)

1. Open `http://localhost:${WEB_PORT:-8080}/play`.
2. **Expect**: a name prompt appears before any gameplay is visible/possible.
3. Try to proceed with the field empty, then with only spaces.
4. **Expect**: the game does not start; the prompt remains.
5. Type a short name (e.g. `kris`) and confirm.
6. **Expect**: gameplay starts immediately (SC-001).

## Scenario 2 — Game Over shows only the player's own score (User Story 2)

1. From Scenario 1, play until death.
2. **Expect**: gameplay stops and a "Game Over" screen appears showing the score for that attempt,
   and nothing else (no other names/scores, no ranking) (SC-002).

## Scenario 3 — Score and name land in the leaderboard store (User Story 3)

1. Immediately after Scenario 2's Game Over screen appears, inspect the stream:
   ```bash
   redis-cli XREVRANGE leaderboard:scores + - COUNT 1
   ```
2. **Expect**: the most recent entry's `name` and `score` fields match what Scenario 2 displayed
   (SC-003).
3. Repeat Scenario 1 and 2 once more under a *different* name in a separate browser/incognito tab,
   without restarting the stack.
4. **Expect**: `XLEN leaderboard:scores` has increased by one for each completed attempt, and both
   entries are present with their distinct names — neither overwrote the other.
5. Play a second attempt under the *same* name as step 1 (see Scenario 4 for how, via Replay).
6. **Expect**: a third distinct entry appears under that same name; the first attempt's entry is
   still present unchanged.

## Scenario 4 — Replay keeps the name, starts immediately (User Story 4)

1. From the Game Over screen (Scenario 2), activate Replay.
2. **Expect**: a new attempt starts immediately, with no name prompt shown (SC-004).
3. Die again.
4. **Expect**: the new Game Over screen's score reflects only this new attempt, and the
   corresponding new leaderboard entry (per Scenario 3) is recorded under the same name as before.

## Scenario 5 — Direct API calls without the credential are rejected (User Story 5)

1. With the stack running, send a request that omits the credential header:
   ```bash
   curl -i -X POST http://localhost:8080/api/leaderboard/scores \
     -H 'Content-Type: application/json' \
     -d '{"name":"forger","score":9999}'
   ```
2. **Expect**: `401 Unauthorized`, and `XLEN leaderboard:scores` is unchanged (SC-005).
3. Repeat with a garbage credential header:
   ```bash
   curl -i -X POST http://localhost:8080/api/leaderboard/scores \
     -H 'Content-Type: application/json' \
     -H 'X-Leaderboard-Token: not-the-real-secret' \
     -d '{"name":"forger","score":9999}'
   ```
4. **Expect**: same `401`, same no-op on the stream.
5. Confirm a normal play-through (Scenario 1-3) still succeeds — the game client's own requests
   carry the real, server-injected credential automatically, with no extra step visible to the
   player.

## Validation-failure spot check

```bash
curl -i -X POST http://localhost:8080/api/leaderboard/scores \
  -H 'Content-Type: application/json' \
  -H "X-Leaderboard-Token: ${LEADERBOARD_API_SECRET:-dev-only-change-me}" \
  -d '{"name":"","score":10}'
```

**Expect**: `400 Bad Request` (empty name), no entry written. Repeat with `"score": -5` and with
the `score` field omitted entirely — both `400`, no entry written.

## Automated coverage

Go tests cover the pieces above that don't require a browser: request validation (name/score
bounds) and credential checking as unit/handler tests against an in-memory fake store, and the
`XADD` write itself against a real Redis via Testcontainers-go (constitution Principle III). The
full name-prompt → play → death → Game Over → submission → Replay flow (Scenarios 1, 2, 4) is
inherently a browser interaction and is validated manually per this quickstart, per constitution
Principle IV.
