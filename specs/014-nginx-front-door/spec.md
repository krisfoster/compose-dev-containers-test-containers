# Feature Specification: Nginx Front-Door Routing Layer

**Feature Branch**: `014-nginx-front-door`

**Created**: 2026-07-09

**Status**: Draft

**Input**: User description: "spec out option A"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Presenter Starts Demo at a Single Entry Point (Priority: P1)

A presenter runs `docker compose up` on their laptop and reaches the full demo through a single
externally-visible address. All game traffic, QR-gated player access, and leaderboard data flow
through one service. The presenter's existing workflow does not change.

**Why this priority**: The demo workflow cannot regress. If a presenter must remember different
ports or change their startup procedure, the feature has failed regardless of the underlying
architectural improvement.

**Independent Test**: Run `docker compose up`, open the demo in a browser at the stack's single
published address, confirm the game lands page loads, the leaderboard is reachable, and QR code
generation works — with no direct internal-service URLs visible to the user.

**Acceptance Scenarios**:

1. **Given** the compose stack is up, **When** a presenter opens the demo address in a browser, **Then** the game landing page loads without the browser ever being redirected to an internal service port
2. **Given** the compose stack is up, **When** a presenter opens the leaderboard page, **Then** the live scores panel and the commit-feed panel both update in real time
3. **Given** `--profile public` is active, **When** ngrok establishes a tunnel, **Then** the tunnel points to the routing layer, and the ngrok public URL delivers the same experience as the local address

---

### User Story 2 - Player Joins via QR Code Through the Routing Layer (Priority: P2)

A player scans a QR code with their phone. The public URL leads through the routing layer, passes
through the QR gate check unchanged, and delivers the game. The player has no awareness of any
routing intermediary.

**Why this priority**: The QR gate is the primary access-control mechanism for the demo. Any
routing change that silently breaks gate enforcement — grant-cookie handling, redirects, or token
injection — would be a critical regression during a live event.

**Independent Test**: With `--profile public` active, scan the QR code on a phone that has no
existing grant; confirm the gate challenge is served and, after grant is obtained, the game loads
correctly with the session token present in the page.

**Acceptance Scenarios**:

1. **Given** a player's phone has no grant cookie, **When** they follow the ngrok URL, **Then** the gate challenge is presented exactly as it was before this change
2. **Given** a player has a valid grant cookie, **When** they follow the ngrok URL, **Then** the game loads with the session token injected into the page
3. **Given** a player's grant has expired, **When** they revisit the ngrok URL, **Then** they are redirected through the gate challenge again

---

### User Story 3 - Static Game and Leaderboard Assets Served Without Backend Involvement (Priority: P3)

Game HTML, JavaScript bundles, 3D models, audio files, and leaderboard component scripts are
served directly by the routing layer without invoking the Go backend process. The Go app only
handles requests that require server-side logic: gate checks, session-token injection into pages,
score submission, and QR management.

**Why this priority**: This is the architectural goal of Option A. It reduces the Go app's serving
footprint and creates a clean boundary between static-file delivery and business logic. The visible
demo behaviour is unchanged; the benefit is operational clarity and reduced load on the backend.

**Independent Test**: Load the game in a browser, then inspect the Go app's access log; static
asset requests (JS bundles, GLB models, audio, CSS) must not appear in it.

**Acceptance Scenarios**:

1. **Given** a browser requests the game's JavaScript bundle, **When** the request arrives at the routing layer, **Then** the file is served directly without the request being forwarded to the Go app
2. **Given** a browser requests a 3D model asset, **When** the request arrives at the routing layer, **Then** it is served directly from the file set without a backend hop
3. **Given** a browser requests the leaderboard React component bundle, **When** the request arrives, **Then** it is served directly without reaching the Go app

---

### Edge Cases

- What happens when the Go backend is slow to start and the routing layer receives a dynamic request before Go is ready? The routing layer must return a clear error rather than hanging silently.
- What happens when the scores microservice or commits microservice is unavailable? The routing layer must propagate a service-unavailable response rather than timing out invisibly.
- What happens when a player's phone does not follow redirects correctly (some older mobile browsers)? Gate redirects must continue to work correctly when forwarded through the routing layer.
- How does the routing layer handle long-lived SSE connections for live scores and commit feed? Streaming connections must not be buffered or closed prematurely — they must remain open for the duration of a browser session.
- What happens when the routing layer itself restarts? The compose `depends_on` ordering must ensure the routing layer starts after the services it forwards to, and the presenter-facing browser reconnects gracefully.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A routing service MUST be added to `docker-compose.yml` as the single externally-published ingress for the demo stack; it MUST start with `docker compose up` and require no host-side configuration beyond Docker Desktop and git
- **FR-002**: The routing service MUST serve game static files (HTML, JavaScript, CSS, 3D model files, audio) directly from the file set without forwarding those requests to the Go backend
- **FR-003**: The routing service MUST serve leaderboard frontend assets (component scripts, stylesheets) directly from the file set without forwarding those requests to the Go backend
- **FR-004**: The routing service MUST forward requests for the gated game play path and host-control path to the Go backend, preserving all request headers, cookies, and response headers exactly so the existing gate middleware functions without modification
- **FR-005**: The routing service MUST forward leaderboard score data requests to the scores microservice, including long-lived SSE stream connections
- **FR-006**: The routing service MUST forward git commit data requests to the commits microservice, including long-lived SSE stream connections
- **FR-007**: The routing service MUST NOT buffer or prematurely close SSE streaming connections — the routing layer must act as a transparent pass-through for streaming responses
- **FR-008**: The ngrok tunnel configuration MUST be updated to target the routing service rather than the Go backend's gated port
- **FR-009**: The Go backend MUST remain the sole authority for gate enforcement, QR management, score submission, and template rendering — no gate logic may be duplicated in or delegated to the routing layer
- **FR-010**: The Go backend's gated port MUST NOT be published directly to the host machine after this change; all external access to gated content flows exclusively through the routing layer
- **FR-011**: Browser-visible service URLs injected into served pages (used by the leaderboard's scores and commits components to connect to SSE streams) MUST be updated so they resolve through the routing layer at the single ingress address
- **FR-012**: The `docker compose up` workflow (without `--profile public`) MUST continue to work for local development; the ngrok tunnel is still an optional addition via `--profile public`

### Key Entities

- **Routing Layer**: The new compose service acting as single public ingress; owns routing rules but contains no business logic; shares file access with the Go backend for static assets
- **Static Asset Set**: Game files (`frontend/game/`) and leaderboard component files (`frontend/leaderboard/`) that the routing layer serves directly; these files are already in the repository and must be accessible to the routing layer at compose start
- **Gated Path**: Requests that carry or require grant cookies and must reach the Go gate middleware intact; the routing layer forwards them verbatim without inspection

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The full demo flow (presenter view + player QR scan + live leaderboard updates) works end-to-end through the single ingress address, with no direct internal-service addresses needed in a browser
- **SC-002**: Game static assets are served without any Go backend process log entries for those file requests — verifiable by inspecting access logs immediately after a page load
- **SC-003**: The QR gate continues to work without any modification to existing gate middleware code — verifiable by the gate test scenarios in User Story 2 passing without code changes to the gate
- **SC-004**: SSE streams for scores and commits remain open and actively delivering events for at least 5 minutes through the routing layer — matching the duration requirement of a full booth demo rotation
- **SC-005**: `docker compose up` reaches a demoable state within the same time window as before this change (routing layer startup does not add meaningful latency to the presenter workflow)
- **SC-006**: Following the ngrok public URL on a mobile device triggers the gate challenge and delivers a playable game — the complete round trip through the routing layer must work on a real phone

## Assumptions

- The Go app will continue to listen on two internal ports: one for ungated presenter-local access (leaderboard, admin) and one for gated player access (game, host controls); the routing layer directs traffic to the appropriate internal port based on path
- Static files are made available to the routing layer via a shared Docker volume or build-time copy — the routing layer does not fetch them from the Go app at runtime
- The routing layer is intended as a permanent addition to the compose stack, not an interim measure; this conflicts with the existing constitution's "Interim Static Hosting carve-out" which only permits a non-Go web server while no Go backend exists for that surface. **This feature requires a constitution amendment** to permit the routing layer as a permanent reverse proxy alongside the existing Go backend
- The React component bundles (scores and commits) are already vendored under `frontend/leaderboard/` and require no build step; the routing layer serves them as static files
- The scores microservice and commits microservice continue to run on their existing internal ports; the routing layer proxies to them by service name within the compose network
- `docker compose up` without `--profile public` exposes the routing layer locally for presenter development; ngrok is still only activated via `--profile public`
- The existing go template variable injection for `SCORES_SERVICE_URL` and `COMMITS_SERVICE_URL` will be updated to produce URLs relative to the single ingress address rather than direct microservice ports
