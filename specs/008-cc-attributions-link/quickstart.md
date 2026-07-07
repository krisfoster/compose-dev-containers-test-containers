# Quickstart Validation Guide: CC Attributions and Leaderboard Link

## Prerequisites

- Docker Desktop running
- Repo cloned and at the root

## Start the stack

```bash
docker compose up --build -d
```

## Validation scenarios

### Scenario 1 — CC attribution visible on start screen

1. Open `http://localhost:8080/play` in a browser.
2. The name-entry start screen (`#name-prompt`) overlays the full viewport.
3. **Verify**: Below the "Play" button, the following credit line is visible:
   - Asset title: "Moby Dock (Docker whale)"
   - Author: Maurice Svay
   - Source link pointing to `https://sketchfab.com/3d-models/moby-dock-docker-whale-b706010291ca46ad8daca2d4aeb79edd`
   - Licence: CC BY 4.0 (with link to `https://creativecommons.org/licenses/by/4.0/`)
   - Modification note: "Scaled and re-oriented for use in this project."
4. **Verify**: The attribution is readable without scrolling.
5. **Verify**: The attribution does not overlap the name-input field or the Play button.

### Scenario 2 — Attribution link opens in a new tab

1. On the start screen at `http://localhost:8080/play`, click the Sketchfab source link in the attribution.
2. **Verify**: A new browser tab opens to the Sketchfab model page.
3. **Verify**: The original game tab remains on the start screen (not navigated away).

### Scenario 3 — Leaderboard link visible and functional

1. On the start screen at `http://localhost:8080/play`, find the leaderboard link.
2. **Verify**: A clearly labelled link to the leaderboard is present below the form.
3. Click the leaderboard link.
4. **Verify**: The leaderboard page opens (in a new tab or same tab, depending on implementation choice documented in research §2).
5. **Verify**: The original game tab remains on the start screen.

### Scenario 4 — Mobile viewport: no overlap or clipping

1. Open `http://localhost:8080/play` in a browser with a mobile emulation viewport (e.g. iPhone SE 375×667).
2. **Verify**: The name-entry form, Play button, leaderboard link, and CC attribution are all visible without scrolling.
3. **Verify**: No elements overlap each other.
4. **Verify**: Attribution text wraps cleanly; no horizontal overflow.

### Scenario 5 — Start screen dismisses normally after name entry

1. Enter any name in the name-input field and press Play (or hit Enter).
2. **Verify**: The `#name-prompt` overlay dismisses and the game starts normally.
3. **Verify**: The CC attribution and leaderboard link disappear with the overlay (they are children of it, so this happens automatically).

## Teardown

```bash
docker compose down
```
