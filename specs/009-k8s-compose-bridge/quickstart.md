# Quickstart Validation Guide: K8s Manifest Generation via Compose Bridge

**Feature**: 009-k8s-compose-bridge | **Date**: 2026-07-07

End-to-end validation steps that prove the feature works as specified. Run from the repository root.

---

## Prerequisites

Install the following before proceeding:

| Tool | Install |
|------|---------|
| Docker Desktop | Running with at least 4 GB RAM allocated |
| `kind` | `brew install kind` · `go install sigs.k8s.io/kind@latest` · [GitHub releases](https://github.com/kubernetes-sigs/kind/releases) |
| `kubectl` | Included with Docker Desktop, or `brew install kubectl` |
| `compose-bridge` | `docker extension install docker/compose-bridge` (Docker Desktop), or download standalone binary from Docker Hub |

Verify:
```bash
kind version
kubectl version --client
compose-bridge version   # or: docker compose bridge version
```

---

## Step 1 — Generate K8s Manifests

```bash
./k8s/generate.sh
```

**Expected outcome**:
- `k8s/manifests/` is populated with YAML files
- Manifests exist for `redis` (Deployment + Service) and `app` (Deployment + Service)
- No ngrok YAML in the output directory
- Script exits 0

**Verify**:
```bash
ls k8s/manifests/
# Expected: redis-deployment.yaml  redis-service.yaml  app-deployment.yaml  app-service.yaml  app-secrets.yaml.example
```

---

## Step 2 — Deploy to Kind Cluster

Set the required secrets as environment variables (dev-only values are fine for local validation):

```bash
export GRANT_COOKIE_SECRET=dev-only-k8s
export LEADERBOARD_API_SECRET=dev-only-k8s
./k8s/deploy.sh
```

**Expected outcome**:
- Kind cluster `whale-runner` is created (first run) or reused (subsequent runs)
- `whale-runner:k8s-local` image is built from `app/Dockerfile`
- Both images are loaded into Kind
- `whale-runner-secrets` K8s Secret is created
- All manifests applied; both Deployments reach `Available` state
- Script prints the `kubectl port-forward` command and exits 0

**Verify pod status**:
```bash
kubectl get pods
# Expected:
# NAME                     READY   STATUS    RESTARTS   AGE
# app-<hash>               1/1     Running   0          Xs
# redis-<hash>             1/1     Running   0          Xs
```

---

## Step 3 — Access the App in a Browser

In a separate terminal, run:

```bash
kubectl port-forward svc/app 8080:8080
```

Open **http://localhost:8080** in a browser.

**Expected outcome**:
- Game start screen loads (Crossy Whale)
- Player name input is present
- No console errors related to missing assets or failed API calls

Open **http://localhost:8080/leaderboard** in a browser.

**Expected outcome**:
- Leaderboard page loads and shows current scores (may be empty on first run)

---

## Step 4 — Verify Game + Leaderboard Flow

1. Enter a player name on the start screen and start the game.
2. Play until the whale dies (or press a key to lose).
3. Confirm the Game Over screen appears with the player's score.
4. Navigate to http://localhost:8080/leaderboard.
5. Confirm the player name and score appear in the leaderboard.

**Expected outcome**: Score submitted to Redis via the leaderboard API; leaderboard page reflects the entry.

---

## Step 5 — Teardown

Stop the port-forward (Ctrl+C in its terminal), then:

```bash
./k8s/teardown.sh --delete-cluster
```

**Expected outcome**:
- All K8s resources removed from the cluster
- `whale-runner-secrets` Secret deleted
- Kind cluster `whale-runner` deleted
- Script exits 0

**Verify**:
```bash
kind get clusters
# Expected: (no output — whale-runner is gone)
```

---

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `compose-bridge: command not found` | Not installed or not on PATH | Run `docker extension install docker/compose-bridge` or add the binary to PATH |
| `ErrImageNeverPull` for `whale-runner:k8s-local` | Image not loaded into Kind | Re-run `./k8s/deploy.sh`; the kind load step is repeated automatically |
| `ImagePullBackOff` for `dhi.io/redis:8-alpine` | Registry connectivity issue | Pre-pull locally: `docker pull dhi.io/redis:8-alpine && kind load docker-image dhi.io/redis:8-alpine --name whale-runner` |
| Redis pod in `CrashLoopBackOff` | Missing `--protected-mode no` in manifest command | Check `k8s/manifests/redis-deployment.yaml` command args; re-run `./k8s/generate.sh` |
| App pod fails with `FRONTEND_DIR` error | Frontend assets not embedded in image | Re-build: `docker build -f app/Dockerfile -t whale-runner:k8s-local . && kind load docker-image whale-runner:k8s-local --name whale-runner` |
| Leaderboard scores not persisting between deploys | `emptyDir` is ephemeral by design | Expected behaviour in local demo context; scores reset when Redis pod restarts |

---

## References

- [Contracts: Scripts Interface](contracts/scripts-interface.md)
- [Data Model: K8s Manifest Entities](data-model.md)
- [Research Decisions](research.md)
- [Feature Spec](spec.md)
