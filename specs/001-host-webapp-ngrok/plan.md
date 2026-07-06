# Implementation Plan: Host Web App with Public Ngrok Access

**Branch**: `001-host-webapp-ngrok` | **Date**: 2026-07-06 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-host-webapp-ngrok/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Bring the existing static frontend (`frontend/game/`) up under `docker compose` behind a minimal
static webserver, with an optional, profile-gated ngrok service that exposes it on a public HTTPS
URL. No application code is introduced: the webserver is an off-the-shelf static file server
serving the existing files unchanged, and the tunnel is the official ngrok agent container. Public
access is opt-in via a compose profile so local hosting always works even if the tunnel is
unavailable, satisfying the resilience requirement in the spec.

## Technical Context

**Language/Version**: N/A — no application code is authored for this feature; only container
configuration (Docker Compose YAML + a one-line static-file-server config).

**Primary Dependencies**: Docker Compose v2 (orchestration); `nginx:alpine` (static file serving);
`ngrok/ngrok:3` official image, pinned to major version 3 (public tunnel).

**Storage**: N/A — static files only, no database or persistent state.

**Testing**: No unit tests (no custom code). Validation is a scripted/manual smoke check
(`docker compose config`, HTTP checks via `curl`, and an in-browser check per constitution
Principle IV), documented in `quickstart.md`.

**Target Platform**: Presenter's laptop (macOS/Linux/Windows) running Docker Desktop; Linux
containers only.

**Project Type**: Infrastructure/orchestration — a single `docker-compose.yml` plus minimal config,
no new source project or build step.

**Performance Goals**: Local reachability within 2 minutes of a cold `docker compose up` (SC-001);
public page load under 5 seconds from an external network (SC-002).

**Constraints**: No host-level installs beyond Docker Desktop + git; a free-tier ngrok account is
sufficient; the existing frontend already loads its dependencies from a CDN via an import map, so
no build step may be introduced.

**Scale/Scope**: One webserver container + one optional tunnel container; booth-demo scale (at most
a few dozen concurrent phone clients), no multi-instance or load-balancing needs.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Status | Notes |
|-----------|------|--------|-------|
| I. Demo-First Delivery | Change must improve the demo or unblock it | PASS | This is the prerequisite for any demo to be shown at all (local view) or shared with attendees (public URL); no invisible-internals work beyond that. |
| II. Compose-Orchestrated Reproducibility | Every runtime component defined as a compose service; `docker compose up` reaches a demoable state with no extra host installs | PASS | Webserver and ngrok are both compose services. Serving static files via `nginx:alpine` before a Go backend exists is permitted by the Technology Stack's "Interim Static Hosting carve-out" (constitution v1.1.0, added specifically for this feature). Per that carve-out, the `webserver` service MUST be retired in whichever future phase introduces a Go backend that can serve `frontend/game/` directly — tracked as TODO(INTERIM-HOSTING-SUNSET) in the constitution. |
| III. Testcontainers Over Mocks | Go unit tests crossing a boundary use Testcontainers | N/A | No Go code, no tests crossing a boundary, introduced by this feature. |
| IV. Visible-in-the-Browser Definition of Done | Done = observed in a browser against the compose stack, with a documented repeatable path | PASS | `quickstart.md` documents the fresh-clone-to-browser path for both local and public access. |
| V. Vendored-Code Hygiene | Vendored code/assets carry attribution | N/A | This feature only references public Docker Hub images (`nginx:alpine`, `ngrok/ngrok`) by tag; it does not vendor any third-party code or assets into the repo. |

No violations. Complexity Tracking is not needed.

*Re-checked after Phase 1 design (`research.md`, `data-model.md`, `quickstart.md`): no new
services, dependencies, or vendored assets were introduced. Table above still holds unchanged.*

## Project Structure

### Documentation (this feature)

```text
specs/001-host-webapp-ngrok/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

No `contracts/` directory is generated for this feature: it does not define any new API, CLI, or
UI contract. The one external interface involved (ngrok's own agent API on port 4040) already
exists and is only consumed for URL discovery, not authored here.

### Source Code (repository root)

```text
docker-compose.yml         # New. Defines the webserver and (profile-gated) ngrok services.
.env.example                # New. NGROK_AUTHTOKEN and any other tunable values.
webserver/
└── nginx.conf               # New. Minimal static-file-serving config, no app logic.

frontend/game/                # Existing, unchanged. Served as-is by the webserver service.
├── index.html
├── script.js
├── style.css
├── favicon.ico
└── ...
```

**Structure Decision**: Single-project, infra-only layout. There is no `backend/` yet (none exists
in the repository), so Option 2's split is not applicable. The feature adds exactly one compose
file, one small webserver config, and an `.env.example` at the repo root; `frontend/game/` is
consumed read-only as the content the webserver serves and is not modified.

## Complexity Tracking

*No Constitution Check violations — table intentionally omitted.*
