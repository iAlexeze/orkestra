# Suites and imports

A test suite is an E2E file whose only job is to run other E2E files. It has no spec of its own — it composes.

---

## Why suites

Each Katalog in a Komposer deserves its own focused E2E: small, fast, easy to debug when it fails. But a consumer pulling the whole pattern — or a CI job verifying a release — should be able to run one command and get a pass/fail for everything.

The `imports` field bridges the two levels. One suite file at the root imports all the sub-tests. `ork e2e -f e2e.yaml` runs the suite. Sub-tests still run individually with `ork e2e -f sub/e2e.yaml`.

---

## Writing a suite file

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: multi-tenancy-suite
  description: >
    Runs all three multi-tenancy sub-examples in the same cluster.

imports:
  - ./01-basic-namespacing/e2e.yaml
  - ./02-cross-access-control/e2e.yaml
  - ./03-shared-platform/e2e.yaml
```

No `spec:` needed. This is a pure aggregator. `ork validate` reports each import as a ✓ line:

```text
✓ multi-tenancy-suite
    imports : 3 file(s)
      ✓ ./01-basic-namespacing/e2e.yaml
      ✓ ./02-cross-access-control/e2e.yaml
      ✓ ./03-shared-platform/e2e.yaml

3 import(s) valid
```

**Try it:**
```bash
ork init --pack use-cases/multi-tenancy
ork e2e -f e2e.yaml
```

---

## String shorthand

A plain path string is equivalent to the full struct form:

```yaml
# shorthand (equivalent)
imports:
  - ./auth-e2e.yaml

# struct form
imports:
  - path: ./auth-e2e.yaml
    freshCluster: false
```

---

## Cluster strategy

### Shared cluster (default)

All imports without `freshCluster: true` run in the same cluster, one after the other. Each import:

1. Applies its own CRD, bundle, and CR
2. Runs its own expectations
3. Cleans up its own resources
4. The next import starts in the same (now clean) cluster

The cluster is created once for the whole suite. Where it comes from:

| How you invoke `ork e2e` | Cluster for imports |
|--------------------------|---------------------|
| No flags (default) | `runImports` provisions a new kind cluster named `<suite-name>-imports` |
| `--use-current` | Active kubectl context — the same cluster the parent used |
| `--cluster myctx` | `myctx` — the same cluster the parent used |

### Fresh cluster (`freshCluster: true`)

An import with `freshCluster: true` provisions its own independent kind cluster, runs its test, and deletes the cluster when done. Use this when the test installs or deletes cluster-scoped resources that would break other tests running in the same cluster.

```yaml
imports:
  - ./01-basic-namespacing/e2e.yaml        # shared cluster
  - ./02-cross-access-control/e2e.yaml     # shared cluster
  - path: ./03-cluster-scoped-crd/e2e.yaml
    freshCluster: true                      # own cluster
```

---

## Imports in a test that has its own spec

`imports` works on E2E files that also have a `spec:`. The parent test runs first. Imports run after the parent's expectations pass, before the parent's cleanup:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: platform-with-addons

spec:
  katalog: ./platform/katalog.yaml
  crd:     ./platform/crd.yaml
  cr:      ./platform/cr.yaml
  expect:
    - name: Platform ready
      after: cr-applied
      timeout: 90s
      resources:
        - kind: Deployment
          name: platform
          namespace: platform
          ready: true

imports:
  - ./addon-monitoring/e2e.yaml
  - ./addon-logging/e2e.yaml
```

---

## Validation

`ork validate` checks every import file before any cluster work starts:

- The file must exist at the path.
- The file must declare `kind: E2E`.

If an import file is missing or has the wrong kind, `ork validate` reports it with a ✗ and the run does not start:

```text
  ✗ import: ./missing-e2e.yaml: open ./missing-e2e.yaml: no such file or directory
  ✗ import: ./komposer.yaml: expected kind E2E, got "Komposer"
```

---

→ Next: [Best practices](04-best-practices.md)
