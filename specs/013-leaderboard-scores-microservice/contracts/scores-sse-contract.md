# SSE Stream Contract: Scores Service

**Endpoint**: `GET http://localhost:8083/scores/stream`

**Protocol**: HTTP/1.1 Server-Sent Events (W3C EventSource specification)

## Connection behaviour

| Phase | Behaviour |
|-------|-----------|
| On connect | Server immediately emits one `standings` event with the current standings |
| While open | Server emits a new `standings` event each time a score-change pub/sub notification arrives |
| On client disconnect | Server closes the goroutine/handler; no cleanup needed on client |
| On server restart | Browser `EventSource` reconnects automatically (browser-native) |
| On SSE unavailable | React component falls back to 5 s polling via `GET /scores` |

## Response headers

```
HTTP/1.1 200 OK
Content-Type: text/event-stream; charset=utf-8
Cache-Control: no-cache
Connection: keep-alive
Access-Control-Allow-Origin: *
```

## Event wire format

Each broadcast is a single SSE event:

```
event: standings
data: {"standings":[{"rank":1,"name":"Alice","score":42},{"rank":2,"name":"Bob","score":37}]}

```

*(One blank line terminates each event block — per the SSE spec.)*

Empty-standings event (no scores have been recorded):

```
event: standings
data: {"standings":[]}

```

## React component consumption

```js
// Primary path: SSE
const source = new EventSource(scoresServiceURL + '/scores/stream');
source.addEventListener('standings', (e) => {
  const { standings } = JSON.parse(e.data);
  setStandings(standings);  // React state update — re-renders the list
});

// Fallback: polling (activated if EventSource fails permanently)
setInterval(async () => {
  const resp = await fetch(scoresServiceURL + '/scores');
  if (resp.ok) {
    const { standings } = await resp.json();
    setStandings(standings);
  }
}, 5_000);
```

## Notes

- The `data` field is always a valid JSON-encoded `StandingsResponse` object (see `scores-openapi.yaml`).
- The React component does NOT handle partial updates — each `standings` event carries the complete current standings and the component replaces its entire rendered list.
- Standings show each player's best score only (one row per player, best-score aggregation).
- A `retry:` SSE field is not set; the browser's default reconnect interval is appropriate.
- Events are emitted only when a score-change notification arrives via Redis pub/sub, not on a periodic timer. A session with no new submissions will see only the initial on-connect event.
- Keepalive comments (`:keepalive\n\n`) are NOT emitted; SSE connections are expected to be held open by the browser's native EventSource reconnect mechanism.
