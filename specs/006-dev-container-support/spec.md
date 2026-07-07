# Feature Specification: Dev Container Support

**Feature Branch**: `006-dev-container-support`

**Created**: 2026-07-07

**Status**: Draft

**Input**: User description: "specify issue 2 within issues.md — Add support for developing, testing, and running the whole app inside a dev container, with matching VS Code config."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Open Project in Dev Container (Priority: P1)

A developer clones the repository and opens it in VS Code. They choose "Reopen in Container" and within a few minutes have a fully functional development environment: Go toolchain, all project dependencies, Docker CLI, and the compose stack accessible — without any host-side installs beyond Docker Desktop and git.

**Why this priority**: This is the core value of the feature. A developer should be able to go from fresh clone to working environment in one step. All other stories depend on this working.

**Independent Test**: Can be fully tested by cloning the repo, opening in VS Code, choosing "Reopen in Container", and verifying the terminal shows a working Go environment with `go version`, `docker compose version`, and the project builds successfully.

**Acceptance Scenarios**:

1. **Given** a fresh clone of the repo and VS Code with the Dev Containers extension, **When** the developer opens the project folder and selects "Reopen in Container", **Then** VS Code builds the dev container and opens a shell with Go, Docker CLI, and git all available on PATH.
2. **Given** the dev container is running, **When** the developer runs `go build ./...` from the project root, **Then** the build succeeds with no errors.
3. **Given** the dev container is running, **When** the developer runs `docker compose version`, **Then** a valid version string is returned, confirming Docker CLI access.

---

### User Story 2 - Run the App Stack from Inside the Dev Container (Priority: P2)

A developer inside the dev container runs `docker compose up` and the full application stack starts: game backend, leaderboard service, Redis, and ngrok tunnel. They can open the app in a host browser and play the game.

**Why this priority**: Being able to run and observe the app from within the dev container is essential for the development workflow. This validates that Docker socket access from inside the container is wired correctly.

**Independent Test**: Can be tested independently by running `docker compose up` from the dev container terminal and verifying the game loads in the host browser at the published URL.

**Acceptance Scenarios**:

1. **Given** the dev container is running, **When** the developer runs `docker compose up` from the project root inside the container, **Then** all services defined in `docker-compose.yml` start successfully.
2. **Given** `docker compose up` has completed, **When** the developer opens the ngrok URL in a browser on the host, **Then** the Whale Runner game loads and is playable.
3. **Given** the compose stack is running inside the dev container, **When** the developer stops it with `docker compose down`, **Then** all containers are stopped and removed cleanly.

---

### User Story 3 - Run Tests from Inside the Dev Container (Priority: P3)

A developer runs the Go test suite from the dev container terminal. Tests that use Testcontainers to spin up Redis or other dependencies work correctly, because the dev container has access to the Docker daemon.

**Why this priority**: Test execution inside the dev container closes the loop on the development workflow. Developers should not need to leave the container to run tests.

**Independent Test**: Can be tested by running `go test ./...` inside the dev container and observing all tests pass, including those that use Testcontainers.

**Acceptance Scenarios**:

1. **Given** the dev container is running, **When** the developer runs `go test ./...` from the project root, **Then** all tests pass, including integration tests that use Testcontainers.
2. **Given** a test that uses Testcontainers to start a Redis container, **When** run inside the dev container, **Then** the test spins up a real Redis container (visible as a sibling container on the host daemon) and completes successfully.
3. **Given** the dev container is running, **When** the developer runs `go test ./...` against a subset package (e.g., `./app/internal/leaderboard/...`), **Then** test output is shown in the terminal with pass/fail status and timing.

---

### Edge Cases

- What happens when the host Docker Desktop is not running when "Reopen in Container" is triggered?
- What if the host environment (e.g. a locked-down corporate Linux machine) does not permit exposing `/var/run/docker.sock` into a container? Docker-in-Docker is the documented fallback in that case, but it runs privileged and is heavier — this edge case should be noted in developer documentation rather than handled automatically.
- What if VS Code's Dev Containers extension is not installed — is there a fallback or a clear error message?
- How does port forwarding work for the compose stack when services are started from inside the dev container?
- What happens if Testcontainers' Ryuk resource-reaper cannot reach the Docker daemon? (Typically a symptom of incorrect socket or host-override configuration, not a Ryuk bug.)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The project MUST include a `.devcontainer/devcontainer.json` configuration that VS Code's Dev Containers extension can use to build and open the project.
- **FR-002**: The dev container MUST include the Go toolchain at the version used by the project, available on PATH without additional manual steps.
- **FR-003**: The dev container MUST access the host Docker daemon via socket mount (Docker-outside-of-Docker), so that `docker compose up` and Testcontainers-based tests talk to the same daemon the host uses. Docker-in-Docker is explicitly not the chosen approach; it is only a fallback when the host environment prohibits socket exposure (see Edge Cases).
- **FR-004**: The dev container MUST NOT duplicate service definitions from `docker-compose.yml`; it MUST consume the existing compose services rather than redefining them.
- **FR-005**: The dev container MUST include common Go development VS Code extensions (language server, test runner, linter) pre-configured so developers get editor support immediately on container open.
- **FR-006**: A developer MUST be able to run the full test suite (`go test ./...`) from inside the dev container, including tests that use Testcontainers, with all tests passing.
- **FR-007**: The dev container MUST reach a usable state from a fresh clone using only Docker Desktop and git on the host — no additional host-side global installs required.
- **FR-008**: The existing `docker compose up` onboarding path MUST continue to work unchanged for developers who prefer not to use a dev container.
- **FR-009**: The dev container workspace MUST be mounted at the same absolute path that the project occupies on the host. Testcontainers forwards bind-mounts into the containers it spawns using the path as seen by the Docker daemon; if the in-container path differs from the host path, fixture directory mounts in tests will silently point at wrong locations.
- **FR-010**: The dev container MUST set `TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal` so that test processes running inside the container can reach services spawned by Testcontainers. Without this, connection attempts to `localhost` from within the container are refused because the spawned service is a sibling container on the host daemon, not a process on the container's own network.

### Key Entities

- **Dev Container Configuration**: The `.devcontainer/devcontainer.json` file and any supporting Dockerfile or compose override that defines the development container environment.
- **VS Code Settings**: Workspace-level settings and recommended extensions that activate inside the dev container.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can go from `git clone` to a running dev container with a working Go build in under 5 minutes on a machine with Docker Desktop already running.
- **SC-002**: 100% of existing tests pass when run inside the dev container (`go test ./...`), including Testcontainers-based integration tests.
- **SC-003**: The dev container configuration is a single `.devcontainer/` directory with no duplication of service definitions already in `docker-compose.yml`.
- **SC-004**: `docker compose up` still works without modification from outside the dev container — the dev container introduction does not break the existing onboarding path.
- **SC-005**: A developer unfamiliar with the project can open the dev container and have editor Go intelligence (autocomplete, go-to-definition, inline errors) active within 2 minutes of the container reaching a running state.

## Assumptions

- Docker Desktop (or an equivalent Docker daemon) is installed and running on the host machine; this is already a prerequisite for the project.
- Developers use VS Code as their primary editor. JetBrains or other IDE support is out of scope for this feature.
- Docker access inside the dev container uses the Docker-outside-of-Docker (DooD) pattern: the host's Docker socket is mounted into the container. This is Testcontainers' recommended approach and the lightest option — no privileged daemon, no image layer duplication. Docker-in-Docker is the documented fallback only for environments where socket exposure is not permitted (see Edge Cases).
- The `TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal` environment variable is required for Docker Desktop hosts (macOS, Windows, WSL2) and is baked into the dev container config from the start. Linux hosts with direct socket access may not need it, but the variable is harmless there.
- The Go version used in the dev container matches the version declared in the project's `go.mod` file.
- Port forwarding from compose services started inside the dev container to the host browser is handled by VS Code's automatic port forwarding — no additional configuration is assumed.
- Dev container configuration will not add a new runtime dependency to the compose stack; it is a developer-environment concern only.
