# Phase 0 Research: DHI Migration

All facts below were pulled from the live DHI catalog (DHI MCP) and the official DHI docs on
2026-07-07. Vulnerability counts on every chosen tag were **0 critical / 0 high / 0 medium / 0 low**
at that time.

## Decision 1 — DHI registry & access model

- **Decision**: Pull DHI **Community** images directly from `dhi.io` after `docker login dhi.io`,
  using a **free Docker account**. No paid subscription, no org-namespace mirroring, no
  build-from-source.
- **Rationale**: Confirmed by the official docs — Community images are free (Apache 2.0), pulled
  from `dhi.io`; mirrored org-namespace repos are a Select/Enterprise-only concern. Matches the
  clarification in spec.md and keeps onboarding to a single new step.
- **Alternatives considered**: (a) Org-namespace mirror (`<ns>/dhi-<repo>`) — rejected, requires a
  paid tier; (b) build-from-source — rejected, heavy setup for zero demo benefit.
- **Reference format**: `dhi.io/<repo>:<tag>` (e.g. `dhi.io/redis:8-alpine`).

## Decision 2 — App build stage (was `golang:1.25-alpine`)

- **Decision**: `FROM dhi.io/golang:1.25-alpine-dev AS build`.
- **Rationale**: DHI framework images split **dev** (build) from **non-dev** (runtime). The non-dev
  `golang:1.25-alpine` runs as **nonroot** with entrypoint `go help` and **no package manager** —
  unsuitable as a build stage. The **dev** variant (`1.25-alpine-dev`, also `1.25.11-alpine-dev`)
  runs as **root**, includes **busybox shell + `apk`** and the Go toolchain (Go 1.25.11, satisfies
  `go.mod`'s `go 1.25.0`), which is what `go mod download` + `go build` need. DHI's own migration
  guide states build stages should use `dev`-tagged images.
- **Exact tags**: `1.25-alpine-dev`, `1.25-alpine3.24-dev`, `1.25.11-alpine-dev`,
  `1.25.11-alpine3.24-dev` (all `supportsHardened: true`, platforms amd64+arm64).
- **Chosen**: `1.25-alpine-dev` (tracks the 1.25 line, matching the prior `golang:1.25-alpine`
  convention). `GOTOOLCHAIN=local` is set in the image; the build stays offline-of-toolchain.
- **Watch-out**: The DHI golang image bundles Socket Firewall (`sfw`) for package installs.
  `go mod download` still needs network egress to the Go module proxy — an existing requirement, not
  new. If the sandbox firewall blocks it, allow the module proxy host (unchanged from today).

## Decision 3 — App runtime stage (was `alpine:3.20`)

- **Decision**: `FROM dhi.io/static:20260611-alpine3.24 AS final` (alias `dhi.io/static:20260611-alpine`).
- **Rationale**: The app is built `CGO_ENABLED=0` (static binary) and the runtime stage only needs
  to hold and exec `/app` — no shell, package manager, or OS userland. DHI `static` is purpose-built
  "for running statically-linked binary executables, such as those built from Go" (per its docs),
  is ~0.23 MB compressed, ships `ca-certificates` + `tzdata`, and runs as **nonroot (uid 65532)`**.
- **Alternatives considered**: (a) `dhi.io/alpine-base` — rejected, larger and includes an unneeded
  userland; (b) `static` **musl/glibc** variants — rejected, only needed for dynamically-linked
  binaries (ours is static, CGO off).
- **Watch-outs / required checks**:
  - **Non-root perms**: `COPY --from=build /out/app /app` preserves the build's `0755` mode, so uid
    65532 can exec it. The frontend is a **read-only bind mount** (`./frontend/game:/frontend:ro`),
    not baked in, and is world-readable on the host → readable by nonroot. No `chown`/`chmod` needed.
  - **Ports**: DHI guidance recommends ports ≥ 1024 for nonroot; app already uses `:8080`/`:8081`.
  - **Entry point**: `static` has no conflicting default; keep explicit `ENTRYPOINT ["/app"]`.
  - **Tag freshness**: `static` uses date-stamped tags (`20260611-*`), not semver. Pin the current
    date tag; refresh it when rebuilding against a newer hardened `static`.

## Decision 4 — Redis (was `redis:7-alpine`) — documented 7→8 major bump

- **Decision**: `dhi.io/redis:8-alpine` for both the compose `redis` service **and** the
  Testcontainers image in `store_test.go` and `window_test.go`.
- **Rationale**: **There is no hardened Redis 7.x.** In the catalog, Redis 7.x exists only on Debian
  and is `supportsHardened: false`; hardened Redis begins at **8.x on Alpine**. To get a genuinely
  hardened image we must move to the 8.x line. `8-alpine` tracks the hardened 8.x line (currently
  8.8.x), mirroring the prior `redis:7-alpine` "track the major" convention. Redis 8.x is
  backward-compatible for every structure this app uses (sorted sets, hashes+TTL, pub/sub, streams,
  HyperLogLog), and `redis/go-redis/v9` supports Redis 8. The spec's version-parity assumption
  explicitly reserved a documented upgrade decision for planning — this is it.
- **Alternatives considered**: (a) `dhi.io/redis:8.6-alpine` pinned — a valid reproducibility-first
  choice; chosen `8-alpine` for convention-parity, with `8.6-alpine`/`8.8-alpine` noted as the
  pin-a-minor alternative. (b) Debian `redis:7` DHI tag — rejected, `supportsHardened: false` (no
  hardening benefit). (c) Keep public `redis:7-alpine` as a second exemption — rejected, a hardened
  equivalent *does* exist (just on 8.x); exempting would violate FR-001.
- **Exact hardened Alpine tags available**: 8.6 → `8.6-alpine`, `8.6.4-alpine` (+ `-alpine3.24`);
  8.8 → `8-alpine`, `8.8-alpine`, `8.8.0-alpine` (+ `-alpine3.24`). Entrypoint `/usr/bin/tini --`,
  command `redis-server /etc/redis/redis.conf`, user nonroot (65532).
- **Testcontainers compatibility (Principle III risk)**:
  - The `testcontainers-go` redis module's default readiness wait keys on the `Ready to accept
    connections` log line, which DHI redis emits normally. Change is only the image string passed to
    `tcredis.Run(ctx, "dhi.io/redis:8-alpine")`.
  - DHI redis runs via `tini` as nonroot and reads `/etc/redis/redis.conf` — the module doesn't
    require root and connects over the standard `6379`. **Validation**: run `go test ./...` after the
    swap; if the module's wait strategy needs adjustment, add an explicit
    `wait.ForLog("Ready to accept connections")` in the test helpers.
  - The test host must be logged in to `dhi.io` so Testcontainers can pull the image (same
    `docker login dhi.io` prerequisite; there is no CI in this repo today).

### Addendum (found during implementation) — DHI redis protected-mode

- **Symptom**: With `dhi.io/redis:8-alpine` unchanged-otherwise, the app's cross-container calls
  and the Testcontainers mapped-port calls failed (`/host` → 503, standings → 500, `DENIED Redis is
  running in protected mode`). The public `redis:7-alpine` did not do this.
- **Cause**: DHI redis ships a hardened `/etc/redis/redis.conf` with `protected-mode` **on**, which
  refuses all non-loopback connections unless a password is set.
- **Decision**: Disable protected-mode rather than add a password. Redis has **no published port**
  (compose publishes only the app's 8080), so it is reachable only on the internal compose network —
  the exposure that protected-mode guards against does not exist here, and this restores the prior
  behaviour without an app-code change. A `requirepass` password was rejected as disproportionate:
  it would require threading a secret through the go-redis client, `.env`, and the tests, for a
  demo whose redis is already network-isolated.
- **Applied in two places** (satisfies FR-006 — "app runs correctly under hardened-image constraints"):
  - `docker-compose.yml`: `command: ["redis-server", "/etc/redis/redis.conf", "--protected-mode", "no"]`
    (keeps the hardened conf; entrypoint stays `tini --`).
  - `store_test.go` / `window_test.go`: `tcredis.Run(ctx, "dhi.io/redis:8-alpine",
    testcontainers.WithCmd("redis-server", "/etc/redis/redis.conf", "--protected-mode", "no"))`.
    `WithCmd` (full command, not `WithCmdArgs`) is required because DHI's entrypoint is `tini --`,
    not a `docker-entrypoint.sh` that prepends `redis-server` — a bare `--protected-mode no` would
    exec as `tini -- --protected-mode no` and fail.
- **Validated**: `go test ./...` green (boundary tests included); live API leaderboard write/read and
  `/host` QR window round-trip succeed against the hardened redis.

## Decision 5 — ngrok exemption (unchanged)

- **Decision**: `ngrok/ngrok:3` stays as-is; recorded as **exempt** in the image inventory.
- **Rationale**: No DHI equivalent exists for ngrok. It runs only under the optional `public`
  compose profile (off the core local demo path). README already documents obtaining a free ngrok
  account/authtoken; the migration reinforces that ngrok is a demo-only exception.

## Decision 6 — Documentation surface

- **Decision**: (a) New inventory at `contracts/image-inventory.md` (FR-008); (b) update the image
  table in `project.md` (FR-011); (c) README: add `docker login dhi.io` + free-account prerequisite
  to "Running the app", and reinforce the ngrok demo-only exception (FR-012); (d) `.env.example`
  needs no change (no new secret — DHI auth is via `docker login`, not an env var).
- **Rationale**: Directly satisfies FR-008/011/012 and SC-002/SC-007; keeps the "fresh clone →
  browser demo" path (Principle II/IV) honest about its one new step.

## Summary of image mapping

| Role | Before | After (DHI) | Notes |
|------|--------|-------------|-------|
| App build | `golang:1.25-alpine` | `dhi.io/golang:1.25-alpine-dev` | dev variant: root, apk, shell |
| App runtime | `alpine:3.20` | `dhi.io/static:20260611-alpine3.24` | static Go binary, nonroot, ~0.23 MB |
| Redis (compose + tests) | `redis:7-alpine` | `dhi.io/redis:8-alpine` | **7→8 documented bump**; no hardened 7.x |
| Tunnel | `ngrok/ngrok:3` | *(exempt)* | no DHI equivalent; demo-only `public` profile |
