# Implementation Plan: Git Commits Microservice

**Branch**: `012-git-commits-microservice` | **Date**: 2026-07-08 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/012-git-commits-microservice/spec.md`

## Summary

Split the existing `handleCommits` handler out of `app/main.go` into a dedicated Go microservice
(`commits-service/`) that exposes structured commit data over both a REST endpoint and a
Server-Sent Events stream. The leaderboard page gains a React component (replacing the current
vanilla-JS commit section) that subscribes to the SSE stream for live updates, with automatic
polling fallback. React + ReactDOM are vendored as static files served by the existing `app`
service, so no build pipeline or CDN dependency is introduced.

## Technical Context

**Language/Version**: Go 1.25 (matching the existing `app` service)

**Primary Dependencies**:
- `github.com/go-git/go-git/v5` (already vendored in `crossywhale/app` — new module pulls same
  version; note the existing `app` go.mod can drop this dependency after the split)
- React 18 + ReactDOM 18 (UMD/production builds, vendored as static JS files; no npm build step)

**Storage**: None — the commits service reads directly from the `.git` directory mounted as a
read-only Docker volume (`./.git:/repo/.git:ro`), same as the existing `app` service.

**Testing**: Go standard testing + Testcontainers-go for any future tests crossing service
boundaries. The commits handler reads from the filesystem (git volume), not a network service, so
unit tests use a real on-disk git fixture repo (no container needed for unit tests; a container
would be needed if a future test exercises the HTTP SSE stream against a real repo clone).

**Target Platform**: Linux container (dhi.io/static base, same pattern as `app`)

**Performance Goals**: Return up to 20 commits in under 200 ms on the SSE initial connection;
subsequent SSE events are push-driven (no polling overhead server-side beyond a file-read on the
refresh interval).

**Constraints**: The commits service MUST NOT serve HTML. It MUST set CORS headers to allow the
leaderboard page (served by `app` on a different port) to fetch from it. It MUST be startable via
`docker compose up` with no additional host steps.

**Scale/Scope**: Booth scale — a handful of concurrent leaderboard viewers, at most 20 commits in
the response.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Assessment | Verdict |
|-----------|------------|---------|
| **I. Demo-First Delivery** | A live commit feed that updates in the browser without a reload is a visible demo improvement — the booth audience sees real-time code activity on the wall display. | PASS |
| **II. Compose-Orchestrated Reproducibility** | The new `commits-service` will be a named service in `docker-compose.yml`, started by `docker compose up` with no additional host-side steps. The git volume (`.git:/repo/.git:ro`) is re-used from the `app` service pattern. | PASS |
| **III. Testcontainers for Boundary Tests** | The commits handler reads from a volume-mounted git directory (filesystem, not a network boundary). Unit tests use an in-process git fixture repo. Any test that exercises the HTTP server layer (SSE stream, CORS headers) uses httptest — no external service boundary, so Testcontainers is not required for the commits service. | PASS |
| **IV. Visible-in-the-Browser DoD** | The leaderboard commits section must be exercised live in the browser against the compose stack before this feature is marked done. | GATE — required at ship |
| **V. Vendored-Code Hygiene** | React 18 + ReactDOM 18 are vendored as static JS files. ATTRIBUTION.md must be updated with author (Meta), source URL (react.dev), licence (MIT), and notation that no modifications were made. | GATE — required pre-ship |
| **Stack amendment: React** | React is not in the current constitution Technology Stack section ("Frontend: three.js r160 loaded via ES-module importmap"). Adding React as a frontend library for the leaderboard page constitutes a new frontend runtime dependency. A constitution amendment (MINOR version bump) is required before ship. | GATE — amendment required |

**Complexity Tracking**:

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| React not in constitution stack | User requirement ("react js component"); the leaderboard's commits section benefits from React's declarative rendering of the SSE event stream | Vanilla JS already exists in the leaderboard for standings — but managing SSE reconnection + state transitions cleanly in vanilla JS adds fragile imperative DOM code. React's component model keeps the commits section self-contained and testable. Constitution amendment resolves the violation formally. |
| New Go module (`commits-service/`) | The microservice needs its own build context and module path to satisfy the spec's architectural requirement | Adding the handler as a new package in `crossywhale/app` keeps it one process (not a microservice) and violates FR-001's explicit separation |

## Project Structure

### Documentation (this feature)

```text
specs/012-git-commits-microservice/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   ├── commits-openapi.yaml     # REST + SSE contract
│   └── commits-sse-contract.md  # SSE stream event spec
└── tasks.md             # Phase 2 output (/speckit-tasks — not created by /speckit-plan)
```

### Source Code (repository root)

```text
# New: commits microservice
commits-service/
├── Dockerfile           # Multi-stage, dhi.io/golang:1.25-alpine-dev → dhi.io/static
├── go.mod               # module crossywhale/commits-service
├── go.sum
├── main.go              # Entry point: HTTP server (REST + SSE)
└── internal/
    └── commits/
        ├── handler.go       # handleCommits (REST) + handleCommitsStream (SSE)
        └── handler_test.go  # Unit tests against on-disk git fixture

# Modified: frontend leaderboard assets
frontend/
└── leaderboard/          # New directory — served by app at /leaderboard-assets/
    ├── react.production.min.js   # Vendored React 18 UMD build
    ├── react-dom.production.min.js  # Vendored ReactDOM 18 UMD build
    └── commits-component.js      # ES module: React component for the commit feed

# Modified: existing app service
app/
├── main.go              # Remove handleCommits + gitRepoPath field + go-git import;
│                        # update leaderboardPageHTML to load React component;
│                        # add /leaderboard-assets/ file route
├── Dockerfile           # Add COPY frontend/leaderboard to image
└── go.mod               # Remove go-git/go-git dependency after handler removal

# Modified: compose
docker-compose.yml       # Add commits-service service; update app environment

# Modified: attribution
ATTRIBUTION.md           # Add React 18 + ReactDOM 18 entries
```
