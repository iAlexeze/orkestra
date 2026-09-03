# Imports and Suite Composition

`imports` is a top-level field (alongside `spec:`) that lists other E2E files to run after the current one completes. It is the building block for test suites.

## Wire format

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: beginner-suite

imports:
  - ./01-hello-website/e2e.yaml
  - path: ./02-with-serviceaccount/e2e.yaml
    wait: 10s
  - path: ./03-infra/e2e.yaml
    freshCluster: true
```

## Fields

| Field | Default | Description |
|-------|---------|-------------|
| `path` | — | Path to another E2E spec file. Relative to this file's directory. |
| `wait` | — | Duration to sleep before this import starts (e.g. `10s`, `1m30s`). Validated at load time. |
| `freshCluster` | `false` | When `true`, provisions a new kind cluster for this import instead of sharing the suite cluster. |

### `wait:` — why you need it

State from the previous import sometimes needs time to clear before the next one starts:

- **Webhook deregistration**: the API server takes ~2–5s to fully remove a `ValidatingWebhookConfiguration` after the previous test uninstalled it. The next test that installs conflicting CRDs will fail without a wait.
- **Namespace termination**: a deleted namespace stays `Terminating` briefly. If the next import creates the same namespace, add `wait: 5s`.
- **Cert provisioning**: cert-manager (installed via `setup.helm`) takes a moment to issue its first certificate after the webhook pod is ready.

## Pure aggregator

An E2E file with `imports:` but no `spec:` is a pure aggregator — it exists only to run a suite:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: beginner-suite

imports:
  - ./01-hello-website/e2e.yaml
  - ./02-with-serviceaccount/e2e.yaml
  - ./03-secret-copy/e2e.yaml
  - ./03b-configmap-copy/e2e.yaml
```

## Shared Orkestra (default)

For non-`freshCluster` imports, the coordinator installs Orkestra **once** before the import loop. Each import updates the bundle in place and syncs the runtime silently. After all imports complete, the coordinator uninstalls once.

A suite of 10 imports produces exactly one `→ Installing Orkestra...` and one `→ Uninstalling Orkestra...`.

## Validation

`ork validate` and `ork e2e` both check every import file before the test runs:
- The file must exist at the resolved path
- The file must declare `kind: E2E`
- Any `wait:` value must be a valid Go duration string

A missing file, wrong kind, or invalid duration is a validation error — the run does not start.
