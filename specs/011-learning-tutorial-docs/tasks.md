---

description: "Task list for Learning Tutorial and Documentation (011-learning-tutorial-docs)"
---

# Tasks: Learning Tutorial and Documentation

**Input**: Design documents from `specs/011-learning-tutorial-docs/`

**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, quickstart.md ✅

**Tests**: Not applicable — this is a pure documentation feature with no runtime code changes.

**Organization**: Tasks are grouped by user story. US1 (inline comments) and US2 (README restructure) are fully independent and can run in parallel. All US1 file tasks are parallel (different files).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to
- No story label = Setup or Polish phase

---

## Phase 1: Setup (Baseline)

**Purpose**: Establish a clean baseline before any changes. Confirms all tests pass so regressions are detectable.

- [x] T001 Run `go test ./...` from `app/` and confirm all tests pass — establishes baseline before any file edits

**Checkpoint**: Baseline confirmed. US1 and US2 can now proceed in parallel.

---

## Phase 2: User Story 1 — Inline Source Code Comments (Priority: P1) 🎯 MVP

**Goal**: Every Docker-adjacent source file and config gains beginner-oriented explanatory comments covering Docker Compose, DHI, Testcontainers, and Redis concepts — so a reader unfamiliar with Docker can understand each file's role without external reference material.

**Independent Test**: Open each file listed in T002–T008 and verify a Docker beginner can understand what it does and why key choices were made, without consulting external docs. All seven files are independently verifiable.

### Implementation for User Story 1

- [x] T002 [P] [US1] Add educational comment blocks to `docker-compose.yml` — top-of-file block explaining Docker Compose purpose; per-service blocks for `redis` (what Redis is, role in the app, DHI image, `protected-mode` workaround explained for beginners), `app` (build context, self-contained image), and `ngrok` (profiles concept, optional service); inline comment on `expose` vs `ports` and `REDIS_ADDR=redis:6379` as inter-service DNS
- [x] T003 [P] [US1] Add educational comments to `app/Dockerfile` — expand existing comments to explain multi-stage builds (build stage compiles; final stage holds only the binary), explain `dhi.io/golang` and `dhi.io/static` as Docker Hardened Images (hardened, minimal base images from `dhi.io` registry, reduced attack surface)
- [x] T004 [P] [US1] Add container DNS comment to `app/main.go` — at the `REDIS_ADDR` line in `loadConfig()`, add a comment explaining that `redis:6379` is a Docker Compose inter-service hostname resolved by Docker's built-in DNS, not a hard-coded IP
- [x] T005 [P] [US1] Expand Testcontainers comment in `app/internal/gate/window.go` — extend the `WindowStore` interface comment to explain the interface-and-fake pattern: the interface enables testing handlers with an in-memory fake; the real Redis implementation (`RedisWindowStore`) is tested separately using Testcontainers-go, which spins up a real Redis container so tests exercise actual Redis behaviour rather than a mock
- [x] T006 [P] [US1] Add Testcontainers explanation block to `app/internal/gate/window_test.go` — expand `newTestRedisStore` comment to explain what Testcontainers-go does (starts a real Docker container as part of the test, tears it down on completion), why a real Redis container is used here (mocks pass tests that break when Redis command behaviour changes), and note the DHI `protected-mode` workaround applied to the test container
- [x] T007 [P] [US1] Add Redis Streams and Testcontainers comments to `app/internal/leaderboard/store.go` — add comment before `scoresStreamKey` explaining what a Redis Stream is (append-only log of entries, each entry is immutable — enabling the never-overwrite requirement); expand `ScoreStore` interface comment to mention Testcontainers exactly as done in `window.go` (T005 sets the pattern)
- [x] T008 [P] [US1] Add Testcontainers explanation block to `app/internal/leaderboard/store_test.go` — same pattern as T006: expand the store's test helper comment to explain Testcontainers-go, why real Redis is used, and the DHI `protected-mode` workaround

**Checkpoint**: US1 complete. Every Docker-adjacent source file has beginner-oriented explanatory comments. Verify by opening each file and checking FR-001 through FR-004 acceptance scenarios.

---

## Phase 3: User Story 2 — README Tutorial Restructure (Priority: P1)

**Goal**: The README becomes a learning-outcomes-first tutorial covering six Docker technologies, ending with a summary. All existing operational content is preserved. `k8s/README.md` gains a brief educational intro.

**Independent Test**: Read the restructured README end-to-end. Confirm: (1) "What you will learn" lists 6 technologies; (2) each section tells the reader to run a command or look at a file — never to write code; (3) tutorial sections progress logically; (4) "What you learned" recap names all 6 technologies; (5) `docker compose up` is findable in under 60 seconds. `k8s/README.md` can be verified independently.

### Implementation for User Story 2

- [x] T009 [US2] Restructure `README.md` skeleton — add "What you will learn" section (list all 6 Docker technologies: Docker Compose, Dockerfile & multi-stage builds, Docker Hardened Images, Testcontainers, Dev Containers, Kubernetes); add six named tutorial section headings in order (Run the App, Understanding Docker Compose, Understanding the Dockerfile, Understanding Testcontainers, Developing inside a Dev Container, Deploying to Kubernetes); add "What you learned" placeholder section; move existing "Develop with Claude" SBX section to the end under a clearly marked "Optional: Advanced Development Setup" heading — do not delete any existing content at this stage, only rearrange headings and add the new structural elements
- [x] T010 [US2] Write "Run the App" tutorial section content in `README.md` — absorb the existing "Running the app" operational content (prerequisites, `docker compose up -d`, stop command, public access, leaderboard); add a short paragraph after the start command explaining that Docker Compose just started three services (redis, app, ngrok-optional) defined in `docker-compose.yml` and that the next section explains how
- [x] T011 [US2] Write "Understanding Docker Compose" tutorial section content in `README.md` — instruct reader to open `docker-compose.yml`; explain what Docker Compose does (one file, one command to define and run a multi-service app); call out: services and dependencies, inter-service DNS (`redis:6379`), `expose` vs `ports`, environment variables as runtime configuration, profiles for optional services; each point should say "look at line X / the `service:` block" to keep it hands-on; do not reproduce the full file contents
- [x] T012 [US2] Write "Understanding the Dockerfile" tutorial section content in `README.md` — instruct reader to open `app/Dockerfile`; explain multi-stage builds (build stage produces the binary, final stage holds only the output — smaller and more secure); explain Docker Hardened Images (`dhi.io/` prefix, hardened configs, reduced attack surface); keep to 3-4 sentences per concept
- [x] T013 [US2] Write "Understanding Testcontainers" tutorial section content in `README.md` — instruct reader to open `app/internal/gate/window_test.go` and look at `newTestRedisStore`; explain Testcontainers-go (starts a real Docker container as part of a Go test, automatically tears it down); explain why real Redis beats a mock for boundary tests (mocks can pass while prod breaks on command argument changes); cross-reference to `app/internal/leaderboard/store_test.go` as a second example
- [x] T014 [US2] Write "Developing inside a Dev Container" tutorial section content in `README.md` — absorb the existing "Develop inside a Dev Container" operational content (VS Code setup, what you get); add a short intro explaining what a dev container is (a Docker container that IS the development environment — Go toolchain, Docker CLI, Testcontainers all pre-installed, defined as code); keep the existing VS Code instructions intact
- [x] T015 [US2] Write "Deploying to Kubernetes" tutorial section content in `README.md` — brief intro (2-3 sentences: what Kubernetes does, that this project includes a Helm chart in `k8s/`, that Docker Desktop's built-in Kubernetes makes a local cluster available without extra infrastructure); then direct the reader to `k8s/README.md` for the full deployment commands; do NOT mention docker compose bridge or Helm chart generation from compose
- [x] T016 [US2] Write "What you learned" summary section in `README.md` — one sentence recap per Docker technology in the order introduced: Docker Compose, Dockerfile/multi-stage builds, Docker Hardened Images, Testcontainers, Dev Containers, Kubernetes; each sentence should complete "You now know that [technology] [what it does / why it matters]"
- [x] T017 [P] [US2] Add brief educational intro to `k8s/README.md` — insert 2-4 sentences before the existing "Prerequisites" section: explain what Kubernetes is (an orchestration system that runs containers across one or more nodes), what Helm does (a package manager for Kubernetes — the `k8s/` directory is a Helm chart), and that Docker Desktop's built-in Kubernetes provides a local single-node cluster with no extra installation; keep it concise, no deep dives into Kubernetes internals

**Checkpoint**: US2 complete. Verify by running through all five acceptance scenarios in the quickstart.md Validation 2 checklist.

---

## Phase 4: Polish & Verification

**Purpose**: Confirm no regressions and that all acceptance criteria pass.

- [x] T018 [P] Run `go test ./...` from `app/` — confirm all tests still pass; no runtime changes should have been made, but this is the safety net
- [x] T019 [P] Walk through `specs/011-learning-tutorial-docs/quickstart.md` Validation 1–5 checklists — tick off each item; note any that fail for remediation before marking this phase done
- [x] T020 [P] Update `specs/011-learning-tutorial-docs/quickstart.md` — add Validation 6 for `k8s/README.md`: confirm it has a 2-4 sentence intro before "Prerequisites" explaining Kubernetes and Helm, and that no Compose Bridge mention appears anywhere in either README

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — run immediately
- **US1 (Phase 2)**: Can start after T001; all T002–T008 are independent (different files) and run in parallel
- **US2 (Phase 3)**: Can start after T001; T009–T016 edit `README.md` in sequence; T017 (`k8s/README.md`) is parallel with any of T009–T016
- **Polish (Phase 4)**: Depends on all Phase 2 and Phase 3 tasks complete

### User Story Dependencies

- **US1 (P1)**: Independent of US2 — different files entirely
- **US2 (P1)**: Independent of US1 — however, US2 tutorial sections reference code that US1 annotates; running them together delivers the best reader experience
- **US3 (P2)**: Verified by Validation 2 in quickstart.md (US2 acceptance scenario 2 + US3 acceptance scenario 1) — no separate implementation tasks; it is a quality constraint on US2

### Within Each Phase

- Phase 2 (US1): All tasks parallel — different source files
- Phase 3 (US2): T009 first (skeleton); T010–T016 sequential (same README.md); T017 parallel with any of T010–T016 (different file)

### Parallel Opportunities

```bash
# After T001 baseline:
# US1 tasks (T002-T008): all in parallel — 7 different files
# US2 structural skeleton (T009): can start alongside US1

# Within US2, once README skeleton (T009) is done:
# T010 → T011 → T012 → T013 → T014 → T015 → T016 (sequential, same file)
# T017 (k8s/README.md): parallel with any README step above
```

---

## Implementation Strategy

### MVP First (User Stories 1 and 2, both P1)

1. Complete Phase 1: baseline check (T001)
2. Complete Phase 2: all inline comments (T002–T008) in parallel
3. Complete Phase 3: README restructure (T009–T017) in sequence
4. **STOP and VALIDATE**: Walk quickstart.md checklists
5. Complete Phase 4: final verification

### Incremental Delivery

Since US1 and US2 are independent, either can ship first:
- Ship US1 alone → codebase is self-documenting; README still needs restructure
- Ship US2 alone → README tutorial is great; inline comments reinforce it
- Ship both together → full learning experience

Recommended: complete both before shipping (they reinforce each other).

### Solo Strategy (single developer)

1. T001: baseline
2. T002–T008: knock out all inline comments in one pass (all parallel)
3. T009: restructure README skeleton
4. T010–T016: write each tutorial section in order
5. T017: update k8s/README.md
6. T018–T020: verify

---

## Notes

- [P] tasks = different files, no dependencies — safe to run concurrently
- [Story] label maps each task to its user story for traceability
- No new Go source files are created; no `go.mod` changes; no runtime code changes
- Comments in Go files use `//` style; never add comment blocks that change package-level doc visibility
- README sections must not duplicate command content from `k8s/README.md` — link, don't copy
- The SBX/Claude workflow is not deleted from README — only moved to the end as optional
- FR-012: `docker compose bridge` and Helm chart generation from compose must never appear in README
