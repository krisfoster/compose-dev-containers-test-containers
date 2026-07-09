# Research: Secure Microservice Port Exposure

**Feature**: 020-secure-service-port-exposure | **Date**: 2026-07-09

## Scope

This feature has no "NEEDS CLARIFICATION" unknowns — the fix is specified in full detail in the arch-issues document and confirmed by reading `docker-compose.yml`. Research covers confirmation of Docker Compose semantics and validation of the chosen approach.

---

## Decision 1: `expose:` vs `ports:` in Docker Compose

**Decision**: Use `expose:` for `commits-service`, `scores-service`, and `qr-service`.

**Rationale**: Docker Compose's `expose:` directive makes a container port reachable by other services on the same Compose network via service DNS (e.g., `scores-service:8083` from nginx), but does not publish the port to the host. `ports:` publishes the port to the host, enabling any process on the host machine to connect directly. The auth gate (`nginx auth_request → GET /auth/check`) exists only in the nginx layer; bypassing nginx means bypassing auth. Switching to `expose:` closes this bypass while preserving all internal routing.

**Evidence in codebase**: `redis` already uses `expose: ["6379"]` with comments explaining exactly this distinction (lines 30–34 of `docker-compose.yml`). The fix makes three services consistent with the existing pattern.

**Alternatives considered**:
- *No change, rely on firewall*: Rejected — the firewall approach is implicit and fragile, especially during demos on shared networks or when the compose stack is exposed via ngrok.
- *Add auth to each microservice*: Rejected — duplicates auth logic across services, violates the constitution's principle that gate enforcement belongs in the Go app (delegated via nginx auth_request). Adds complexity for a simple compose-config fix.
- *Docker network policy*: Not available in standard Docker Compose; requires Docker Swarm or external tools. Overkill.

---

## Decision 2: Scope — all three services vs scores only

**Decision**: Apply `expose:` to all three microservices (`commits-service`, `scores-service`, `qr-service`).

**Rationale**: While only `scores-service` has a write path (the data integrity risk), `commits-service` and `qr-service` exposing host ports is inconsistent with the nginx-as-single-ingress principle. All three services are internal implementation details; none should be directly reachable from the host in normal operation. The arch-issues document explicitly recommends applying the change to all three. Doing so now prevents future confusion about which services are "safe" to expose.

**Alternatives considered**:
- *Scores only*: Addresses the write-path risk but leaves read-only bypass open and creates inconsistency within the compose file.

---

## Decision 3: Debug access mechanism

**Decision**: Document that a compose override file (e.g., `docker-compose.override.yml`) or environment-specific override can temporarily restore host port access for debugging. Do not add a `--profile` guard in this phase.

**Rationale**: The arch-issues document suggests a `--profile debug` guard as an option. This is more complex than needed for an MVP fix. A compose override file is the standard Docker Compose pattern for environment-specific changes and requires no changes to `docker-compose.yml` itself. A comment in the compose file explaining how to re-enable direct access is sufficient documentation for developers.

**Alternatives considered**:
- *`--profile debug` guard*: Valid approach but adds template complexity (requires profile annotations on the service blocks). A follow-up can implement this if the team regularly needs direct microservice access during development.
- *No documentation*: Rejected — a future developer who needs direct access should not have to read git history to understand why ports aren't exposed.

---

## Summary Table

| Question | Answer |
|----------|--------|
| Which directive to use? | `expose:` (same as redis) |
| Which services? | All three: commits-service, scores-service, qr-service |
| Internal routing affected? | No — Docker Compose DNS still resolves service names |
| nginx routing affected? | No — nginx connects via internal network using service names |
| Debug access? | Document compose override approach in inline comment |
| Go code changes needed? | None |
| Constitution amendment needed? | None |
