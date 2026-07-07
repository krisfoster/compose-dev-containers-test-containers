# Kubernetes Deployment

Whale Runner is deployed to Kubernetes via a Helm chart in this directory.

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) with Kubernetes enabled
  (Settings → Kubernetes → Enable Kubernetes)
- [Helm](https://helm.sh/docs/intro/install/)

## Install

```bash
helm install whale-runner ./k8s/
```

## Uninstall

```bash
helm uninstall whale-runner --namespace default
kubectl patch service app-published -n whale-runner \
  -p '{"metadata":{"finalizers":[]}}' --type=merge 2>/dev/null
kubectl wait --for=delete namespace/whale-runner --timeout=60s
```

## Access

```bash
kubectl port-forward -n whale-runner service/app-published 8080:8080
```

Then open http://localhost:8080.
