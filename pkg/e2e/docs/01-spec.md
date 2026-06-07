# E2E Spec Format

The spec file is a YAML document with `kind: E2E`. All paths are relative to the spec file's directory.

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: my-operator-e2e
  description: What this test verifies

spec:
  katalog: ./katalog.yaml   # required (optional when customOperator: true)
  crd: ./crd.yaml           # required — the operator's CRD
  cr: ./cr.yaml             # required — the CR to apply

  customOperator: false     # true = skip bundle + Orkestra install (see below)

  cluster:
    provider: kind           # only "kind" is supported
    name: ork-e2e            # kind cluster name (default: ork-e2e)
    reuse: false             # false = delete and recreate if exists

  setup:                     # optional — applied before CR, in order
    apply:
      - ./setup.yaml
    helm:
      - repo: https://charts.example.com
        chart: my-prereq
        namespace: my-prereq-system
        createNamespace: true
    wait:
      - kind: Deployment
        name: my-prereq
        namespace: my-prereq-system
        ready: true
        timeout: 120s

  expect:
    - name: CR created
      after: cr-applied      # or: cr-deleted
      timeout: 60s
      resources:
        - kind: MyApp
          name: my-cr
          namespace: default
    - name: Cleanup verified
      after: cr-deleted
      timeout: 30s
      resources:
        - kind: MyApp
          name: my-cr
          namespace: default
          count: 0
```

## Fields

### `spec.katalog`
Path to the Katalog (or Komposer) file. The runner resolves `crdFile` entries in it and embeds them in the bundle so the in-cluster runtime doesn't need local file access.

Optional when `spec.customOperator: true`.

### `spec.crd`
Path to the CRD YAML to apply before the operator starts. If omitted, the runner falls back to `crdFile` entries in the Katalog.

### `spec.cr`
Path to the CR to apply. Applied when the first `after: cr-applied` expectation is reached, deleted when the first `after: cr-deleted` expectation is reached.

### `spec.customOperator`
When `true`, skips bundle generation and Orkestra helm install/uninstall. Use when your operator is installed via `setup.helm` or is already present in the cluster. `spec.katalog` is optional. Everything else (CRD apply, setup, CR apply, assertions, cleanup) runs unchanged.

See [05-custom-operator.md](05-custom-operator.md).

### `spec.setup`
Prerequisites applied after the cluster is ready but before the CR. Shorthand (plain list of strings) applies each file:

```yaml
setup:
  - ./namespaces.yaml
  - ./secret.yaml
```

Struct form adds Helm chart installation and resource waiting:

```yaml
setup:
  apply:
    - ./namespaces.yaml
  helm:
    - repo: https://charts.jetstack.io
      chart: cert-manager
      namespace: cert-manager
      createNamespace: true
      version: v1.14.5
      values:
        installCRDs: true
  wait:
    - kind: Deployment
      name: cert-manager-webhook
      namespace: cert-manager
      ready: true
      timeout: 120s
```

`setup.wait` uses `kubectl rollout status` for Deployments and `kubectl wait --for=condition=Ready` for other kinds.

### `spec.cluster.reuse`
When `true`, the runner reuses an existing kind cluster with the same name instead of deleting and recreating it. Useful for iterating locally without waiting for cluster creation each time.

## Imports (suite composition)

`imports` is a top-level field (alongside `spec:`) that lists other E2E files to run after the current one:

```yaml
imports:
  - ./01-basic/e2e.yaml
  - path: ./02-advanced/e2e.yaml
    wait: 10s          # sleep before this import starts
  - path: ./03-infra/e2e.yaml
    freshCluster: true # provision a new cluster for this import
```

See [05-imports.md](05-imports.md).

## Using an existing cluster

```bash
ork e2e -f e2e.yaml --cluster my-existing-context
ork e2e -f e2e.yaml --use-current
```

## Discovery mode

```bash
ork e2e ./...                          # all *e2e.yaml files recursively
ork e2e ./examples/beginner/...        # scoped to a subtree
ork e2e ./... --wait 2s --skip vendor,cr-e2e.yaml
```

See [06-discovery.md](06-discovery.md).

## Dev server (external gate examples)

```bash
ork e2e -f e2e.yaml --dev-server
```

Deploys `ghcr.io/orkspace/orkestra-dev-server:latest` as a Deployment+Service into `orkestra-system` before Orkestra installs. CRs in `use-cases/external` use `orkestra-dev-server.orkestra-system.svc:9999` as the gate endpoint instead of `localhost:9999`, so they work from inside the cluster without host networking.

## Keeping the cluster after the test

```bash
ork e2e -f e2e.yaml --keep-cluster
```

## Dry run

```bash
ork e2e -f e2e.yaml --dry-run          # runs ork validate on the file
ork e2e ./... --dry-run                # lists files that would be discovered
```
