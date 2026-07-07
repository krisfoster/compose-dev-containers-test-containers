# Quickstart / Validation Guide: Learning Tutorial and Documentation

**Feature**: 011-learning-tutorial-docs  
**Date**: 2026-07-07

This guide describes how to validate that the feature is complete. There is no new runtime behaviour — validation is a review of the README tutorial and the inline comments.

---

## Prerequisites

- Docker Desktop installed and running
- A Docker account (free); logged into the DHI registry: `docker login dhi.io`
- A clone of this repository

---

## Validation 1 — App still starts

Confirm the documentation changes have not touched any runtime file by verifying the app starts and the tests pass.

```bash
docker compose up -d
# navigate to http://localhost:8080 — landing page should load
docker compose down
```

```bash
cd app
go test ./...
# all tests should pass
```

---

## Validation 2 — README renders correctly

Open `README.md` on GitHub (or in a local Markdown renderer). Confirm:

- [ ] The first substantive section is "What you will learn" and lists at least: Docker Compose, Docker images/Dockerfile, Docker Hardened Images, Testcontainers, Dev Containers
- [ ] Headings progress in a logical tutorial order (run → compose → dockerfile → testcontainers → dev containers → summary)
- [ ] No tutorial step asks the reader to write or edit code — each step either runs a command or points at a file to look at
- [ ] The closing "What you learned" section explicitly names and summarises each Docker technology introduced in the body
- [ ] `docker compose up -d` is findable in under 60 seconds by a first-time reader
- [ ] Dev container setup instructions are present and findable
- [ ] Troubleshooting section remains intact
- [ ] "Develop with Claude Code" section is present but clearly marked as optional/advanced

---

## Validation 3 — Inline comments (docker-compose.yml)

Open `docker-compose.yml`. Confirm:

- [ ] A comment block near the top explains what Docker Compose does and how one command starts all services
- [ ] The `redis` service has a comment explaining what Redis is and its role in the app
- [ ] The `redis` service has a comment explaining Docker Hardened Images and the `protected-mode` workaround
- [ ] The `app` service has a comment explaining the build process (context, multi-stage)
- [ ] The `app` service (or its environment section) has a comment explaining `REDIS_ADDR=redis:6379` as container DNS
- [ ] The `ngrok` service and its `profiles` field have a comment explaining Docker Compose profiles

---

## Validation 4 — Inline comments (Dockerfile)

Open `app/Dockerfile`. Confirm:

- [ ] A comment block explains multi-stage builds: what the `build` stage does and why the final image doesn't contain the Go toolchain
- [ ] A comment explains Docker Hardened Images and the `dhi.io` registry

---

## Validation 5 — Inline comments (Go files)

Open each of the following files and confirm the indicated comment is present:

**`app/main.go`**:
- [ ] The `REDIS_ADDR` reference (or `loadConfig`) has a comment explaining that `redis:6379` is a Compose inter-service DNS hostname, not a hard-coded IP

**`app/internal/gate/window.go`**:
- [ ] The `WindowStore` interface comment (or a nearby block) explains the interface-plus-fake pattern and explicitly mentions Testcontainers as the tool used to test the real Redis implementation

**`app/internal/gate/window_test.go`**:
- [ ] `newTestRedisStore` has an expanded comment explaining what Testcontainers-go does (spins up a real container for the test) and why (avoids mock/real divergence at the Redis boundary)

**`app/internal/leaderboard/store.go`**:
- [ ] A comment explains what a Redis Stream is and why it is the right data structure for an append-only score log
- [ ] The `ScoreStore` interface comment mentions Testcontainers in the same way as `WindowStore`

**`app/internal/leaderboard/store_test.go`**:
- [ ] Same Testcontainers explanation as `window_test.go`'s `newTestRedisStore`

---

## Validation 6 — k8s/README.md educational intro

Open `k8s/README.md`. Confirm:

- [ ] A brief intro paragraph (2-4 sentences) appears before the "Prerequisites" section
- [ ] The intro explains what Kubernetes is (container orchestration)
- [ ] The intro explains what Helm does (package manager for Kubernetes; `k8s/` is a Helm chart)
- [ ] The intro mentions Docker Desktop's built-in Kubernetes as the local cluster option
- [ ] Neither `k8s/README.md` nor `README.md` contains the phrase "compose bridge" or mentions Helm chart generation from a compose file

---

## Success criteria mapping

| Spec SC | Validation check |
|---|---|
| SC-001: Reader identifies ≥6 Docker technologies without external docs | Validation 2, "What you will learn" bullet |
| SC-002: Every relevant file has at least one explanatory comment | Validations 3–5 |
| SC-003: `docker compose up` findable in <60 seconds | Validation 2, heading scan |
| SC-004: No tutorial step requires writing code | Validation 2, step-by-step check |
| SC-005: Summary section recaps every technology | Validation 2, "What you learned" bullet |
| SC-006: No Compose Bridge mention in README or k8s/README.md | Validation 6, last bullet |
