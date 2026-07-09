# Implementation Plan: Secure Microservice Port Exposure

**Branch**: `020-secure-service-port-exposure` | **Date**: 2026-07-09 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/020-secure-service-port-exposure/spec.md`

## Summary

The three microservices (`commits-service`, `scores-service`, `qr-service`) currently publish host-reachable ports (8082, 8083, 8084), allowing any client on the demo host to bypass the nginx authentication gate. The fix is a Compose-only change: replace `ports:` with `expose:` on all three services, matching the pattern already used by `redis`. No Go service code changes are required. All nginx-fronted access paths remain fully functional; only direct host-to-microservice connections are closed.

## Technical Context

**Language/Version**: Docker Compose (compose spec, v2 format)

**Primary Dependencies**: Docker Compose, Docker Engine

**Storage**: N/A — no storage changes

**Testing**: Manual validation against the running compose stack; no automated test framework applies to compose configuration

**Target Platform**: Linux (Docker Desktop / Docker Engine on demo host)

**Project Type**: Compose-configuration change (infrastructure only)

**Performance Goals**: N/A

**Constraints**: Change must not break any nginx-proxied access path; `docker compose up` must reach a demoable state with no additional flags

**Scale/Scope**: Single file change (`docker-compose.yml`), three service blocks, plus inline documentation updates

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Demo-First Delivery | **PASS** | No visible change to any demo path. All attendee-facing flows (QR, leaderboard, scores) continue through nginx on port 80. |
| II. Compose-Orchestrated Reproducibility | **PASS** | Change stays within `docker-compose.yml`. `docker compose up` continues to reach a demoable state without host-side prerequisites. Internal service-to-service routing is unchanged (Docker Compose DNS is unaffected by `expose:` vs `ports:`). |
| III. Testcontainers over Mocks | **N/A** | No Go code changes. |
| IV. Visible-in-the-Browser Definition of Done | **PASS** | Done condition is: all existing browser-visible flows work after applying the change, AND direct host port connections are refused. Both are observable. |
| V. Vendored-Code Hygiene | **N/A** | No third-party assets introduced. |
| Stack Change Gate | **PASS** | No new runtime dependency introduced. `expose:` is a built-in Docker Compose directive. |
| Permanent Routing Layer carve-out | **PASS** | Reinforces the principle: nginx remains the sole public ingress. This change removes the accidental bypass that undermined the carve-out. |

**Gate result**: All applicable gates pass. No constitution amendment required.

## Project Structure

### Documentation (this feature)

```text
specs/020-secure-service-port-exposure/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created by /speckit-plan)
```

### Source Code Changes

```text
docker-compose.yml       # Single file; three service blocks modified
```

No new files are created in the source tree. No existing Go modules, nginx config, or other files are touched.

## Complexity Tracking

> No constitution violations — section not required.
