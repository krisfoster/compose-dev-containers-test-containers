# Implementation Plan: Leaderboard Display Page

**Branch**: `004-leaderboard-page` | **Date**: 2026-07-07 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/004-leaderboard-page/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add a `GET /leaderboard` page and a `GET /api/leaderboard/scores` read endpoint to the existing
`app` Go service, so a wall/booth display can show the top-scoring completed attempts and refresh
itself automatically by polling. Per the Clarifications recorded in `spec.md`, this feature is
responsible for building the read side of the leaderboard API — the prior feature
(003-leaderboard-score-submission) only built score submission (`POST`) and explicitly deferred
reading. The read endpoint extends the existing `leaderboard` package's `ScoreStore` with a new
top-N read over the same `leaderboard:scores` Redis Stream, requires no credential (per the
Clarifications), and is registered on both of `app`'s listeners so the page works identically
locally and through the public tunnel. No new compose service, Go module, or storage engine is
introduced.

## Technical Context

**Language/Version**: Go (same toolchain as `app` today — 1.25+); the leaderboard page itself is a
small inline HTML/CSS/JS response (no build step), following the same pattern `handleHost`
already uses for the presenter's host page rather than adding a new static frontend project.

**Primary Dependencies**: Go standard library `net/http`, `encoding/json`, and `sort` (ranking
results in-process); the already-vendored `github.com/redis/go-redis/v9` (`XRange` read) and
`github.com/testcontainers/testcontainers-go/modules/redis` (boundary tests). No new Go module
dependencies.

**Storage**: Redis — reads (does not write) the same `leaderboard:scores` Stream that
003-leaderboard-score-submission's `XADD` writes to. No new persistent storage or Redis structure;
top-N-by-score is computed in-process from a bounded `XRange` read rather than maintained
incrementally (see `research.md` §1 for why).

**Testing**: Go tests for the extended `internal/leaderboard` package: the new read method against
a real Redis via Testcontainers-go (constitution Principle III), and the `GET` handler branch
(empty list, ranking, tie-breaking, limit bound) as unit/handler tests against an in-memory fake
store, extending the existing `leaderboardtest.FakeScoreStore`. The browser-visible auto-refresh
and stale-data resilience behavior is validated manually per `quickstart.md`, per constitution
Principle IV.

**Target Platform**: Presenter's laptop (macOS/Linux/Windows) running Docker Desktop; Linux
containers only — unchanged from prior features.

**Project Type**: Web service — extends the existing single Go backend (`app/`); no new service,
module, or frontend project.

**Performance Goals**: A newly recorded score must appear on an already-open page within 10 seconds
(SC-001) — the page polls the read endpoint every few seconds (well under that bound) to leave
margin for network/render latency.

**Constraints**: No host installs beyond Docker Desktop + git, unchanged from prior features. The
page must keep showing its last successfully retrieved standings across a failed poll (FR-007) and
must run for hours unattended without manual reload (SC-002) — the polling loop must not leak
timers/listeners across repeated cycles.

**Scale/Scope**: Booth-demo scale — the same order of entries as 003 produces (a few dozen players
× a handful of attempts each, at most a few hundred total). The read endpoint bounds its response
to a fixed top-N (default 10, hard-capped) regardless of stream size (FR-004).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Status | Notes |
|-----------|------|--------|-------|
| I. Demo-First Delivery | Change must improve the demo or unblock it | PASS | A live-updating wall leaderboard is one of the core demo beats described in `crossy.md` (steps 1 and 5 of the player flow) — attendees see their score land on the wall display without the presenter doing anything. |
| II. Compose-Orchestrated Reproducibility | Every runtime component is a compose service; `docker compose up` reaches a demoable state with no extra host installs | PASS | No new compose service and no new environment variable — the read endpoint needs no credential (Clarifications) and reuses the existing `app`/`redis` services as-is. |
| III. Testcontainers Over Mocks (NON-NEGOTIABLE) | Go tests crossing a boundary use Testcontainers | PASS (binding on implementation) | The new Redis `XRange`-based read method MUST be tested against a real Redis via Testcontainers-go, extending `store_test.go`'s existing pattern; ranking/limit/tie-break logic is pure and covered by handler tests against the existing in-memory fake. Tracked for `/speckit-tasks`. |
| IV. Visible-in-the-Browser Definition of Done | Done = observed in a browser against the compose stack, with a documented repeatable path | PASS | `quickstart.md` documents opening `/leaderboard`, confirming ranked standings, confirming auto-refresh after a new score is submitted via the game, and confirming resilience when Redis is briefly stopped. |
| V. Vendored-Code Hygiene | Vendored code/assets carry attribution | N/A | No new third-party code or assets. |

No unjustified violations. Complexity Tracking is not needed.

*Re-checked after Phase 1 design (`research.md`, `data-model.md`, `contracts/`, `quickstart.md`): no
additional services, dependencies, or vendored assets beyond those already listed above. The one
notable design call — computing top-N by reading the whole (small, booth-scale) stream and sorting
in-process rather than maintaining a separate ranked structure — is documented and justified in
`research.md` §1 and does not change this table.*

## Project Structure

### Documentation (this feature)

```text
specs/004-leaderboard-page/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/
│   └── leaderboard-openapi.yaml   # Phase 1 output — supersedes
│                                    # specs/003-leaderboard-score-submission/contracts/
│                                    # leaderboard-openapi.yaml as the canonical contract
│                                    # (adds the new GET operation; POST is carried over
│                                    # unchanged)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
app/                            # Existing Go module — extended, not replaced.
├── main.go                     # Updated: registers GET /leaderboard (inline HTML/JS page,
│                                # same pattern as handleHost) on both ungatedMux and
│                                # gatedMux, per FR-011/FR-013 (no gating on viewing/reads).
├── main_test.go                # Updated: handler tests for the new route's wiring.
└── internal/
    ├── gate/                   # Existing, unchanged.
    ├── qrcode/                 # Existing, unchanged.
    └── leaderboard/            # Existing package — extended.
        ├── store.go            # Updated: ScoreStore interface gains a bounded top-N read
        │                       # method, implemented via Redis XRange + in-process sort.
        ├── store_test.go       # Updated: Testcontainers-go Redis tests for the new read
        │                       # method (Principle III).
        ├── handler.go          # Updated: ServeHTTP gains a GET branch (no credential
        │                       # check) returning ranked JSON; POST branch unchanged.
        ├── handler_test.go     # Updated: GET branch tests against the fake store (empty,
        │                       # ranked, tie-break, limit bound).
        └── leaderboardtest/
            └── fake_store.go   # Updated: fake gains the new read method for handler tests.

specs/003-leaderboard-score-submission/
└── contracts/leaderboard-openapi.yaml   # Updated: one-line note pointing to
                                           # specs/004-leaderboard-page/contracts/
                                           # leaderboard-openapi.yaml as the current
                                           # canonical contract (history preserved, not
                                           # deleted — same pattern research.md already
                                           # uses for superseded decisions).
```

**Structure Decision**: Extend the existing single Go module (`app/`) and its existing
`internal/leaderboard` package rather than introducing a second Go service or a separate frontend
project — see `research.md` §2 for why a literal second "standalone app" isn't warranted here. The
leaderboard page is served as an inline HTML/JS response from `main.go`, mirroring how the existing
`/host` page is already served, keeping the project's "no frontend build step" posture intact.

## Complexity Tracking

*No unjustified Constitution Check violations — table intentionally omitted.*
