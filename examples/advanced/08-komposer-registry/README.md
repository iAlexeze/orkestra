# 08 — Komposer with Registry

Three sources. One runtime. A production Komposer that pulls versioned operator
patterns from an OCI registry, combines them with a local Katalog and a Helm
chart source, and applies environment-specific overrides.

**What you learn:** OCI registry sources, version pinning, Helm chart sources,
multi-source composition, production Komposer structure, disabling CRDs.

**Builds on:** [06 — Basic Komposer](../../intermediate/06-komposer-basic/README.md)

---

## The composition

```
OCI:  ghcr.io/orkspace/orkestra-registry/patterns/katalogs/postgres:v1.0.0
        → postgres CRD with production-tested operator behavior

File: ./website-katalog.yaml
        → your team's custom website operator

Helm: https://github.com/orkspace/charts.git
        → additional internal CRDs from a Helm chart

Overrides (spec.crds):
        → postgres: workers 8, resync 30s
        → website: workers 6, resync 15s
```

Three sources, two overrides, one Orkestra instance.

---

## Understanding OCI patterns

An OCI pattern is a versioned artifact in a container registry. It can contain the following files:

```text
postgres/v1.0.0/
  crd.yaml        the PostgreSQL CRD definition
  katalog.yaml    operator behavior (what Orkestra reads)   → Required
  komposer.yaml   example import reference
  cr.yaml         example PostgreSQL CR
  simulate.yaml   in-memory test
  e2e.yaml        end-to-end test
  README.md       pattern documentation
```

> [!NOTE]
> Only `katalog.yaml` is required.

The `v1.0.0` pin in the URL is immutable — once published, that version's
content cannot change. This is the guarantee that makes OCI distribution safe
for production: you know exactly what you pulled.

> To learn more about OCI patterns including publishing, see: `ork init --pack registry-guide`

---

## Steps

### 0. OCI Inspection

- Inspect the registry pattern:

```bash
ork inspect postgres:v1.0.0
```

Expected:
```text
postgres:v1.0.0
  Registry:    ghcr.io
  Kind:        Katalog
  Digest:      sha256:3b070e8b7aaac6009d7d617db70d08b96d21953784c45fcb1767e29ba2783679
  Pushed:      2026-06-13T05:50:33Z
  Size:        2.5 KB
  ...
  Simulate:    ✓ Verified · 6 assertions · 862ms · tested 14d ago
  E2E:         ✓ Verified · 5 assertions · 2m21s · tested 14d ago

  Files:
    katalog.yaml                   1.5 KB
    crd.yaml                       808 B
    cr.yaml                        160 B
    e2e.yaml                       1.2 KB
    simulate.yaml                  723 B
```

- Preview any of the files:

```bash
ork inspect postgres:v1.0.0 --view katalog.yaml
```

The inspection stage is useful in understanding what is to be pulled from a public registry. You can view the tests that ran and replicate it in your environment:

#### Optional
```bash
# View unit tests and end-to-end tests
ork inspect postgres:v1.0.0 --view simulate.yaml,e2e.yaml

# Pull to a temporary directory
ork pull postgres:v1.0.0 -o test-pattern

# Run the tests
cd test-pattern
ork simulate    # Quick in-memory test. No cluster
ork e2e         # 
```

Once satisfied, you can proceed.

### 1. Preview the merged configuration

First pull all dependencies to cache:

```bash
ork pull -f komposer.yaml
```

Preview the merged configuration:

```bash
ork template
```

```bash
# To see the startup order:
ork template --graph
```

This shows the merged result of all three sources with overrides applied —
without pulling anything or touching a cluster.

### 2. Validate

```bash
ork validate
```

Expected:
```text
● database
    kind: Database / group: demo.orkestra.io / version: v1alpha1 / plural: databases / scope: Namespaced
    mode: dynamic / workers: 2 / resync: 1m0s

● cache
    kind: Cache / group: demo.orkestra.io / version: v1alpha1 / plural: caches / scope: Namespaced
    mode: dynamic / workers: 2 / resync: 30s

● postgres
    kind: Postgres / group: postgres.orkestra.io / version: v1 / plural: postgreses / scope: Namespaced
    mode: dynamic / workers: 8 / resync: 30s

● website
    kind: Website / group: demo.orkestra.io / version: v1alpha1 / plural: websites / scope: Namespaced
    mode: dynamic / workers: 6 / resync: 15s

────────────────────────────────────────────────────────────
4 CRDs valid (0 built-in, 4 custom)
```

Validates all sources, checks for dependency cycles, verifies pattern structure.
If the OCI registry is unreachable, `ork validate` reports which source failed.

#### 2.1 Disable Database CRD
Let's disable the `database` CRD. To do this, add an override block under `spec.crds` in [`komposer.yaml`](komposer.yaml) after the `website` block:


```yaml
  database:
    enabled: false
```

Run `ork validate` again.

Expected:
```text
● cache
    kind: Cache / group: demo.orkestra.io / version: v1alpha1 / plural: caches / scope: Namespaced
    mode: dynamic / workers: 2 / resync: 30s

● postgres
    kind: Postgres / group: postgres.orkestra.io / version: v1 / plural: postgreses / scope: Namespaced
    mode: dynamic / workers: 8 / resync: 30s

● website
    kind: Website / group: demo.orkestra.io / version: v1alpha1 / plural: websites / scope: Namespaced
    mode: dynamic / workers: 6 / resync: 15s

────────────────────────────────────────────────────────────
3 CRDs valid (0 built-in, 3 custom)
```

Now we have 3 valid CRDs to work with.

> [!TIP]
> If you want to enable the database CRD, uncomment the CRD block in [`crd.yaml`](crd.yaml) and [`cr.yaml`](cr.yaml).


#### 2.2 Generate runtime bundle
```bash
ork generate bundle -o bundle.yaml
```
This generates the following:
- Namespace
- Service accounts (runtime and control center)
- Cluster role    _**(just enough to manage your CRDs)**_
- Cluster role binding
- ConfigMap with komposer as content

This will be needed in the next step.

> [!NOTE]
> This is Orkestra's security-first approach.
> **RBAC enforced from day 1**

---

### 3. Deploy

```bash
# Install CRDs
kubectl apply -f crd.yaml
# (This file contains all CRDs)

# Apply Bundle generated in the previous step
kubectl apply -f bundle.yaml

# Deploy Orkestra
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --wait --timeout 120s
```

### 4. Verify both CRDs are managed

```bash
ork proxy

curl localhost:8080/katalog | jq '.crds[] | {name: .name, workers: .workers,}'
```

Expected:
```json
{"name": "postgres",  "workers": 8}
{"name": "website",   "workers": 6}
```

---

### 5. Apply the CR and Watch the Control Center
```bash
kubectl apply -f cr.yaml

# open the control center
ork proxy
```

Open http://localhost:8081

> [!NOTE]
> You might notice this error in `Website CRD: 
> "Last Error
> validation denied: field "spec.image": images must come from the internal registry (got "nginx:latest")"
>
> This is Orkestra's runtime validation denying the creation of Kubernetes resources since image is not from internal registry `myorg/`. See [`Website Katalog`](website-katalog.yaml) for validation details.

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
to a version. The CI pipeline runs `ork validate --file komposer.yaml` on
every pull request. No cluster needed for validation. Orkestra action provides easy E2E for your operator. Check it out here.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
