# Tasks: Go HTML Template Extraction with Live Reload

**Input**: Design documents from `specs/015-extract-html-templates/`

**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ping-contract.md ✅, quickstart.md ✅

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Exact file paths are included in every task description

---

## Phase 1: Setup (Template Files on Disk)

**Purpose**: Create the external template files that all subsequent work depends on. These replace the three inline HTML constants in `app/main.go`: `gettingStartedPageHTML`, `hostPageHTML`, and `leaderboardPageTemplate`.

- [x] T001 Create `templates/` directory at the repository root
- [x] T002 [P] Create `templates/getting-started.html` — copy the full HTML content of the `gettingStartedPageHTML` const from `app/main.go` into this file (valid Go `html/template` syntax; no dynamic fields, so no `{{.}}` needed)
- [x] T003 [P] Create `templates/host.html` — copy the full HTML content of the `hostPageHTML` const from `app/main.go` into this file (valid Go `html/template` syntax; no dynamic fields)
- [x] T004 [P] Create `templates/leaderboard.html` — copy the full HTML content of the `leaderboardPageTemplate` const from `app/main.go` into this file, preserving the `{{.ScoresServiceURL}}` and `{{.CommitsServiceURL}}` template actions

**Checkpoint**: Three `.html` files exist in `templates/`; each contains valid Go template syntax and produces correct HTML when rendered.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Extend the `Config` struct and `App` struct in `app/main.go` with the new fields required by all user story phases. No user story work can begin until these are in place.

**⚠️ CRITICAL**: Phases 3, 4, and 5 all depend on these two tasks completing first.

- [x] T005 Add `TemplatesDir string` field to the `Config` struct in `app/main.go`; add `TEMPLATES_DIR` env var to `loadConfig()` with default `"/templates"`; add `templatesDir string` field to the `App` struct and wire it from `cfg.TemplatesDir` in `main()`
- [x] T006 Add `templateVersion atomic.Int64` field to the `App` struct in `app/main.go` (import `"sync/atomic"` if not already present); no initialization needed — zero value is correct

**Checkpoint**: `app/main.go` compiles cleanly with `go build ./...`; `TEMPLATES_DIR` env var is read at startup.

---

## Phase 3: User Story 2 — All Inline HTML Extracted to Template Files (Priority: P1) 🎯 MVP

**Goal**: Replace the three inline HTML string constants with calls that read `.html` files from `templatesDir` at request time, using `html/template`. This is the structural prerequisite for live reload (US1) and makes page HTML independently editable.

**Independent Test**: All three previously-inline pages (`/`, `/host`, `/leaderboard`) render correctly in a browser after `docker compose up --build`. Inspecting `app/main.go` shows zero multi-line HTML string constants.

### Implementation for User Story 2

- [x] T007 [P] [US2] Update `handleRootOrAsset` in `app/main.go`: replace the inline `gettingStartedPageHTML` write with `template.ParseFiles(filepath.Join(a.templatesDir, "getting-started.html"))` → `tmpl.Execute(w, nil)`; wire `handleRootOrAsset` as a method on `*App` or pass `a.templatesDir` as an argument so it can read from disk; return `http.Error(..., 500)` + `log.Printf(...)` with the file path on any error
- [x] T008 [P] [US2] Update `handleHost` in `app/main.go`: replace `w.Write([]byte(hostPageHTML))` with `template.ParseFiles(filepath.Join(a.templatesDir, "host.html"))` → `tmpl.Execute(w, nil)`; return `http.Error(..., 500)` + `log.Printf(...)` with the file path on any error
- [x] T009 [US2] Update `handleLeaderboardPage` in `app/main.go`: replace `template.New("leaderboard").Parse(leaderboardPageTemplate)` with `template.ParseFiles(filepath.Join(a.templatesDir, "leaderboard.html"))`; keep the same data struct (`CommitsServiceURL`, `ScoresServiceURL`); return `http.Error(..., 500)` + `log.Printf(...)` with the file path on any error (depends on T007, T008 being structurally consistent)
- [x] T010 [US2] Remove the `gettingStartedPageHTML`, `hostPageHTML`, and `leaderboardPageTemplate` const strings from `app/main.go`; confirm `go build ./...` succeeds and no references remain
- [x] T011 [US2] Add `COPY templates/ /templates` to `app/Dockerfile` (after the existing `COPY frontend/ /frontend` line) so template files are baked into the image for production
- [x] T012 [US2] Update `docker-compose.yml` app service: add `- TEMPLATES_DIR=/templates` to `environment` and `- ./templates:/templates` to `volumes` (bind mount so host edits reach the running container immediately, same pattern as the existing `.git` volume in `commits-service`)

**Checkpoint**: `docker compose up --build` succeeds; visiting `/`, `/host`, and `/leaderboard` in a browser renders correctly; `/play` continues to work unchanged; `app/main.go` has no multi-line HTML string constants.

---

## Phase 4: User Story 1 — Live Reload on Template Edit (Priority: P1)

**Goal**: A developer editing any template file in `templates/` sees the browser automatically reload within ~5 seconds — no `docker compose restart` needed. The existing `/api/ping` polling mechanism is extended with a composite `id` field that changes when the watcher detects a file modification.

**Independent Test**: Open `http://localhost/` in a browser. Edit `templates/getting-started.html` and save. The browser reloads within ~5 seconds and displays the change. DevTools → Network → filter `ping` shows `id` incrementing after the edit.

### Implementation for User Story 1

- [x] T013 [US1] Add `watchTemplates(ctx context.Context)` method to `*App` in `app/main.go`: every 1 second call `os.Stat` on each file in `a.templatesDir` (`getting-started.html`, `host.html`, `leaderboard.html`); store a `map[string]time.Time` of last-seen mtimes; when any file's `ModTime()` advances, call `a.templateVersion.Add(1)` and update the baseline; log any `os.Stat` errors but do not crash the goroutine
- [x] T014 [US1] In `main()` in `app/main.go`, after creating the `App` struct and before starting the HTTP listeners, add `go app.watchTemplates(context.Background())` to start the background polling goroutine
- [x] T015 [US1] Convert `handlePing` from a package-level function to an `(a *App)` method in `app/main.go`; change the response from `` fmt.Fprintf(w, `{"id":%q}`, startupID) `` to `` fmt.Fprintf(w, `{"id":%q}`, startupID+"."+strconv.FormatInt(a.templateVersion.Load(), 10)) ``; add `"strconv"` to imports
- [x] T016 [US1] Update `ungatedMux()` and `gatedMux()` in `app/main.go` to register `a.handlePing` (the new App method) instead of the old package-level `handlePing` at the `/api/ping` route

**Checkpoint**: After `docker compose up --build`, edit any `.html` file in `templates/`, observe the browser reload automatically within ~5 seconds. `/api/ping` response `id` value includes a dot-separated version suffix (e.g., `"1234567890.1"`).

---

## Phase 5: User Story 3 — Clear Error Reporting for Template Issues (Priority: P2)

**Goal**: A template syntax error or missing file produces a visible browser error message and a logged file path + error detail — not a blank page or silent hang.

**Independent Test**: Introduce a `{{.Unclosed` syntax error in `templates/host.html`; open `http://localhost/host`; see an HTTP error page; check `docker compose logs app` for a log entry naming the file and parse error. Fix the file; reload the page; it renders correctly.

### Implementation for User Story 3

- [x] T017 [US3] Add startup validation in `main()` in `app/main.go`: after creating the `App` struct (and before starting listeners), call `os.Stat` on each expected template file (`getting-started.html`, `host.html`, `leaderboard.html`) in `cfg.TemplatesDir`; `log.Fatalf(...)` with the file path if any file is missing, so the app fails fast at startup rather than silently returning 500s on first request
- [x] T018 [US3] Audit the three handler updates from Phase 3 (T007, T008, T009) and confirm each returns `http.Error(w, "...", http.StatusInternalServerError)` and calls `log.Printf("template error %s: %v", filePath, err)` on both parse errors and execute errors; make any corrections needed

**Checkpoint**: Missing or malformed template file → 500 response with logged file path within 2 seconds. Fixed file → next request renders correctly.

---

## Phase 6: Tests

**Purpose**: Unit tests for the modified and new handlers, validating correct rendering, dynamic data injection, and watcher behaviour.

- [x] T019 [P] Write unit tests for `handleRootOrAsset` in `app/main_test.go`: create a `t.TempDir()` with a minimal `getting-started.html`, confirm `/` returns 200 with expected content; confirm it returns 500 when the file is missing or malformed
- [x] T020 [P] Write unit tests for `handleHost` in `app/main_test.go`: create a `t.TempDir()` with a minimal `host.html`, confirm `/host` returns 200 with expected content; confirm 500 on missing/malformed file
- [x] T021 [P] Write unit tests for `handleLeaderboardPage` in `app/main_test.go`: create a `t.TempDir()` with a `leaderboard.html` containing `{{.ScoresServiceURL}}` and `{{.CommitsServiceURL}}`, confirm both values are injected correctly in the response body; confirm 500 on missing/malformed file
- [x] T022 [P] Write unit tests for `handlePing` (App method) in `app/main_test.go`: confirm response JSON `id` field is `"<startupID>.0"` on fresh App; increment `templateVersion` manually; confirm `id` becomes `"<startupID>.1"`
- [x] T023 [P] Write unit tests for `watchTemplates` in `app/main_test.go`: create a `t.TempDir()` with template files; start the watcher; advance a file's mtime by writing to it; confirm `templateVersion.Load()` increments within 2 seconds (use `time.Sleep` or a short-poll retry loop in the test)
- [x] T024 Run `go test ./...` from `app/` and confirm all tests pass (existing tests + new tests)

**Checkpoint**: `go test ./...` exits 0; all three handler tests, the ping test, and the watcher test pass.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: End-to-end browser validation and final cleanup.

- [ ] T025 Run all five scenarios from `specs/015-extract-html-templates/quickstart.md` against the live compose stack: (1) pages render correctly, (2) live reload works for all three templates, (3) syntax error handling, (4) dynamic values injected on `/leaderboard` and `/play`, (5) fresh-clone `docker compose up` works without extra steps; record observed outcomes — requires `docker compose up --build` and browser observation (Principle IV)

**Checkpoint**: All five quickstart scenarios pass in the browser — the feature is done per Principle IV (Visible-in-the-Browser Definition of Done).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately; T002, T003, T004 are parallel
- **Phase 2 (Foundational)**: Depends on Phase 1; T005 and T006 are parallel; **blocks Phases 3–5**
- **Phase 3 (US2)**: Depends on Phase 2; T007 and T008 are parallel; T009 depends on T007/T008 (structural consistency); T010 depends on T007–T009; T011 and T012 can run in parallel with T007–T010
- **Phase 4 (US1)**: Depends on Phase 3; T013 and T014 depend on T006; T015 and T016 depend on T013/T014
- **Phase 5 (US3)**: Depends on Phase 3 (handlers must exist before auditing them); T017 depends on T005/T006; T018 depends on T007–T009
- **Phase 6 (Tests)**: T019–T023 are parallel and depend on Phases 3–5; T024 depends on T019–T023
- **Phase 7 (Polish)**: Depends on all prior phases

### User Story Dependencies

- **US2 (P1 — Extract HTML)**: Depends on Phase 2 foundational; no dependency on US1 or US3
- **US1 (P1 — Live Reload)**: Depends on US2 (templates must be on disk before the watcher is meaningful)
- **US3 (P2 — Error Handling)**: Depends on US2 (handlers must read from disk before error paths apply)

### Parallel Opportunities

Within Phase 1: T002, T003, T004 (three different files, no deps)
Within Phase 2: T005, T006 (different concerns in same file — coordinate to avoid conflicts)
Within Phase 3: T007 and T008 (different handlers in same file — coordinate); T011 and T012 (different files)
Within Phase 6: T019, T020, T021, T022, T023 (all different test functions in the same file — coordinate)

---

## Parallel Example: Phase 3 (US2)

```text
# Parallel: two handler updates (different functions, same file — one dev or two sequential edits)
Task T007: Update handleRootOrAsset in app/main.go
Task T008: Update handleHost in app/main.go

# Parallel with handler work: infrastructure tasks (different files entirely)
Task T011: Update app/Dockerfile
Task T012: Update docker-compose.yml
```

---

## Implementation Strategy

### MVP First (User Story 2 Only — Template Extraction without Live Reload)

1. Complete Phase 1: Create template files
2. Complete Phase 2: Config + struct fields
3. Complete Phase 3: Handler updates + Dockerfile + Compose
4. **STOP and VALIDATE**: All pages render correctly from external templates; `main.go` has no inline HTML constants
5. Demo-ready: template files are editable (changes take effect on next browser load, no auto-refresh yet)

### Full Feature Delivery

1. Phase 1 + 2 → foundation ready
2. Phase 3 (US2) → extraction complete → validate in browser
3. Phase 4 (US1) → live reload working → validate edit→reload cycle in browser
4. Phase 5 (US3) → error handling → validate syntax error scenario
5. Phase 6 (Tests) → all unit tests pass
6. Phase 7 (Polish) → all quickstart scenarios confirmed in browser

---

## Notes

- [P] tasks operate on different files or non-conflicting sections — safe to run in parallel
- Template files in `templates/` are project-authored HTML; no attribution entry needed
- `frontend/index.html` (served at `/play`) is already read from disk and is unchanged by this feature
- The watcher goroutine uses `context.Background()` — if the app is later refactored to support graceful shutdown, pass a cancellable context instead
- Commit after Phase 3 checkpoint (extracting templates is independently useful even without live reload)
