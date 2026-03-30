# 08 — Komposer with Registry

Three sources. One runtime. A production Komposer that pulls versioned operator
patterns from an OCI registry, combines them with a local Katalog and a Helm
chart source, and applies environment-specific overrides.

**What you learn:** OCI registry sources, version pinning, Helm chart sources,
multi-source composition, `critical: true`, production Komposer structure.

**Builds on:** [06 — Basic Komposer](../../intermediate/06-komposer-basic/)

---

## The composition

```
OCI:  ghcr.io/konduktor-io/orkestra-registry/postgres@v14.2.0
        → postgres CRD with production-tested operator behavior

File: ./website-katalog.yaml
        → your team's custom website operator

Helm: charts.myorg.io/platform-crds:2.1.0
        → additional internal CRDs from a Helm chart

Overrides (spec.crds):
        → postgres: workers 8, resync 30s
        → website: workers 6, resync 15s, critical: true
```

Three sources, two overrides, one Orkestra instance.

---

## Understanding OCI patterns

An OCI pattern is a versioned artifact in a container registry. It contains
exactly five files:

```
postgres/v14.2.0/
  crd.yaml        the PostgreSQL CRD definition
  katalog.yaml    operator behavior (what Orkestra reads)
  komposer.yaml   example import reference
  cr.yaml         example PostgreSQL CR
  README.md       pattern documentation
```

The `@v14.2.0` pin in the URL is immutable — once published, that version's
content cannot change. This is the guarantee that makes OCI distribution safe
for production: you know exactly what you pulled.

---

## Steps

### 1. Preview the merged configuration

```bash
ork template --katalog komposer.yaml
```

This shows the merged result of all three sources with overrides applied —
without pulling anything or touching a cluster.

### 2. Validate

```bash
ork validate --katalog komposer.yaml
```

Validates all sources, checks for dependency cycles, verifies pattern structure.
If the OCI registry is unreachable, `ork validate` reports which source failed.

### 3. Deploy

```bash
# Install CRDs
kubectl apply -f website-crd.yaml
# (postgres CRD comes from the OCI pattern — applied by Orkestra at startup)

# Apply Katalog ConfigMap
kubectl apply -f orkestra-configmap.yaml

# Deploy Orkestra
kubectl apply -f ../../installation/install-webhook-support.yaml

kubectl wait --for=condition=available deployment/orkestra \
  -n orkestra-system --timeout=60s
```

### 4. Verify both CRDs are managed

```bash
kubectl port-forward svc/orkestra 8080:8080 -n orkestra-system &

curl localhost:8080/katalog | jq '.crds[] | {name: .name, workers: .workers, critical: .critical}'
```

Expected:
```json
{"name": "postgres",  "workers": 8, "critical": false}
{"name": "website",   "workers": 6, "critical": true}
```

### 5. Test `critical: true`

Scale the Orkestra deployment replicas to observe critical CRD behavior:

```bash
# The website CRD is marked critical: true
# If it degrades, the entire operator degrades
curl localhost:8080/health   # 200 when healthy
```

---

## The production pattern

This example shows the canonical structure for a production Orkestra deployment:

```
team-platform/
  infrastructure/
    orkestra/
      komposer.yaml      ← environment-specific composition
      website.katalog.yaml ← team's own Katalogs
  applications/
    websites/
      my-site.yaml       ← the CRs
```

The `komposer.yaml` is the production artifact. It pins every external source
to a version. The CI pipeline runs `ork validate --katalog komposer.yaml` on
every pull request. No cluster needed for validation.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
