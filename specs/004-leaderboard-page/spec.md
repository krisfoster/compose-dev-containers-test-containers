# Feature Specification: Leaderboard Display Page

**Feature Branch**: `004-leaderboard-page`

**Created**: 2026-07-07

**Status**: Draft

**Input**: User description: "From issues.md, open issue 5: A standalone Go-based app/page that
dynamically refreshes by polling/calling the leaderboard API, so current leaderboard standings can
be shown (e.g., on a wall display) without anyone manually reloading the page."

## Clarifications

### Session 2026-07-07

- Q: Is there a read API defined yet for the leaderboard API — was it built previously, is it
  included in this spec, or has it not been defined yet? → A: It has not been built yet (the prior
  score-submission feature explicitly deferred it); this feature is responsible for defining and
  building the leaderboard read/list API endpoint itself, in addition to the display page.
- Q: Should the new leaderboard read/list endpoint require the same credential as the write
  endpoint, or be open? → A: Open/unauthenticated — reads require no credential, matching the
  ungated leaderboard page.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Viewer sees current leaderboard standings (Priority: P1)

Someone looking at the leaderboard page sees a ranked list of player names and their scores,
highest score first.

**Why this priority**: This is the entire purpose of the feature. Without a readable, ranked
standings list, there is nothing else to build on.

**Independent Test**: With leaderboard entries already recorded, open the leaderboard page and
confirm it shows player names and scores in descending score order.

**Acceptance Scenarios**:

1. **Given** the leaderboard store already contains recorded entries, **When** the leaderboard page
   is opened, **Then** it displays those entries ranked from highest score to lowest.
2. **Given** two entries have different scores, **When** they both appear on the page, **Then** the
   higher-scoring entry is shown above the lower-scoring one.
3. **Given** no leaderboard entries exist yet, **When** the leaderboard page is opened, **Then** it
   shows a clear, friendly empty state instead of an error or a blank screen.

---

### User Story 2 - Standings update automatically without a manual reload (Priority: P1)

While the leaderboard page is left open (e.g., on a wall display at a booth), newly completed game
attempts appear in the standings on their own, without anyone reloading the page.

**Why this priority**: The feature is explicitly meant to be a "dynamically refreshing" display —
this is what makes it usable as an unattended, continuously-running booth display rather than a
page someone has to babysit and refresh by hand.

**Independent Test**: Leave the leaderboard page open, record a new leaderboard entry through the
game, and confirm the new entry appears on the already-open page without reloading it.

**Acceptance Scenarios**:

1. **Given** the leaderboard page is already open and showing standings, **When** a new score is
   recorded, **Then** the page reflects that new entry within a short, bounded amount of time with
   no manual reload.
2. **Given** the leaderboard page has been open and unattended for an extended period, **When** it
   continues running, **Then** it keeps refreshing standings on its own indefinitely, without
   needing to be restarted or reloaded.

---

### User Story 3 - Display stays usable when the leaderboard data is briefly unavailable (Priority: P2)

If the leaderboard page can't retrieve current standings for a moment, it keeps showing the last
standings it successfully loaded rather than going blank or showing an error to whoever is looking
at it.

**Why this priority**: This is a booth/wall display meant to run unattended for hours. A single
missed refresh shouldn't turn it into a broken-looking error screen in front of attendees. It's P2
because the core viewing and auto-refresh behavior (User Stories 1–2) must exist first before this
resilience behavior matters.

**Independent Test**: While the leaderboard page is open and showing standings, make the
leaderboard data temporarily unreachable, confirm the page keeps showing the last known standings
through that period, then restore access and confirm it resumes updating normally.

**Acceptance Scenarios**:

1. **Given** the leaderboard page is showing standings, **When** a refresh attempt fails because the
   leaderboard data is temporarily unreachable, **Then** the page continues showing the last
   successfully retrieved standings instead of clearing or erroring.
2. **Given** the leaderboard data becomes reachable again after a failure, **When** the next refresh
   happens, **Then** the page resumes showing up-to-date standings automatically.

---

### Edge Cases

- What happens when the same player name appears multiple times (e.g., from repeated Replay
  attempts)? Each completed attempt is its own entry, so the same name may appear more than once at
  different ranks with different scores — this mirrors classic arcade high-score tables and is
  expected, not a bug.
- What happens when there are far more recorded entries than can be reasonably displayed at once?
  The page shows a bounded top slice of the standings rather than every entry ever recorded.
- What happens when a player's display name is unusually long? The page renders it without breaking
  the layout, truncating if necessary.
- What happens when two entries have the exact same score? Both are shown; a consistent tiebreaker
  (e.g., most recent first) is used so the ordering doesn't visibly jump between refreshes.
- What happens the very first time the page is opened, before any refresh has completed? The page
  shows a brief loading state rather than an empty or broken one.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a leaderboard page that is separate and standalone from the game
  itself.
- **FR-002**: System MUST display, for each shown leaderboard entry, the player's name and the
  score they achieved.
- **FR-003**: System MUST order displayed entries by score, from highest to lowest.
- **FR-004**: System MUST limit the displayed standings to a bounded number of top entries rather
  than showing an unbounded, ever-growing list.
- **FR-005**: System MUST retrieve leaderboard standings by calling a leaderboard API read/list
  endpoint, not by reading the underlying data store directly.
- **FR-006**: System MUST automatically refresh the displayed standings on a recurring basis for as
  long as the page remains open, with no manual reload required.
- **FR-007**: System MUST continue displaying the most recently successfully retrieved standings
  when a refresh attempt fails, rather than clearing the display or showing an error in its place.
- **FR-008**: System MUST show a clear empty state when there are no recorded leaderboard entries.
- **FR-009**: System MUST show a clear loading state before the first successful retrieval of
  standings completes.
- **FR-010**: System MUST render unusually long player names without breaking the page's layout.
- **FR-011**: System MUST NOT require any player action, credential, or gating step to simply view
  the leaderboard page.
- **FR-012**: System MUST define and expose a leaderboard read/list API endpoint — extending the
  existing leaderboard API's contract (documented via OpenAPI, consistent with the existing
  score-submission endpoint) — since no such read capability exists yet; this feature is
  responsible for delivering it, not merely consuming a pre-existing one.
- **FR-013**: System MUST NOT require a credential on the leaderboard read/list endpoint — unlike
  the score-submission (write) endpoint, reads are open to any caller that can reach the API.

### Key Entities

- **Leaderboard Standing**: A single ranked row shown on the page — a player display name paired
  with the score from one completed attempt, plus its rank relative to the other displayed entries.
  Sourced from the existing Leaderboard Entry data recorded by the leaderboard API; this feature
  adds the read/list capability needed to serve that data and displays it, but does not change how
  entries are written.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A newly recorded score appears on an already-open leaderboard page within 10 seconds,
  with zero manual action taken on the page.
- **SC-002**: The leaderboard page can run continuously for a multi-hour event without requiring any
  manual restart or reload to keep showing current standings.
- **SC-003**: 100% of temporary leaderboard-data retrieval failures result in the last known
  standings remaining visible, rather than an error or blank screen being shown.
- **SC-004**: A first-time visitor to the leaderboard page sees either populated standings or a
  clear empty state within 3 seconds of the page loading.

## Assumptions

- This feature builds on the leaderboard storage and score-recording capability delivered by a
  prior feature (name + score recorded to a Redis-backed store via a Go-based API); this feature
  adds only the read/viewing side and does not change how entries are written.
- Per Clarifications above, the leaderboard read/list capability does not exist yet — the prior
  score-submission feature explicitly deferred it — so this feature's scope includes defining and
  building that read endpoint, not just consuming a pre-existing one.
- Consistent with the append-only recording model from the prior leaderboard feature, this page
  displays individual completed attempts (not one best-score-per-player rollup) — the same player
  name can legitimately appear more than once in the standings, similar to a classic arcade
  high-score table.
- Viewing the leaderboard page has no access gate of its own (unlike the playable game, which is
  QR-gated on the public endpoint) — it's treated as a public/wall-display view, not gameplay.
- A reasonable default of showing roughly the top 10 entries and refreshing on the order of every
  few seconds is assumed; exact numbers are tuning details left to implementation, not fixed
  requirements of this spec.
- No player identity, authentication, or personalization is introduced by this page — it shows the
  same standings to every viewer.
