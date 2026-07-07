---
description: "Task list for K8s Manifest Generation via Compose Bridge"
---

# Tasks: K8s Manifest Generation via Compose Bridge

**Input**: Design documents from `specs/009-k8s-compose-bridge/`

**Prerequisites**: plan.md ✅, spec.md ✅

**Tests**: No automated test tasks — manual browser verification is the DoD per Principle IV and plan.md.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Exact file paths in all task descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the `k8s/` directory scaffold so all subsequent tasks have a clear target.

- [x] T001 Create `k8s/` directory with empty subdirectory `k8s/manifests/` and stub files: `k8s/generate.sh` (empty), `k8s/deploy.sh` (empty), `k8s/teardown.sh` (empty), `k8s/README.md` (heading only), `k8s/Dockerfile` (empty)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core artifacts that every user story depends on. No story work can begin until this phase is complete.

**⚠️ CRITICAL**: US2 (deploy) cannot proceed without a buildable K8s-specific image; US1 (generate) needs the secrets example to confirm the manifest baseline.

- [x] T002 Create `k8s/Dockerfile` — multi-stage build from repo root that embeds frontend: stage 1 copies `app/go.mod`, `app/go.sum`, `app/` source, builds binary from `dhi.io/golang:1.25-alpine-dev`; stage 2 uses `dhi.io/static:20260611-alpine3.24`, copies binary to `/app` and `frontend/game/` to `/frontend`; ENTRYPOINT `["/app"]`
- [x] T003 [P] Create `k8s/manifests/app-secrets.yaml.example` — K8s Secret template (not applied as-is) with apiVersion, kind: Secret, metadata.name: whale-runner-secrets, stringData keys: GRANT_COOKIE_SECRET and LEADERBOARD_API_SECRET, each with placeholder value `CHANGE_ME`

**Checkpoint**: k8s/Dockerfile buildable from repo root; secrets template committed.

---

## Phase 3: User Story 1 — Generate K8s Manifests (Priority: P1) 🎯 MVP

**Goal**: A developer can run `k8s/generate.sh` to produce a committed set of Kubernetes manifests for the redis and app services from `docker-compose.yml`.

**Independent Test**: Run `k8s/generate.sh` from the repo root; confirm `k8s/manifests/` contains YAML files for redis and app with no ngrok manifest; re-run and confirm idempotency.

### Implementation for User Story 1

- [x] T004 [US1] Implement `k8s/generate.sh`: shebang + `set -euo pipefail`; prerequisite check for `compose-bridge` (exit 1 with install hint if missing); run `compose-bridge convert -f docker-compose.yml --output k8s/manifests/` (or equivalent Docker plugin form); remove any ngrok-related YAML from `k8s/manifests/`; patch app Deployment manifest to set `image: whale-runner:k8s-local` and `imagePullPolicy: Never`; make script executable (`chmod +x`)
- [x] T005 [US1] Hand-author committed baseline manifests in `k8s/manifests/` if compose-bridge is not available in the current environment — create `k8s/manifests/redis-deployment.yaml` (Deployment, 1 replica, image `dhi.io/redis:8-alpine`, command `["redis-server","/etc/redis/redis.conf","--protected-mode","no"]`, emptyDir volume), `k8s/manifests/redis-service.yaml` (ClusterIP, port 6379), `k8s/manifests/app-deployment.yaml` (Deployment, 1 replica, image `whale-runner:k8s-local`, imagePullPolicy: Never, env from Secret `whale-runner-secrets` for GRANT_COOKIE_SECRET and LEADERBOARD_API_SECRET, REDIS_ADDR=redis:6379, FRONTEND_DIR=/frontend, APP_WEB_PORT=8080, APP_GATED_PORT=8081, NGROK_API_URL="" ), `k8s/manifests/app-service.yaml` (ClusterIP, port 8080)
- [x] T006 [US1] Review all four manifests in `k8s/manifests/` for correctness: verify redis `command` override is present, app image is `whale-runner:k8s-local` with `imagePullPolicy: Never`, app Service exposes port 8080, no ngrok YAML exists, all env var names match those in `docker-compose.yml`
- [x] T007 [US1] Write `k8s/README.md` Prerequisites and Manifest Generation sections: list required tools (Docker Desktop, kind, kubectl, compose-bridge) with install links; document `k8s/generate.sh` usage; note ngrok exclusion; note image-loading requirement (FR-009)

**Checkpoint**: `k8s/manifests/` contains four committed YAML files; README covers prerequisites and generation.

---

## Phase 4: User Story 2 — Deploy to Local Kind Cluster (Priority: P2)

**Goal**: A developer can run `k8s/deploy.sh` to go from zero to a running app in a local Kind cluster, with both redis and app pods reaching `Running` state.

**Independent Test**: With Kind installed, run `k8s/deploy.sh`; confirm cluster `whale-runner` exists, both pods reach `Running`, and `kubectl port-forward svc/app 8080:8080` exposes the game in a browser at `http://localhost:8080`.

### Implementation for User Story 2

- [x] T008 [US2] Implement `k8s/deploy.sh`: shebang + `set -euo pipefail`; prerequisite checks for `kind` and `kubectl` (exit 1 with install hints); accept optional `--cluster-name NAME` flag (default: `whale-runner`); create Kind cluster if `kind get clusters` does not include the name; build image `docker build -t whale-runner:k8s-local -f k8s/Dockerfile .` from repo root; `kind load docker-image whale-runner:k8s-local --name <cluster>`; pull and load `dhi.io/redis:8-alpine` via `docker pull` + `kind load docker-image`; create K8s Secret `kubectl create secret generic whale-runner-secrets --from-literal=GRANT_COOKIE_SECRET="${GRANT_COOKIE_SECRET:-dev-only-k8s}" --from-literal=LEADERBOARD_API_SECRET="${LEADERBOARD_API_SECRET:-dev-only-k8s}" --dry-run=client -o yaml | kubectl apply -f -`; `kubectl apply -f k8s/manifests/`; `kubectl wait --for=condition=Ready pod -l app=app --timeout=120s` and `kubectl wait --for=condition=Ready pod -l app=redis --timeout=120s`; print success message with `kubectl port-forward svc/app 8080:8080` command; make script executable
- [x] T009 [US2] Update `k8s/README.md` with Deployment section: Kind cluster creation (manual and via deploy.sh), environment variables for secrets (GRANT_COOKIE_SECRET, LEADERBOARD_API_SECRET), `k8s/deploy.sh` usage, accessing the app via `kubectl port-forward svc/app 8080:8080`, troubleshooting for `ImagePullBackOff` and `ErrImageNeverPull`

**Checkpoint**: `k8s/deploy.sh` creates cluster, loads images, applies manifests, pods reach Running; README covers the full deploy flow.

---

## Phase 5: User Story 3 — Teardown Local K8s Deployment (Priority: P3)

**Goal**: A developer can run `k8s/teardown.sh` to cleanly remove all K8s resources, with an optional `--delete-cluster` flag to also delete the Kind cluster.

**Independent Test**: After a successful deploy, run `k8s/teardown.sh`; confirm all app-related pods and services are removed; run with `--delete-cluster` and confirm `kind get clusters` no longer lists `whale-runner`.

### Implementation for User Story 3

- [x] T010 [US3] Implement `k8s/teardown.sh`: shebang + `set -euo pipefail`; accept optional `--delete-cluster` flag and optional `--cluster-name NAME` (default: `whale-runner`); check if cluster exists (exit 0 with "nothing to do" if not); `kubectl delete -f k8s/manifests/ --ignore-not-found=true`; `kubectl delete secret whale-runner-secrets --ignore-not-found=true`; if `--delete-cluster` flag set: `kind delete cluster --name <cluster>`; print completion status; make script executable
- [x] T011 [US3] Update `k8s/README.md` with Teardown section: `k8s/teardown.sh` usage, `--delete-cluster` flag, expected output

**Checkpoint**: All three scripts are complete; README is a self-contained guide from prerequisites to teardown.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Error handling consistency, constitution gate, and end-to-end validation.

- [x] T012 [P] Harden error handling across all three scripts in `k8s/generate.sh`, `k8s/deploy.sh`, `k8s/teardown.sh`: ensure each missing-prerequisite check emits the tool name, a one-line install hint, and exits with code 1; ensure each script prints a clear success/failure summary line as its last output
- [x] T013 Update `.specify/memory/constitution.md` to add K8s/Kind carve-out to the Technology Stack section (required per Constitution Check gate in plan.md): add a sentence under Orchestration noting that Kind is an optional local Kubernetes deployment target for demonstrating compose→k8s portability, does not replace docker compose for development or primary demo use, and requires a compose-bridge conversion step; bump version from 1.1.0 to 1.2.0 and update Last Amended date to 2026-07-07; add a Sync Impact Report at the top
- [ ] T014 End-to-end validation per Principle IV: ⚠️ REQUIRES HOST — kind/kubectl/compose-bridge not available in sandbox; run manually on host with Docker Desktop installed from a clean state run `k8s/generate.sh`, then `k8s/deploy.sh`, then `kubectl port-forward svc/app 8080:8080`, open `http://localhost:8080` in a browser, confirm the game loads and leaderboard is accessible at `/leaderboard`, then run `k8s/teardown.sh --delete-cluster` and confirm clean state

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2 — delivers independently testable manifest generation
- **US2 (Phase 4)**: Depends on Phase 2 and US1 manifests — requires manifests to exist
- **US3 (Phase 5)**: Depends on US2 being implementable (tests teardown of what US2 deploys)
- **Polish (Phase 6)**: Depends on all user story phases

### User Story Dependencies

- **US1 (P1)**: Depends on Foundational only; independently testable without Kind
- **US2 (P2)**: Depends on US1 manifests being committed in `k8s/manifests/`
- **US3 (P3)**: Logically depends on US2 (teardown of US2 deploy); script itself is independent

### Within Each User Story

- US1: generate.sh (T004) → manifests baseline (T005) → review/correct (T006) → README (T007)
- US2: deploy.sh (T008) → README (T009)
- US3: teardown.sh (T010) → README (T011)

### Parallel Opportunities

- T002 and T003 can run in parallel (different files, both in Phase 2)
- T012 and T013 can run in parallel (different files, both in Phase 6)
- T005 and T007 can start as soon as T004 is complete (different files)

---

## Parallel Example: User Story 1

```bash
# After T004 (generate.sh) is done, these can run in parallel:
Task T005: "Hand-author manifests in k8s/manifests/"
Task T007: "Write k8s/README.md prerequisites and generation sections"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001)
2. Complete Phase 2: Foundational (T002, T003)
3. Complete Phase 3: User Story 1 (T004–T007)
4. **STOP and VALIDATE**: `k8s/generate.sh` produces correct manifests; manifests committed and reviewable
5. Share for review — manifests can be used even without Kind

### Incremental Delivery

1. Setup + Foundational → directory structure and Dockerfile ready
2. US1 → Manifests generated and committed; README covers prerequisites
3. US2 → Full deploy-to-Kind flow; app verified in browser
4. US3 → Clean teardown completes the lifecycle
5. Polish → Constitution amended; end-to-end validated

---

## Notes

- [P] tasks = different files, no shared-state dependencies
- [Story] label traces each task to its user story
- US1 is independently valuable: committed manifests can be used on any K8s cluster without Kind
- US2 requires Docker Desktop + Kind + internet access to pull DHI Redis image
- T013 (constitution amendment) is a ship-gate per plan.md Constitution Check — do not mark feature Done until complete
- Verify the browser (`http://localhost:8080`) responds before reporting T014 complete (Principle IV)
