# Tasks: App Serves Templates Only

**Feature**: 017-app-serves-templates-only  
**Input**: plan.md, research.md

---

## Phase 1: Setup

- [x] T001 Run `go test ./...` from the `app/` directory and confirm all tests
  pass before making any changes; note the test count as the baseline.

---

## Phase 2: Add index.html to templates/

**Purpose**: Establish the new canonical location for `index.html` before changing
any code that reads it. Keeps the build green throughout.

- [x] T002 Copy `frontend/game/index.html` to `templates/index.html`. Do not
  delete `frontend/game/index.html` — nginx's `COPY frontend/game` still needs it
  there. This is a copy, not a move.

**Checkpoint**: `ls templates/` shows four files: `getting-started.html`,
`host.html`, `leaderboard.html`, `index.html`.

---

## Phase 3: Update Go source

**Purpose**: Wire the app to read `index.html` from `templatesDir`, remove both
asset-serving paths, and clean up all dead config.

- [x] T003 In `app/main.go`, update `handlePlayIndex`:
  - Change `template.ParseFiles(filepath.Join(a.frontendDir, "index.html"))` to
    `template.ParseFiles(filepath.Join(a.templatesDir, "index.html"))`.
  - Replace the doc comment (currently references the removed leaderboard credential
    injection from spec 003) with: `// handlePlayIndex serves the game's index page.
    // The template lives in templatesDir alongside the other page templates.`

- [x] T004 In `app/main.go`, update `watchTemplates`: add
  `filepath.Join(a.templatesDir, "index.html")` to the `files` slice (alongside
  `getting-started.html`, `host.html`, `leaderboard.html`) so browser auto-reload
  fires when the game page changes on disk.

- [x] T005 In `app/main.go`, update `ungatedMux()`:
  - Delete the line `fileServer := http.FileServer(http.Dir(a.frontendDir))`.
  - Change `mux.Handle("/", a.handleRootOrAsset(fileServer))` to
    `mux.HandleFunc("/", a.handleRootOrAsset)`.
  - Delete `mux.Handle("/leaderboard-assets/", http.StripPrefix("/leaderboard-assets/",
    http.FileServer(http.Dir(a.leaderboardAssetsDir))))`.

- [x] T006 In `app/main.go`, update `handleRootOrAsset`:
  - Remove the `assets http.Handler` parameter from the function signature.
  - Replace the `assets.ServeHTTP(w, r)` call in the non-root branch with
    `http.NotFound(w, r)`.
  - The function now takes only `(w http.ResponseWriter, r *http.Request)` and its
    signature must match `http.HandlerFunc` (since T005 registers it with
    `mux.HandleFunc`).

- [x] T007 In `app/main.go`, remove `frontendDir` and `leaderboardAssetsDir` from
  the `App` struct (two field deletions).

- [x] T008 In `app/main.go`, remove `FrontendDir` and `LeaderboardAssetsDir` from
  the `Config` struct (two field deletions).

- [x] T009 In `app/main.go`, remove from `loadConfig()`:
  - `FrontendDir: envOr("FRONTEND_DIR", "/frontend"),`
  - `LeaderboardAssetsDir: envOr("LEADERBOARD_ASSETS_DIR", "/leaderboard-assets"),`

- [x] T010 In `app/main.go`, remove from the `app := &App{...}` literal in `main()`:
  - `frontendDir: cfg.FrontendDir,`
  - `leaderboardAssetsDir: cfg.LeaderboardAssetsDir,`

- [x] T011 In `app/main.go`, update the package-level doc comment (lines 1–9):
  remove the stale reference to leaderboard API spec files
  (`specs/004-leaderboard-page/contracts/leaderboard-openapi.yaml`) — that package
  is deleted.

**Checkpoint**: `go build ./...` from `app/` — must compile cleanly with no
references to the removed fields.

---

## Phase 4: Update tests

**Purpose**: Align the test suite with the new code. The behaviour under test is
unchanged; only the setup plumbing moves.

- [x] T012 In `app/main_test.go`, add `"index.html"` to the `files` map inside
  `newTestTemplatesDir()` with value `<html><body>game</body></html>` — this is the
  fake game page the handler tests assert against.

- [x] T013 In `app/main_test.go`, update `newTestApp()`:
  - Delete the `frontendDir` temp dir creation block (lines 81–87: `frontendDir :=
    t.TempDir()`, both `os.WriteFile` calls, and the `t.Fatalf` lines).
  - Remove `frontendDir: frontendDir` from the `App` struct literal.
  - The `testScriptJS` constant (line 20) and its comment (lines 17–19) become
    unused; delete them.

- [x] T014 In `app/main_test.go`, delete the test function
  `TestHandleRootFallsThroughToStaticAssets` (lines 400–415). The behaviour it
  tested — non-root paths falling through to the file server — no longer exists;
  the app now returns 404 for unmatched paths on direct :8080 access.

- [x] T015 In `app/main_test.go`, update `TestHandlePlayIndexWhenTemplateFileMissing`
  (line 424): change `app.frontendDir = t.TempDir()` to
  `app.templatesDir = t.TempDir()` — an empty templatesDir now triggers the missing
  template error, same behaviour as before.

- [x] T016 In `app/main_test.go`, update `appWithErroringStore()` (lines 668–682):
  - Delete `frontendDir := t.TempDir()` (line 672).
  - Remove `frontendDir: frontendDir` from the returned `&App{...}` literal (line 676).

- [x] T017 In `app/main_test.go`, update `TestHandleHostWhenActivateFails`
  (lines 715–728): remove `frontendDir: t.TempDir(),` from the inline `&App{...}`
  literal (line 720).

- [x] T018 In `app/main_test.go`, update `TestLoadConfigDefaults` (line 609):
  remove `"FRONTEND_DIR"` from the `[]string{...}` slice passed to the `t.Setenv`
  loop — `FRONTEND_DIR` is no longer a recognised config key.

**Checkpoint**: `go test ./...` from `app/` — all tests pass; no compile errors.

---

## Phase 5: Remove assets from app Dockerfile and Compose

- [x] T019 In `app/Dockerfile`, delete these two lines:
  ```dockerfile
  COPY frontend/game /frontend
  COPY frontend/leaderboard /leaderboard-assets
  ```
  Also remove or update the Dockerfile header comment that says the build context
  copies `frontend/game` into the image.

- [x] T020 In `docker-compose.yml`, remove from the `app` service `environment`
  block:
  - `- FRONTEND_DIR=/frontend`
  - `- LEADERBOARD_ASSETS_DIR=/leaderboard-assets`

---

## Phase 6: Validate

- [x] T021 Run `go test ./...` from `app/` — all tests must pass.

- [x] T022 Run `docker compose up --build -d` — stack starts with no errors.

- [x] T023 Verify the game loads end-to-end: open `http://localhost:8081/play-local`,
  confirm the game page renders and scripts load (network tab shows 200s for
  `script.js`, `style.css`, model assets).

- [x] T024 Verify the leaderboard loads: open `http://localhost:8081/leaderboard`,
  confirm the page renders with live score data.

- [x] T025 Confirm the app image no longer carries the redundant assets:
  ```bash
  docker exec whale-runner-app-1 ls /
  ```
  Expected: no `/frontend` or `/leaderboard-assets` directories.

---

## Dependencies

- T002 must complete before T003 (the file must exist before Go reads it in tests).
- T003–T011 are independent of each other within Phase 3 but all must compile before
  Phase 4 tests can run.
- T012–T018 are independent of each other within Phase 4 (all edit `main_test.go`
  but at different locations).
- T019–T020 are independent of each other.
- Phase 6 requires all prior phases complete.

## Parallelism

| Task | Can run in parallel with |
|---|---|
| T003 (handlePlayIndex) | T004, T007, T008, T009, T010, T011 |
| T004 (watchTemplates) | T003, T007, T008, T009, T010, T011 |
| T019 (Dockerfile) | T020 (docker-compose.yml) |
| T012–T018 (test updates) | Each other (different line ranges in the same file — apply sequentially in practice) |
