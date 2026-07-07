# Testcontainers inside Dev Containers

Notes on the accepted approaches for running Testcontainers-based tests inside a Dev Container, gathered while looking at whether `awesome-compose` had a sample that combines Go, Redis, and Testcontainers (it does not; nothing in `awesome-compose` uses Testcontainers at all, and no Go sample uses Redis).

## Context

Testcontainers spawns real Docker containers on demand from your test process. That requires the test process to reach *some* Docker daemon. Inside a Dev Container, the test process is itself running in a container, so you have to decide which daemon it talks to and how.

## The three accepted approaches

### 1. Docker-in-Docker (DinD)

Add the `docker-in-docker` devcontainer feature. The dev container gets its own Docker daemon.

```json
"features": {
  "ghcr.io/devcontainers/features/docker-in-docker:2": { "moby": true }
}
```

The `"moby": true` matters. Without it some Testcontainers clients (notably .NET's `DockerApiClient`) throw a `NullReferenceException` at startup.

Tradeoffs:
- Simplest to add; fully isolated from the host daemon.
- Runs the container privileged.
- Heavier resource footprint; no sharing of the host image layer cache.
- Testcontainers' own docs describe DinD as "an instrument of last resort."

### 2. Docker-outside-of-Docker (DooD / socket mount)

Mount the host's `/var/run/docker.sock` into the dev container so tests spawn sibling containers on the host daemon. Testcontainers calls this the "Docker wormhole" and treats it as the preferred pattern.

There is a `docker-outside-of-docker` devcontainer feature that wires this up.

Requirements from the Testcontainers docs:
- Mount the socket: `-v /var/run/docker.sock:/var/run/docker.sock`
- Mount the workspace at the same absolute path used on the host: `-v $PWD:$PWD -w $PWD`. If the paths differ, Testcontainers cannot forward bind-mounts into the containers it spawns.
- On Docker Desktop (macOS, Windows, WSL2) set `TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal` so the test process can reach the spawned services. Without this you get "connection refused" against `localhost`.

Tradeoffs:
- Lightest option: reuses the host daemon and its image cache.
- Test-spawned containers are siblings, not children, so they show up in `docker ps` on the host.
- Path-mapping is the sharp edge; the mount-path-equals-host-path rule catches people out.
- Officially preferred by Testcontainers.

### 3. Testcontainers Cloud (TCC)

The dev container talks to a remote Docker environment provided by Testcontainers Cloud. This is what Docker's own blog recommends for the devcontainer use case.

```json
"containerEnv": {
  "TC_CLOUD_TOKEN": "${localEnv:TC_CLOUD_TOKEN}",
  "TCC_PROJECT_KEY": "your-project-name"
},
"features": {
  "ghcr.io/eddumelendez/test-devcontainer/tcc:0.0.2": {}
}
```

Tradeoffs:
- No local Docker load, no privileged container, no socket exposure.
- Requires a TCC account + service-account token; paid SaaS.
- Dashboard visibility and per-session tagging via `TCC_PROJECT_KEY`.
- Overkill for a public sample, useful for teams already on TCC.

## Common failure modes

- **`NullReferenceException` in `DockerApiClient` (DinD)**: add `"moby": true` to the DinD feature.
- **`connection refused` from tests to spawned services (DooD on Docker Desktop)**: set `TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal`.
- **Testcontainers can't mount a fixture directory (DooD)**: workspace is not bind-mounted at the same absolute path as on the host. Fix the mount, don't try to work around it in the test.
- **Ryuk cleanup errors**: usually a symptom of the daemon connection being wrong, not a Ryuk bug. Fix the connectivity first.

## Community state

The `devcontainers/discussions#198` thread shows this is still under-documented at the community level. People do get it working, but the "which approach and why" question rarely gets a clear answer in one place. The Testcontainers docs are opinionated (DooD > DinD), Docker's blog is opinionated (TCC > DinD), and the two don't fully overlap.

## Recommendation for a Go + Redis + Testcontainers sample in `awesome-compose`

DooD (socket mount) is the natural fit:

- The dev container is already running on a host with Docker, so pointing at the host daemon costs nothing extra.
- It matches Testcontainers' preferred pattern.
- It keeps the sample cheap and reproducible; no paid SaaS, no privileged DinD.
- Bake in `TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal` and workspace-at-host-path from day one so contributors don't rediscover the gotchas.

DinD is the fallback if the surrounding sandbox policy forbids exposing the host socket. TCC only makes sense if the team is already using it.

## Sources

- [Streamlining Local Development with Dev Containers and Testcontainers Cloud — Docker blog](https://www.docker.com/blog/streamlining-local-development-with-dev-containers-and-testcontainers-cloud/)
- [Using TestContainers inside a Dev Container — Jason Penniman](https://www.jasonpenniman.com/testcontainers-devcontainers)
- [Patterns for running tests inside a Docker container — Testcontainers docs](https://java.testcontainers.org/supported_docker_environment/continuous_integration/dind_patterns/)
- [Running testcontainer in devcontainer — devcontainers GitHub Discussion #198](https://github.com/orgs/devcontainers/discussions/198)
- [Testcontainers — Docker Docs](https://docs.docker.com/testcontainers/)
