# Constitution Amendment: v1.3.0 → v1.4.0

**Feature**: 014-nginx-front-door
**Amendment type**: MINOR (new carve-out added; no existing principles removed or narrowed)
**Status**: Ready to apply — paste the Sync Impact Report block below into the TOP of
`.specify/memory/constitution.md` before implementation begins.

---

## Sync Impact Report Block (paste at top of constitution.md)

```
<!--
Sync Impact Report
Version change: 1.3.0 -> 1.4.0
Modified principles: (none renamed or removed)
Added sections:
  Technology Stack: new "Permanent Routing Layer carve-out" permitting a dedicated reverse-proxy
    service (nginx:alpine or DHI equivalent) as a permanent single-ingress layer in the compose
    stack, distinct from the Interim Static Hosting carve-out. The routing service's permitted
    roles are static file serving (game assets, leaderboard bundles) and transparent proxying
    to Go services. It MUST contain no business logic. This supersedes the Interim Static Hosting
    carve-out's "bridge, not permanent" restriction for configurations where the routing layer
    is the intentional long-term architecture.
Removed sections: (none)
Rationale: Feature 014-nginx-front-door adds nginx as a permanent front-door routing service
  alongside the existing Go app backend. The Go app continues to own all gate enforcement,
  template rendering, and API logic. nginx routes /play to the Go gated port, proxies APIs and
  dynamic pages to the ungated Go port, and serves game + leaderboard static assets directly.
  The existing "Interim Static Hosting carve-out" was scoped to "while no Go backend exists"
  and explicitly forbids permanent co-residence — this amendment extends the permitted model
  to include permanent routing layers, provided they contain no business logic.
Templates requiring updates:
  OK  .specify/templates/plan-template.md: Constitution Check placeholder is design-compatible;
      no constitution-derived language to update
  OK  .specify/templates/spec-template.md: no constitution reference; no update needed
  OK  .specify/templates/tasks-template.md: no constitution reference; no update needed
Follow-up TODOs:
  TODO(ROUTING-LAYER-SCOPE): If the routing layer grows beyond simple routing (e.g., adds rate
    limiting, auth delegation, or per-route transformations), revisit whether it has become a
    business-logic layer and whether further amendments are required.
  TODO(INTERIM-HOSTING-SUNSET): The original Interim Static Hosting carve-out was added for
    001-host-webapp-ngrok. Once nginx is in place as the permanent routing layer, the interim
    carve-out should be retired (its purpose is superseded by this amendment).
-->
```

---

## New carve-out text (add AFTER the existing "Leaderboard React carve-out" paragraph in the Technology Stack section)

```
**Permanent Routing Layer carve-out**: A dedicated reverse-proxy service (for example,
`nginx:alpine` from DHI — `dhi.io/nginx:1-alpine3.24`) MAY be added as a permanent compose
service acting as the single public ingress for the demo stack. Unlike the Interim Static Hosting
carve-out, this service co-exists permanently with the Go backend rather than bridging a pre-backend
gap. The routing service's permitted scope is strictly: (a) serving game and leaderboard static
assets directly from files copied into its image at build time; (b) reverse-proxying requests
that require business logic to the appropriate Go service or microservice via `proxy_pass`. The
routing service MUST NOT implement any business logic — no gate enforcement, no token generation,
no session management, no request rewriting beyond path-based routing. Gate enforcement MUST
remain in the Go app and is reached by proxying to the Go gated internal port. This carve-out
does not extend to OpenResty, Caddy, Traefik, or other routing platforms unless amended.
```

---

## Updated version metadata (replace the last line of constitution.md)

Old:
```
**Version**: 1.3.0 | **Ratified**: 2026-07-06 | **Last Amended**: 2026-07-08
```

New:
```
**Version**: 1.4.0 | **Ratified**: 2026-07-06 | **Last Amended**: 2026-07-09
```
