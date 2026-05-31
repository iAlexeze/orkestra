# 05 — OR Conditions (anyOf:)

`anyOf:` fires a resource when any one of several conditions is true. Combined with `when:` (AND), the full logic is: `(when conditions) AND (any one of anyOf conditions)`. This replaces multi-branch `if a || b` hooks in Go with a declarative block on the resource itself.

**What you learn:** `anyOf:` semantics, combining `when:` AND `anyOf:` OR on the same resource, and how phase transitions cascade through a Job sequence without custom state machine code.

---

## Step 1 — Apply the CRD

```bash
kubectl apply -f crd.yaml
```

---

## Step 2 — Validate

```bash
ork validate
```

Expected:
```
✓ flex-app
    kind: FlexApp
    group: advanced.orkestra.io / version: v1alpha1 / plural: flexapps
    mode: dynamic / workers: 2 / resync: 30s
```

---

## Step 3 — Start the operator

```bash
ork run
```

---

## Step 4 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **flex-app**, then select the **FlexApp** CRD.

---

## Step 5 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

The CR starts in `Running`. No Jobs yet — neither cleanup nor notify has a matching phase.

```bash
kubectl get flexapp my-flex-app
# NAME           PHASE
# my-flex-app    Running
```

```bash
kubectl get jobs
# No resources found.
```

---

## Step 6 — Enable notifications

Patch `spec.notify` to `"true"`:

```bash
kubectl patch flexapp my-flex-app --type=merge -p '{"spec":{"notify":"true"}}'
```

The notify Job's combined condition now passes:
- `when: spec.notify == "true"` ✓
- `anyOf: phase == Running` ✓

```bash
kubectl get jobs
# NAME                    COMPLETIONS
# my-flex-app-notify      1/1
```

Watch the phase transition — once the notify Job completes, phase becomes `Succeeded`:

```bash
kubectl get flexapp my-flex-app
# NAME           PHASE
# my-flex-app    Succeeded
```

---

## Step 7 — Cascade to cleanup

Phase is now `Succeeded` — the cleanup Job's `anyOf:` fires:

```bash
kubectl get jobs
# NAME                    COMPLETIONS
# my-flex-app-cleanup     1/1
# my-flex-app-notify      1/1
```

Both Jobs appeared in sequence without writing any state machine logic. The phase field drove the cascade.

---

## Cleanup

```bash
kubectl delete flexapp my-flex-app --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
```
