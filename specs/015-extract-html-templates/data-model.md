# Data Model: Go HTML Template Extraction with Live Reload

**Feature**: 015-extract-html-templates
**Date**: 2026-07-09

---

## Entities

### 1. Template File

A named HTML file on disk containing Go `html/template` syntax for one page of the app.

| Field       | Type     | Description                                              |
|-------------|----------|----------------------------------------------------------|
| `path`      | string   | Absolute path to the `.html` file on the container FS   |
| `name`      | string   | Template name as registered with `html/template` (`"getting-started"`, `"host"`, `"leaderboard"`) |
| `mtime`     | time.Time| Last modification time, used by the watcher for change detection |

**Files** (resolved via `TEMPLATES_DIR`):

| Page     | File                           | Dynamic fields injected at render time |
|----------|--------------------------------|----------------------------------------|
| `/`      | `getting-started.html`         | None — fully static HTML               |
| `/host`  | `host.html`                    | None — fully static HTML               |
| `/leaderboard` | `leaderboard.html`       | `.ScoresServiceURL`, `.CommitsServiceURL` |
| `/play`  | `frontend/index.html` (existing, in `FRONTEND_DIR`) | `.LeaderboardToken` |

---

### 2. Template Watcher State

Internal-only state maintained by the background polling goroutine. Not persisted; lives only in the running process.

| Field           | Type                  | Description                                                 |
|-----------------|-----------------------|-------------------------------------------------------------|
| `baselines`     | `map[string]time.Time`| Maps each template file path to its mtime at last check     |
| `pollInterval`  | time.Duration         | How often the goroutine wakes to check mtimes (1 second)    |

---

### 3. Template Version Counter

A process-scoped monotonically increasing integer, atomically updated when the watcher detects a template change.

| Field            | Type          | Description                                          |
|------------------|---------------|------------------------------------------------------|
| `templateVersion`| `atomic.Int64`| Starts at 0 on process boot; incremented on each detected template change |

---

### 4. Ping Response

The JSON object returned by `GET /api/ping`. Extended from `{"id": "<startupID>"}` to a composite that incorporates both the process boot ID and the current template version.

| Field | Type   | Description                                                               |
|-------|--------|---------------------------------------------------------------------------|
| `id`  | string | Composite: `startupID + "." + strconv.FormatInt(templateVersion, 10)`. Browsers compare against their stored value; any change triggers `location.reload()`. |

**Before this feature**: `{"id": "1234567890"}`  
**After this feature**: `{"id": "1234567890.0"}` (`.0` on boot; `.1`, `.2`, … as templates change)

The browser-side check `if (data.id !== knownID) { location.reload(); }` (already present in all three page templates and in the new external template files) continues to work without modification.

---

## State Transitions

### Template Version Lifecycle

```
Process starts
    │
    ▼
templateVersion = 0
id = "<startupID>.0"
    │
    ▼
Watcher goroutine polls every 1s
    │
    ├─ No mtime change → (no-op)
    │
    └─ Any mtime advances
           │
           ▼
       templateVersion.Add(1)
       id = "<startupID>.<n>"
           │
           ▼
       Browser polls /api/ping within 2s
       Sees new id → location.reload()
           │
           ▼
       Page re-renders from fresh template on disk
```

---

## Validation Rules

- **Template file missing at startup**: The app MUST fail to start with a logged error naming the missing file. It must not start and silently return 500s.
- **Template file missing mid-run** (deleted after startup): The handler for that page returns `500 Internal Server Error` and logs the file path and OS error. The watcher goroutine logs the missing file but does not crash the process — the file may be transiently absent during an editor save cycle.
- **Template syntax error**: `template.ParseFiles` / `template.New().Parse()` returns a non-nil error; the handler returns `500 Internal Server Error` and logs the parse error with the file path. The browser shows the error page.
- **Template directory missing at startup**: Fail to start; logged with the `TEMPLATES_DIR` value.
- **Concurrent template change + in-flight request**: Acceptable to serve either the old or new content for that one request; no partial content, no crash.

---

## Configuration

| Environment Variable | Default              | Description                                                                 |
|----------------------|----------------------|-----------------------------------------------------------------------------|
| `TEMPLATES_DIR`      | `/templates`         | Container-side path to the directory containing page template files         |
| `FRONTEND_DIR`       | `/frontend` (existing) | Container-side path to game assets + `index.html`; unchanged by this feature |
