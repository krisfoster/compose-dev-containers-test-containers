# Research: Learning Tutorial and Documentation

**Feature**: 011-learning-tutorial-docs  
**Date**: 2026-07-07

## 1. Docker Technologies Inventory

The codebase already uses five distinct Docker technologies that form the learning outcomes for the tutorial. Each technology appears in specific, well-bounded files.

| Technology | Where it appears | Beginner concept to introduce |
|---|---|---|
| **Docker Compose** | `docker-compose.yml` | Defining a multi-service app in a single file; one command to start everything |
| **Multi-stage Dockerfile** | `app/Dockerfile` | Two-stage builds: "build" stage compiles the binary, "final" stage holds only the output |
| **Docker Hardened Images (DHI)** | `dhi.io/` image refs in `docker-compose.yml` and `Dockerfile` | Hardened, minimal base images that reduce attack surface; `dhi.io` registry |
| **Testcontainers** | `app/internal/gate/window_test.go`, `app/internal/leaderboard/store_test.go` | Spinning up real containers during tests; why a real Redis beats a mock for boundary tests |
| **Dev Containers** | `.devcontainer/` config | A reproducible development environment defined as code; the same Docker runtime the app uses |

Additionally, two Compose-specific concepts deserve explicit tutorial stops:
- **Inter-service DNS**: how `redis:6379` in `REDIS_ADDR` resolves to the Redis container without a hard-coded IP
- **Compose profiles**: how `ngrok` only starts when `--profile public` is passed

---

## 2. Current State of Inline Documentation

### Assessment: what already exists

The codebase has substantial inline comments, but they are written for readers who already know Docker — they reference spec numbers, explain *why* design decisions were made, and describe cross-cutting concerns. They do not explain Docker concepts to a Docker beginner.

| File | Existing comment quality | Gap |
|---|---|---|
| `docker-compose.yml` | Has comments on `protected-mode` workaround and build context | No explanation of what Compose is, what `expose` vs `ports` means, how services discover each other |
| `app/Dockerfile` | Comments on build context and self-contained image | No explanation of multi-stage builds or what DHI is |
| `app/main.go` | Good package-level comment; function comments reference FR numbers | `REDIS_ADDR=redis:6379` container DNS not explained for a beginner |
| `app/internal/gate/window.go` | Interface comment mentions Testcontainers-go but briefly | Testcontainers pattern not explained; seam concept not introduced |
| `app/internal/gate/window_test.go` | `newTestRedisStore` has a brief Testcontainers comment | No explanation of *why* a real container vs a mock |
| `app/internal/leaderboard/store.go` | ScoreStore interface mentions Testcontainers briefly | Redis Streams not explained for a beginner |
| `app/internal/leaderboard/store_test.go` | Implicit — no Testcontainers explanation | Mirrors the gap in window_test.go |
| `app/internal/gate/grant.go` | HMAC pattern well-commented | No gap for Docker purposes |
| `app/internal/gate/middleware.go` | HTTP middleware well-commented | No gap for Docker purposes |
| `app/internal/leaderboard/handler.go` | Well-commented | No gap for Docker purposes |
| `app/internal/qrcode/qrcode.go` | Simple; package comment is sufficient | No gap |

### Files needing new educational comments

Five files need beginner-oriented Docker concept explanations added:

1. `docker-compose.yml` — explain Compose itself, each service, `expose` vs `ports`, inter-service DNS, DHI, profiles
2. `app/Dockerfile` — explain multi-stage build, DHI base images
3. `app/main.go` — explain `redis:6379` as container DNS (the only Docker-facing gap; other comments are good)
4. `app/internal/gate/window.go` — expand Testcontainers mention in interface doc
5. `app/internal/gate/window_test.go` — add paragraph explaining what Testcontainers-go does and why
6. `app/internal/leaderboard/store.go` — explain Redis Streams and expand Testcontainers mention
7. `app/internal/leaderboard/store_test.go` — add Testcontainers explanation (same pattern as window_test.go)

---

## 3. README Restructuring Plan

### Decision: learning-outcomes-first structure

The README will be restructured as a tutorial. The existing operational content (start commands, public access, dev container setup, troubleshooting) is retained but repositioned within the tutorial flow. The SBX/sandbox development workflow ("Develop with Claude") is moved to the end as an optional advanced section.

**Rationale**: The operational content is both prerequisite (readers need to run the app before understanding how it works) and context (each tutorial section points at a running service or source file). Keeping it integrated is better than separating it into a separate doc.

### Tutorial progression (section order)

1. **Header + one-liner** — what the app is; what the tutorial covers
2. **What you will learn** — explicit learning outcomes, one per Docker technology
3. **Prerequisites** — Docker Desktop, `docker login dhi.io` (moved from "Running the app")
4. **Step 1 — Run the app** — `docker compose up -d`; describe what Compose just started
5. **Step 2 — Understand Docker Compose** — tour `docker-compose.yml`: services, `expose` vs `ports`, inter-service DNS, DHI images, profiles
6. **Step 3 — Understand the Dockerfile** — tour `app/Dockerfile`: multi-stage build, DHI base images, self-contained image
7. **Step 4 — Understand Testcontainers** — tour `app/internal/gate/window_test.go`; explain real containers in tests
8. **Step 5 — Develop inside a Dev Container** — tour `.devcontainer/`; explain what dev containers do
9. **What you learned** — summary, one sentence per Docker technology
10. *(Operational appendix)* **Make it publicly accessible** — ngrok profile; existing content retained
11. *(Operational appendix)* **Leaderboard scores** — existing content retained
12. *(Operational appendix)* **Troubleshooting** — existing content retained
13. *(Optional advanced)* **Develop with Claude Code (SBX)** — clearly marked optional; existing content retained but moved to the end

### Navigation contract

A contributor can find `docker compose up` in Step 1 (under Prerequisites/Run the app) — within the first few sections, well under 60 seconds. Dev container setup is Step 5. No operational content is removed.

---

## 4. Comment Writing Rules

Established from reviewing existing comments and the spec's quality criteria:

- **Explain the why, not the what**: code is readable; comments explain constraints, design choices, and Docker concepts a first-time reader would not already know
- **One comment block per technology introduction**: the first time a concept appears (e.g., Testcontainers in `window_test.go`), include a full beginner explanation; subsequent appearances in other files can be shorter or cross-reference the first
- **No multi-paragraph docstrings**: a block of at most 3-4 lines per concept; if more is needed, link to the relevant spec or the README tutorial section
- **Write in full English sentences**: not bullet fragments

---

## 5. Decisions with Rationale

| Decision | Rationale |
|---|---|
| Combine issues 8 and 9 into one feature | Inline docs and README restructure are complementary: the README points readers at specific source files, which need the inline explanations to be present. Doing them independently would either produce a README tutorial with unexplained code, or inline comments with no tutorial to contextualise them. |
| Keep SBX workflow at the end, not deleted | The spec says exclude it from the tutorial; it doesn't say remove it. Existing users depend on it. |
| No new documentation files | All documentation lives either in the README or as inline comments. Separate doc files (DOCKER_GUIDE.md etc.) create maintenance drift. The README is where readers look first. |
| Tutorial "look at code" steps point at specific files and line ranges | Readers don't have to search; each step is self-contained. |
| No `.devcontainer/` source file comments | Dev container JSON/YAML config is already minimal and well-named; the README tutorial section is sufficient context. Adding comments to JSON would require JSON5-style hacks that could break tooling. |
