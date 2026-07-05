# 17 — API Type Override

A vendor ships a Katalog. Your company has its own API group. This example shows
how to import the vendor Katalog with a Komposer and override only the `apiTypes`
block — your operator runs identical logic under your own CRD identity.

**What you learn:** `apiTypes` override in a Komposer, white-label operator pattern,
how inherited fields (reconciler, status, onCreate) are untouched by the override.

**Builds on:** [06 — Basic Komposer](../../intermediate/06-komposer-basic/README.md)

---

## The problem

A vendor ships:
```yaml
# vendor-katalog.yaml
spec:
  crds:
    db-cluster:
      apiTypes:
        group: vendor.example.io   # vendor's group
        kind: DatabaseCluster
```

You want to run the same operator but watch `platform.acme.io/DatabaseCluster` —
your company's API surface. You cannot modify the vendor file.

---

## The solution

```yaml
# komposer.yaml
imports:
  files:
    - ./vendor-katalog.yaml

spec:
  crds:
    db-cluster:
      apiTypes:
        group: platform.acme.io    # your group — only this changes
        version: v1alpha1
        kind: DatabaseCluster
        plural: databaseclusters
      # workers, resync, operatorBox all inherited from vendor-katalog.yaml
```

The Komposer replaces `apiTypes` for `db-cluster`. Everything else — workers,
resync interval, status fields, onCreate resources — is inherited from the vendor
Katalog unchanged.

---

## Steps

### 1. Install your CRD

```bash
kubectl apply -f crd-mine.yaml
```

> The vendor CRD (`crd-vendor.yaml`) is included for reference. You do not install
> it — your Komposer points the operator at your CRD instead.

### 2. Validate

```bash
ork validate --file komposer.yaml
```

Expected:
```text
● db-cluster
    kind: DatabaseCluster
    group: platform.acme.io / version: v1alpha1 / plural: databaseclusters
    mode: dynamic / workers: 2 / resync: 45s
```

The group is `platform.acme.io` — not `vendor.example.io`. The workers and
resync values come from the vendor Katalog.

### 3. Start the runtime

```bash
ork run --file komposer.yaml
```

### 4. Apply a CR

```bash
kubectl apply -f cr.yaml
```

`cr.yaml` uses `apiVersion: platform.acme.io/v1alpha1` — your API group.

### 5. Verify

```bash
kubectl get databaseclusters
kubectl get statefulsets | grep prod-db
kubectl get services   | grep prod-db
```

The operator created a StatefulSet and Service for `prod-db` under your API group.
The vendor's group (`vendor.example.io`) is never involved at runtime.

---

## What the override covers

| Field | Source | Notes |
|-------|--------|-------|
| `apiTypes.group` | Komposer (overrides) | Your company's API group |
| `apiTypes.kind` | Komposer (overrides) | Same kind name |
| `workers` | Vendor Katalog (inherited) | Can be overridden |
| `resync` | Vendor Katalog (inherited) | Can be overridden |
| `operatorBox` | Vendor Katalog (inherited) | Full reconciler logic reused |

The Komposer override wins field-by-field. You only declare what changes.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
