# Architectural Issues

Issues found during a review of the Crossy Whale architecture. Each entry contains the exact
location, a description of the problem, and a concrete fix so an implementing agent has enough
context to act without re-reading the whole codebase.

---

## Issue 1: Dead `/host/rotate` call in leaderboard page (bug)

**Severity:** Bug — feature silently broken  
**File:** `templates/leaderboard.html`  
**Lines:** 91, 99

### Problem

The leaderboard page calls `fetch('/host/rotate', { method: 'POST' })` in two places: on the
manual "Refresh QR" button click and on a 60-second auto-rotate timer. No `/host/rotate` route
is registered anywhere in `app/main.go`'s `ungatedMux()`. Every call returns a 404, so the QR
code never rotates — it only ever shows the window that was auto-activated on the first `/qr.png`
load. The failure is silent because the `fetch` chain only refreshes the image on `resp.ok`.

### Fix

Add a `/host/rotate` route to `app/main.go`'s `ungatedMux()` that:
1. Calls `a.store.Activate(ctx, a.qrWindowTTL)` to generate and store a new window.
2. Returns `204 No Content` on success, `500` on store error.

```go
mux.HandleFunc("/host/rotate", a.handleHostRotate)
```

```go
func (a *App) handleHostRotate(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.Header().Set("Allow", http.MethodPost)
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    if _, err := a.store.Activate(r.Context(), a.qrWindowTTL); err != nil {
        http.Error(w, "failed to rotate window", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
```

Add a corresponding unit test in `app/main_test.go` that:
- POSTs to `/host/rotate` and asserts 204.
- Confirms `store.Current()` returns a new (non-empty) window ID after the call.
- Asserts a GET to `/host/rotate` returns 405.

---

## Issue 2: Score submission auth bypassed at direct port (security)

**Severity:** Security gap — write path unprotected at direct port  
**File:** `docker-compose.yml`  
**Line:** 112 (`ports: - "8083:8083"` under `scores-service`)

### Problem

`scores-service` publishes port 8083 to the host. `POST /scores` on that port goes directly to
`serveSubmit` in `scores-service/internal/scores/handler.go` with no authentication. The
`auth_request` gate exists only in nginx at `/api/leaderboard/scores`. Any client that can reach
`localhost:8083` can submit arbitrary scores without holding a valid `cw_grant` cookie.

The same pattern applies to `commits-service` (8082) and `qr-service` (8084), though those have
no write paths so the risk is read-only data exposure rather than data integrity.

### Fix

Change `scores-service` (and optionally `commits-service`, `qr-service`) from `ports:` to
`expose:` in `docker-compose.yml`, matching how `redis` is configured. `expose:` makes the port
reachable by other Compose services on the internal network but not from the host.

```yaml
# Before
scores-service:
  ports:
    - "8083:8083"

# After
scores-service:
  expose:
    - "8083"
```

Apply the same change to `commits-service` and `qr-service` if developer convenience of direct
port access is not required. If direct access is wanted for debugging, document it explicitly in
`README.md` as a known bypass and add a `--profile debug` guard so the ports are only published
when intentionally enabled.

> Note: `nginx` already maps port 80, so all normal access paths remain functional after this change.

---

## Issue 3: Template re-parsed on every request (performance)

**Severity:** Performance — unnecessary filesystem + parse work per request  
**File:** `app/main.go`  
**Lines:** 230 (`handlePlayIndex`), 283 (`handleRootOrAsset`), 303 (`handleLeaderboardPage`)

### Problem

Every request to `/play`, `/`, and `/leaderboard` calls `template.ParseFiles()` from scratch,
reading the file from disk and re-parsing on every hit. The `watchTemplates` goroutine already
tracks mtime changes and increments `templateVersion` — this information is available but unused
for caching.

### Fix

Add a template cache to `App` that stores the last-parsed `*template.Template` per file keyed by
the `templateVersion` at parse time. On each request, compare the stored version to
`a.templateVersion.Load()`; re-parse only when the version has advanced.

Simplest implementation — add two fields to `App`:

```go
type App struct {
    // ... existing fields ...
    tmplMu    sync.RWMutex
    tmplCache map[string]cachedTemplate
}

type cachedTemplate struct {
    version int64
    tmpl    *template.Template
}
```

Add a helper:

```go
func (a *App) getTemplate(name string) (*template.Template, error) {
    ver := a.templateVersion.Load()
    a.tmplMu.RLock()
    if c, ok := a.tmplCache[name]; ok && c.version == ver {
        a.tmplMu.RUnlock()
        return c.tmpl, nil
    }
    a.tmplMu.RUnlock()

    tmpl, err := template.ParseFiles(filepath.Join(a.templatesDir, name))
    if err != nil {
        return nil, err
    }

    a.tmplMu.Lock()
    a.tmplCache[name] = cachedTemplate{version: ver, tmpl: tmpl}
    a.tmplMu.Unlock()
    return tmpl, nil
}
```

Initialize `tmplCache` in `main()`:

```go
app := &App{
    // ... existing fields ...
    tmplCache: make(map[string]cachedTemplate),
}
```

Replace the three inline `template.ParseFiles` calls with `a.getTemplate(name)`.

---

## Issue 4: ngrok API queried on every `/qr.png` request (performance)

**Severity:** Performance — redundant network call per QR load  
**File:** `app/main.go`  
**Lines:** 257 (`handleQRPNG` calls `discoverPublicHost`), 406 (`discoverPublicHost`)

### Problem

`handleQRPNG` calls `discoverPublicHost()` on every request, which makes an HTTP GET to the
ngrok local inspection API (`http://ngrok:4040/api/tunnels`). The leaderboard page re-fetches
`/qr.png` every 3 seconds on error. The public URL does not change during a session. This is a
redundant network call on the hot path.

### Fix

Cache the discovered host in `App` with a short TTL (e.g. 30 seconds) using an atomic value or
a mutex-guarded struct. On cache hit, return the cached value immediately; on miss or expiry,
query ngrok and update the cache.

```go
type cachedHost struct {
    host    string
    fetchedAt time.Time
}

type App struct {
    // ... existing fields ...
    publicHostMu   sync.Mutex
    publicHostCache cachedHost
}
```

```go
const publicHostCacheTTL = 30 * time.Second

func (a *App) discoverPublicHostCached(ctx context.Context) (string, error) {
    a.publicHostMu.Lock()
    defer a.publicHostMu.Unlock()
    if a.publicHostCache.host != "" && time.Since(a.publicHostCache.fetchedAt) < publicHostCacheTTL {
        return a.publicHostCache.host, nil
    }
    host, err := a.discoverPublicHost(ctx)
    if err != nil {
        return "", err
    }
    a.publicHostCache = cachedHost{host: host, fetchedAt: time.Now()}
    return host, nil
}
```

Replace the `discoverPublicHost` call in `handleQRPNG` with `discoverPublicHostCached`.

---

## Issue 5: `envOr` duplicated across all four services

**Severity:** Maintenance — four identical copies that can diverge  
**Files:**
- `app/main.go:120`
- `commits-service/main.go:29`
- `scores-service/main.go:21`
- `qr-service/main.go:26`

### Problem

Each service defines an identical `envOr(key, fallback string) string` function. Because each
service is its own Go module (separate `go.mod`) there is no shared module to import from. If
the function ever needs to change (e.g., to support a different empty-string treatment), all four
copies must be updated.

### Fix — Option A (preferred): add a comment declaring the copy intentional

Since the function is four lines and the services are intentionally independent modules, the
simplest fix is to add a comment above each copy:

```go
// envOr is intentionally copied per-service (each service is its own Go module).
func envOr(key, fallback string) string { ... }
```

This makes the duplication deliberate and documented, preventing well-meaning consolidation
attempts that would introduce cross-module coupling.

### Fix — Option B: extract a shared internal module

Create `lib/envutil/envutil.go` at the repo root with a `go.mod` of its own, and add a
`replace` directive in each service's `go.mod` pointing to it. This is the "correct" Go
monorepo pattern but adds coordination overhead for a four-line function.

**Recommendation:** Apply Option A unless further shared code accumulates.

---

## Issue 6: `scores-service` missing `ReadTimeout` and `IdleTimeout`

**Severity:** Defensive — slow clients and idle connections accumulate unbounded  
**File:** `scores-service/main.go`  
**Lines:** 53–58

### Problem

The `http.Server` in `scores-service` sets only `WriteTimeout: 0` (correct for SSE streams).
`ReadTimeout` and `IdleTimeout` are not set, meaning:
- A POST to `/scores` with a slow or stalled request body has no timeout.
- Idle keep-alive connections are never closed by the server.

`commits-service/main.go:46-50` correctly sets all three. The two services are inconsistent.

### Fix

Add the missing timeouts to match `commits-service`:

```go
srv := &http.Server{
    Addr:         cfg.ListenAddr,
    Handler:      mux,
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 0,          // SSE connections are long-lived
    IdleTimeout:  60 * time.Second,
}
```

---

## Issue 7: `hostOnly` uses manual string slicing instead of `url.Parse`

**Severity:** Robustness — breaks silently on unexpected URL formats  
**File:** `app/main.go`  
**Lines:** 435–441

### Problem

`hostOnly` strips the `https://` prefix by checking string length and slicing:

```go
func hostOnly(rawURL string) string {
    const httpsPrefix = "https://"
    s := rawURL
    if len(s) >= len(httpsPrefix) && s[:len(httpsPrefix)] == httpsPrefix {
        s = s[len(httpsPrefix):]
    }
    return s
}
```

If ngrok ever returns a URL with a path component (e.g. `https://abc.ngrok-free.app/some-path`)
or a port (e.g. `https://abc.ngrok-free.app:443`), this function silently returns the wrong
value. The standard library handles all these cases correctly.

### Fix

```go
func hostOnly(rawURL string) string {
    u, err := url.Parse(rawURL)
    if err != nil || u.Host == "" {
        return rawURL // fall back to the raw value rather than returning empty
    }
    return u.Host
}
```

This correctly handles URLs with paths, ports, and other components. The `url` package is already
imported in `app/main.go`.

---

## Implementation Order

Suggested order for a follow-up implementing agent:

1. **Issue 1** (dead `/host/rotate` route) — functional bug, highest impact
2. **Issue 7** (`hostOnly` robustness) — two-line fix, zero risk
3. **Issue 6** (scores-service timeouts) — three-line fix, zero risk
4. **Issue 3** (template cache) — moderate complexity, clear test path
5. **Issue 4** (ngrok host cache) — moderate complexity, clear test path
6. **Issue 2** (scores port exposure) — requires compose change + decision on debug profile
7. **Issue 5** (envOr duplication) — add comment only (Option A)
