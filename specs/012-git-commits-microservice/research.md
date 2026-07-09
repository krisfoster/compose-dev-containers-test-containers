# Research: Git Commits Microservice

All items below were open questions in the Technical Context; none remain as NEEDS CLARIFICATION.

## 1. Push mechanism choice: SSE vs WebSockets vs polling-only

**Decision**: Server-Sent Events (SSE) as the primary delivery mechanism for the React component,
with automatic polling fallback (30 s interval, matching the existing inline JS) for environments
where SSE connections are dropped by intermediaries (e.g., some proxy setups).

**Rationale**: SSE is the right fit for this one-directional, server-to-client use case:
- Unidirectional (server pushes commit events to the browser — no bidirectional messaging needed)
- Native HTTP — no WebSocket upgrade, no special proxy config, works through standard HTTP/1.1
- Built-in browser reconnection (`EventSource` auto-reconnects on disconnect, no client code needed)
- Simpler to implement in Go than WebSockets: `flusher.Flush()` in a loop, no gorilla/websocket dependency
- SSE shares the same `net/http` primitives already used everywhere in this project

Polling fallback is a one-liner on the React component: if `EventSource` is unavailable (it is
supported in all modern browsers; this guard is for paranoia) or the SSE connection fails
permanently, the component switches to `setInterval`-based `fetch` at 30 s.

**Alternatives considered**:
- **WebSockets**: More complex handshake, requires a goroutine per connection, frame encoding/decoding,
  and a new dependency (gorilla/websocket) or more Go standard library work. No benefit here since
  the connection is server→client only.
- **Polling only**: Already implemented in the existing vanilla JS (30 s interval). Works but does
  not satisfy the spec's requirement for a push mechanism. Keeping polling-only would also mean the
  React component adds no improvement over the current implementation.

## 2. How the React component is loaded without a build pipeline

**Decision**: React 18 + ReactDOM 18 are vendored as pre-built UMD production bundles
(`react.production.min.js`, `react-dom.production.min.js`) in `frontend/leaderboard/`. They are
served as static files by the existing `app` service under a new file route `/leaderboard-assets/`.
The commits component (`commits-component.js`) is loaded as an ES module via a `<script
type="module">` tag, using an importmap that maps `"react"` and `"react-dom"` to the vendored
paths. No npm, no bundler, no build step.

**Rationale**:
- Vendoring removes the CDN runtime dependency — critical for a booth environment where network is
  unreliable or the CDN is rate-limited
- UMD bundles exposed via importmap is the same pattern the game already uses for three.js — this
  stays consistent with the project's existing ES-module-importmap approach (constitution
  Technology Stack)
- The `app` service already serves static files from `frontend/game` via `http.FileServer`; a
  second route for `frontend/leaderboard` requires one `mux.Handle` line and one `COPY` in the
  Dockerfile
- The `commits-component.js` module is plain JS (no JSX) using `React.createElement` — no Babel
  needed

**Alternatives considered**:
- **CDN (esm.sh, unpkg, jsDelivr)**: Rejected — runtime CDN dependency breaks the demo if the
  network is down or the CDN is slow. Constitution Principle II requires no host-side installs;
  a CDN dependency isn't a host-side install but it is an uncontrolled runtime dependency.
- **npm build step producing a bundle**: Would introduce npm as a build-time dependency and a
  bundler (Vite, esbuild) into the Dockerfile, increasing image build time and complexity. Nothing
  in this feature requires JSX or TypeScript, so there is no pay-off for the complexity cost.
- **Inline React in the Go string constant**: Downloading React's full source inline is
  impractical; vendored files served separately keep the Go string manageable.

## 3. New Go module structure for the commits service

**Decision**: `commits-service/` at the repo root contains a standalone Go module
(`module crossywhale/commits-service`). It has its own `go.mod`, `go.sum`, and Dockerfile.
The `internal/commits` package owns the handler logic.

**Rationale**: A separate module gives the commits service a fully independent build context (its
Dockerfile can reference just the `commits-service/` subtree, keeping image layers small). It also
allows the `go-git/go-git/v5` dependency to be removed from `crossywhale/app` after the handler is
relocated, reducing the `app` image's dependency footprint.

The `internal/commits` package follows the same pattern as `app/internal/leaderboard` and
`app/internal/gate` — the handler struct and its test live together, separated from `main.go`'s
wire-up.

**Alternatives considered**:
- **Single module (`crossywhale/app`) with a new `cmd/commits-service` entry point**: Would share
  the same `go.mod`, making it harder to prune the `app` binary's dependencies (go-git stays in the
  module even if `app/main.go` no longer imports it). Separate modules give cleaner dependency
  isolation.
- **Placing commits handler in `app/internal/` and calling it from a new `cmd/`**: Same issue as
  above — shared module, same image build context.

## 4. CORS strategy

**Decision**: The commits service adds permissive CORS headers (`Access-Control-Allow-Origin: *`)
on all responses. No credential or cookie is involved in the commits API (read-only, public data),
so wildcard CORS is safe.

A `OPTIONS` preflight handler is included for completeness, though the leaderboard's `fetch` and
`EventSource` calls are simple requests (no custom headers) and do not trigger preflights in
practice.

**Rationale**: The leaderboard page is served by `app` (e.g., `localhost:8080`); the commits
service is on a different port (e.g., `localhost:8082`). From the browser's same-origin policy,
these are different origins, so CORS headers are required. Wildcard is appropriate here because:
- The data is public (git commits — no PII, no auth)
- The commits service has no POST/write surface, so CSRF is not a concern
- Restricting the origin to the `app` service URL would require environment-specific config (the
  URL differs between localhost dev and a deployed demo)

**Alternatives considered**:
- **Restrict to `http://localhost:8080`**: Creates a config dependency between the two services;
  breaks if `WEB_PORT` is changed or the leaderboard is accessed via ngrok's public URL.
- **Proxy commits API through `app`**: Would re-introduce the coupling between `app` and the commits
  service that this feature is designed to remove. Defeats the microservice split.

## 5. Port and service naming in docker-compose.yml

**Decision**: The commits service listens on port `8082` internally and publishes `8082:8082` to
the host. The compose service name is `commits-service`. The React component uses a configurable
base URL (defaulting to `http://localhost:8082` for local dev) injected by the leaderboard page
template, so future environments (staging, demo with a different port) can override without code
changes.

**Rationale**: Port `8080` is the `app` web port; `8081` is the `app` gated port; `8082` is
clearly the next slot in sequence and avoids conflicts. Publishing the port allows direct testing
via curl/browser without going through `app`.

**Alternatives considered**:
- **Internal-only (no published port)**: Would mean the browser cannot reach the commits service
  directly — CORS is meaningless if the browser can't form the request. Published port is required.
- **Re-using port 8080 via a path prefix on `app` (reverse proxy)**: This is the "proxy commits
  through app" alternative rejected in §4.

## 6. Fate of the existing `/api/commits` handler in `app/main.go`

**Decision**: Remove `handleCommits` and the `gitRepoPath` field from `app/main.go` entirely in
the same PR that introduces the commits service. Do not leave a compatibility shim — there is no
external caller that requires the old path at the time of this change (only the leaderboard page's
own JS, which is updated in the same PR to point to the new service).

**Rationale**: A shim that proxies to the new service would add complexity, a new internal
dependency, and a test surface for no gain. Atomic removal and replacement keeps `app/main.go`
clean and removes the `go-git` dependency from the `app` module, shrinking the image.

**Alternatives considered**:
- **Leave `/api/commits` as a proxy to `commits-service`**: Adds an HTTP round-trip on each
  leaderboard poll and keeps `go-git` in `app/go.mod` for no reason. Rejected.
- **Deprecate gradually (leave in place, update leaderboard JS, remove in follow-up PR)**:
  Unnecessary — there are no other callers and no external consumers of this internal endpoint.

## 7. React version and vendored file source

**Decision**: React 18.3.x (latest stable at plan time). Vendored files are the official
production minified UMD builds from the React npm package
(`react/umd/react.production.min.js`, `react-dom/umd/react-dom.production.min.js`). Files are
extracted from the npm package tarball — no `node_modules` checked in, just the two JS files.
ATTRIBUTION.md entries reference the npm package URLs and MIT licence.

**Rationale**: React 18 is the current stable release and the version with the most mature
concurrent rendering support. UMD builds are designed to be loaded directly via `<script>` or
importmap — they expose `window.React` and `window.ReactDOM` globals when loaded as UMD, which is
how the importmap binding works. React 19 exists as of 2024-12 but its UMD build is not yet widely
vetted in importmap contexts; 18 is the safer choice for a demo-critical feature.

**Alternatives considered**:
- **Preact**: Smaller (3 kB vs 44 kB), compatible React API. Rejected because the spec explicitly
  says "React JS component" and Preact, while API-compatible, is a different library. Using Preact
  would require clarification from the user.
- **React 19**: Emerging ecosystem; UMD production bundles work, but less vetted. Deferred to a
  future amendment if there is a concrete reason to upgrade.
