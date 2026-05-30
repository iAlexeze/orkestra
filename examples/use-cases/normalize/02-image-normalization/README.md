# Normalize 02 — Image Normalization

Three CRs. Three different image formats. One Deployment image in every case: `registry.internal/name:tag`. The `onCreate` Deployment template never branches — it just uses `.spec.image` knowing normalize already made it canonical.

**What you learn:** Multi-step normalize using template variables (`$base`, `$tagged`), `ternary`, `contains`, `trimPrefix`, `printf`.

---

## Step 1 — Validate

```bash
ork validate
```

Expected:
```
✓ app
    kind: App
    group: demo.orkestra.io / version: v1 / plural: apps
    mode: dynamic / workers: 2 / resync: 30s
```

---

## Step 2 — Start the operator

```bash
ork run
```

---

## Step 3 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081).

Select **app-operator**, then select the **App** CRD. Keep this open — you will watch CRs appear as you apply them.

---

## Step 4 — Apply the bare image CR

```bash
kubectl apply -f cr-bare.yaml
```

This CR has `image: nginx` — no tag, no registry.

Watch the Control Center: `app-bare` appears. Click it, then click **top-right** to see child resources. Open the `app-bare` Deployment.

```bash
kubectl get app app-bare -o yaml | grep "image:" 
```

Status:
```yaml
status:
  image: registry.internal/nginx:latest   # ← tag and registry added
```

Deployment:
```bash
kubectl get deployment app-bare -o jsonpath='{.spec.template.spec.containers[0].image}'
# registry.internal/nginx:latest
```

---

## Step 5 — Apply the tagged CR

```bash
kubectl apply -f cr-tagged.yaml
```

This CR has `image: nginx:1.25` — tag present, no registry.

```bash
kubectl get app app-tagged -o yaml | grep "image:"
# image: registry.internal/nginx:1.25
```

---

## Step 6 — Apply the full CR

```bash
kubectl apply -f cr-full.yaml
```

This CR has `image: registry.internal/nginx:1.25` — already fully qualified.

```bash
kubectl get app app-full -o yaml | grep "image:"
# image: registry.internal/nginx:1.25   ← unchanged, idempotent
```

---

## What normalize did

| CR | User wrote | After normalize |
|---|---|---|
| `app-bare` | `nginx` | `registry.internal/nginx:latest` |
| `app-tagged` | `nginx:1.25` | `registry.internal/nginx:1.25` |
| `app-full` | `registry.internal/nginx:1.25` | `registry.internal/nginx:1.25` |

The Deployment template uses `image: "{{ .spec.image }}"` — one expression, three formats handled.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
