# Research: Go HTML Template Extraction with Live Reload

**Feature**: 015-extract-html-templates
**Date**: 2026-07-09

---

## Decision 1: Templating Technology

**Decision**: Use Go's standard `html/template` package throughout — no third-party library.

**Rationale**: `html/template` is already used in this codebase for `/play` (via `template.ParseFiles`) and `/leaderboard` (via `template.New().Parse()`). Moving the remaining inline HTML constants to disk-resident `.html` files parsed with the same package adds zero new dependencies. Go's built-in template syntax (`{{.Field}}`) already covers all dynamic value injection in use (leaderboard token, service URLs).

**Alternatives considered**:
- **Templ** (code-generation approach): generates Go functions from `.templ` files; compile-time safety but requires a build step and a new dependency — adds friction with no measurable value for this feature's scope.
- **Pongo2 / Jet / Plush**: Django-like or Shopify Liquid-inspired engines; richer syntax but foreign to Go stdlib conventions and unnecessary for pages that are mostly static HTML with 1–2 dynamic fields each.

---

## Decision 2: Live Reload Mechanism (Browser Refresh on Template Change)

**Decision**: Extend the existing `/api/ping` polling mechanism by making its `id` field a composite of `startupID` (set at process boot, unchanged) and an atomic `templateVersion` counter (incremented when any template file on disk changes). No WebSocket or SSE channel is added.

**Rationale**:
- The leaderboard page and host page already embed a JavaScript snippet that polls `/api/ping` every 2 seconds and calls `location.reload()` when `data.id` changes. This snippet is present in both `leaderboardPageTemplate` and `hostPageHTML` in `main.go`.
- The getting-started page at `/` also polls `/api/ping` (the same snippet is in `gettingStartedPageHTML`).
- By changing the `id` field from `startupID` alone to `startupID + "." + strconv.FormatInt(templateVersion, 10)`, the existing browser logic detects the change on the very next poll (within ~2 seconds) and reloads — with zero changes to the browser-side JavaScript.
- An `atomic.Int64` in Go is cheap, thread-safe, and requires no new dependency.

**Alternatives considered**:
- **WebSocket push**: Would eliminate the 2-second poll delay but requires a new WebSocket hub, more complex server-side state, and browser JS changes. Not justified for a demo tool where 2-second latency is imperceptible.
- **Server-Sent Events (SSE) channel**: Simpler than WebSocket but still requires a new endpoint, keep-alive goroutine, and browser-side EventSource wiring. The polling approach already present in the codebase is sufficient.
- **Hot template cache invalidation only (no browser refresh)**: Changes would be visible on the very next manual page load or browser refresh but no automatic notification. Acceptable for CI/production; not sufficient for the "live update" UX goal of this feature.

---

## Decision 3: Change Detection Method (How the App Detects Template File Edits)

**Decision**: A single background goroutine polls the template directory every **1 second** using `os.Stat` on each template file, comparing `ModTime()` against a stored baseline. When any file's mtime advances, the goroutine increments the `templateVersion` counter and updates the baseline.

**Rationale**:
- No new dependency required (`os.Stat` is stdlib).
- Works reliably with Docker bind mounts (volume-mounted directories on macOS and Linux via Docker Desktop), where `inotify`-based watchers frequently fail to fire because the kernel events are generated on the host FS, not propagated into the container.
- 1-second polling granularity means changes are detected within ~1 second and the browser reloads within ~3 seconds of saving a file (1s detection + up to 2s poll interval in the browser). This meets SC-001 (5-second budget) with margin.

**Alternatives considered**:
- **`fsnotify`** (github.com/fsnotify/fsnotify): inotify/kqueue/FSEvents based. Would detect changes instantly, but: (a) requires a new module dependency; (b) known to be unreliable with Docker Desktop bind mounts on macOS and Windows because file-change events originate on the host kernel, not inside the Linux VM the container runs in. This is a documented open issue in the fsnotify project.
- **Hash-based polling** (hash file contents rather than mtime): More correct for catching content changes where the mtime doesn't advance (e.g., touch then revert), but overkill for a developer workflow where mtime is a reliable proxy.

---

## Decision 4: Template File Layout

**Decision**: New top-level `templates/` directory in the repository root, with one `.html` file per page. The directory is mounted into the app container at a path configured by a new `TEMPLATES_DIR` environment variable, following the same pattern as `FRONTEND_DIR`.

**Files**:
```
templates/
├── getting-started.html    # was: gettingStartedPageHTML const in main.go
├── host.html               # was: hostPageHTML const in main.go
└── leaderboard.html        # was: leaderboardPageTemplate const in main.go
```

`frontend/index.html` (served at `/play`) is already read from disk via `template.ParseFiles`. It stays in `frontend/` alongside the game assets it references. The `FRONTEND_DIR` mount already covers it.

**Rationale**: Keeping templates in a dedicated `templates/` directory separates page structure from game assets (`frontend/`) and avoids commingling HTML with `.js`, `.glb`, and `.css` files that serve entirely different purposes. A single `TEMPLATES_DIR` variable (consistent with `FRONTEND_DIR`) makes the configuration pattern self-explanatory.

---

## Decision 5: Template Re-read Strategy

**Decision**: Templates are **re-read from disk on every request** in the current implementation (same as `handlePlayIndex` today, which calls `template.ParseFiles` on each request). The file watcher only triggers the `templateVersion` bump for browser notification; the actual template content is always fetched fresh from disk.

**Rationale**: Parsing a small HTML template file is microseconds of work. Caching and invalidation logic adds complexity with no measurable latency benefit at the request volumes this demo stack sees (tens of users at a conference booth). The simplicity of "always read from disk" also means a template fix is visible on the very next page load even if the background goroutine hasn't fired yet, which is strictly more responsive than a cache with delayed invalidation.

**Alternatives considered**:
- **Cache + invalidation**: Store the parsed `*template.Template` in a struct field, invalidate when the watcher goroutine detects a change. More efficient at scale but unnecessary and harder to test.

---

## Summary: No New Dependencies

All decisions converge on **zero new runtime dependencies**:
- `html/template` — stdlib, already in use
- `os.Stat` / `time.Since` — stdlib
- `sync/atomic` — stdlib
- `strconv.FormatInt` — stdlib

No amendment to `go.mod`, no constitution amendment required.
