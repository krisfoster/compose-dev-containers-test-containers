# Quickstart & Validation Guide: Go HTML Template Extraction with Live Reload

**Feature**: 015-extract-html-templates
**Date**: 2026-07-09

---

## Prerequisites

- Docker Desktop running
- Repository cloned: `git clone <repo>`
- `.env` file present (copy `.env.example` and fill in `GRANT_COOKIE_SECRET`, `LEADERBOARD_API_SECRET`)
- No prior `docker compose up` session running

---

## Setup

```bash
# From repo root
docker compose build
docker compose up
```

Wait until all services are healthy (nginx, app, scores-service, commits-service, redis).

---

## Scenario 1: Pages Render Correctly (No Regression)

**Goal**: Confirm all pages render after the template extraction.

| URL | Expected Content |
|-----|-----------------|
| `http://localhost/` | Getting-started landing page with "Play the game", "Host: show the QR code", "View the leaderboard" buttons |
| `http://localhost/host` | Host presenter page with QR image and "Rotate QR" button |
| `http://localhost/leaderboard` | Leaderboard page with standings and recent commits columns |
| `http://localhost/play?w=<token>` | Game loads; leaderboard token injected (check DevTools → Sources → `window.__LEADERBOARD_TOKEN__`) |

---

## Scenario 2: Template Live Reload — Edit and See Change in Browser

**Goal**: Confirm that editing a template file triggers an automatic browser reload.

**Steps**:

1. Open `http://localhost/` in a browser. Note the heading text.
2. In the repository on the host, open `templates/getting-started.html`.
3. Edit the `<h1>` tag — e.g., change `Crossy Whale` to `Crossy Whale (edited)`.
4. Save the file.
5. Watch the browser tab. Within ~5 seconds it should reload automatically and display the new heading.

**Repeat** for the other templates:
- Edit `templates/host.html` → verify at `http://localhost/host`
- Edit `templates/leaderboard.html` → verify at `http://localhost/leaderboard`

**Expected `/api/ping` behaviour** (optional deep check):

Open DevTools → Network. Filter for `ping`. Observe:
- Before edit: `{"id": "1234567890.0"}` (or similar, version = 0)
- After edit: `{"id": "1234567890.1"}` (version incremented)
- Page reloads automatically at this point

---

## Scenario 3: Template Syntax Error Handling

**Goal**: Confirm the app handles a malformed template file gracefully.

**Steps**:

1. Open `templates/host.html`.
2. Introduce a syntax error: add `{{.Unclosed` anywhere in the file body and save.
3. Open `http://localhost/host` (or reload if already open).

**Expected**:
- Browser shows an HTTP error page (not a blank page, not a hang).
- Terminal log (`docker compose logs app`) shows an entry like: `template parse error: <file path>: <error detail>`.

4. Fix the file (remove the bad line). Reload the browser. Page renders normally.

---

## Scenario 4: Dynamic Values Still Injected

**Goal**: Confirm template rendering with data still works after extraction.

**Check `/leaderboard`**:

1. Open `http://localhost/leaderboard`.
2. View page source (`Ctrl+U`).
3. Verify the `ScoresComponent` script tag contains `scoresServiceURL: 'http://localhost'` (or the value of `SCORES_SERVICE_URL` from `.env`).
4. Verify the `CommitsComponent` script tag contains `commitsServiceURL: 'http://localhost'` (or `COMMITS_SERVICE_URL`).

**Check `/play`** (requires an active QR window):

1. Open `http://localhost/host` to activate a QR window.
2. Open `http://localhost/play` in a second tab (or scan the QR code on a phone).
3. Open DevTools → Console.
4. Confirm `window.__LEADERBOARD_TOKEN__` is set to a non-empty string (not the literal `{{.LeaderboardToken}}`).

---

## Scenario 5: Stack Starts Clean from Fresh Clone

**Goal**: Confirm Compose-Orchestrated Reproducibility (Principle II) is maintained.

```bash
docker compose down -v
docker compose build --no-cache
docker compose up
```

Visit all pages in Scenario 1. Everything should work without any extra steps.

---

## What the Verification Does NOT Cover

- The templates directory baked into the Docker image for production (non-volume build): confirmed by `docker compose build` succeeding and pages rendering at startup.
- Performance under load: not a requirement for this feature.
- The `commits-service` and `scores-service` microservices: out of scope (no HTML changes).
