# Research: Host Web App with Public Ngrok Access

All items below were open questions in the Technical Context; none remain as
NEEDS CLARIFICATION.

## 1. Static webserver container

**Decision**: `nginx:alpine`, serving `frontend/game/` as a bind-mounted (or `COPY`'d) document
root via a minimal `nginx.conf` — no custom Dockerfile build step required beyond mounting files.

**Rationale**: The frontend is already a pure static site (plain HTML/CSS/JS, dependencies loaded
from a CDN via an `<script type="importmap">` — see `frontend/game/index.html`). It needs nothing
more than a file server. `nginx:alpine` is a small (~40MB), widely-trusted image that serves a
static directory with a one-line `location / { try_files $uri $uri/ =404; }` config, requires zero
custom application code, and satisfies constitution Principle II (compose-defined, no host
installs) directly.

**Alternatives considered**:
- **Caddy**: auto-HTTPS is a selling point Caddy has that isn't needed here — ngrok already
  terminates public TLS, and local access is plain HTTP on the laptop. Adds a dependency with no
  corresponding benefit for this feature.
- **`python:3-alpine -m http.server`**: no gzip, no cache headers, not meant for anything beyond
  throwaway scripts; nginx is barely more setup for a materially more solid result.
- **`node:alpine` + `http-server`**: requires an `npm install` layer at build/start time, which is
  unnecessary weight for serving files that need no processing.

## 2. Public tunnel container

**Decision**: `ngrok/ngrok:3` (official image, pinned to the major-version-3 tag), run as
`ngrok http webserver:80` (or the compose `command:` equivalent), configured via the
`NGROK_AUTHTOKEN` environment variable. This matches the tunnel service already named in this
project's design docs (`project.md`, `crossy.md`). The major-version tag is pinned rather than
`:latest` so a booth-day `docker compose pull` can't silently pick up a breaking agent CLI change;
`:3` still tracks patch/minor updates within the current major version.

**Rationale**: ngrok publishes and maintains this image; pulling it is simpler and more reliable
than reimplementing tunnel logic. The project constitution already fixes ngrok as "the default
tunnel" in the Technology Stack section, so this is not a new stack decision — it's applying an
existing one.

**Alternatives considered**:
- **`cloudflare/cloudflared`**: constitution allows this as an acceptable alternative when a stable
  reserved URL is needed for a specific event, but ngrok is the stated default and is what the
  user asked for by name. No reason to deviate for this feature.

## 3. Discovering the current public URL (FR-005)

**Decision**: Point the presenter at ngrok's own built-in web inspection UI/API, exposed on the
container's port 4040 (`http://localhost:4040`), rather than building a custom endpoint. The
Compose service publishes 4040 to the host for this purpose.

**Rationale**: The ngrok agent already exposes the live tunnel URL at `http://localhost:4040` (UI)
and `http://localhost:4040/api/tunnels` (JSON) with zero custom code. Building a `/public-url`
endpoint (as sketched for the eventual Go backend in `project.md`) requires a backend that does not
exist yet in this repository, and would be out of scope per this feature's Assumptions (backend
hosting is a separate future feature).

**Alternatives considered**:
- **Custom backend endpoint**: deferred — no backend exists yet; adding one only to expose a URL
  would be disproportionate to this feature's scope and would duplicate work the eventual backend
  feature will already need to do.

## 4. Making public access optional (FR-004)

**Decision**: Docker Compose profiles. The `ngrok` service is tagged `profiles: ["public"]`, so
`docker compose up` starts local-only by default, and `docker compose --profile public up` adds
the tunnel.

**Rationale**: A single compose file with a profile is simpler to maintain than a base file plus an
overlay file, and it matches the toggle already described in this project's own design docs.

**Alternatives considered**:
- **Separate `docker-compose.public.yml` overlay**: more files, more combinations to keep in sync;
  rejected in favor of one file with a profile flag.

## 5. Resilience when the tunnel is unavailable (FR-006)

**Decision**: The webserver service has no `depends_on` relationship to the ngrok service (and
vice versa), so a missing/invalid `NGROK_AUTHTOKEN` or a failed ngrok container never blocks or
restarts the webserver. `restart: unless-stopped` is set on the ngrok service so transient
provider or network hiccups self-heal without presenter action.

**Rationale**: Directly satisfies the spec's requirement that local access survive a public-tunnel
outage, with no custom health-check logic needed — this falls out of *not* wiring a dependency
between the two services.

**Alternatives considered**:
- **Health-checked `depends_on`**: would couple the services and risk local access failing to
  start if ngrok is slow or down — the opposite of the desired resilience.

## 6. Port conflicts on the host (FR-008)

**Decision**: Rely on Docker Compose's native "port is already allocated" failure, which already
aborts startup with a clear, actionable message. Document the expected message and the fix
(stop the conflicting process, or change the published port in `.env`) in `quickstart.md`.

**Rationale**: Compose's existing error is already specific and actionable; adding a custom
pre-flight port-check script for a booth-demo-scale feature would be extra surface area for no
material improvement in clarity.

**Alternatives considered**:
- **Custom pre-flight script**: rejected as unnecessary complexity for the scale of this feature.
