# Quickstart Validation Guide: Mobile Support

**Feature**: `007-mobile-support`  
**Date**: 2026-07-07

This guide documents the runnable validation scenarios that prove the mobile support feature works end-to-end. Run these after implementation; all require the compose stack to be up.

---

## Prerequisites

- Docker Desktop running on the host
- Compose stack up: `docker compose up` from the repo root
- ngrok tunnel established (required for QR code scenarios)
- A smartphone (iOS Safari 15+ or Android Chrome 108+) on the same network, or Chrome DevTools mobile emulation

---

## Setup Commands

```bash
# Start the full stack
docker compose up

# Verify the app is reachable
curl -s http://localhost:8080/ | grep "Crossy Whale"

# Open the presenter host page to activate a QR window
open http://localhost:8080/host        # macOS
# or: xdg-open http://localhost:8080/host   # Linux
```

---

## Scenario 1: Mobile Tap Controls — Real Device

**Goal**: Confirm tap controls move the whale on a physical phone.

1. Open the ngrok URL from `/host` on a smartphone, or navigate to the app's local IP on port 8080 (the phone must be on the same Wi-Fi network).
2. Hold the phone in **landscape** orientation.
3. Enter a name and tap **Play**.
4. Tap each of the four directional buttons (↑ ↓ ← →) one at a time.

**Expected outcome**: The whale moves one step per tap in the corresponding direction. No double-movement or missed taps. No JavaScript errors in the browser console (use Safari Web Inspector or Chrome remote debugging to check).

---

## Scenario 2: Portrait Mode Prompt — Real Device or DevTools

**Goal**: Confirm the portrait overlay appears and disappears correctly.

### Using Chrome DevTools (fastest path):
1. Open `http://localhost:8080/play` in Chrome.
2. Open DevTools → Toggle Device Toolbar (Ctrl+Shift+M / Cmd+Shift+M).
3. Select an iPhone or Android preset.
4. Set orientation to **Portrait**.

**Expected outcome**: A full-screen overlay appears with a "rotate your device" message. The game canvas and controls are not interactive beneath it.

5. Switch orientation to **Landscape**.

**Expected outcome**: Overlay disappears. Game canvas is visible and the name prompt or game controls are accessible.

### Using a real device:
1. Open the game URL on the phone.
2. Hold in portrait; confirm overlay appears.
3. Rotate to landscape; confirm overlay disappears.
4. Rotate back to portrait mid-game; confirm overlay reappears without a JS error.

---

## Scenario 3: Full-Screen Landscape Layout — DevTools

**Goal**: Confirm the canvas fills the viewport with no scroll bars.

1. Open `http://localhost:8080/play` in Chrome DevTools mobile emulation.
2. Select iPhone 14 Pro or similar (390 × 844 logical pixels). Set to **landscape** (844 × 390).
3. Inspect the `<canvas>` element.

**Expected outcome**:
- Canvas computed width: 844px (or full viewport width).
- Canvas computed height: 390px (or full viewport height minus any browser chrome).
- No horizontal or vertical scroll bars visible.
- No white gap at the bottom from the address bar overflow.

---

## Scenario 4: Home Page Two-Column Layout

**Goal**: Confirm the home page at `/` shows a 2-column layout with the QR code on the right.

1. Open `http://localhost:8080/` in a desktop browser.

**Expected outcome**: Two visible columns — left column has the Play / Host / Leaderboard links; right column shows the QR code image.

2. Resize the browser window to below 600px wide (or use DevTools mobile emulation).

**Expected outcome**: Layout collapses to a single column; no content is clipped or overflowed.

---

## Scenario 5: Home Page QR Auto-Refresh

**Goal**: Confirm the QR code on the home page updates when the QR window is rotated.

1. Open `http://localhost:8080/` on the presenter's laptop and note the QR code image.
2. In a separate tab, open `http://localhost:8080/host` and click **Rotate QR**.
3. Wait up to 5 seconds.

**Expected outcome**: The QR code image on the home page updates to the new code automatically, without a manual page reload.

---

## Scenario 6: QR Missing State (No Active Window)

**Goal**: Confirm the home page handles the case where no QR window has been activated yet.

1. Stop and restart the compose stack (`docker compose restart app`), which clears Redis.
2. Open `http://localhost:8080/` immediately, before opening `/host`.

**Expected outcome**: The QR column shows a placeholder or empty image area (not a broken-image icon). No JavaScript error is thrown. After opening `/host` and waiting up to 5 seconds, the QR image appears.

---

## References

- Touch event handler contract: [data-model.md §5](data-model.md#5-touch-event-handler-contract)
- Orientation state machine: [data-model.md §1](data-model.md#1-orientation-state-machine)
- QR poller contract: [data-model.md §4](data-model.md#4-home-page-qr-poller-contract)
- Research decisions: [research.md](research.md)
