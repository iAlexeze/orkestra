# 04 — Drift Correction

A `Database` CRD creates a `BackupPolicy` child CR with `reconcile: true`. This means Orkestra enforces the child's spec on every reconcile cycle. If someone manually deletes or edits the `BackupPolicy`, Orkestra recreates or resets it within the resync window — no human intervention needed.

---

## What This Shows

1. `Database` CR is created.
2. Orkestra creates `BackupPolicy` child CR via `onReconcile.custom`.
3. `reconcile: true` on the child means Orkestra applies the spec on every Database reconcile (every 30s).
4. Manually deleting the `BackupPolicy` → Orkestra recreates it within 30s.
5. Manually changing `backuppolicy.spec.schedule` → Orkestra resets it on the next cycle.

---

## New Concepts Introduced

### `reconcile: true` on child blocks

Without `reconcile: true`, child CRs are created once (`onCreate`) or verified to exist (`onReconcile`) but their spec is not actively corrected.

With `reconcile: true`, Orkestra performs a full apply (patch) of the declared spec on every parent reconcile cycle. This enforces spec as a continuous invariant.

```yaml
onReconcile:
  custom:
    - apiVersion: storage.example.io/v1alpha1
      kind: BackupPolicy
      metadata:
        name: "{{ .metadata.name }}-backup"
        namespace: "{{ .metadata.namespace }}"
        namespaced: true
      spec:
        schedule: "{{ .spec.backup.schedule }}"
        retention: "{{ .spec.backup.retention }}"
        engine: "{{ .spec.engine }}"
      reconcile: true   # <-- enforce spec on every cycle
      hasStatus: false
```

### `onCreate.custom` vs `onReconcile.custom`

| Block | When it runs | Typical use |
|---|---|---|
| `onCreate.custom` | Only when the parent CR is first created | Cheap one-time child provisioning |
| `onReconcile.custom` | Every reconcile cycle | Children that must always match spec |

Use `onReconcile.custom` with `reconcile: true` when the child must stay in sync with the parent spec at all times.

### Why this matters

Without drift correction, an operator has a "reconcile gap": a human or another controller can modify a child resource and the parent operator won't notice until something else triggers a reconcile. `reconcile: true` closes that gap continuously.

---

## Prerequisites

- `kubectl` configured to a running cluster (Kind works)
- Ork CLI:
  ```bash
  curl get.orkestra.sh | bash
  ```

---

## Run the Example

### 1. Apply the CRDs

```bash
kubectl apply -f crd-database.yaml
kubectl apply -f crd-backuppolicy.yaml
```

### 2. Start the operator

```bash
ork run -f katalog.yaml 
```

### 3. Apply the Database CR

```bash
kubectl apply -f cr.yaml
```

Check that the BackupPolicy was created:

```bash
kubectl get database,backuppolicy -n default
```

Expected:

```
NAME                                      ENGINE    VERSION   PHASE   AGE
database.storage.example.io/prod-postgres postgres  15.4      Ready   10s

NAME                                             SCHEDULE    RETENTION   ENGINE     PHASE   AGE
backuppolicy.storage.example.io/prod-postgres-backup   0 2 * * *   14          postgres   Active  8s
```

### 4. Delete the BackupPolicy manually

```bash
kubectl delete backuppolicy prod-postgres-backup -n default
```

Watch it come back:

```bash
kubectl get backuppolicy -n default -w
```

Within the resync window (30s), Orkestra detects the missing child and recreates it.

### 5. Mutate the BackupPolicy spec

```bash
kubectl patch backuppolicy prod-postgres-backup -n default --type=merge \
  -p '{"spec":{"schedule":"0 6 * * *"}}'
```

Check the current schedule:

```bash
kubectl get backuppolicy prod-postgres-backup -n default -o jsonpath='{.spec.schedule}'
# Outputs: 0 6 * * *  (the bad value)
```

Wait 30s, then check again:

```bash
kubectl get backuppolicy prod-postgres-backup -n default -o jsonpath='{.spec.schedule}'
# Outputs: 0 2 * * *  (corrected by Orkestra)
```

---

## What to Observe

- The BackupPolicy is recreated within the `resync: 30s` window after deletion.
- Spec mutations are silently corrected on the next reconcile without any alert or error — the declared state wins.
- The operator logs show a "drift detected, applying" message on each correction cycle.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
