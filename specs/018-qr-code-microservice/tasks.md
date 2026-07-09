---

description: "Task list for QR Code Microservice extraction"
---

# Tasks: QR Code Microservice

**Input**: Design documents from `specs/018-qr-code-microservice/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/qr-http-contract.md, quickstart.md

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to
- Exact file paths are included in all descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the `qr-service` directory skeleton so all subsequent tasks have a place to write files.

- [x] T001 Create directory tree `qr-service/` and `qr-service/internal/qrcode/` (no files yet — just the directories for Go package layout)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Initialize the Go module so `go build` and `go test` work in `qr-service/`. This unblocks all US1 implementation tasks.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T002 Create `qr-service/go.mod` declaring `module crossywhale/qr-service`, `go 1.25.0`, and `require github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e`; run `go mod tidy` inside `qr-service/` to generate `qr-service/go.sum`

**Checkpoint**: `qr-service/go.mod` and `qr-service/go.sum` present — implementation can now begin.

---

## Phase 3: User Story 1 — QR Code Rendered by Dedicated Service (Priority: P1) 🎯 MVP

**Goal**: The dynamic QR code at `/qr.png` is rendered by `qr-service` instead of the app process. The presenter's `/host` page continues to show a scannable QR code; behaviour at the browser is unchanged.

**Independent Test**: Start the compose stack; open `http://localhost/host` — a QR code image appears. `docker compose logs qr-service` shows it handled the render request.

### Implementation for User Story 1

- [x] T003 [P] [US1] Implement `qr-service/internal/qrcode/handler.go`: define `Handler` struct with `ServeHTTP` method; read `content` query param (return 400 if empty); read `size` param (default 320, clamp to [64, 1024]); call `qr.Encode(content, qr.Medium, size)` from `github.com/skip2/go-qrcode`; set `Content-Type: image/png`; write PNG bytes; return 500 on encode failure
- [x] T004 [P] [US1] Write `qr-service/internal/qrcode/handler_test.go`: table-driven tests covering — valid content+size returns 200 + decodable PNG; absent `content` returns 400; zero/negative `size` defaults to 320; oversized `size` is clamped to 1024; different content produces different PNG bytes
- [x] T005 [US1] Implement `qr-service/main.go`: `Config` struct with `ListenAddr` field; `loadConfig()` reading `QR_LISTEN_ADDR` env var (default `:8084`); `envOr` helper; `main()` wires `http.NewServeMux()` → `qrcode.Handler{}` at `/qr.png` and starts `http.Server` with `ReadTimeout: 5s` following the `commits-service/main.go` pattern (depends on T003)
- [x] T006 [P] [US1] Write `qr-service/Dockerfile`: two-stage build — `FROM dhi.io/golang:1.25-alpine-dev AS build`, `WORKDIR /src`, `COPY go.mod go.sum ./`, `RUN go mod download`, `COPY . .`, `RUN CGO_ENABLED=0 go build -o /out/qr-service .`; final `FROM dhi.io/static:20260611-alpine3.24`, `COPY --from=build /out/qr-service /qr-service`, `ENTRYPOINT ["/qr-service"]` — mirror `commits-service/Dockerfile` exactly
- [x] T007 [US1] Update `docker-compose.yml`: add `qr-service` block with `build: context: qr-service`, `image: whale-runner-qr:local`, `environment: [QR_LISTEN_ADDR=:8084]`, `ports: ["8084:8084"]`; add `QR_SERVICE_URL=http://qr-service:8084` to the `app` service's `environment` list; add `qr-service: condition: service_started` to nginx `depends_on`
- [x] T008 [US1] Update `app/main.go`: add `QRServiceURL string` to `Config`; add `qrServiceURL string` to `App`; set default `http://qr-service:8084` in `loadConfig()`; convert `handleRepoQRPNG` from a package-level function to an `*App` method (so it can access `qrServiceURL`); update `handleQRPNG` to call `http.Get` (via `a.httpClient`) to `a.qrServiceURL+"/qr.png?content=<url-encoded-PlayURL>&size=320"`, proxy the PNG body to the browser, return 503 on any non-200 or error; remove the `qrcode` import from `app/main.go` (depends on T007 for env var contract)
- [x] T009 [US1] Update `app/main_test.go`: add a `qrServiceStub httptest.NewServer` in `newTestApp` that returns a minimal 1×1 PNG response with `Content-Type: image/png`; set `app.qrServiceURL = qrServiceStub.URL`; update `TestHandleQRPNGWithActiveWindow` to assert on the proxied PNG response; add `TestHandleQRPNGWhenQRServiceDown` that sets `app.qrServiceURL` to an unreachable address and expects 503 (depends on T008)

**Checkpoint**: `go test ./...` passes in `app/`. `docker compose up` shows `/host` with a QR code. `docker compose logs qr-service` shows it received a render request.

---

## Phase 4: User Story 2 — Repo QR Code Migrated (Priority: P2)

**Goal**: The static `/repo-qr.png` route also delegates PNG rendering to `qr-service`, completing the extraction and leaving no direct `RenderPNG` call in the app.

**Independent Test**: `curl -s -o /tmp/repo-qr.png http://localhost/repo-qr.png && file /tmp/repo-qr.png` prints `PNG image data`.

### Implementation for User Story 2

- [x] T010 [US2] Update `app/main.go` `handleRepoQRPNG` (now an `*App` method after T008): call `a.httpClient.Get(a.qrServiceURL + "/qr.png?content=<url-encoded-repoURL>&size=320")`; proxy PNG body to browser with `Cache-Control: public, max-age=86400`; return 503 on non-200 or error (depends on T008)
- [x] T011 [US2] Add to `app/main_test.go`: `TestHandleRepoQRPNGReturnsValidPNG` (stub returns PNG → 200 + `image/png` content-type); `TestHandleRepoQRPNGWhenQRServiceDown` (qr-service unreachable → 503); `TestHandleRepoQRPNGCacheControl` (response carries `Cache-Control: public, max-age=86400`) (depends on T009 for stub setup pattern)

**Checkpoint**: `go test ./...` passes. Both `/qr.png` and `/repo-qr.png` are rendered by `qr-service`. `app/main.go` no longer imports the `qrcode` package.

---

## Phase 5: User Story 3 — Standalone Service Access (Priority: P3)

**Goal**: A developer can call `qr-service` directly on port 8084 to generate a QR code for any URL, without the app service in the path. This is fully satisfied by the Phase 3 implementation; this phase verifies handler coverage and confirms the port is reachable.

**Independent Test**: `curl -s -o /tmp/direct.png "http://localhost:8084/qr.png?content=https%3A%2F%2Fexample.com&size=256" && file /tmp/direct.png` prints `PNG image data`.

### Implementation for User Story 3

- [x] T012 [US3] Review `qr-service/internal/qrcode/handler_test.go` against US3 acceptance scenarios (direct valid request → PNG; missing `content` → 400; invalid `size` → default applied); add any missing test cases not already covered by T004; confirm `go test ./...` passes in `qr-service/`

**Checkpoint**: All three user stories are independently functional. qr-service is reachable directly on port 8084.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Complete the extraction by removing the now-unused `RenderPNG` function and `go-qrcode` dependency from the app module; run full test suite; validate end-to-end.

- [x] T013 Remove `RenderPNG` and its import of `github.com/skip2/go-qrcode` from `app/internal/qrcode/qrcode.go`; remove the three `TestRenderPNG*` tests from `app/internal/qrcode/qrcode_test.go`; run `go mod tidy` in `app/` to drop `go-qrcode` from `app/go.mod` and `app/go.sum` (depends on T010 — app must no longer call RenderPNG anywhere)
- [x] T014 [P] Run `cd qr-service && go test ./...` and confirm all handler tests pass with no compilation errors
- [x] T015 [P] Run `cd app && go test ./...` and confirm all app tests pass; verify `grep "skip2/go-qrcode" app/go.mod` returns empty (depends on T013)
- [x] T016 Run `docker compose build && docker compose up`; execute quickstart.md Scenarios 1–5 to validate end-to-end: QR code on `/host`, valid PNG from `/qr.png`, valid PNG from `/repo-qr.png`, direct call to port 8084, and `go-qrcode` absent from `app/go.mod`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2 — core extraction; blocks US2 handler changes
- **US2 (Phase 4)**: Depends on T008 (qrServiceURL on App) from Phase 3
- **US3 (Phase 5)**: Depends on Phase 3 being complete (implementation delivered by T003/T004)
- **Polish (Phase 6)**: Depends on Phase 3 + Phase 4 (no callers of RenderPNG remain)

### User Story Dependencies

- **US1 (P1)**: Can start after Phase 2 — no dependencies on other stories
- **US2 (P2)**: Depends on T008 from US1 (needs `qrServiceURL` field on `App` and `httpClient` already delegating for the `handleQRPNG` pattern)
- **US3 (P3)**: Fully delivered by US1 + T007 (port published); Phase 5 is verification only

### Within Each Phase

- T003 and T004 are parallel (different files, handler.go and handler_test.go)
- T005 depends on T003 (imports the handler package)
- T006 is parallel with T003/T004/T005 (Dockerfile has no Go dependency)
- T007 depends on T006 (needs Dockerfile to exist before build context makes sense)
- T008 depends on T007 (env var name `QR_SERVICE_URL` is set in compose)
- T009 depends on T008 (tests the new HTTP delegation in handleQRPNG)
- T010 depends on T008 (uses `qrServiceURL` on `App`)
- T011 depends on T009 (follows stub setup pattern)
- T013 depends on T010 (no callers of RenderPNG remain after both handlers migrated)
- T014 and T015 are parallel (different modules)

---

## Parallel Opportunities

### Phase 3 (US1) — parallel group A

```
T003: qr-service/internal/qrcode/handler.go
T004: qr-service/internal/qrcode/handler_test.go
T006: qr-service/Dockerfile
```

All three touch different files and can be written concurrently.

### Phase 6 — parallel group B

```
T014: go test in qr-service/
T015: go test in app/  (after T013)
```

Different module directories; no file conflicts.

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001)
2. Complete Phase 2: Foundational (T002)
3. Complete Phase 3: US1 (T003–T009)
4. **STOP and VALIDATE**: Open `/host` in browser — QR code appears; logs confirm qr-service rendered it
5. Demo if ready — the extraction is functionally complete even with `RenderPNG` still in app

### Incremental Delivery

1. T001 → T002 → Foundation ready
2. T003–T009 → US1 complete → `/qr.png` delegated to qr-service → Demo MVP
3. T010–T011 → US2 complete → `/repo-qr.png` delegated to qr-service
4. T012 → US3 verified (standalone port access confirmed)
5. T013–T016 → Polish → `go-qrcode` removed from app; end-to-end validated

---

## Notes

- `[P]` tasks touch different files and have no incomplete-task dependencies — safe to parallelize
- `[US1]`, `[US2]`, `[US3]` labels map tasks to spec user stories for traceability
- Each user story checkpoint is independently testable before moving to the next
- T013 (remove RenderPNG from app) is the cleanest proof of extraction completion — `grep "skip2/go-qrcode" app/go.mod` returning empty is SC-003
- `BuildPlayURL` is NOT moved to qr-service; it remains in `app/internal/qrcode/qrcode.go` as an app-level URL assembly helper (see research.md Decision 2)
- qr-service has no Redis, no ngrok, no external network calls — no Testcontainers needed
- App handler tests use `httptest.NewServer` stubs (not Testcontainers) because the app's own handler logic is under test, not qr-service rendering
