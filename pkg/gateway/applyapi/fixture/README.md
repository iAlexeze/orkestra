# pkg/gateway/fixture

Living integration fixture for the Gateway Apply API and IDP features.

## Why this exists

The gateway's Apply API, schema endpoint, IDP form, and admission webhooks only work
against a running operator with a real API server. Unit tests cover rule parsing; this
fixture covers the full stack — from a CR submitted via the Apply API through to the
reconciled child resource in the cluster.

The fixture uses a `PlatformResource` CRD with three workload types (`app`, `cert`,
`monitoring`), giving one CRD that exercises every IDP feature: conditional field hints,
conditional admission rules, type-specific child resources, and status projection.

## What this covers

| Feature | Where |
|---------|-------|
| Apply API (`POST /api/v1/apply`) | `katalog.yaml` → `gateway.applyAPI` |
| Schema catalog endpoint (`GET /api/v1/schema/`) | `katalog.yaml` → `idp.category` / `idp.description` |
| Admission webhook — unconditional deny | `admission/platformresource.yaml` rules 1–3 |
| Admission webhook — `deny when:` | `admission/platformresource.yaml` rule 4 (domain for cert) |
| Admission webhook — `deny anyOf:` | `admission/platformresource.yaml` rule 5 (repoURL for app/monitoring) |
| Admission webhook — `warn when:` | `admission/platformresource.yaml` rule 7 (productionApproval) |
| IDP field categories (`idp.fields.category`) | `idp/platformresource.yaml` |
| Conditional field visibility (`idp.fields.when`) | `idp/platformresource.yaml` |
| Conditional field visibility (`idp.fields.anyOf`) | `idp/platformresource.yaml` |
| Disabled/locked fields (`idp.fields.disabled`) | `idp/platformresource.yaml` |
| `ignoreFields` — hide system fields from form | `katalog.yaml` → `idp.ignoreFields` |
| CRD schema defaults pre-populating form inputs | `crd.yaml` |
| `dryRun=true` — violation preview without admission | `e2e.yaml` |
| Conditional child resource (`when:`) | `katalog.yaml` → `onReconcile.custom` |
| Status field projection | `status/platformresource.yaml` |

## Running

### Step 1 — build

```bash
make ork
```

### Step 2 — validate

```bash
ork validate -f pkg/gateway/fixture/katalog.yaml
```

This exercises `StrictUnmarshal` on all included sub-files — typos in field names
surface here, before any cluster is involved.

### Step 3 — simulate

```bash
ork simulate -f pkg/gateway/fixture/simulate.yaml
```

Simulate runs the reconciler in-memory against a fake API server — no cluster needed.
`cr-sim.yaml` contains all three workload types so every `custom:` branch is exercised:
`payments-api` (app → Application), `api-cert` (cert → Certificate),
`api-monitor` (monitoring → ServiceMonitor), all asserted on cycle 1.

### Step 4 — full cluster e2e

```bash
ork e2e -f pkg/gateway/fixture/e2e.yaml
```

`ork e2e` provisions a kind cluster, installs Orkestra via `helm upgrade --install`
with `charts/orkestra`, applies `setup.yaml` (namespaces), installs ArgoCD,
applies the CRD and CR, runs all assertions, and tears down. Assertions include:

- `PlatformResource` accepted and reaches `Ready`
- ArgoCD `Application` created and removed on deletion
- Admission blocks missing `spec.team`
- Admission blocks invalid `spec.workloadType`

To iterate against an existing cluster without reprovisioning:

```bash
ork e2e -f pkg/gateway/fixture/e2e.yaml --use-current
```

For manual runs against an existing cluster, apply the namespaces first:

```bash
kubectl apply -f pkg/gateway/fixture/setup.yaml
```

To clean up after a `--use-current` run:

```bash
bash pkg/gateway/fixture/cleanup.sh
```

### Step 5 — token secret resilience

Verify the housekeeper detects and recreates a deleted Apply API token secret.

Deploy the fixture against a running cluster first:

```bash
ork generate bundle | kubectl apply -f -

helm upgrade --install orkestra ~/orkestra/charts/orkestra \
  --values ~/orkestra/unknown/values.yaml \
  --namespace orkestra-system \
  --create-namespace \
  --set gateway.enabled=true \
  --set controlCenter.gatewayToken.secretRef.name=ork-apply-token \
  --wait --timeout 120s
```

Then delete the token secret — the housekeeper should recreate it within one safety
ticker interval (default 30 s) with a fresh token value:

```bash
kubectl get secret ork-apply-token -n orkestra-system \
  -o jsonpath='{.data.token}' | base64 -d && echo

kubectl delete secret ork-apply-token -n orkestra-system

kubectl get secret ork-apply-token -n orkestra-system
```

The gateway logs will show:

```
housekeeper: apply API token secret missing — will reload
housekeeper: apply API tokens reloaded
```

After recreation the token value is new — compare it against the one captured above:

```bash
kubectl get secret ork-apply-token -n orkestra-system \
  -o jsonpath='{.data.token}' | base64 -d
```

### Step 6 — conditional admission rules

The CRs in [crs/](./crs/) each target one specific admission rule. Use them to
verify conditional `when:`/`anyOf:` behaviour against a running cluster:

```bash
# No violations — happy path
kubectl apply -f pkg/gateway/fixture/crs/app-staging.yaml

# Accepted with ValidationWarning (production, no approval ticket)
kubectl apply -f pkg/gateway/fixture/crs/app-production-warn.yaml

# Accepted, warn rule skipped (approval ticket present)
kubectl apply -f pkg/gateway/fixture/crs/app-production-approved.yaml

# Accepted, when: deny skipped (domain present)
kubectl apply -f pkg/gateway/fixture/crs/cert-valid.yaml

# Rejected — spec.domain is required for cert workloads
kubectl apply -f pkg/gateway/fixture/crs/cert-missing-domain.yaml

# Rejected — spec.repoURL is required for app and monitoring workloads
kubectl apply -f pkg/gateway/fixture/crs/monitoring-missing-repo.yaml
```

See [crs/README.md](./crs/README.md) for the full rule matrix.

## Fixture layout

```
pkg/gateway/fixture/
  katalog.yaml                 — entry point: gateway + CRD + operator config
  crd.yaml                     — PlatformResource CRD (schema + defaults)
  cr-default.yaml              — default happy-path CR (payments-api, app workload)
  cr-rejected.yaml             — CR that should be blocked by admission
  simulate.yaml                — in-memory simulation spec
  e2e.yaml                     — full cluster e2e spec
  cleanup.sh                   — manual teardown for --use-current runs
  admission/platformresource.yaml  — validation rules (deny, deny when, deny anyOf, warn when)
  idp/platformresource.yaml        — field hints, categories, conditions, disabled fields
  status/platformresource.yaml     — status field projections
  crs/                             — targeted CRs for conditional rule testing
```

## Adding a new feature

1. Add the feature to the appropriate sub-file (`idp/`, `admission/`, `status/`),
   or extend `katalog.yaml` directly for gateway-level config.
2. Add a row to the table above.
3. If it is an admission rule, add a CR to `crs/` and a row to `crs/README.md`.
4. If it changes reconciler behaviour, add a `simulate` op to `simulate.yaml`.
5. Add an e2e assertion to `e2e.yaml`.
6. Run `ork validate` then `ork simulate` locally before opening the PR.
