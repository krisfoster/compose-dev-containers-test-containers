# Quickstart Validation Guide: Leaderboard Scores Microservice

This guide describes how to validate that the feature works end-to-end against the compose stack. Run these scenarios in order; each one builds on the previous.

## Prerequisites

- Docker Desktop running
- `docker compose` available
- A terminal at the repo root
- The leaderboard page open in a browser at `http://localhost:8080/leaderboard`

## Scenario 1: Service starts and is reachable

**Validates**: FR-002, FR-009, SC-003

```bash
docker compose up --build
```

Expected outcome:
- `scores-service` appears in the compose output and prints a startup log line
- `GET http://localhost:8083/scores` returns HTTP 200 with `{"standings":[]}`
- No HTML content in the response (`Content-Type: application/json`)

```bash
curl -s http://localhost:8083/scores | python3 -m json.tool
# Expected: { "standings": [] }
```

---

## Scenario 2: Leaderboard page shows empty-state message

**Validates**: FR-007, SC-002

1. Open `http://localhost:8080/leaderboard` in a browser.
2. The standings column renders: **"No scores yet — be the first to play!"** (or equivalent).
3. No error message or blank area is visible.

---

## Scenario 3: SSE stream connects and delivers initial standings

**Validates**: FR-004, FR-004a

```bash
curl -N -H "Accept: text/event-stream" http://localhost:8083/scores/stream
```

Expected output immediately on connect (empty standings):
```
event: standings
data: {"standings":[]}
```

The connection should remain open (no auto-close). Press Ctrl-C to exit.

---

## Scenario 4: Submitting a score updates the leaderboard live

**Validates**: FR-006, SC-001, FR-004a, FR-004b

1. Open `http://localhost:8080/leaderboard` in a browser (standings column visible).
2. In a separate terminal, submit a score via the app's write endpoint:

```bash
curl -s -X POST http://localhost:8080/api/leaderboard/scores \
  -H "Content-Type: application/json" \
  -H "X-Leaderboard-Token: dev-only-change-me" \
  -d '{"name":"TestPlayer","score":99}'
```

Expected outcome:
- The leaderboard standings column updates within 5 seconds (no page reload needed).
- **TestPlayer** appears at rank 1 with score 99.
- The empty-state message disappears.

---

## Scenario 5: Best-score-per-player aggregation

**Validates**: Data model (best-score aggregation)

Submit a second, lower score for the same player:

```bash
curl -s -X POST http://localhost:8080/api/leaderboard/scores \
  -H "Content-Type: application/json" \
  -H "X-Leaderboard-Token: dev-only-change-me" \
  -d '{"name":"TestPlayer","score":10}'
```

Expected outcome:
- **TestPlayer** still appears at rank 1 with score **99** (the higher score is retained).
- **TestPlayer** appears exactly once in the standings (no duplicate rows).

---

## Scenario 6: Configurable standings limit

**Validates**: FR-008, Q3 (configurable `SCORES_LIMIT`)

Submit 12 unique players via the API (or adjust `SCORES_LIMIT` to a low value in `.env`):

```bash
for i in $(seq 1 12); do
  curl -s -X POST http://localhost:8080/api/leaderboard/scores \
    -H "Content-Type: application/json" \
    -H "X-Leaderboard-Token: dev-only-change-me" \
    -d "{\"name\":\"Player${i}\",\"score\":${i}}" > /dev/null
done
```

Expected outcome:
- The leaderboard shows at most 10 rows (the default `SCORES_LIMIT`).
- To verify configurability: add `SCORES_LIMIT=5` to `.env`, restart the scores-service, and confirm only 5 rows appear.

---

## Scenario 7: Transient service restart recovery

**Validates**: FR-010, SC-005

1. Open `http://localhost:8080/leaderboard` in a browser with standings visible.
2. Restart the scores-service:

```bash
docker compose restart scores-service
```

3. Within 15 seconds, the standings column should resume showing current data without a page reload.
4. The React component should NOT show a permanent error state during the outage window.

---

## Scenario 8: GET /api/leaderboard/scores returns 405

**Validates**: FR-014 (read endpoint removed from app)

```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/leaderboard/scores
# Expected: 405 (Method Not Allowed — GET removed; only POST remains)
```

---

## Scenario 9: CORS headers present (browser-direct access)

**Validates**: FR-011 (browser reaches scores-service directly)

```bash
curl -s -I http://localhost:8083/scores | grep -i "access-control"
# Expected: Access-Control-Allow-Origin: *
```

---

## References

- API contract: `contracts/scores-openapi.yaml`
- SSE wire format: `contracts/scores-sse-contract.md`
- Data model: `data-model.md`
- Research decisions: `research.md`
