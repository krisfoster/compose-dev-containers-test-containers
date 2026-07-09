# Contract: GET /api/ping (Extended)

**Feature**: 015-extract-html-templates
**Status**: Amendment to existing contract (first defined in feature 001-host-webapp-ngrok)

---

## Overview

`GET /api/ping` is the live-reload polling endpoint. Browsers on all pages poll this endpoint every 2 seconds and call `location.reload()` when the `id` field changes value. This feature extends the `id` field to incorporate both the process boot ID and the current template version counter.

---

## Request

```
GET /api/ping HTTP/1.1
```

No query parameters, no request body, no authentication required.

---

## Response

**Status**: `200 OK`  
**Content-Type**: `application/json`  
**Cache-Control**: `no-store`

### Response Body

```json
{
  "id": "<startupID>.<templateVersion>"
}
```

| Field | Type   | Description |
|-------|--------|-------------|
| `id`  | string | Composite of two dot-separated values: `startupID` (Unix nanosecond timestamp as a decimal string, set once at process boot) and `templateVersion` (decimal integer, starts at `0`, incremented each time the watcher detects a template file change). Example: `"1720521600000000000.3"` |

### Before This Feature

```json
{ "id": "1720521600000000000" }
```

### After This Feature (on fresh boot, no template changes yet)

```json
{ "id": "1720521600000000000.0" }
```

### After Two Template Changes

```json
{ "id": "1720521600000000000.2" }
```

---

## Browser-Side Behaviour (unchanged)

The existing poll loop in all page templates:

```javascript
var knownID = null;
setInterval(function () {
  fetch('/api/ping')
    .then(function (r) { return r.json(); })
    .then(function (data) {
      if (knownID === null) { knownID = data.id; return; }
      if (data.id !== knownID) { location.reload(); }
    })
    .catch(function () {});
}, 2000);
```

This code requires **no change**: it stores the first `id` it receives (which now has the `.version` suffix) and reloads when it changes. Both process restarts (which change `startupID`) and template file edits (which change `templateVersion`) trigger a reload.

---

## Availability

- Available on both the ungated listener (`app:8080`) and the gated listener (`app:8081`).
- Reachable via nginx at `GET /api/ping` (nginx proxies `/api/` to `app:8080`).

---

## Error Responses

This endpoint has no failure modes beyond the process being down. If the process is unreachable, the `fetch()` call rejects and the `.catch` swallows it silently (existing browser behaviour, unchanged).
