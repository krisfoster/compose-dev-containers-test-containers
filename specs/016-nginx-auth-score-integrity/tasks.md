---

description: "Task list for 016-nginx-auth-score-integrity"
---

# Tasks: nginx auth_request for Score Integrity

**Input**: Design documents from `specs/016-nginx-auth-score-integrity/`

**Prerequisites**: plan.md ✓ spec.md ✓ research.md ✓ data-model.md ✓ contracts/ ✓ quickstart.md ✓

**Note**: All changes ship atomically — a half-migrated state where both auth paths co-exist is
explicitly disallowed (spec Assumptions). The task order is sequenced so the build stays green
at every checkpoint.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no inter-task dependencies at that point)
- **[Story]**: Which user story this task belongs to (US1–US4 from spec.md)
- Exact file paths and line numbers are included in each description

---

## Phase 1: Setup

**Purpose**: Record the baseline before making any changes.

- [x] T001 Run `cd app && go test ./...` from repo root — confirm all tests pass before making any changes; note the test count as the baseline

---

## Phase 2: Foundational — `/auth/check` Handler

**Purpose**: Add the new Go endpoint that nginx will call as an `auth_request` sub-request. Everything in Phases 3–6 depends on this handler existing.

**⚠️ CRITICAL**: US1 (Phase 3) cannot be wired until this phase is complete.

- [x] T002 In `app/main.go`: add `signer *gate.Signer` field to the `App` struct (after the existing `gate *gate.Gate` field); populate it in `main()` by storing the `signer` variable alongside the existing `gate.NewGate(store, signer)` call (`app := &App{ ..., gate: g, signer: signer, ... }`); then add a new `handleAuthCheck(w http.ResponseWriter, r *http.Request)` method on `*App` — it must: (a) reject non-GET with 405, (b) read the `cw_grant` cookie via `r.Cookie(gate.GrantCookieName)` and return 401 if absent, (c) call `a.signer.Verify(cookie.Value)` and return 401 on any error, (d) return 200 with an empty body on success

- [x] T003 In `app/main.go`, register `mux.HandleFunc("/auth/check", a.handleAuthCheck)` on the single mux (the `ungatedMux()` function) — add it after the `/api/leaderboard/scores` route registration

- [x] T004 [P] In `app/main_test.go`: update `newTestApp()` to populate the new `signer` field (`signer: gate.NewSigner([]byte("test-secret"), time.Hour)` — same value used for the `Gate`); then add five unit tests for `handleAuthCheck`: `TestHandleAuthCheckWithValidCookie` (sign a fresh grant with the test signer, set it as cookie → expect 200), `TestHandleAuthCheckWithNoCookie` (no cookie → 401), `TestHandleAuthCheckWithExpiredCookie` (use a signer configured with `-time.Second` lifetime so every grant is instantly expired → 401), `TestHandleAuthCheckWithInvalidCookie` (set cookie to `"not.valid"` → 401), `TestHandleAuthCheckRejectsNonGet` (POST to `/auth/check` → 405); all tests call through `app.ungatedMux()`

**Checkpoint**: `cd app && go test ./...` — all tests pass (no mux tests broken yet; `/auth/check` is a new route).

---

## Phase 3: US1 (P1) — Score Submission Requires QR Grant 🎯 MVP

**Goal**: nginx rejects any score submission that lacks a valid `cw_grant` cookie before the Go handler runs.

**Independent Test**: `curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost/api/leaderboard/scores -H "Content-Type: application/json" -d '{"name":"x","score":1}'` → must return `401` (no cookie). With a valid cookie → `201`.

- [x] T005 [US1] In `nginx/nginx.conf`: (a) add a dedicated `location /api/leaderboard/scores` block above the existing `location /api/` block — it must include `auth_request /auth/check;`, `proxy_pass http://app:8080;`, `proxy_set_header Host $host;`, and `proxy_set_header Cookie $http_cookie;`; (b) add a `location = /auth/check { internal; proxy_pass http://app:8080/auth/check; proxy_pass_request_body off; proxy_set_header Content-Length ""; proxy_set_header Cookie $http_cookie; }` block; the specific `/api/leaderboard/scores` block takes precedence over the general `/api/` prefix match so `/api/ping` remains ungated

**Checkpoint**: `docker compose build nginx && docker compose up -d` — run quickstart.md Scenario 1 (no cookie → 401) and Scenario 2 (valid cookie → 201).

---

## Phase 4: US2 (P1) — No Extractable Token in Browser

**Goal**: The game page source contains no leaderboard credential. Score submissions carry no `X-Leaderboard-Token` header.

**Independent Test**: `curl -s http://localhost/play | grep -i "leaderboard_token"` → zero output. Browser DevTools → `window.__LEADERBOARD_TOKEN__` → `undefined`.

- [x] T006 [P] [US2] Remove the line `  <script>window.__LEADERBOARD_TOKEN__ = "{{.LeaderboardToken}}";</script>` (line 14) from `frontend/game/index.html`

- [x] T007 [P] [US2] Remove the line `        'X-Leaderboard-Token': window.__LEADERBOARD_TOKEN__ || '',` (line 68) from `frontend/game/script.js`

- [x] T008 [US2] In `handlePlayIndex` in `app/main.go` (around line 271): remove the `data := struct{ LeaderboardToken string }{LeaderboardToken: a.leaderboardSecret}` struct literal; change `tmpl.Execute(w, data)` to `tmpl.Execute(w, nil)` (the template now has no variables after T006)

- [x] T009 [US2] In `app/main.go`: remove the `leaderboardSecret string` field from the `App` struct and its assignment (`leaderboardSecret: cfg.LeaderboardAPISecret`) in the `main()` `app := &App{...}` initializer; remove the `leaderboardSecret` argument from `leaderboard.NewHandler(scoreStore, cfg.LeaderboardAPISecret, scoreStore)` — update to `leaderboard.NewHandler(scoreStore, scoreStore)` (the handler no longer takes a secret; this is a compile error until T018 below, so do T018 immediately after or defer the `NewHandler` call update to T018)

  > **Note**: `leaderboard.NewHandler` still has the secret param until T018. To keep the build green, change the call site first by passing `""` as the secret arg: `leaderboard.NewHandler(scoreStore, "", scoreStore)`. T018 removes the param from the function signature.

- [x] T010 [US2] In `app/main.go`: remove `LeaderboardAPISecret string` from the `Config` struct; remove the `LeaderboardAPISecret: envOr("LEADERBOARD_API_SECRET", "dev-only-change-me"),` line from `loadConfig()`

- [x] T011 [US2] In `docker-compose.yml`, remove the `- LEADERBOARD_API_SECRET=${LEADERBOARD_API_SECRET:-dev-only-change-me}` line from the `app` service's `environment` block

**Checkpoint**: `cd app && go build ./...` — must compile (leaderboard handler still accepts secret param with empty string pass-through). Run quickstart.md Scenario 3.

---

## Phase 5: US3 (P2) — Game Access and Score Submission Use One Credential

**Goal**: A single `cw_grant` cookie gates both `/play` and score submission. The Go app runs on one port (8080) with no second listener.

**Independent Test**: The compose stack starts with one app listener log line. `/play` without a cookie → 403. `/play?w=<windowID>` → grant issued + redirect. Score submission with the same cookie → 201.

- [x] T012 [US3] In `app/main.go` in the `ungatedMux()` function: change the `/play` registration from `mux.HandleFunc("/play", a.handlePlayIndex)` to `mux.Handle("/play", a.gate.Middleware(http.HandlerFunc(a.handlePlayIndex)))` — this enforces the `cw_grant` gate on the single mux, matching the behavior the old `gatedMux` provided

- [x] T013 [US3] In `app/main.go`: delete the entire `gatedMux()` method (lines 244–255 in the current file); the method is no longer called after T014

- [x] T014 [US3] In `main()` in `app/main.go`: remove the gated listener goroutine (`go func() { log.Printf("gated listener starting on :%s", cfg.GatedPort); errc <- http.ListenAndServe(":"+cfg.GatedPort, app.gatedMux()) }()`); reduce the `errc` channel from capacity 2 to capacity 1 (`errc := make(chan error, 1)`); remove `GatedPort string` from the `Config` struct and the `GatedPort: envOr("APP_GATED_PORT", "8081"),` line from `loadConfig()`

- [x] T015 [P] [US3] In `docker-compose.yml`, remove the `- APP_GATED_PORT=8081` line from the `app` service's `environment` block

- [x] T016 [P] [US3] In `nginx/nginx.conf`, change `proxy_pass http://app:8081;` to `proxy_pass http://app:8080;` in the `location = /play` block — the gated port is gone; the single port serves all routes

- [x] T017 [P] [US3] In `app/main_test.go`: add `TestSingleMuxGatesPlayWithMiddleware` — sends `GET /play` with no cookie through `app.ungatedMux()` and asserts the response is 403 (gate.Middleware rejects it); add `TestSingleMuxIssuesGrantOnValidToken` — activates a window on the fake store, sends `GET /play?w=<windowID>` through `app.ungatedMux()`, asserts the response is 302 (grant issued + redirect) and the `Set-Cookie` header contains `cw_grant`

**Checkpoint**: `cd app && go test ./...` — T017 tests must pass. `docker compose logs app | grep "listener starting"` → one line only.

---

## Phase 6: US4 (P2) — Dead Auth Path Removed

**Goal**: Zero references to the old leaderboard API secret remain in source code, tests, or configuration.

**Independent Test**: `grep -r "LEADERBOARD_API_SECRET\|X-Leaderboard-Token\|CredentialHeader\|validCredential\|leaderboardSecret\|LeaderboardToken\|__LEADERBOARD_TOKEN__" app/ frontend/ docker-compose.yml nginx/` → zero matches.

- [x] T018 [US4] In `app/internal/leaderboard/handler.go`: (a) remove the `CredentialHeader = "X-Leaderboard-Token"` const; (b) remove the `secret string` field from the `Handler` struct; (c) remove the `secret string` parameter from `NewHandler` and its assignment in the function body (`return &Handler{store: store, notifier: notifier}`); (d) remove the `validCredential()` function; (e) remove the `if !validCredential(r.Header.Get(CredentialHeader), h.secret) { writeError(...); return }` block from `serveSubmit` — the handler now performs payload validation only, no credential check

- [x] T019 [US4] In `app/main.go`, update the `leaderboard.NewHandler(...)` call — remove the `""` pass-through added in T009 so the call becomes `leaderboard.NewHandler(scoreStore, scoreStore)` (now that T018 removed the second parameter from the function signature)

- [x] T020 [US4] In `app/main_test.go`: (a) remove constants `testLeaderboardSecret`, `testIndexHTMLTemplate`, `testIndexHTMLRendered`; (b) in `newTestApp()`, remove the `leaderboardSecret: testLeaderboardSecret` field from the `App` struct literal and change `leaderboard.NewHandler(scoreStore, testLeaderboardSecret, &leaderboardtest.FakeScoreNotifier{})` to `leaderboard.NewHandler(scoreStore, &leaderboardtest.FakeScoreNotifier{})`; (c) apply the same `NewHandler` update in `appWithErroringStore()`; (d) update the fake `index.html` written in `newTestApp()` — the template no longer has `{{.LeaderboardToken}}`, so replace `testIndexHTMLTemplate` content with a simple `<html><body>game</body></html>` string literal inline

- [x] T021 [US4] In `app/main_test.go`, delete these test functions entirely: `TestHandlePlayIndexInjectsLeaderboardToken`, `TestGatedPlayRejectsWithNoGrantOrToken`, `TestGatedPlayAllowsValidToken`, `TestGatedListenerDoesNotExposeHostRoutes`, `TestHandleLeaderboardPageOnBothListeners`

- [x] T022 [US4] In `app/main_test.go`, make these targeted updates: (a) in `TestOldCommitsEndpointRemoved` — remove the `app.gatedMux()` assertion block (the second `req2`/`rec2` block), keeping only the ungated mux assertion; (b) in `TestLoadConfigDefaults` — remove `"APP_GATED_PORT"` and `"LEADERBOARD_API_SECRET"` from the `t.Setenv(key, "")` loop slice and remove `cfg.GatedPort != "8081"` and `cfg.LeaderboardAPISecret != "dev-only-change-me"` from the assertion; (c) in `TestLoadConfigReadsOverrides` — remove the `t.Setenv("LEADERBOARD_API_SECRET", "super-secret")` and its assertion; (d) in `TestUngatedPlayRequiresNoGate` — update the body assertion: remove the `rec.Body.String() != testIndexHTMLRendered` check; instead assert only that `rec.Code == http.StatusOK` and that `rec.Body.String()` contains `"game"` (or any static string from the new minimal template); (e) add `TestScoreSubmissionNoTokenRequired` — sends `POST /api/leaderboard/scores` through `app.ungatedMux()` with a valid JSON body but NO `X-Leaderboard-Token` header, asserts 201 (the handler now accepts it with no credential check)

**Checkpoint**: `cd app && go test ./...` — all tests pass. `grep -r "LEADERBOARD_API_SECRET\|CredentialHeader\|validCredential\|leaderboardSecret\|__LEADERBOARD_TOKEN__" app/ frontend/ nginx/ docker-compose.yml` → zero matches.

---

## Phase 7: Polish & Constitution

**Purpose**: Constitution amendment and end-to-end validation.

- [x] T023 In `.specify/memory/constitution.md`: write a PATCH-level Sync Impact Report at the top of the file documenting version 1.4.0 → 1.4.1; update the Permanent Routing Layer carve-out section — specifically the sentence "Gate enforcement MUST remain in the Go app and is reached by proxying to the Go gated internal port" to reflect the post-016 model: score-submission auth is enforced by nginx `auth_request` to `GET /auth/check` on port 8080; play-path auth is enforced by `gate.Middleware` applied per-route on the single mux on port 8080; the second (gated) internal port (8081) is removed; bump the `Version: 1.4.0` to `1.4.1` and update `Last Amended` to `2026-07-09`

- [x] T024 Run `cd app && go test ./...` — all tests must pass with no skips or failures before ship

- [ ] T025 Run `docker compose up --build` and execute quickstart.md Scenarios 1-7 in order; record the HTTP status code returned for each scenario; confirm all match the expected values documented in quickstart.md

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 baseline only — BLOCKS Phase 3
- **Phase 3 (US1)**: Depends on Phase 2 complete (`/auth/check` handler must exist before nginx wires to it)
- **Phase 4 (US2)**: Depends on Phase 3 (auth_request in place before removing token — avoids a window where the token is absent but not yet gated)
- **Phase 5 (US3)**: Depends on Phase 4 (token removal done before mux consolidation to avoid double-auth window)
- **Phase 6 (US4)**: Depends on Phase 5 (gatedMux gone before removing tests that reference it)
- **Phase 7 (Polish)**: Depends on Phases 1–6 all complete

### Within-Phase Parallelism

| Task | Parallel with |
|------|---------------|
| T004 (auth-check tests) | Can start as soon as T003 is done; T004a–e can run in parallel with each other |
| T006 (index.html) | T007 (script.js) — different files |
| T015 (compose APP_GATED_PORT) | T016 (nginx /play proxy) | T017 (test additions) — different files |

### User Story Dependencies

- **US1 (P1)**: Requires Foundational phase only
- **US2 (P1)**: Requires US1 (score submissions already gated before token removed)
- **US3 (P2)**: Requires US1 + US2 (architecture consolidation after security is in place)
- **US4 (P2)**: Requires US1 + US2 + US3 (dead-code removal last)

---

## Parallel Execution Example: Phase 2

```bash
# T002 and T003 are sequential (register after adding method)
Task: "Add signer field + handleAuthCheck to app/main.go"    # T002
Task: "Register /auth/check on the mux in app/main.go"      # T003

# T004a-e can run in parallel after T003:
Task: "TestHandleAuthCheckWithValidCookie in app/main_test.go"
Task: "TestHandleAuthCheckWithNoCookie in app/main_test.go"
Task: "TestHandleAuthCheckWithExpiredCookie in app/main_test.go"
Task: "TestHandleAuthCheckWithInvalidCookie in app/main_test.go"
Task: "TestHandleAuthCheckRejectsNonGet in app/main_test.go"
```

## Parallel Execution Example: Phase 4 (US2)

```bash
# T006 and T007 are independent:
Task: "Remove __LEADERBOARD_TOKEN__ from frontend/game/index.html"   # T006
Task: "Remove X-Leaderboard-Token from frontend/game/script.js"      # T007

# T008-T011 are sequential (same file or build-order dependencies):
Task: "Remove LeaderboardToken injection from handlePlayIndex"        # T008
Task: "Remove leaderboardSecret from App struct"                      # T009
Task: "Remove LeaderboardAPISecret from Config"                       # T010
Task: "Remove LEADERBOARD_API_SECRET from docker-compose.yml"        # T011
```

---

## Implementation Strategy

### MVP (US1 + US2 together — both P1)

US1 and US2 are treated as a single MVP unit because the two together eliminate the vulnerability:
US1 blocks unauthenticated submissions; US2 removes the now-unnecessary token from the browser.
Shipping US1 without US2 leaves a useless (but still visible) token in the page — worse than the
status quo from a UX perspective.

1. Complete Phase 1 (Setup)
2. Complete Phase 2 (Foundational — `/auth/check` handler)
3. Complete Phase 3 (US1 — nginx auth_request wiring)
4. Complete Phase 4 (US2 — token removal)
5. **STOP and VALIDATE**: run quickstart.md Scenarios 1, 2, 3 — score integrity working, no token in page
6. Proceed to Phase 5 (US3) and Phase 6 (US4) for cleanup

### Full Delivery (all 4 user stories)

1. Phases 1–4 → MVP validated
2. Phase 5 (US3) → Single-listener consolidation; validate quickstart.md Scenario 6
3. Phase 6 (US4) → Dead code gone; grep check clean
4. Phase 7 (Polish) → Constitution amended; all tests green; quickstart.md Scenarios 1–7 complete

---

## Notes

- `[P]` tasks touch different files — safe to work on concurrently within the same phase
- `[USn]` label maps each task to its user story for traceability
- The build must stay green at every checkpoint — the `""` pass-through in T009 is a deliberate
  intermediate step to avoid a compile break between T009 and T018
- Constitution amendment (T023) is a prerequisite for ship — not optional
- `go test ./...` must pass (T024) before running the compose validation (T025)
