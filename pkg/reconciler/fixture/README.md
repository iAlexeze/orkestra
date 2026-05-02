# pkg/reconciler/fixture

Living integration fixture for the reconciler's `run_*` resource types.

## Why this exists

The `run_*` functions in `pkg/reconciler/` apply Kubernetes resources from
katalog templates. There is no meaningful way to unit-test them: the logic that
matters — template rendering, server-side apply, status propagation, condition
evaluation — only works against a real API server. Mocking it tests the mock,
not Orkestra.

This fixture is the right vehicle. It declares one `ReconcilerProbe` CRD and a
companion CR that triggers every supported `run_*` code path in a single
reconcile cycle. Failures surface as a missing resource, a wrong status field,
or an operator crash — all observable without reading code.

## What each block covers

| Block in `katalog.yaml`            | `run_*` file exercised                |
|------------------------------------|---------------------------------------|
| `validation.rules`                 | `run_validations.go`                  |
| `mutation.rules`                   | `run_mutations.go`                    |
| `serviceAccounts`                  | `run_serviceaccounts.go`              |
| `configMaps`                       | `run_configmaps.go`                   |
| `secrets` (once: true)             | `run_secrets.go`, `run_secrets_once.go` |
| `deployments` (main)               | `run_deployments.go`                  |
| `services`                         | `run_services.go`                     |
| `cronJobs`                         | `run_cronjobs.go`                     |
| `jobs`                             | `run_jobs.go`                         |
| `deployments` with `when:`         | `run_deployments.go` + `conditions.go` |
| `status.fields`                    | `run_status.go`                       |

Resource types not yet in the fixture (PR welcome):
`statefulSets`, `ingresses`, `hpa`, `pdb`, `persistentVolumeClaims`,
`namespaces`, `replicaSets`, `run_foreach.go`, `run_cross.go`

## Running locally

Requires: `kind`, `helm`, `kubectl`, `ork` — all installed and on `$PATH`.

```bash
# From the repo root
make test-fixture-reconciler
```

That target:
1. Creates a kind cluster (`orkestra-reconciler-fixture`)
2. Applies `fixture/crd.yaml`
3. Generates and applies the Orkestra bundle from `fixture/katalog.yaml`
4. Installs Orkestra via the local Helm chart
5. Applies `fixture/cr.yaml`
6. Waits for `status.phase` to be set
7. Asserts expected resources exist
8. Deletes the cluster

To iterate manually without tearing down the cluster:

```bash
scripts/setup-kind.sh orkestra-reconciler-fixture

kubectl apply -f pkg/reconciler/fixture/crd.yaml
ork bundle --katalog pkg/reconciler/fixture/katalog.yaml | kubectl apply -f -

helm install orkestra charts/orkestra --namespace default

kubectl apply -f pkg/reconciler/fixture/cr.yaml
kubectl get reconcilerprobe probe -o yaml -w
```

Clean up:

```bash
cd pkg/reconciler/fixture && bash cleanup.sh
scripts/setup-kind.sh delete orkestra-reconciler-fixture
```

## Adding a new `run_*`

When you add a new resource type to the reconciler (e.g. `run_widgets.go`):

1. **Add a block to `katalog.yaml`** that exercises it. Use the existing blocks
   as a template. Name resources `{{ .metadata.name }}-widget` to avoid collisions.

2. **Add a row to the table above** so the coverage map stays accurate.

3. **Add an assertion** in the CI workflow
   (`.github/workflows/validate-pr.yml`, job `fixture-tests`) that confirms the
   resource was created:
   ```yaml
   - name: Assert Widget created
     run: kubectl get widget probe-widget
   ```

4. **Run `make test-fixture-reconciler` locally** before opening the PR.

The fixture is the contract that a new `run_*` function works end-to-end.
If the block renders and the resource appears, the function works. If Orkestra
crashes or the resource is absent, the function has a bug.
