# Implementation Plan: Dev Container Support

**Branch**: `006-dev-container-support` | **Date**: 2026-07-07 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/006-dev-container-support/spec.md`

## Summary

Add a `.devcontainer/devcontainer.json` that lets VS Code open this project in a fully-equipped Go 1.25 development environment. Docker access uses the **Docker-outside-of-Docker (DooD)** pattern — the host's Docker socket is mounted into the container — so `docker compose up` and Testcontainers-based tests run against the host daemon without a privileged inner daemon. Two DooD-specific requirements are baked in unconditionally: the workspace is mounted at the same absolute path as on the host (required for Testcontainers bind-mount forwarding), and `TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal` is set in the container environment (required for test processes on Docker Desktop to reach Testcontainers-spawned sibling containers). No compose service definitions are duplicated; the existing `docker compose up` onboarding path is untouched.

## Technical Context

**Language/Version**: Go 1.25.0 (declared in `app/go.mod`; matched by `dhi.io/golang:1.25-alpine-dev` in production Dockerfile)

**Primary Dependencies**:
- `ghcr.io/devcontainers/features/docker-outside-of-docker:1` — DooD socket mount devcontainer feature
- `ghcr.io/devcontainers/features/go:1` pinned to `"version": "1.25"` — Go toolchain install (see research.md §Decision 2)
- `golang.go` VS Code extension — Go language server, formatter, test runner
- `ms-azuretools.vscode-docker` VS Code extension — container management from the editor

**Storage**: N/A — this feature introduces no new storage. Application state remains in Redis (unchanged).

**Testing**: `go test ./...` run from inside the dev container; existing Testcontainers-go tests (`github.com/testcontainers/testcontainers-go/modules/redis v0.43.0`) must pass without modification.

**Target Platform**: macOS and Linux hosts running Docker Desktop or an equivalent Docker daemon. Windows (WSL2) should work via Docker Desktop's WSL2 backend but is not an explicit validation target for this feature.

**Project Type**: Developer tooling configuration — a single `.devcontainer/devcontainer.json` file and no new source code or compose services.

**Performance Goals**: Dev container build time under 5 minutes on a cold start (no cached layers).

**Constraints**:
- Workspace MUST be mounted at the same absolute path as on the host (`${localWorkspaceFolder}` as both source and target) — Testcontainers DooD bind-mount requirement
- `TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal` MUST be set — DooD on Docker Desktop host-reachability requirement
- MUST NOT duplicate `docker-compose.yml` service definitions (Constitution Principle II, FR-004)
- Existing `docker compose up` onboarding path MUST remain unchanged (FR-008)

**Scale/Scope**: One new file (`.devcontainer/devcontainer.json`). No changes to `app/`, `docker-compose.yml`, or the frontend.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

**Principle I — Demo-First Delivery**: CONDITIONAL PASS. The dev container is a developer infrastructure feature and does not directly improve the live demo. Justified: reliable development tooling reduces the risk of regressions in demo-facing code and enables faster iteration on demo features. The constitution's "secondary" category explicitly acknowledges that code quality and developer infrastructure serve demo goals; this is squarely in that category.

**Principle II — Compose-Orchestrated Reproducibility**: PASS. The dev container is not a compose service and introduces no compose service definitions. FR-004 prohibits duplication; the design (a single `devcontainer.json` that calls `docker compose up` against the existing file) satisfies this. FR-008 preserves the existing `docker compose up` path unchanged.

**Principle III — Testcontainers Over Mocks**: PASS. This feature's primary test-related goal is making the existing Testcontainers tests work inside the dev container. DooD socket mount (FR-003), workspace path matching (FR-009), and `TESTCONTAINERS_HOST_OVERRIDE` (FR-010) together satisfy the requirement.

**Principle IV — Visible-in-Browser Definition of Done**: PASS. The quickstart validation guide documents the path from `git clone` → dev container → `docker compose up` → Whale Runner visible in the host browser. That browser observation is the definition of done.

**Principle V — Vendored-Code Hygiene**: PASS. No third-party assets are vendored. `devcontainer.json` references devcontainer features and VS Code extensions by URI; no code is copied into the repository.

**Technology Stack — Constitution Amendment**: NOT REQUIRED. The dev container base image is not a new compose service or runtime dependency. It is a developer tooling artifact that exists entirely outside the application's runtime stack.

## Project Structure

### Documentation (this feature)

```text
specs/006-dev-container-support/
├── plan.md              # This file
├── research.md          # Docker access pattern + base image decisions
├── data-model.md        # devcontainer.json schema and environment contract
├── quickstart.md        # End-to-end validation guide
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
.devcontainer/
└── devcontainer.json    # VS Code dev container configuration (only new file)
```

No changes to:
- `app/` (Go source, tests, go.mod, Dockerfile)
- `docker-compose.yml`
- `frontend/`
- `bin/`

**Structure Decision**: Single flat `.devcontainer/devcontainer.json` using the devcontainer features API. No supporting Dockerfile is needed — the Go toolchain is installed via the `ghcr.io/devcontainers/features/go:1` feature, which allows version pinning without maintaining a custom Dockerfile.

## Complexity Tracking

> No Constitution Check violations requiring justification.
