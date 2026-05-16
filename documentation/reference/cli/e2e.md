# ork e2e

Run declarative end-to-end tests against a real cluster.

```bash
ork e2e -f e2e.yaml
```

`ork e2e` is the complement to `ork simulate` — it runs against a real kind cluster, applying the full operator lifecycle and verifying that resources reach the expected state. The same command runs locally and in CI.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | `e2e.yaml` | Path to the E2E spec file |
| `--keep-cluster` | | `false` | Keep the kind cluster after the test completes |
| `--cluster` | | *(create new)* | Use an existing kubectl context instead of creating a cluster |

## What it does

1. Creates a kind cluster (or reuses `--cluster` context)
2. Installs operator dependencies (CRDs, namespaces, Helm charts)
3. Applies the CRD and operator bundle
4. Starts the Orkestra runtime
5. Applies the CR
6. Polls each `spec.expect` block until all resources reach the expected state or the timeout expires
7. Cleans up the cluster (unless `--keep-cluster`)

## e2e.yaml spec

```yaml
apiVersion: orkestra.io/v1alpha1
kind: E2E
metadata:
  name: hello-website
  description: Verifies the website operator creates a deployment and service
spec:
  katalog: katalog.yaml
  crd: website
  cr: cr.yaml
  expect:
    - name: deployment ready
      after: cr-applied
      timeout: 60s
      resources:
        - kind: Deployment
          name: hello-website-deployment
          namespace: default
          conditions:
            - type: Available
              status: "True"
```

## Examples

```bash
# Run e2e tests defined in e2e.yaml
ork e2e -f e2e.yaml

# Keep the cluster for debugging after the test
ork e2e -f e2e.yaml --keep-cluster

# Run against an existing cluster context
ork e2e -f e2e.yaml --cluster my-dev-context

# Validate the e2e spec before running
ork validate -f e2e.yaml
```

## Related

- [ork simulate](./simulate.md) — in-memory simulation, no cluster required
- [ork validate](./validate.md) — validate the e2e.yaml spec before running
