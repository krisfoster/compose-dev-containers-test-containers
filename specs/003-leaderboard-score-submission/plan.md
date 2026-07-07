# Implementation Plan: Player Name Entry, Game Over Score Display, and Leaderboard Score Submission

**Branch**: `003-leaderboard-score-submission` | **Date**: 2026-07-07 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/003-leaderboard-score-submission/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add a name prompt and a Game Over screen (with a Replay control) to `frontend/game/`, and add a
new `POST /api/leaderboard/scores` route to the existing `app` Go service that appends each
completed attempt's name and score to a Redis Stream (`leaderboard:scores`). The route lives on
both of `app`'s existing listeners, is documented as its own OpenAPI contract, and is protected by
a shared credential (a new env var) that `app` injects into the served game page server-side so the
legitimate client can present it automatically with no visible extra step for the player. No new
compose service, Go module, or storage engine is introduced — this reuses the Redis instance and
Go service `002-qr-gated-access` already established. Leaderboard viewing/browsing is explicitly
out of scope (spec FR-015); this phase only covers capture and durable recording.

## Technical Context

**Language/Version**: Go (same toolchain as `app` today — 1.25+); frontend is plain JS/HTML/CSS,
no build step (unchanged from existing `frontend/game/`).

**Primary Dependencies**: Go standard library `net/http` (new routes) and `html/template` (index
page credential injection, replacing the current raw `http.FileServer` pass-through for that one
file); the already-vendored `github.com/redis/go-redis/v9` (Stream `XADD`) and
`github.com/testcontainers/testcontainers-go/modules/redis` (boundary tests). No new Go module
dependencies.

**Storage**: Redis — one new Stream key, `leaderboard:scores`, appended to via `XADD` (see
`data-model.md`). No other new persistent storage; reuses the same Redis instance `app` already
depends on for QR window state.

**Testing**: Go tests for the new `internal/leaderboard` package: request validation and credential
checking as unit/handler tests against an in-memory fake store; the `XADD` write itself tested
against a real Redis via Testcontainers-go, per constitution Principle III (same pattern
`internal/gate`'s `WindowStore` already establishes). The full browser flow (name prompt → play →
death → Game Over → submission → Replay) is validated manually per `quickstart.md`, per
constitution Principle IV.

**Target Platform**: Presenter's laptop (macOS/Linux/Windows) running Docker Desktop; Linux
containers only — unchanged from `001-host-webapp-ngrok` and `002-qr-gated-access`.

**Project Type**: Web service — extends the existing single Go backend (`app/`) and existing static
frontend (`frontend/game/`); no new service or project.

**Performance Goals**: Name-entry-to-gameplay under 10 seconds (SC-001); score write is a single
Redis `XADD`, effectively instant relative to the game's own death-to-Game-Over transition.

**Constraints**: No host installs beyond Docker Desktop + git, unchanged from prior features. The
credential check (FR-012) must not add a visible step for the player (FR-014) and must work
identically on the ungated (local/presenter) listener, which has no QR-grant concept at all.

**Scale/Scope**: Booth-demo scale — the same order of concurrent players as `002-qr-gated-access`'s
grants (a few dozen), each producing at most a handful of leaderboard entries per visit via Replay.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Status | Notes |
|-----------|------|--------|-------|
| I. Demo-First Delivery | Change must improve the demo or unblock it | PASS | Capturing a player's name and score, and showing it back to them at Game Over with an instant Replay, is a direct payoff moment for attendees and the prerequisite data for the future leaderboard wall display (`crossy.md` player flow steps 3, 5, 6). |
| II. Compose-Orchestrated Reproducibility | Every runtime component is a compose service; `docker compose up` reaches a demoable state with no extra host installs | PASS | No new compose service — reuses the existing `app` and `redis` services. Adds one new environment variable (`LEADERBOARD_API_SECRET`) to `app`'s existing environment block and `.env.example`, same pattern as `GRANT_COOKIE_SECRET`. |
| III. Testcontainers Over Mocks (NON-NEGOTIABLE) | Go tests crossing a boundary use Testcontainers | PASS (binding on implementation) | The new `leaderboard` package's Redis `XADD` write MUST be tested against a real Redis via Testcontainers-go; validation/credential logic is pure and may use standard unit/handler tests against an in-memory fake, mirroring `internal/gate`. Tracked for `/speckit-tasks`. |
| IV. Visible-in-the-Browser Definition of Done | Done = observed in a browser against the compose stack, with a documented repeatable path | PASS | `quickstart.md` documents five scenarios (name prompt, Game Over display, store verification, Replay, credential rejection) run against the live compose stack. |
| V. Vendored-Code Hygiene | Vendored code/assets carry attribution | N/A | No new third-party code or assets — reuses `go-redis` and `testcontainers-go`, both already present and already accounted for by `002-qr-gated-access`. |

No unjustified violations. Complexity Tracking is not needed.

*Re-checked after Phase 1 design (`research.md`, `data-model.md`, `contracts/`, `quickstart.md`):
no additional services, dependencies, or vendored assets beyond those already listed above. The one
notable design call — keeping the leaderboard API as a package inside `app` rather than a separate
service — is documented and justified in `research.md` §1 and does not introduce a constitution
violation (Principle II is satisfied by *not* adding a service, not by adding one). Table holds
unchanged.*

## Project Structure

### Documentation (this feature)

```text
specs/003-leaderboard-score-submission/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/
│   └── leaderboard-openapi.yaml   # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
docker-compose.yml            # Updated. `app` service gains LEADERBOARD_API_SECRET env var.
.env.example                   # Updated. Adds LEADERBOARD_API_SECRET, documented like
                                # GRANT_COOKIE_SECRET.

app/                            # Existing Go module — extended, not replaced.
├── main.go                     # Updated: registers /api/leaderboard/scores on both muxes
│                                # (research.md §2); switches index.html serving from raw
│                                # http.FileServer to an html/template render that injects
│                                # the credential (research.md §4).
├── main_test.go                # Updated: handler tests for the new route's wiring.
└── internal/
    ├── gate/                   # Existing, unchanged.
    ├── qrcode/                 # Existing, unchanged.
    └── leaderboard/            # New package.
        ├── store.go            # ScoreStore interface + Redis Stream (XADD)-backed implementation.
        ├── store_test.go       # Testcontainers-go Redis tests (Principle III).
        ├── store_fake_test.go  # In-memory fake ScoreStore, for handler tests.
        ├── handler.go          # POST /api/leaderboard/scores: validation, credential check,
        │                       # calls ScoreStore.
        └── handler_test.go     # Unit tests against the fake store: validation, credential
                                 # accept/reject, success path.

frontend/game/                  # Existing, updated.
├── index.html                  # Updated: adds name-prompt and Game-Over/Replay markup.
├── script.js                   # Updated: name-prompt flow before game start; on death, shows
│                                # Game Over with score, submits to the new API using
│                                # window.__LEADERBOARD_TOKEN__, wires the Replay control.
└── style.css                   # Updated: styling for the new overlays.
```

**Structure Decision**: Extend the existing single Go module (`app/`) with one new internal
package (`internal/leaderboard`) rather than introducing a second Go service — see `research.md`
§1 for why a separate service isn't warranted at this scale. `frontend/game/` gains new UI states
(name prompt, Game Over/Replay) within its existing files rather than new pages, consistent with
the game already being a single-page app.

## Complexity Tracking

*No unjustified Constitution Check violations — table intentionally omitted. The one structural
decision worth flagging (leaderboard writes as a package inside `app` rather than a new service) is
a simplification relative to a literal "separate service" reading of the source issue, not an added
violation — see `research.md` §1.*
