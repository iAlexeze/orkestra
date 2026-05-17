# E2E Spec Format

The spec file is a YAML document with `kind: E2E`. All paths are relative to the spec file's directory.

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: my-operator-e2e
  description: What this test verifies

spec:
  katalog: ./katalog.yaml   # required
  crd: ./crd.yaml           # required — the operator's CRD
  cr: ./cr.yaml             # required — the CR to apply

  cluster:
    provider: kind           # only "kind" is supported
    name: ork-e2e            # kind cluster name (default: ork-e2e)
    reuse: false             # false = delete and recreate if exists

  setup:                     # optional — applied before CR, in order
    - ./setup.yaml

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

### `spec.crd`
Path to the CRD YAML to apply before the operator starts. If omitted, the runner falls back to `crdFile` entries in the Katalog.

### `spec.cr`
Path to the CR to apply. Applied when the first `after: cr-applied` expectation is reached, deleted when the first `after: cr-deleted` expectation is reached.

### `spec.setup`
List of YAML files applied after the cluster is ready but before the CR. Use for namespaces, Secrets, or fixture CRDs the test depends on.

### `spec.cluster.reuse`
When `true`, the runner reuses an existing kind cluster with the same name instead of deleting and recreating it. Useful for iterating locally without waiting for cluster creation each time.

## Using an existing cluster

Pass `--cluster` to skip cluster creation entirely:

```bash
ork e2e -f e2e.yaml --cluster my-existing-context
```

## Keeping the cluster after the test

```bash
ork e2e -f e2e.yaml --keep-cluster
```
