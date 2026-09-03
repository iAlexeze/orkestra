# Bootstrap fixture

Manual integration test for `ork clusters bootstrap --config`.

## Setup

```bash
ork create cluster --name ork-bt --count 3
# ork-bt-1 → gateway cluster (current context)
# ork-bt-2 → staging
# ork-bt-3 → prod
```

## Validate the config (no cluster calls)

```bash
ork clusters bootstrap --validate fixture/cluster-config.yaml
```

Expected:
```
✓ bootstrap config valid (2 clusters)
  staging  →  kind-ork-bt-2
  prod     →  kind-ork-bt-3
```

## Dry run

```bash
ork clusters bootstrap --config fixture/cluster-config.yaml --dry-run
```

## Run

```bash
ork clusters bootstrap --config fixture/cluster-config.yaml
```

Verify secrets on the gateway cluster:
```bash
kubectl get secret orkestra-staging orkestra-prod -n default
```

## Cleanup

```bash
kind delete cluster --name ork-bt-1
kind delete cluster --name ork-bt-2
kind delete cluster --name ork-bt-3
```
