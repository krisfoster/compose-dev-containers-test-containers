# Tasks: Dev Container Support

**Input**: Design documents from `specs/006-dev-container-support/`

**Prerequisites**: plan.md ✓ spec.md ✓ research.md ✓ data-model.md ✓ quickstart.md ✓

**Tests**: No explicit test tasks — this feature's validation is live observation (Constitution Principle IV). Each user story phase ends with a validation task that exercises the running dev container, not a unit test.

**Note on parallelism**: Almost all tasks modify `.devcontainer/devcontainer.json`. Tasks marked [P] touch distinct top-level JSON keys (e.g., `features`, `customizations.vscode.extensions`, `customizations.vscode.settings`, `containerEnv`) and are safe to run in parallel once the Phase 1 skeleton is in place.

---

## Phase 1: Setup

**Purpose**: Create the `.devcontainer/` directory and a complete, well-structured skeleton so all subsequent tasks add to well-defined sections rather than creating them.

- [x] T001 Create `.devcontainer/devcontainer.json` with a complete skeleton: top-level keys `name` ("Whale Runner Dev"), `image`, `features` (empty object), `workspaceMount`, `workspaceFolder`, `containerEnv` (empty object), `forwardPorts` (empty array), `portsAttributes` (empty object), `customizations.vscode.extensions` (empty array), `customizations.vscode.settings` (empty object), `postCreateCommand` (empty string). All keys must be present so parallel tasks in Phase 3+ can fill them without structural conflicts.

---

## Phase 2: Foundational (Blocking Prerequisite)

**Purpose**: Add Docker socket access to the dev container. Without this, both US2 (compose stack) and US3 (Testcontainers) are completely blocked. US1 technically works without it but the DooD feature must be present before validating any user story.

**⚠️ CRITICAL**: No user story validation can proceed until this phase is complete.

- [x] T002 Add `"ghcr.io/devcontainers/features/docker-outside-of-docker:1": {}` to the `features` object in `.devcontainer/devcontainer.json`. This mounts the host's `/var/run/docker.sock` and installs the Docker CLI inside the container.

**Checkpoint**: Skeleton + DooD feature wired. User story implementation can now begin in parallel.

---

## Phase 3: User Story 1 — Open Project in Dev Container (Priority: P1) 🎯 MVP

**Goal**: Developer opens VS Code → "Reopen in Container" → working Go 1.25 environment with editor intelligence, no host-side setup beyond Docker Desktop and git.

**Independent Test**: `go version` shows 1.25.x, `docker version` shows both client and server, `go build ./...` exits 0, all from inside the dev container.

### Implementation for User Story 1

- [x] T003 [P] [US1] Set `image` to `"mcr.microsoft.com/devcontainers/base:bookworm"` and add `"ghcr.io/devcontainers/features/go:1": {"version": "1.25"}` to the `features` object in `.devcontainer/devcontainer.json`. The Go version must match the `go` directive in `app/go.mod` (currently `1.25.0`).
- [x] T004 [P] [US1] Add `["golang.go", "ms-azuretools.vscode-docker"]` to `customizations.vscode.extensions` in `.devcontainer/devcontainer.json`.
- [x] T005 [P] [US1] Add Go editor settings to `customizations.vscode.settings` in `.devcontainer/devcontainer.json`: `"editor.formatOnSave": true`, `"[go]": {"editor.defaultFormatter": "golang.go", "editor.formatOnSave": true}`, `"go.toolsManagement.autoUpdate": true`, `"go.testFlags": ["-v", "-count=1"]`.
- [x] T006 [US1] Set `postCreateCommand` to `"cd app && go mod download"` in `.devcontainer/devcontainer.json`. This pre-downloads Go module dependencies after container creation so the first build is fast.
- [x] T007 [US1] Validate User Story 1 against `specs/006-dev-container-support/quickstart.md` §"User Story 1" by opening the dev container in VS Code and running: `go version` (expect 1.25.x), `docker version` (expect client + server), `pwd` (expect host absolute path), `echo $TESTCONTAINERS_HOST_OVERRIDE` (will be checked again in US3 — can skip for now), `go build ./...` (expect exit 0). Report any failures before proceeding.

**Checkpoint**: At this point US1 is fully functional — dev container opens with Go toolchain + Docker CLI.

---

## Phase 4: User Story 2 — Run the App Stack from Inside the Dev Container (Priority: P2)

**Goal**: `docker compose up` from inside the dev container starts all compose services; Whale Runner game loads and is playable in the host browser.

**Independent Test**: `docker compose ps` shows redis and app running; http://localhost:8080 loads the landing page in the host browser.

### Implementation for User Story 2

- [x] T008 [US2] Populate `forwardPorts` with `[8080, 8081, 4040]` and populate `portsAttributes` in `.devcontainer/devcontainer.json`:
  ```json
  "portsAttributes": {
    "8080": {"label": "App (ungated)", "protocol": "http"},
    "8081": {"label": "App (gated)", "protocol": "http"},
    "4040": {"label": "ngrok admin UI", "protocol": "http"}
  }
  ```
  VS Code uses these to automatically forward ports and label them in the Ports panel.
- [x] T009 [US2] Validate User Story 2 against `specs/006-dev-container-support/quickstart.md` §"User Story 2": from the dev container terminal run `docker compose up`, verify `docker compose ps` shows both services running, open http://localhost:8080 in the host browser and confirm the Whale Runner landing page loads, then run `docker compose down` and verify clean teardown. Report any failures before proceeding.

**Checkpoint**: At this point US1 + US2 both work — dev container opens, compose stack runs, game is playable.

---

## Phase 5: User Story 3 — Run Tests from Inside the Dev Container (Priority: P3)

**Goal**: `go test ./...` passes inside the dev container, including Testcontainers-based tests that spin up real Redis containers via the host Docker daemon.

**Independent Test**: All packages show `ok` with no "connection refused" errors; during the run, `docker ps` on the host shows short-lived Redis sibling containers appearing and disappearing.

### Implementation for User Story 3

- [x] T010 [P] [US3] Set `workspaceMount` to `"source=${localWorkspaceFolder},target=${localWorkspaceFolder},type=bind"` and set `workspaceFolder` to `"${localWorkspaceFolder}"` in `.devcontainer/devcontainer.json`. This ensures the workspace path inside the container is identical to the host path — required for Testcontainers' bind-mount forwarding to work correctly (see `specs/006-dev-container-support/research.md` §Decision 3).
- [x] T011 [P] [US3] Add `"TESTCONTAINERS_HOST_OVERRIDE": "host.docker.internal"` to the `containerEnv` object in `.devcontainer/devcontainer.json`. Without this, test processes on Docker Desktop cannot reach services spawned by Testcontainers (they are sibling containers on the host daemon, not on the dev container's network — see `specs/006-dev-container-support/research.md` §Decision 4).
- [x] T012 [US3] Validate User Story 3 against `specs/006-dev-container-support/quickstart.md` §"User Story 3": rebuild the dev container to apply T010/T011 (Command Palette → `Dev Containers: Rebuild Container`), then run `go test ./... -v -count=1` from `app/`, verify all packages show `ok`, confirm no "connection refused" errors, and observe Redis sibling containers in `docker ps` on the host during the run. Report any failures before proceeding.

**Checkpoint**: All three user stories fully functional. The dev container is complete.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and final end-to-end validation.

- [x] T013 [P] Add a "Dev Container" section to `README.md` after the existing quick-start instructions. The section should: (1) state that VS Code with the Dev Containers extension is the recommended development environment, (2) describe the one-step `Reopen in Container` flow, (3) note that `docker compose up` and `go test ./...` both work inside the container, (4) link to `specs/006-dev-container-support/quickstart.md` for full validation steps. Keep it brief — this is an entry point, not a tutorial.
- [x] T014 Run the complete definition-of-done checklist from `specs/006-dev-container-support/quickstart.md` (the final checklist block) and confirm all items pass. This is the Constitution Principle IV gate: the change is not complete until the running app has been observed in the browser from inside the dev container.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user story validation
- **US1 (Phase 3)**: Depends on Phase 2 — can begin once T002 is complete
- **US2 (Phase 4)**: Depends on Phase 3 validation passing (T007) — requires working Docker CLI
- **US3 (Phase 5)**: Depends on Phase 4 validation passing (T009) — requires working compose stack for context, even though Testcontainers is independent at the code level
- **Polish (Phase 6)**: Depends on T012 passing — requires all three user stories working

### User Story Dependencies

- **US1 (P1)**: No story dependencies — validate independently after Phase 3
- **US2 (P2)**: Depends on US1 (needs working Docker CLI access) — builds on top of US1
- **US3 (P3)**: Depends on US1 (needs working Docker access) — US2 is not strictly required but should be validated first to confirm the Docker socket is routing correctly before testing Testcontainers

### Within Each Phase

- T003, T004, T005 (within Phase 3): All marked [P] — modify different JSON keys, safe to do in parallel once the T001 skeleton exists
- T010, T011 (within Phase 5): Both marked [P] — modify different JSON top-level keys
- T013 (Phase 6): Marked [P] — different file from T014 (README vs live validation)

---

## Parallel Example: User Story 1

Three tasks (T003, T004, T005) modify distinct sections of `.devcontainer/devcontainer.json` and can proceed simultaneously once T001 and T002 are complete:

```text
T003: features["ghcr.io/devcontainers/features/go:1"] → Go toolchain
T004: customizations.vscode.extensions → ["golang.go", "ms-azuretools.vscode-docker"]
T005: customizations.vscode.settings → formatOnSave, testFlags, etc.
```

## Parallel Example: User Story 3

```text
T010: workspaceMount + workspaceFolder → path matching
T011: containerEnv.TESTCONTAINERS_HOST_OVERRIDE → host override
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001)
2. Complete Phase 2: Foundational (T002) — CRITICAL
3. Complete Phase 3: US1 (T003–T007)
4. **STOP and VALIDATE**: `go version`, `go build ./...`, `docker version` all pass inside dev container
5. Ship if that's enough — the remaining stories add value but don't block the core dev workflow

### Incremental Delivery

1. Phase 1 + 2 → skeleton + DooD wired
2. Phase 3 (US1) → dev container opens with Go + editor intelligence → **Demo: open project in VS Code**
3. Phase 4 (US2) → compose stack runs → **Demo: game playable from inside dev container**
4. Phase 5 (US3) → Testcontainers tests pass → **Demo: full test suite green inside container**
5. Phase 6 → README updated, definition-of-done signed off

### Single-Developer Sequence

This feature has a single implementation file, so parallel team strategy doesn't apply. The recommended order is exactly the phase order: T001 → T002 → T003+T004+T005 (in any order) → T006 → T007 → T008 → T009 → T010+T011 (in any order) → T012 → T013 → T014.

---

## Notes

- **One file, low risk**: The entire implementation is `.devcontainer/devcontainer.json`. There is no risk of breaking existing source code, tests, or the compose stack.
- **Rebuild after changes**: Any change to `devcontainer.json` requires a dev container rebuild before the change takes effect (`Dev Containers: Rebuild Container` from the Command Palette).
- **Validation tasks are mandatory**: T007, T009, T012, T014 are not optional polish — they are the Constitution Principle IV gate for each user story. Do not skip them.
- **DinD fallback not implemented**: Docker-in-Docker is documented as a fallback in `research.md` but is not part of this implementation. If a developer's environment requires DinD, they can add it manually by swapping the feature URI.
- **Go version must stay in sync**: The `"version": "1.25"` in the Go feature must match the `go` directive in `app/go.mod`. If `go.mod` is updated, update `devcontainer.json` in the same commit.
