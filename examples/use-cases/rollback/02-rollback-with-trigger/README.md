# 02 — Rollback with Custom Trigger

`rollBackOnError: true` with an explicit trigger threshold. All three resource types (Deployment, Service, ConfigMap) roll back together automatically.

**What you learn:** combining `rollBackOnError: true` with `rollback.trigger`, multi-resource derived rollback, the `withinDuration` window.

---

## The pattern

```yaml
operatorBox:
  rollBackOnError: true
  rollback:
    trigger:
      consecutiveFailures: 5
      withinDuration: 10m
```

`rollBackOnError: true` derives the rollback templates from `reconcile: true` resources.  
`rollback.trigger` overrides the default threshold (3 failures) without requiring a full `rollback:` block.  
`withinDuration: 10m` ensures transient cluster errors — API server blips, brief network issues — do not trigger rollback.

---

## What gets rolled back

All three resources declared with `reconcile: true` are included in the derived rollback:

| Resource | Template | On rollback |
|---|---|---|
| Deployment | `image: "{{ .spec.image }}"` | restores previous image |
| Service | `targetPort: "{{ .spec.port }}"` | restores previous port |
| ConfigMap | `config: "{{ .spec.configData }}"` | restores previous config |

No `onRollback:` block. No `.previous.spec.*` syntax. When rollback runs, `.spec.*` resolves to the previous spec values automatically.

---

## Step 1 — Apply the CRD

```bash
kubectl apply -f crd.yaml
```

---

## Step 2 — Start the operator

```bash
ork run -k katalog.yaml
```

---

## Step 3 — Apply the good CR

In a second terminal:

```bash
kubectl apply -f cr.yaml
```

Verify:

```bash
kubectl get apiserver my-api -o yaml
```

Expected status:

```yaml
status:
  phase: Ready
  image: nginx:1.25
  replicas: 2
  configVersion: v1-stable-config
```

Check the ConfigMap was created with the correct data:

```bash
kubectl get configmap my-api-config -o yaml
```

---

## Step 4 — Apply the bad spec

```bash
kubectl apply -f cr-bad.yaml
```

This changes both the image (to a non-existent tag) and the config value. Watch the operator terminal. After 5 failures within 10 minutes, rollback activates:

```
{"level":"warn","name":"my-api","message":"rollback: threshold reached — marking rollback active"}
{"level":"info","name":"my-api","derived":true,"message":"rollback: previous state re-applied"}
```

Verify the rollback restored all three resources:

```bash
# Deployment is back on the previous image
kubectl get deployment my-api -o jsonpath='{.spec.template.spec.containers[0].image}'

# ConfigMap reverted to v1-stable-config
kubectl get configmap my-api-config -o jsonpath='{.data.config}'

# CR status shows RolledBack
kubectl get apiserver my-api
```

```
NAME     PHASE       IMAGE        REPLICAS   AGE
my-api   RolledBack  nginx:1.25   2          4m
```

---

## Step 5 — Exit rollback

Fix the spec and reapply:

```bash
kubectl apply -f cr.yaml
```

The generation increments. On the next reconcile cycle, rollback clears and normal reconciliation resumes.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
