# Research: CC Attributions and Leaderboard Link

## §1 — CC BY 4.0 Attribution Requirements

**Decision**: The attribution must include: (1) the asset title, (2) the author name, (3) a hyperlink to the original source, and (4) the licence name with a hyperlink to the licence text.

**Rationale**: CC BY 4.0 requires you to give appropriate credit, provide a link to the licence, and indicate if changes were made. "Appropriate credit" means, at minimum, the title, the author, the source URL, and the licence name. The ATTRIBUTION.md file already captures the exact required credit line:

> "Moby Dock (Docker whale)" by Maurice Svay, https://sketchfab.com/3d-models/moby-dock-docker-whale-b706010291ca46ad8daca2d4aeb79edd, licensed under CC BY 4.0. Scaled and re-oriented for use in this project.

That line is the canonical attribution text. Modifications are documented: the model was rotated +90° around the X axis and auto-scaled to a 60 world-unit longest axis.

**Alternatives considered**: A footer-only credit (invisible on the start screen) — rejected, because the constitution (Principle V) requires attribution on a *demo-visible surface* before ship. A separate credits page — rejected, because players might not find it; the start screen is the guaranteed touchpoint.

---

## §2 — Leaderboard URL (relative to the game)

**Decision**: Use `/leaderboard` as the href for the leaderboard link. Open it in a new tab (`target="_blank" rel="noopener"`).

**Rationale**: The Go backend (`app/main.go`) registers `/leaderboard` on both the ungated mux (line 141) and the gated mux (line 158), so the route is always available regardless of which listener served the game. Because the game fills the entire viewport, opening the leaderboard in a new tab preserves the game session rather than navigating the player away from it.

**Alternatives considered**: Using a relative path (`../leaderboard`) — rejected, because the game is served at `/play`, and a relative path from `/play` to a sibling `/leaderboard` would require `./leaderboard` which resolves correctly, but `/leaderboard` is clearer and works from both `/` and `/play`. Opening in the same tab — rejected, because it destroys the player's game state (name entered, game in progress).

---

## §3 — UI Placement on the Start Screen

**Decision**: Append the attribution and leaderboard link inside the existing `#name-prompt` overlay, below the `<form>` element.

**Rationale**: The `#name-prompt` div is a full-screen overlay (`position: absolute; min-width: 100%; min-height: 100%; z-index: 20`) with a centred flex column. Adding elements inside it keeps them visually grouped with the name-entry form. The form's flex column layout means new children stack naturally below the Play button. The overlay dismisses when the player submits the form, so the attribution is visible exactly during the pre-game window — both criteria (visible before play starts, not obstructing gameplay) are met simultaneously.

The existing `.credits` div (bottom-right, `position: fixed; font-size: 13px; width: 160px`) is too small for the full CC credit and does not interact with the start screen z-index, so it is not the right surface. It can remain unchanged.

**Alternatives considered**: Modifying the `.credits` div — rejected; the div is too small (160px wide) to hold the full attribution and was designed for a one-line "Made with ❤️" note. A separate modal — rejected; adds complexity for a simple static text addition. Placing it above the form — rejected; the label "Enter your name to play" should remain the first thing the player sees.

---

## §4 — CSS Styling Approach

**Decision**: Use small-print styling (font-size ~0.35em relative to body, `font-family: inherit`), white text with reduced opacity on the start-screen overlay background. Links styled with muted contrast, underlined.

**Rationale**: The `#name-prompt` background is `rgba(0, 0, 0, 0.85)` — a dark overlay — so white-on-dark text is readable without any additional background. The attribution is supporting information, not primary UI, so it should be visually subordinate to the form. The existing label at `0.6em` sets the precedent; the attribution can be slightly smaller (`0.35em`) to indicate lower visual hierarchy.

Mobile-viewport constraints: the start screen uses `align-items: center; justify-content: center` on a flex column. The attribution paragraph needs `text-align: center` and `max-width` to avoid overflowing narrow viewports.

**Alternatives considered**: A tooltip on hover — rejected; not discoverable on mobile. A collapse/expand toggle — rejected; the text is short enough to display inline.

---

## §5 — Scope Boundaries

- **In scope**: `frontend/game/index.html` and `frontend/game/style.css`. No backend (Go) changes.
- **Out of scope**: Changes to `ATTRIBUTION.md` (it is already correct). Changes to the leaderboard page itself. Changes to the `.credits` div (it can remain as-is). Any changes to the game logic (`script.js`).
- **No new services, routes, or API contracts** are introduced by this feature.
