# Feature Specification: Player Name Entry, Game Over Score Display, and Leaderboard Score Submission

**Feature Branch**: `003-leaderboard-score-submission`

**Created**: 2026-07-07

**Status**: Draft

**Input**: User description: "From issues.md, open issue 3: Add support for storing a user's name
and score in a leaderboard that is stored within Redis - do not create the leaderboard page yet.
The game will require updating so that before it starts it asks the user to enter their name. When
the game finishes, when they die, it says game over and displays their score, and the score and
name are written to the leaderboard store from the game. A replay button is underneath which
restarts the game. The leaderboard is its own Go-based API, using OpenAPI. Security note: only the
game can write to the API, so a secret, or similar, must be used to stop arbitrary people writing
to it."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Player enters their name before playing (Priority: P1)

Before a round starts, the player is asked to type in a display name. Only once they've entered a
name does the game actually begin.

**Why this priority**: Every other part of this feature — the game-over score display and the
leaderboard write — depends on a name existing to attach the score to. Without this, there is
nothing to record.

**Independent Test**: Load the game, confirm a name prompt appears before any gameplay is
possible, type a name, and confirm the game starts only after a name has been entered.

**Acceptance Scenarios**:

1. **Given** a player has just loaded the game, **When** the game screen appears, **Then** they are
   prompted to enter a display name before gameplay begins.
2. **Given** the name prompt is showing, **When** the player tries to proceed without typing
   anything (or only whitespace), **Then** the game does not start and they remain on the name
   prompt.
3. **Given** the player has typed a non-empty name and confirmed it, **When** the prompt is
   dismissed, **Then** gameplay starts immediately.

---

### User Story 2 - Game Over screen shows the player their score (Priority: P1)

When the player dies, the game stops and a "Game Over" screen appears showing the score they just
achieved.

**Why this priority**: This is the moment of payoff for playing — the player needs to see how they
did. It's also the trigger point for writing the score to the leaderboard store, which is the core
purpose of this feature.

**Independent Test**: Play until death and confirm a "Game Over" screen appears showing the
numeric score for that attempt.

**Acceptance Scenarios**:

1. **Given** a player is mid-game with a running score, **When** they die, **Then** gameplay stops
   and a "Game Over" screen appears displaying that final score.
2. **Given** the Game Over screen is showing, **When** the player looks at it, **Then** they see
   only their own name and score for that attempt — no other players' scores or any ranking (the
   leaderboard viewing page is a separate, future feature).

---

### User Story 3 - Score and name are recorded to the leaderboard store (Priority: P1)

The moment a player dies, their entered name and the score they achieved are sent to a
Redis-backed leaderboard store via a dedicated API, without the player having to do anything extra.

**Why this priority**: This is the actual feature being requested — capturing player results
durably so they exist for a future leaderboard display. Without this, the game-over screen is
just a local display with nothing persisted.

**Independent Test**: Play a round to completion (die), then confirm — independently of the game
UI — that a new entry with the matching name and score exists in the leaderboard store.

**Acceptance Scenarios**:

1. **Given** a player has entered a name and dies with a given score, **When** the Game Over screen
   appears, **Then** a leaderboard entry containing that name and score has been submitted without
   any extra action from the player.
2. **Given** two different players complete attempts with different names, **When** both scores are
   submitted, **Then** both are recorded as distinct entries and neither overwrites the other.
3. **Given** the same player plays multiple attempts (via Replay) in one visit, **When** each
   attempt ends, **Then** each attempt's score is recorded as its own leaderboard entry under that
   player's name, rather than only the best or only the most recent being kept.

---

### User Story 4 - Replay restarts the game without losing the entered name (Priority: P2)

Underneath the Game Over screen's score, a Replay button lets the player immediately start another
attempt without having to re-type their name.

**Why this priority**: This keeps repeat play frictionless, which matters for a booth/demo setting
where attendees want another go right away. It's lower priority than the scoring/recording flow
because the game is still usable (just less convenient) without it.

**Independent Test**: From the Game Over screen, activate Replay and confirm a fresh game attempt
starts immediately, using the same previously entered name, with no name prompt shown again.

**Acceptance Scenarios**:

1. **Given** the Game Over screen is showing after a completed attempt, **When** the player
   activates Replay, **Then** a new game attempt starts immediately.
2. **Given** the player used Replay, **When** the new attempt later ends, **Then** the score for
   that new attempt is recorded under the same name as before, without asking the player to
   re-enter it.

---

### User Story 5 - Arbitrary clients cannot write scores to the leaderboard (Priority: P2)

The score-write API only accepts submissions that carry a credential the legitimate game client
has; a request sent directly against the API without that credential is rejected and nothing is
recorded.

**Why this priority**: Without this, anyone who finds the API endpoint could flood the future
leaderboard with fake entries, undermining the whole point of recording real play results. It's P2
rather than P1 because the core recording flow (User Story 3) has to exist before there's anything
worth protecting.

**Independent Test**: Send a score-submission request directly to the API without the game's
credential and confirm it is rejected and no entry is recorded; then confirm a normal play-through
via the actual game still succeeds.

**Acceptance Scenarios**:

1. **Given** a request to the score-write API that does not include a valid credential, **When**
   it is submitted, **Then** the API rejects it and no leaderboard entry is created.
2. **Given** a normal game session playing through to death, **When** the game submits the score on
   the player's behalf, **Then** the submission includes a valid credential and succeeds.

---

### Edge Cases

- What happens if the leaderboard store is temporarily unavailable when a score is submitted? The
  player still sees their own score on the Game Over screen regardless of whether the write
  succeeded, and Replay is never blocked by a failed or pending submission.
- What happens if two different players enter the exact same name? Both are recorded as separate
  leaderboard entries; names are not required to be unique.
- What happens if the player enters an extremely long name? The system enforces a reasonable
  maximum length, silently trimming or refusing input beyond it rather than allowing unbounded
  text.
- What happens if the player closes the browser tab immediately after dying, before a submission
  can complete? This is a best-effort submission; no requirement is placed on guaranteeing delivery
  after the page is gone.
- What happens if a score-write request presents an invalid or expired credential (as opposed to a
  missing one)? It is treated the same as a missing credential — rejected, nothing recorded.
- What happens if a player tries to submit a score without ever having played (e.g., a direct API
  call crafted to look like a real attempt but bypassing the game client)? If it carries a valid
  credential the system cannot distinguish it from a real play-through; the credential check is the
  only defense this feature provides against illegitimate submissions.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST prompt the player to enter a display name before a game attempt can
  start.
- **FR-002**: System MUST prevent the game from starting if the entered name is empty or contains
  only whitespace.
- **FR-003**: System MUST enforce a reasonable maximum length on the entered name.
- **FR-004**: System MUST stop gameplay and show a "Game Over" screen immediately when the player
  dies, displaying the score achieved in that attempt.
- **FR-005**: System MUST display only the current player's own name/score on the Game Over screen
  — not other players' entries or any ranking.
- **FR-006**: System MUST submit the player's entered name and the score achieved to the
  leaderboard store automatically when the game ends, with no additional action required from the
  player.
- **FR-007**: System MUST record each completed attempt as its own leaderboard entry, so that
  multiple attempts by the same player (e.g., via Replay) each produce a separate recorded entry
  rather than overwriting a prior one.
- **FR-008**: System MUST NOT let one player's leaderboard entry overwrite a different player's
  entry, regardless of matching or differing names.
- **FR-009**: System MUST provide a Replay control on the Game Over screen that immediately starts
  a new attempt when activated.
- **FR-010**: System MUST retain the previously entered name across a Replay-triggered attempt so
  the player is not asked to re-enter it.
- **FR-011**: System MUST expose leaderboard score submission as a Go-based API, with its
  request/response contract documented via an OpenAPI specification.
- **FR-012**: System MUST require every score-submission request to carry a valid credential known
  only to the legitimate game client, and MUST reject and discard any submission that lacks one or
  presents an invalid one, without recording an entry.
- **FR-013**: System MUST persist accepted leaderboard entries (name and score) in a Redis-backed
  store.
- **FR-014**: System MUST NOT require the player to take any visible action related to the
  credential check — obtaining and presenting it is handled transparently as part of normal play.
- **FR-015**: System MUST NOT provide a page or view for browsing/displaying the leaderboard's
  contents as part of this feature — recording is in scope, viewing is a separate future feature.

### Key Entities

- **Leaderboard Entry**: One recorded result of a completed game attempt — a player-entered
  display name paired with the score achieved. Entries are independent of one another; a new
  attempt always produces a new entry rather than modifying an existing one.
- **Score Submission Credential**: The credential that authorizes writing a Leaderboard Entry.
  Held only by the legitimate game client; presented automatically with every submission from real
  gameplay. Any request lacking it, or presenting an invalid one, is refused.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A player can go from the name prompt appearing to gameplay actually starting in
  under 10 seconds of normal use (time to type a short name and confirm).
- **SC-002**: 100% of completed game attempts (player reaches death) result in the player seeing
  their own score on a Game Over screen, regardless of leaderboard store availability.
- **SC-003**: At least 99% of completed game attempts with the leaderboard store available result
  in a matching leaderboard entry (correct name and score) existing in the store, verifiable
  independently of the game UI.
- **SC-004**: A player can go from the Game Over screen to a new attempt already in progress via
  Replay in under 3 seconds, with zero re-entry of their name required.
- **SC-005**: 100% of score-submission attempts that do not carry a valid credential are rejected
  with zero leaderboard entries created from them.
- **SC-006**: Multiple players (at least 20) completing attempts around the same time each get
  their own correctly attributed leaderboard entry, with zero entries lost or overwritten due to
  name collisions or concurrent submissions.

## Assumptions

- The leaderboard-viewing/display page described elsewhere in the project's broader vision is
  explicitly out of scope for this feature, per the source request ("do not create the leaderboard
  page yet"). This feature only covers capturing name, prompting, scoring display, and durable
  submission.
- Name entry has no uniqueness requirement across players and no profanity/content filtering in
  this feature; any non-empty, length-bounded text is accepted.
- Every completed attempt is recorded as its own leaderboard entry (append-only). Deciding how to
  roll multiple attempts by the same name into a single ranked view (e.g., best score, most recent,
  full history) is left to the future leaderboard-viewing feature, not this one.
- "Its own Go-based API" is read as: a distinct, independently addressable API surface for score
  submission (documented via OpenAPI), rather than folding score-writing into an unrelated existing
  endpoint. This mirrors how the project's broader vision also describes a future leaderboard
  viewing app as its own Go-based app.
- The score-submission credential is a shared-secret style protection against casual/scripted
  abuse of the public write endpoint. As with the project's existing QR-access gating, this is a
  practical deterrent, not a cryptographically unbeatable anti-cheat system — a sufficiently
  determined party who fully reverse-engineers the game client could still forge a credential. That
  tradeoff is accepted, consistent with how the rest of this project's access controls are framed.
- The name entered here is a simple display name for the leaderboard, independent of any
  QR-access-grant identifier from prior access-gating work; no additional login or identity concept
  is introduced by this feature.
- Redis persistence follows the project's existing default (ephemeral is acceptable); no new
  durability guarantee beyond what Redis is already configured for elsewhere in the project is
  required by this feature.
