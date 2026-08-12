# simulate-envtest

Declarative integration tests that run `ork simulate --envtest` — the same reconciler loop as regular simulate, but against a real `kube-apiserver` + `etcd` started locally.

```bash
ork simulate -f tests/simulate-envtest/simulate.yaml --envtest
```

Binaries are auto-downloaded to `~/.ork/envtest-bins` on first run.

These tests replace Go integration tests in `tests/integration/` for scenarios that only need CRD schema enforcement, status subresource behaviour, and namespace filtering — the same scenarios expressed as `simulate.yaml` files that any operator author can read and extend. New API-server-level assertions belong here as `simulate.yaml` first; Go integration tests remain the home for lifecycle tests that require programmatic setup (startup/shutdown sequences, error injection, multi-step state machines).

## Tests

| Directory | What it verifies | Replaces |
|-----------|-----------------|----------|
| `01-probe-reconcile` | Deployment applied via SSA on cycle 1; loop reaches steady state | `tests/integration/kubeclient` create path |
| `02-status-patch` | Status patch reaches the real `/status` subresource endpoint | `tests/integration/kubeclient` status patch |
| `03-namespace-filter` | CR in `default`, operator allows only `staging` — no Deployment created | `tests/integration/informer` namespace filter |
| `04-conditional-reconciliation` | Gate pass: Deployment created when `spec.enabled: true`. Gate discard: no Deployment when `spec.enabled: false` — kordinator never calls the reconciler | kordinator pre-reconcile gate |

Tests 01–03 share root fixtures (`katalog.yaml`, `crd.yaml`, `cr.yaml`). Test 04 has its own fixtures in its directory.

## Shared fixtures

- `crd.yaml` — Probe CRD with `subresources.status` and `x-kubernetes-preserve-unknown-fields: true` on the status schema
- `katalog.yaml` — operator that creates a Deployment and patches status on reconcile
- `cr.yaml` — Probe CR in the `default` namespace

## See also

- [Declarative Integration Testing](../../documentation/concepts/envtest/index.md)
- [`ork simulate --envtest` reference](../../documentation/reference/cli/05-simulate.md)
