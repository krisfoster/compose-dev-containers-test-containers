# Feature Specification: Add CC Attributions and Leaderboard Link to Game

**Feature Branch**: `008-cc-attributions-link`

**Created**: 2026-07-07

**Status**: Draft

**Input**: User description: "Add CC attributions to the game. They should be there, but they are not. If allowed, add the CC attributions on the start view of the app, where the user enters their name. Also add a link to the game to get to the leader-board."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View CC Attribution on Start Screen (Priority: P1)

A player loads the game and lands on the name-entry start screen. Before playing, they can clearly see the Creative Commons attribution for the Moby Dock whale model used in the game — including the author's name, the asset title, the source URL, and the CC BY 4.0 licence notice. They can click the attribution link to visit the original model page on Sketchfab.

**Why this priority**: The CC BY 4.0 licence legally requires visible attribution in any app that uses the asset. The project constitution (Principle V) also makes this non-negotiable. The ATTRIBUTION.md already documents the required credit line and flags it as something that must surface before shipping.

**Independent Test**: Can be fully tested by loading the game start screen and verifying the credit line is visible, readable, and links correctly to the Sketchfab model page — without needing to play the game at all.

**Acceptance Scenarios**:

1. **Given** the game start screen is displayed, **When** the player views the page, **Then** they see the credit line: "Moby Dock (Docker whale)" by Maurice Svay, licensed under CC BY 4.0, with a visible link to the Sketchfab source URL.
2. **Given** the credit line is displayed, **When** the player clicks the attribution link, **Then** they are taken to the Sketchfab model page in a new browser tab (without leaving the game).
3. **Given** the start screen is viewed on a mobile device, **When** the player looks at the attribution, **Then** it is legible and does not obscure the name-entry form or Play button.

---

### User Story 2 - Navigate from Game to Leaderboard (Priority: P2)

A player who is on the game start screen (name-entry view) can see a link to the leaderboard and click it to navigate there. This lets curious players check the current standings before committing to a game, and gives presenters an easy path to show the leaderboard to an audience.

**Why this priority**: Improves the in-game navigation flow and supports demo use cases, but the game is fully functional without it. The leaderboard already exists as a standalone page; this is a discoverability improvement.

**Independent Test**: Can be fully tested by loading the game start screen and verifying a leaderboard link is present and navigates to the leaderboard page.

**Acceptance Scenarios**:

1. **Given** the game start screen is displayed, **When** the player views the page, **Then** they see a clearly labelled link or button to the leaderboard.
2. **Given** the leaderboard link is present, **When** the player clicks it, **Then** they are taken to the leaderboard page.
3. **Given** the start screen is viewed on mobile, **When** the player looks at the leaderboard link, **Then** it is tappable and does not overlap the name-entry form or Play button.

---

### Edge Cases

- What happens when the leaderboard service is unreachable — does the start screen still load and show the attribution and link normally?
- How is the attribution displayed if the whale model fails to load (the primitive whale fallback is shown instead)?
- Is the attribution text readable at high contrast and on projectors (the game is shown at booth demos on projected walls)?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The game start screen MUST display the CC BY 4.0 attribution for the Moby Dock whale model.
- **FR-002**: The attribution MUST include: the asset title ("Moby Dock (Docker whale)"), the author (Maurice Svay), the licence (CC BY 4.0), and a link to the Sketchfab source URL.
- **FR-003**: The attribution link MUST open the Sketchfab model page in a new browser tab so the player does not leave the game.
- **FR-004**: The attribution MUST be visible on the start screen without scrolling, on both desktop and mobile viewports.
- **FR-005**: The attribution MUST NOT obstruct or displace the name-entry form, name input field, or Play button.
- **FR-006**: The start screen MUST include a link to the leaderboard page.
- **FR-007**: The leaderboard link MUST be clearly labelled so players understand where it leads.
- **FR-008**: The attribution text MUST exactly match the required credit line recorded in `frontend/game/ATTRIBUTION.md` (title, author, source URL, licence).

### Key Entities

- **Start Screen**: The name-entry view (`#name-prompt` in `index.html`) displayed before the game begins. This is where both the attribution and the leaderboard link will appear.
- **CC Attribution**: The legally required credit line for the Moby Dock (Docker whale) model, sourced from `frontend/game/ATTRIBUTION.md`.
- **Leaderboard Link**: A navigation element linking to the existing leaderboard page served by the app.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of players see the CC BY 4.0 attribution before the game begins (it is on the start screen every player must pass through).
- **SC-002**: The attribution satisfies all four required CC BY 4.0 display elements: asset title, author name, source URL, and licence name — verifiable by visual inspection against the credit line in `ATTRIBUTION.md`.
- **SC-003**: A player can reach the leaderboard from the game start screen in a single click or tap, on both desktop and mobile.
- **SC-004**: The start screen layout — including attribution and leaderboard link — is visually clean and does not introduce any overlapping or displaced UI elements, verified by loading the page in a browser at desktop and mobile viewport sizes.

## Assumptions

- The leaderboard page already exists and is accessible at a known relative URL within the same app; no new leaderboard backend work is required.
- The required credit line from `frontend/game/ATTRIBUTION.md` is authoritative; the spec does not re-evaluate licence compliance beyond what is documented there.
- The attribution will appear on the start screen (#name-prompt) rather than in a separate credits page — this is the most demo-visible surface that every player passes through.
- No other CC-licensed or attribution-required assets beyond the Moby Dock whale model currently need visible in-app attribution (the game code is MIT-licensed, which does not require visible credit in the UI).
- The leaderboard link destination (URL) will be confirmed at implementation time based on the running compose stack's routing.
