# Implementation Plan: Nginx Front-Door Routing Layer

**Branch**: `014-nginx-front-door` | **Date**: 2026-07-09 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/014-nginx-front-door/spec.md`

## Summary

Add an nginx reverse-proxy service to the compose stack as the single externally-visible ingress
point for the Crossy Whale demo. nginx serves game static assets and leaderboard component
bundles directly (no Go backend hop), routes `/play` through the Go app's existing QR gate on the
gated port (8081), and proxies all other dynamic paths to the ungated Go app (8080) or the
appropriate microservice. ngrok is reconfigured to tunnel to `nginx:80` instead of `app:8081`.
`SCORES_SERVICE_URL` and `COMMITS_SERVICE_URL` injected into the leaderboard template are updated
to reflect the single ingress base URL.

No existing Go middleware is modified. The gate remains entirely in Go, reached by the nginx
proxy for `/play` requests.

## Technical Context

**Language/Version**: Go 1.25.0 (unchanged for app and microservices); nginx for routing layer

**Primary Dependencies**:
- `dhi.io/nginx:1-alpine3.24` — routing service base image (DHI hardened nginx, matches existing alpine3.24 base)
- `dhi.io/golang:1.25-alpine-dev` → `dhi.io/static:20260611-alpine3.24` — app/microservice base images (unchanged)

**Storage**: N/A — no new storage; static files bundled into nginx image at build time

**Testing**: Manual end-to-end browser validation per Principle IV; no new Go tests required (no Go code changes)

**Target Platform**: Linux container (`linux/amd64` + `linux/arm64`), Docker Compose

**Project Type**: Infrastructure / compose service addition — new service and routing config only

**Performance Goals**: Zero added latency for static asset delivery compared to Go file serving (nginx is faster at static file serving); no latency regression for proxied routes

**Constraints**:
- No business logic in nginx — routing only (Principle III equivalent at the infra layer)
- No host-side installs beyond Docker Desktop; compose-only change (Principle II)
- Constitution amendment required before merge (new Routing Layer carve-out supersedes the
  "bridge, not permanent" clause of the Interim Static Hosting carve-out)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Demo-First Delivery | PASS | Presenter workflow unchanged; single-port access is simpler |
| II. Compose-Orchestrated Reproducibility | PASS | nginx added as compose service; fresh clone + `docker compose up` still works |
| III. Testcontainers Over Mocks | PASS | No new Go boundary tests; no changes to existing tests |
| IV. Visible-in-the-Browser DoD | PASS | End-to-end browser validation required before this is done |
| V. Vendored-Code Hygiene | PASS | No third-party assets vendored; nginx config and Dockerfile only |
| Technology Stack (new runtime) | **AMENDMENT REQUIRED** | nginx is a new permanent runtime dependency alongside Go. The existing "Interim Static Hosting carve-out" permits nginx only while no Go backend exists; this feature adds nginx as a permanent routing layer alongside an existing Go backend. A new "Routing Layer carve-out" must be added to the constitution at version 1.4.0 before implementation begins. See `constitution-amendment.md` in this spec directory. |

**Gate outcome**: ONE violation requiring a constitution amendment. Amendment drafted in
`specs/014-nginx-front-door/constitution-amendment.md`. Must be applied to
`.specify/memory/constitution.md` before the implementation task list is executed.

## Project Structure

### Documentation (this feature)

```text
specs/014-nginx-front-door/
├── plan.md                       # This file
├── research.md                   # Phase 0: nginx SSE, DHI image, volume strategy
├── contracts/
│   └── routing-table.md          # nginx path → upstream mapping
├── quickstart.md                 # Validation scenarios
├── constitution-amendment.md     # Amendment text for v1.4.0 (apply before implementation)
└── tasks.md                      # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
nginx/
├── Dockerfile       # new — builds nginx image from dhi.io/nginx:1-alpine3.24,
│                    #        copies frontend/game + frontend/leaderboard into image
└── nginx.conf       # new — routing rules: static files, proxy rules per contracts/routing-table.md

docker-compose.yml   # modified — add nginx service; update app SCORES_SERVICE_URL,
                     #            COMMITS_SERVICE_URL env vars; nginx ports: ["80:80"]

ngrok.yml            # modified — addr: http://nginx:80  (was: http://app:8081)

.specify/memory/constitution.md   # modified — apply v1.4.0 amendment from constitution-amendment.md
```

**Structure Decision**: Single new `nginx/` directory at the repo root, mirroring the `commits-service/`
and `scores-service/` pattern (each service owns its own directory). No Go code changes anywhere.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| nginx as permanent co-resident with Go (new carve-out) | Demo requires single public entry point; Go app has two internal ports for gate/ungated split that cannot be served by a single listener; nginx is the standard solution for this routing pattern | Merging both listeners into one Go port would require rearchitecting the gate middleware — larger scope, higher risk, violates the "minimum change for maximum effect" principle of Principle I |
