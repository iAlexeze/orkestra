# Imports

`imports` is a top-level field (same level as `spec:`) that lists other E2E files to run after the current one completes. It is the building block for test suites.

---

## Wire format

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E

metadata:
  name: full-stack-app-suite

imports:
  - ./01-multi-region/e2e.yaml
  - ./03-cross-crd/e2e.yaml
  - path: ./infra-e2e.yaml
    freshCluster: true
```

---

## Shorthand

A plain string is equivalent to `{path: <string>, freshCluster: false}`:

```yaml
# shorthand
imports:
  - ./auth-e2e.yaml

# equivalent struct form
imports:
  - path: ./auth-e2e.yaml
    freshCluster: false
```

---

## Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `path` | yes | — | Path to another E2E spec file. Relative to this file's directory. |
| `freshCluster` | no | `false` | When `true`, provisions a new kind cluster for this import instead of sharing the suite cluster. |
| `wait` | no | — | Duration to sleep before this import starts (e.g. `"10s"`, `"1m30s"`). Validated at load time — invalid durations are caught before the cluster starts. |

---

## When to use `wait`

The common cases where state from the previous import needs time to clear:

- **Webhook deregistration** — the API server takes a few seconds to fully remove a `ValidatingWebhookConfiguration` or `MutatingWebhookConfiguration` after the previous test uninstalls it. The next test that installs conflicting CRDs will fail without a brief wait.
- **Namespace termination** — a deleted namespace stays in `Terminating` briefly. If the next import creates the same namespace, add `wait: 5s`.
- **Certificate provisioning** — cert-manager (installed via `setup.helm`) takes a moment to issue its first certificate after the webhook pod is running.

```yaml
imports:
  - ./admission/e2e.yaml
  - path: ./deletion-protection/e2e.yaml
    wait: 10s   # wait for admission webhook to deregister
  - path: ./namespace-protection/e2e.yaml
    freshCluster: true
```

---

## Validation

`ork validate` and `ork e2e` both check every import file before the test runs:

- The file must exist at the resolved path.
- The file must declare `kind: E2E`.

A missing file or wrong kind is a validation error — the run does not start.

---

## Cluster strategy

**Shared cluster (default — `freshCluster: false`):**
All shared imports run in the same cluster, one after the other. Each import:
- Applies its own CRD, bundle, and CR
- Runs its own expectations
- Cleans up its own resources before the next import starts

The cluster is provisioned once (either by the parent E2E or by `runImports` itself) and torn down after all imports finish.

**How the cluster is determined:**
- If the parent was invoked with `--use-current` or `--cluster`: imports reuse the same active context.
- If the parent created its own kind cluster (no flags): `runImports` provisions a separate `<name>-imports` cluster for all shared imports to use together.

**Fresh cluster (`freshCluster: true`):**
A new kind cluster is created for this specific import, independent of all other imports and the parent test. Use this when an import cannot safely share state — for example, a test that installs and uninstalls cluster-scoped resources that would affect other tests running concurrently.

---

## Pure aggregator

An E2E file with `imports:` but no `spec:` is a valid pure aggregator. It has no test of its own — it exists only to run a suite:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: multi-tenancy-suite
  description: Runs all three multi-tenancy sub-examples in the same cluster.

imports:
  - ./01-basic-namespacing/e2e.yaml
  - ./02-cross-access-control/e2e.yaml
  - ./03-shared-platform/e2e.yaml
```

`ork validate -f e2e.yaml` shows each import as a ✓ line. `ork e2e -f e2e.yaml` creates a cluster and runs all three imports in it.

---

→ Back: [03-expect.md](03-expect.md) | [Schema index](index.md)
