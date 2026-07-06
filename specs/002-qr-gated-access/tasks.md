---

description: "Task list for QR-Gated Public Access to Crossy Whale"
---

# Tasks: QR-Gated Public Access to Crossy Whale

**Input**: Design documents from `/specs/002-qr-gated-access/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/gate-http-contract.md, quickstart.md

**Tests**: Included, with an explicit coverage target. The constitution's Principle III
(Testcontainers Over Mocks) is non-negotiable for any Go code touching Redis, so Redis-boundary
tests are mandatory; everything above that boundary is made testable via a `WindowStore` interface
so unit/handler tests can run against a fast in-memory fake instead of a live container. Target:
≥80% statement coverage across all `app/` Go code (`go test ./... -cover`), excluding the
camera-scan flow itself, which cannot be automated (constitution Principle IV — validated manually
via `quickstart.md` instead).

**Organization**: Tasks are grouped by user story (US1/US2/US3, matching spec.md's P1/P1/P2
priorities) so each can be implemented and demoed independently once Foundational is done.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- File paths are relative to the repository root unless otherwise noted

## Path Conventions

Single Go module at `app/`, alongside the existing `frontend/game/` static content, per
`plan.md`'s Project Structure. `webserver/` is retired by this feature.

---

## Phase 1: Setup

**Purpose**: Stand up the new Go module and its dependency/config surface

- [X] T001 Create the `app/` Go module skeleton: `app/go.mod` (module path, Go 1.22+), `app/main.go`
      stub, and empty `app/internal/gate/` and `app/internal/qrcode/` directories, per `plan.md`'s
      Project Structure
- [X] T002 [P] Add `github.com/redis/go-redis/v9`, an MIT-licensed QR-encoding library (e.g.
      `github.com/skip2/go-qrcode`), and `github.com/testcontainers/testcontainers-go` plus its
      Redis module to `app/go.mod`/`app/go.sum`
- [X] T003 [P] Add `GRANT_COOKIE_SECRET` and `QR_WINDOW_TTL` variables (with short explanatory
      comments) to `.env.example`, alongside the existing `WEB_PORT`/`NGROK_AUTHTOKEN`

**Checkpoint**: `app/` module exists and builds (even if `main.go` is a stub); dependencies resolve.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The shared gate mechanism every user story depends on — Redis window state (behind a
testable interface), grant cookie signing, the gate decision itself, and the two-listener server
that replaces `webserver`

**⚠️ CRITICAL**: No user story task can be verified end-to-end until this phase is complete

- [X] T004 Update `docker-compose.yml`: remove the `webserver` service; add a `redis` service
      (no published port needed, internal only) and an `app` service (builds from `app/`, publishes
      the ungated port as `${WEB_PORT:-8080}:8080`, exposes an internal gated port, depends on
      `redis` but not on `ngrok`); repoint the `ngrok` service's `command` at `app`'s gated port
      instead of `webserver:80`
- [X] T005 [P] Delete `webserver/nginx.conf` and remove the now-empty `webserver/` directory
      (retires the interim static-hosting carve-out per `plan.md`'s Constitution Check)
- [X] T006 Define a `WindowStore` interface (get current window ID; activate/rotate with a TTL) and
      its Redis-backed implementation against `access:window:current` in `app/internal/gate/window.go`,
      per `data-model.md`'s QR Access Code — this interface is what makes T009's middleware and
      T0xx handler tests testable without a live container
- [X] T007 [P] Provide an in-memory fake `WindowStore` implementation for tests. **Implemented at
      `app/internal/gate/gatetest/fake_store.go`, not `window_fake_test.go` as originally planned**:
      Go does not allow a package's `_test.go` symbols to be imported by another package's tests, and
      this fake needs to be usable both by `gate`'s own tests and by `app/main_test.go`, so it had to
      be a regular (non-test) file in its own subpackage instead. It is never imported by production
      code — only by tests, including its own (`gatetest/fake_store_test.go`, added during the
      coverage pass in Phase 6).
- [X] T008 [P] Implement the grant cookie payload type and HMAC-SHA256 sign/verify functions in
      `app/internal/gate/grant.go` — payload is `{grant_id (UUIDv4), issued_window_id, issued_at}`,
      per `data-model.md`'s Visitor Access Grant and `research.md` §4
- [X] T009 Implement the gate decision in `app/internal/gate/middleware.go` (depends on T006,
      T008), coded against the `WindowStore` interface (not a concrete Redis client): valid grant
      cookie → allow; no valid cookie but a `w` query parameter matching the current window → mint
      a new grant, set the `cw_grant` cookie (HttpOnly, Secure, SameSite=Lax), redirect to the
      clean `/play` URL; otherwise → reject (reject response body is filled in by US2's task)
- [X] T010 Implement `app/main.go`: two `http.Server` instances — an ungated one serving
      `frontend/game/` via `http.FileServer` directly, and a gated one serving the same content
      wrapped by the T009 middleware — per `contracts/gate-http-contract.md` (depends on T009)
- [X] T011 [P] Write a Testcontainers-go test for the Redis-backed `WindowStore` implementation
      (activate/rotate/lookup against a real Redis) in `app/internal/gate/window_test.go`
      (constitution Principle III — no mocked Redis client for this test)
- [X] T012 [P] Write unit tests for grant cookie sign/verify, including tamper detection (mutated
      payload fails verification) and expiry (payload older than the fixed grant lifetime fails),
      in `app/internal/gate/grant_test.go`

**Checkpoint**: `docker compose up` brings up `redis` + `app`; `/play` is reachable on the ungated
port; the gated port exists but grants nothing yet (no way to activate a window) — correctly
fail-closed per FR-009. Foundation ready for user story work.

---

## Phase 3: User Story 1 - Attendee scans the QR code and lands in the game (Priority: P1) 🎯 MVP

**Goal**: A displayed QR code opens the game directly on scan (FR-001, FR-002), access persists
for the rest of the visit without re-scanning (FR-005), and concurrent scanners each get their own
independent, identifiable access (FR-010, FR-011)

**Independent Test**: Display the QR code, scan it with a phone camera, and confirm the phone
reaches a playable game and keeps playing without being asked to scan again; repeat from a second
device concurrently and confirm both work independently

### Tests for User Story 1

- [X] T013 [P] [US1] Write a middleware test against the fake `WindowStore` (from T007) in
      `app/internal/gate/middleware_test.go` confirming a request bearing a valid `w` token
      receives a `cw_grant` cookie and a redirect to clean `/play`, and that a follow-up request
      with only that cookie succeeds with no token present
- [X] T014 [P] [US1] Write a middleware test confirming two simulated concurrent grant requests
      each receive distinct `grant_id` values with no collision, in
      `app/internal/gate/middleware_test.go`
- [X] T015 [P] [US1] Write a unit test for QR PNG rendering in `app/internal/qrcode/qrcode_test.go`
      confirming the encoded payload is the expected `/play?w=<window_id>` URL and that a valid PNG
      is produced
- [X] T016 [P] [US1] Write handler tests for `GET /qr.png` and `GET /host` against the fake
      `WindowStore` in `app/main_test.go`, covering: no window active yet (auto-activation path),
      and a window already active (image/page render successfully)

### Implementation for User Story 1

- [X] T017 [US1] Implement QR PNG rendering in `app/internal/qrcode/qrcode.go`, encoding
      `https://<public-host>/play?w=<current window_id>`; discover `<public-host>` via ngrok's
      local inspection API (`http://ngrok:4040/api/tunnels`), matching the URL-discovery approach
      already used for the presenter in `001-host-webapp-ngrok`
- [X] T018 [US1] Implement `GET /qr.png` and `GET /host` on the ungated listener in `app/main.go`:
      `/host` renders a minimal page embedding the QR image; the first-ever visit to `/host`
      auto-activates a window via T006 if none is currently active
- [X] T019 [US1] Implement `GET /play` and `GET /play/*` static asset routes on the gated listener
      in `app/main.go`, using the T009 gate middleware ahead of `http.FileServer`
- [ ] T020 [US1] Run `quickstart.md` Scenarios 1, 2, and 5 against `docker compose --profile
      public up` (manual phone-camera validation — constitution Principle IV; Scenario 5 needs two
      devices/browsers) — **NOT completed by implementation**: requires a real phone camera and a
      real ngrok public URL, neither available in the environment this feature was implemented in.
      The underlying grant-issuance and concurrency logic this scenario exercises is covered by
      automated tests (`TestMiddlewareValidTokenGrantsAccessAndRedirects`,
      `TestMiddlewareConcurrentGrantsAreDistinct`) and was additionally smoke-tested end-to-end
      against the live compose stack via the internal Docker network (equivalent HTTP requests in
      place of an actual scan). The presenter (with a phone and a real `NGROK_AUTHTOKEN`) still
      needs to run this scenario for real before a live booth.

**Checkpoint**: User Story 1 is fully functional and independently demoable — this is the MVP.

---

## Phase 4: User Story 2 - Public access is blocked without a valid scan (Priority: P1)

**Goal**: Anyone reaching the public endpoint without a valid grant or token sees a clear block
message instead of the game (FR-003, FR-009), while local access on the presenter's machine is
never gated (FR-004)

**Independent Test**: With public access enabled, open the bare public URL in a fresh browser with
no prior scan and confirm a "scan the QR code" message appears instead of the game; confirm the
local URL still loads the game directly with no gate

### Tests for User Story 2

- [X] T021 [P] [US2] Write a middleware test against the fake `WindowStore` in
      `app/internal/gate/middleware_test.go` confirming the gated listener rejects a request with
      neither a valid grant cookie nor a valid `w` token, and also rejects when no window has ever
      been activated at all (fail closed)
- [X] T022 [P] [US2] Write a handler test confirming the ungated listener serves `/play` with no
      cookie and no `w` token required, in `app/main_test.go`

### Implementation for User Story 2

- [X] T023 [US2] Implement the reject-path response body in `app/internal/gate/middleware.go`:
      `403` with a short HTML page reading "Scan the QR code to play" — the same response whether
      no window is active at all or the presented token/cookie is simply invalid (per
      `contracts/gate-http-contract.md`'s Error responses table, so nothing is leaked either way)
- [ ] T024 [US2] Run `quickstart.md` Scenario 3 against `docker compose --profile public up` —
      **NOT completed by implementation**: same real-device/real-tunnel limitation as T020. Covered
      instead by `TestGatedPlayRejectsWithNoGrantOrToken` and a live smoke-test confirming the
      gated listener returns `403` with no grant/token, and the ungated listener serves `/play`
      unconditionally, against the actual running compose stack.

**Checkpoint**: User Stories 1 and 2 both work independently — the gate now has real teeth and a
real message, without touching local access.

---

## Phase 5: User Story 3 - Presenter rotates the QR code to cut off stale access (Priority: P2)

**Goal**: The presenter can invalidate the current QR code on demand (FR-007), stale codes also
expire on their own (FR-006), and neither action interrupts a visitor's already-granted access
(FR-008)

**Independent Test**: With a QR code active and already used, trigger rotation and confirm the old
code's link stops working while a freshly scanned code still works, and confirm a visitor who
already had access keeps playing uninterrupted

### Tests for User Story 3

- [X] T025 [P] [US3] Write a Testcontainers-go test in `app/internal/gate/window_test.go`
      confirming rotation immediately invalidates the previous window's token while a freshly
      generated token is accepted (extends the real-Redis suite from T011 — rotation semantics are
      exactly the kind of "atomic overwrite" behavior Principle III requires testing against a real
      Redis rather than a fake)
- [X] T026 [P] [US3] Write a Testcontainers-go test in `app/internal/gate/window_test.go`
      confirming a window's TTL expiry (using a short TTL for the test) stops granting new access
      with no manual action taken
- [X] T027 [P] [US3] Write a middleware test against the fake `WindowStore` in
      `app/internal/gate/middleware_test.go` confirming a visitor's existing grant cookie keeps
      working even when the fake reports the window as rotated/expired
- [X] T028 [P] [US3] Write a handler test for `POST /host/rotate` against the fake `WindowStore` in
      `app/main_test.go`, confirming it overwrites the current window and redirects to `/host`

### Implementation for User Story 3

- [X] T029 [US3] Implement `POST /host/rotate` on the ungated listener in `app/main.go`: generates
      a fresh window via T006, overwriting any currently active one, then redirects (303) back to
      `/host`
- [ ] T030 [US3] Run `quickstart.md` Scenarios 4 and 6 against `docker compose --profile public up`
      — **NOT completed by implementation**: same real-device/real-tunnel limitation as T020.
      Covered instead by `TestRedisWindowStoreRotateInvalidatesPreviousToken`,
      `TestRedisWindowStoreExpiresOnItsOwn` (both against a real Redis via Testcontainers-go), and
      `TestMiddlewareGrantSurvivesWindowRotationAndExpiry`, plus a live smoke-test confirming
      rotation against the actual running compose stack (verified via the internal Docker network:
      old window token stops matching, new one is issued).

**Checkpoint**: All three user stories are independently functional and demoable together.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Bring the rest of the repo's documentation in line with the new architecture, and
confirm the coverage target is actually met

- [X] T031 [P] Update `README.md`'s hosting section (currently describes the retired `webserver`
      service) to describe the `app`/`redis` services, the QR-gated public endpoint, and how to
      reach `/host` to display/rotate the code
- [X] T032 Run `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out` in
      `app/`; confirm ≥80% total statement coverage. For any package below target, add the missing
      test(s) — expected coverage gaps at this point should be limited to trivial code (e.g. the
      `main()` bootstrap itself), not to `gate`, `qrcode`, or handler logic, all of which have
      dedicated tests from earlier phases. **Result: 88.7% total** (`main` 81.1%, `gate` 94.5%,
      `gate/gatetest` 100%, `qrcode` 100%). Remaining gaps are `main()`'s own bootstrap (starts real
      listeners, calls `log.Fatal` — not safely unit-testable) and a handful of defensive
      `if err != nil` branches guarding failures that cannot be triggered honestly in a test
      (`crypto/rand.Read` failing, `json.Marshal` failing on a plain struct).
- [ ] T033 Run the full `quickstart.md` validation (all six scenarios) end-to-end against a fresh
      `docker compose --profile public up`, confirming no regressions across stories — **NOT
      completed by implementation**: Scenarios 2 and 5 require a real phone camera and a real
      ngrok public URL. What was verified: `docker compose build app` succeeds; `docker compose up`
      brings up `redis` + `app` cleanly; `/play` serves the game on the ungated port; `/qr.png`
      correctly 503s before `/host` is ever visited; `/host` auto-activates a window and returns
      200; the gated port (reached over the internal Docker network, simulating what `ngrok` would
      forward) correctly returns `403` with no token, `302` plus a correctly-formed signed
      `cw_grant` cookie (`HttpOnly`, `Secure`, `SameSite=Lax`) with a valid token, and rotating via
      `/host/rotate` invalidates the old token. The full `go test ./...` suite (41 tests) passes.
      **Remaining for the user**: re-add a real `NGROK_AUTHTOKEN` to `.env` (see note below) and
      run the six `quickstart.md` scenarios with an actual phone before a live booth.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories (this is where the actual
  gate mechanism, its testability seam (`WindowStore`), and the compose/service topology change
  all live)
- **User Stories (Phase 3-5)**: All depend on Foundational completion
  - US1 and US2 are both P1 and independent of each other; either can go first
  - US3 (P2) is independent of US1/US2's *implementation* but is most meaningfully demoed after
    US1 exists (there's nothing to rotate away from otherwise)
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational — no dependency on US2/US3
- **User Story 2 (P1)**: Can start after Foundational — no dependency on US1/US3 (its tests use the
  Foundational middleware directly; T023's response copy doesn't require US1's routes to exist)
- **User Story 3 (P2)**: Can start after Foundational — no code dependency on US1/US2, though
  `quickstart.md` Scenario 4 is easiest to run manually once US1's `/host` and `/play` exist

### Within Each User Story

- Tests before the implementation tasks they cover
- Foundational pieces (`WindowStore`, grant signing, gate decision) before anything that routes
  through them
- Story complete before moving to the next priority, if working sequentially

### Parallel Opportunities

- T002 and T003 (Setup) can run in parallel
- T005, T007, T008, T011, T012 (Foundational) can run in parallel with each other once T004/T006
  land (T005 touches an unrelated directory; T007/T008 are separate files; T011/T012 are tests
  against already-landed code)
- Within each user story, all tasks marked [P] — typically the test tasks, since they exercise
  independent behaviors even where they share a file — can be written in parallel
- US1, US2, and US3 can be staffed to different people once Foundational is merged, since none of
  their implementation tasks share a file with another story's implementation tasks (T018/T019 vs.
  T023 vs. T029 all touch different routes in `app/main.go`, which is a coordination point to
  sequence rather than truly parallelize if one person owns it)

---

## Parallel Example: Foundational Phase

```bash
# After T004 and T006 land, these can proceed together:
Task: "Delete webserver/nginx.conf and remove the webserver/ directory"
Task: "Provide an in-memory fake WindowStore implementation in app/internal/gate/window_fake_test.go"
Task: "Implement grant cookie payload type and HMAC-SHA256 sign/verify in app/internal/gate/grant.go"
Task: "Write Testcontainers-go test for the Redis-backed WindowStore in app/internal/gate/window_test.go"
Task: "Write unit tests for grant cookie sign/verify in app/internal/gate/grant_test.go"
```

## Parallel Example: User Story 1

```bash
Task: "Middleware test: valid w token grants a cookie and redirects to clean /play"
Task: "Middleware test: two concurrent grant requests get distinct, non-colliding grant IDs"
Task: "Unit test: QR PNG encodes the expected URL and produces a valid image"
Task: "Handler test: GET /qr.png and GET /host against the fake WindowStore"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (critical — this is most of the feature's real complexity)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: run `quickstart.md` Scenarios 1, 2, 5 with a real phone
5. At this point the QR-scan-to-play flow works, but the public endpoint isn't yet
   enforcing rejection copy (US2) or supporting rotation (US3) — acceptable for an internal demo
   of the mechanism, not yet for a live booth

### Incremental Delivery

1. Setup + Foundational → gate mechanism exists, nothing reachable via QR yet
2. Add US1 → scan-to-play works end to end (MVP demoable internally)
3. Add US2 → the public endpoint is now actually safe to expose at a booth
4. Add US3 → presenter has manual control for a multi-day or multi-session event
5. Polish → docs catch up to the new architecture, coverage target verified

### Note on Sequencing for a Live Booth

Because US2's core protection (fail-closed, reject unknown requests) is already implemented by
Foundational's gate decision (T009) — US2's own tasks only add the *user-facing message* and its
tests — a booth-ready state technically exists as soon as US1 and US2 are both done, even before
US3. US3 (rotation) upgrades operational control but isn't a prerequisite for the endpoint being
safe to expose.

### Note on the Coverage Target

The 80% target (see plan.md's Testing section) is achievable without any test touching a live
container beyond what Principle III already requires, because `WindowStore` isolates the one real
external boundary (Redis) behind an interface: T011/T025/T026 give that implementation itself
thorough Testcontainers-go coverage, while every consumer of the interface (middleware, HTTP
handlers) is fully covered by fast tests against the in-memory fake from T007. T032 is the single
point where actual coverage is measured and any gap is closed before calling the feature done.

---

## Notes

- [P] tasks touch different files, or exercise independent behaviors against a shared fake/test
  file without blocking each other's authorship
- [Story] labels map tasks to spec.md's user stories for traceability
- Redis-touching tests MUST use Testcontainers-go, never a mocked client (constitution Principle
  III, non-negotiable); tests against the `WindowStore` interface from a consumer's perspective
  (middleware, handlers) use the in-memory fake instead, which is not the same thing as mocking
  Redis itself
- Commit after each task or logical group
- Manual quickstart.md steps (camera scans) cannot be automated — they are the Definition of Done
  per constitution Principle IV and are called out explicitly as their own tasks rather than
  folded silently into implementation tasks
