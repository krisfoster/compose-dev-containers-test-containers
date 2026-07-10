# Quickstart Validation Guide: Phone Touch Controls

## Prerequisites

- Docker Desktop running
- `docker compose up` started from repo root (game served via nginx at `/`)
- A phone (or browser DevTools with touch emulation) reachable at the game URL

## Exposing the game to a phone

The sandbox is not directly reachable from a phone. To expose it, run on the host:

```bash
sbx ports <sandbox-name> --publish 80:80/tcp
```

Then open `http://<host-ip>:80/` on the phone. Alternatively use the existing ngrok tunnel if
`docker compose --profile ngrok up` is running.

## Scenario 1: On-screen buttons are hidden on a phone

**Setup**: Open the game on a physical phone or in Chrome DevTools (F12 → Toggle device toolbar → select a phone preset).

**Run**:
1. Load the game page
2. Observe the bottom-left corner

**Expected**: The D-pad control buttons (`#controls` div) are not visible.

**Fail if**: The four arrow buttons are visible in the bottom-left corner.

---

## Scenario 2: On-screen buttons remain visible on desktop

**Setup**: Open the game in a desktop browser (no touch emulation active).

**Run**:
1. Load the game page
2. Observe the bottom-left corner

**Expected**: The four directional arrow buttons are visible.

**Fail if**: The buttons are hidden.

---

## Scenario 3: Tap to the right of the whale moves it right

**Setup**: Phone or touch emulation, name entered (game active).

**Run**:
1. Identify the whale's position on screen
2. Tap clearly to the right of the whale (at least 20px to the right, minimal vertical offset)

**Expected**: The whale rotates (if not already facing right) then jumps one step to the right.

**Fail if**: No movement, or whale moves in any other direction.

---

## Scenario 4: Tap to the left of the whale moves it left

**Run**: Tap clearly to the left of the whale.

**Expected**: Whale rotates to face left (if needed) and jumps one step left.

---

## Scenario 5: Tap above the whale moves it forward

**Run**: Tap clearly above the whale on screen (minimal horizontal offset).

**Expected**: Whale moves forward one step (lane count increments by 1).

---

## Scenario 6: Tap below the whale moves it backward

**Run**: Tap clearly below the whale on screen.

**Expected**: Whale rotates to face backward (if needed) and moves one step back.

**Fail if**: Movement is blocked only if the whale is already at lane 0 (correct behaviour — matches keyboard).

---

## Scenario 7: Diagonal tap resolves to the dominant axis

**Run**: Tap at roughly 30° diagonal (e.g., slightly up and more to the right).

**Expected**: Whale moves right (dominant horizontal axis), not diagonally.

---

## Scenario 8: Tap directly on the whale does nothing

**Run**: Tap as close to the centre of the whale as possible.

**Expected**: No movement. Whale stays still.

---

## Scenario 9: Multi-touch is ignored (only first touch counts)

**Run**: Touch with two fingers simultaneously — one to the right of the whale, one to the left.

**Expected**: Whale moves in one direction only (whichever finger landed first). No double-move or cancel.

---

## Scenario 10: Page does not scroll when tapping the game canvas

**Run**: On a phone where the page might scroll (e.g., embedded in a taller page), tap rapidly in different directions on the canvas.

**Expected**: No page scroll occurs. The game canvas remains fixed. The whale responds to taps.

---

## Scenario 11: Landscape and portrait both work

**Run**: Start in portrait, play a few taps, rotate phone to landscape, play a few more taps.

**Expected**: Tap controls work in both orientations. Buttons remain hidden in both.

---

## Orientation Checks After Each Scenario

- Lane counter increments correctly for each forward move
- Whale collision and boundary rules are unchanged (cannot move past column 0 or column 16, or into a forest obstacle)
- Game-over screen appears normally when a vehicle hits the whale

## Known Constraints

- Touch controls are only active when `pointer: coarse` is detected at page load. Switching input devices mid-session (e.g., connecting a mouse to a phone) does not re-evaluate detection.
- The dead zone (10 CSS pixels centred on the whale's projected screen position) means tapping the whale itself is intentionally a no-op.
