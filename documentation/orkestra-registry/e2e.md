# E2E

The E2E framework verifies that a pattern works against a real cluster before it is published. When `e2e.yaml` is present in a pattern directory, `ork registry push` runs it automatically and blocks publication if any expectation fails.

## How it gates publication

```bash
# E2E runs automatically if e2e.yaml is present
ork registry push postgres:v14 ./patterns/postgres/

# Skip the gate — use --force
ork registry push postgres:v14 ./patterns/postgres/ --force
```

Whether the gate passed, was skipped, or was force-overridden, the result is baked into the OCI artifact as annotations:

```
io.orkestra.e2e.status     passed | skipped | forced
io.orkestra.e2e.duration   45s
io.orkestra.e2e.tested_at  2026-05-24T10:00:00Z
io.orkestra.e2e.runner     local | github-actions | gitlab-ci
```

`ork registry info` and `ork registry list` show this status for every pattern — a `✓ passed (45s · tested 2h ago)` gives consumers confidence before they import.

## Running E2E locally

Pull a pattern and run its E2E before importing it:

```bash
ork registry pull postgres:v14
ork e2e --file ~/.orkestra/registry/ghcr.io/orkestra-registry/postgres/v14/e2e.yaml
```

Or run against a pattern you are developing locally:

```bash
ork e2e --file ./patterns/postgres/e2e.yaml
```

By default, `ork e2e` provisions a temporary kind cluster, runs all expectations, then tears it down. Use `--use-current` to run against your current kubeconfig context:

```bash
ork e2e --file ./e2e.yaml --use-current
```

## e2e.yaml structure

```yaml
# Provision (optional — omit to use current cluster)
cluster:
  name: e2e-postgres

# Setup applied before the operator starts
setup:
  - rbac/namespace.yaml

# The CR to apply after the operator is running
cr: examples/postgres-sample.yaml

# Expectations — polled until timeout or failure
expectations:
  - name: StatefulSet is ready
    resource:
      kind: StatefulSet
      name: my-postgres
      namespace: "default"
      ready: true

  - name: Service exists
    resource:
      kind: Service
      name: my-postgres
      namespace: default

  - name: Custom health check
    command:
      run: "kubectl exec -n default postgres-0 -- pg_isready"
      exitCode: 0
      outputContains: "healthy"
```

The runner applies the setup manifests, installs Orkestra, applies the CR, then polls each expectation until it passes or the timeout expires. On teardown it deletes the CR, uninstalls Orkestra, removes setup manifests, and deletes the cluster (unless `--keep-cluster`).

## E2E in CI

The `io.orkestra.e2e.runner` annotation records the environment. When `ork registry push` runs inside GitHub Actions, it records `github-actions`. Consumers can see that the pattern was verified in CI, not just locally.

Use `--no-e2e` to skip the gate in CI steps that only build artifacts without running tests:

```bash
ork registry push postgres:v14 ./patterns/postgres/ --no-e2e
```
