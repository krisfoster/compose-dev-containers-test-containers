# SSE Stream Contract: Commits Service

**Endpoint**: `GET http://localhost:8082/commits/stream`

**Protocol**: HTTP/1.1 Server-Sent Events (W3C EventSource specification)

## Connection behaviour

| Phase | Behaviour |
|-------|-----------|
| On connect | Server immediately emits one `commits` event with the current feed |
| While open | Server emits a new `commits` event every 30 seconds |
| On client disconnect | Server closes the goroutine/handler; no cleanup needed on client |
| On server restart | Browser `EventSource` reconnects automatically (browser-native) |
| On SSE unavailable | React component falls back to 30 s polling via `GET /commits` |

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
event: commits
data: {"commits":[{"hash":"a2c6757","author":"Kris Foster","date":"2026-07-07 22:15","message":"Leaderboard redesign, ngrok fix, and mobile camera improvements"},{"hash":"84cc9c6","author":"Kris Foster","date":"2026-07-07 18:42","message":"Improve README readability and completeness"}]}

```

*(One blank line terminates each event block — per the SSE spec.)*

Empty-feed event (no commits in the repository):

```
event: commits
data: {"commits":[]}

```

## React component consumption

```js
// Primary path: SSE
const source = new EventSource('http://localhost:8082/commits/stream');
source.addEventListener('commits', (e) => {
  const { commits } = JSON.parse(e.data);
  setCommits(commits);  // React state update — re-renders the list
});

// Fallback: polling (activated if EventSource is unavailable or fails permanently)
setInterval(async () => {
  const resp = await fetch('http://localhost:8082/commits');
  if (resp.ok) {
    const { commits } = await resp.json();
    setCommits(commits);
  }
}, 30_000);
```

## Notes

- The `data` field is always a valid JSON-encoded `CommitFeed` object (see `commits-openapi.yaml`).
- The React component does NOT need to handle partial updates — each `commits` event carries the
  complete current feed and the component replaces its entire rendered list.
- A `retry:` SSE field is not set; the browser's default reconnect interval (a few seconds) is
  appropriate.
- Keepalive comments (`:keepalive\n\n`) are NOT emitted — the 30 s data interval is frequent
  enough to prevent proxy timeouts at booth scale.
