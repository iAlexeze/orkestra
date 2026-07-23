# Cluster Lifecycle

## Default behaviour

By default the runner creates a fresh kind cluster named `ork-e2e`, runs the test, then deletes it. The `kubectl` context is restored to whatever it was before the run.

## Reusing a cluster

Set `reuse: true` in the spec to keep the cluster between runs:

```yaml
cluster:
  provider: kind
  name: ork-e2e
  reuse: true
```

Combined with `--keep-cluster`, this makes local iteration fast — the cluster stays up, the bundle is updated in place, and only the CR lifecycle is re-run.

## Using an existing cluster

```bash
ork e2e -f e2e.yaml --cluster my-context
ork e2e -f e2e.yaml --use-current
```

The runner switches to the given context and skips cluster creation entirely. The cluster is never deleted at the end. The original context is still restored via defer.

## Suite mode — shared cluster across imports

When a root e2e file has `imports:`, all non-`freshCluster` imports share the same cluster. Orkestra is installed **once** by the coordinator before the import loop, then each import:

1. Applies its own bundle (updates the ConfigMap in place via `kubectl apply`)
2. Syncs the runtime silently
3. Runs its test
4. Cleans up CRDs and setup — but does **not** delete the bundle or uninstall Orkestra

After all imports complete, the coordinator uninstalls Orkestra once. This means a suite of 5 imports shows exactly one `→ Installing Orkestra...` and one `→ Uninstalling Orkestra...`, regardless of how many imports it contains.

For imports that need full isolation, set `freshCluster: true`:

```yaml
imports:
  - ./01-basic/e2e.yaml
  - path: ./02-needs-isolation/e2e.yaml
    freshCluster: true
```

## Context restore

The runner captures `kubectl config current-context` before any context switch and restores it at the end via `defer` — so cleanup errors, test failures, and early returns all restore the context correctly.
