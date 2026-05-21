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

Combined with `--keep-cluster`, this makes local iteration fast — the cluster stays up, Orkestra stays installed, and only the CR lifecycle is re-run.

## Using an existing cluster

```bash
ork e2e -f e2e.yaml --cluster my-context
```

The runner switches to `my-context` and skips cluster creation entirely. The cluster is never deleted at the end. The original context is still restored via defer.

## Context restore

The runner captures `kubectl config current-context` before any context switch and restores it at the end via `defer` — so cleanup errors, test failures, and early returns all restore the context correctly.

Before this fix, deleting the kind cluster left `kubectl` pointing at the deleted cluster. Any subsequent `kubectl` command would fail with "the server could not find the requested resource" until the context was switched manually.
