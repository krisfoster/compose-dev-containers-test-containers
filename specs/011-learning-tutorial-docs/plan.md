# Implementation Plan: Learning Tutorial and Documentation

**Branch**: `011-learning-tutorial-docs` | **Date**: 2026-07-07 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/011-learning-tutorial-docs/spec.md`

## Summary

Restructure the README as a learning-outcomes tutorial covering five core Docker technologies (Compose, images/Dockerfile, DHI, Testcontainers, Dev Containers), and add beginner-oriented explanatory comments to the key source files and configuration where Docker or idiomatic-Go concepts appear. No runtime code changes; no new dependencies.

## Technical Context

**Language/Version**: Go 1.25 (source), Markdown (documentation)

**Primary Dependencies**: None new — this is a pure documentation feature

**Storage**: N/A

**Testing**: `go test ./...` (no change); README validation by manual walkthrough

**Target Platform**: GitHub Markdown rendering (README) and source code viewers (inline comments)

**Project Type**: Documentation enhancement of an existing web-service + compose demo

**Performance Goals**: N/A

**Constraints**: Comments must add educational value without lengthening files to the point of obscuring the code being explained. README must render cleanly on GitHub. No steps in the tutorial may require the reader to write or edit code.

**Scale/Scope**: The file surface is small and well-defined — one README, one docker-compose.yml, one Dockerfile, and seven Go source files containing Docker-adjacent logic.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
|------|--------|-------|
| **Principle I** (Demo-First) | PASS | Educational framing increases the project's value to Docker booth attendees and workshop participants — directly improves the demo's reach. |
| **Principle II** (Compose-Orchestrated Reproducibility) | PASS | No service definitions change; docker-compose.yml comments are additive only. |
| **Principle III** (Testcontainers) | PASS | No test logic changes; comments explain the pattern but do not alter it. |
| **Principle IV** (Visible-in-browser definition of done) | PASS | Done = README renders correctly on GitHub and the app still runs via `docker compose up`. The "browser verification" for this feature is navigating the rendered README on GitHub. |
| **Principle V** (Vendored-Code Hygiene) | PASS | No new assets vendored. |
| **Stack** | PASS | No new runtime dependencies. |

No violations requiring justification.

## Project Structure

### Documentation (this feature)

```text
specs/011-learning-tutorial-docs/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

(No contracts/ or data-model.md — this feature has no API surface or data entities.)

### Source Code (repository root)

Files touched by this feature — no new files created:

```text
README.md                                            # restructured as tutorial
docker-compose.yml                                   # service-level educational comments
app/Dockerfile                                       # build-stage educational comments
app/main.go                                          # Docker-concept and Go-pattern comments
app/internal/gate/window.go                          # Redis + Testcontainers seam comments
app/internal/gate/window_test.go                     # Testcontainers usage comment block
app/internal/gate/grant.go                           # Go pattern (HMAC signing) comment
app/internal/gate/middleware.go                      # HTTP middleware pattern comment
app/internal/leaderboard/store.go                    # Redis Streams + Testcontainers comments
app/internal/leaderboard/store_test.go               # Testcontainers usage comment block
app/internal/leaderboard/handler.go                  # HTTP handler pattern comment
app/internal/qrcode/qrcode.go                        # dependency role comment
```

**Structure Decision**: Pure documentation overlay — all changes are to Markdown content and Go `//` comments. Directory layout does not change.

## Complexity Tracking

No constitution violations to justify.
