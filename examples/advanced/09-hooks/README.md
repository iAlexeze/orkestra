# 09 — Go Hooks

Declarative templates cover the common case. When your operator needs type-safe
struct access, complex conditional logic, or calls to external APIs, hooks are
the answer. Orkestra still provides everything else — you write only the logic
that genuinely requires code.

**What you learn:** `apiTypes.location`, typed mode, `ork generate runtime`,
typed hooks, OrkestraRegistry from Go, the difference between hooks and a
full constructor.

**Builds on:** [07 — Validation and Mutation](../07-validation-mutation/)

---

## Why typed hooks exist

In dynamic mode (no `apiTypes.location`), the reconcile loop receives
`*unstructured.Unstructured`. Accessing `spec.engine` looks like this:

```go
engine, _, _ := unstructured.NestedString(obj.Object, "spec", "engine")
```

In typed mode, the same field is:

```go
engine := obj.Spec.Engine
```

Type-safe, IDE-autocompleted, compile-time checked. Typed mode is what you
reach for when hooks need to make decisions based on multiple spec fields,
or when the spec is complex enough that map navigation becomes error-prone.

---

## What Orkestra still provides

Hooks do not replace Orkestra's runtime. The hook receives control for
reconcile and delete logic only. Orkestra still handles:

- Informer watching `demo.orkestra.io/v1alpha1, Kind=Database`
- Workqueue with deduplication and rate-limited backoff
- Worker pool (3 workers per the Katalog)
- Finalizer management — the CR is protected from dirty deletion
- Kubernetes events — `Reconciled` and `ReconcileError` events emitted per cycle
- Status Layer 1 — `Ready` condition written after every reconcile
- Status Layer 2 — `phase`, `engine`, `endpoint` written on success
- Prometheus metrics — reconcile total, duration, queue depth

The hook provides: type-safe access to spec fields, the business logic that
cannot be expressed in templates, and direct calls to OrkestraRegistry for
Kubernetes child resources.

---

## Step 1 — Code generation

Because `apiTypes.location` is set, Orkestra needs a generated file that
registers the Go types at startup. Run this once, and again whenever you
change `apiTypes` fields:

```bash
ork generate runtime --katalog katalog.yaml --output ./cmd/
```

This produces `cmd/zz_generated_runtime_registry.go`:

```go
func init() {
    orktypes.ObjectRegistry[schema.GroupVersionKind{
        Group:   "demo.orkestra.io",
        Version: "v1alpha1",
        Kind:    "Database",
    }] = func() runtime.Object { return &apiv1.Database{} }

    orktypes.SchemeRegistry = append(orktypes.SchemeRegistry, apiv1.AddToScheme)

    orktypes.HookRegistry[schema.GroupVersionKind{
        Group:   "demo.orkestra.io",
        Version: "v1alpha1",
        Kind:    "Database",
    }] = hooks.DatabaseHooks
}
```

---

## Step 2 — Build the binary

```bash
go mod tidy
go build ./...
```

If code generation was not run, this fails with:

```
error: no object factory for demo.orkestra.io/v1alpha1, Kind=Database
  — run: ork generate runtime --katalog katalog.yaml
```

---

## Step 3 — Install and run

```bash
kubectl apply -f crd.yaml

# Option A: local development
ork run --katalog katalog.yaml

# Option B: in-cluster (after building and pushing your image)
kubectl apply -f orkestra-configmap.yaml
kubectl apply -f ../../installation/install-webhook-support.yaml
kubectl wait --for=condition=available deployment/orkestra \
  -n orkestra-system --timeout=60s
```

---

## Step 4 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

---

## Step 5 — Verify hook behavior

```bash
# Deployment created by the hook via orkdeploy.Update
kubectl get deployment my-db

# Service created by the hook via orksvc.Update
kubectl get service my-db-svc

# Backup CronJob — created because spec.backup: true
kubectl get cronjob my-db-backup
```

If `backup: false`, the CronJob is not created — the conditional in the hook
prevents it. This is the equivalent of a `when:` condition, expressed in Go
because the CronJob spec also varies by engine type.

---

## Step 6 — Verify status

```bash
kubectl get database my-db -o yaml | grep -A15 "status:"
```

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
  phase: Running           ← Layer 2: from katalog status.fields
  engine: postgres         ← Layer 2: from .spec.engine
  endpoint: my-db-svc.default.svc.cluster.local
```

Layer 1 and Layer 2 status work even when hooks are used — Orkestra applies
them after the hook returns.

---

## Step 7 — Update the CR

Change backup from true to false:

```bash
kubectl patch database my-db --type=merge -p '{"spec":{"backup":false}}'
```

The CronJob is **not** deleted automatically — the hook does not delete it,
it only creates it. This is a deliberate design: hooks are responsible for
what they explicitly manage. If the hook should remove the CronJob when
`backup` becomes false, add explicit deletion logic:

```go
if !obj.Spec.Backup {
    orkcron.Delete(ctx, kube, obj, cronSpec)
}
```

Owner references handle deletion when the CR itself is deleted — the CronJob
is garbage-collected when `my-db` is deleted.

---

## Hooks vs templates: when to use each

| | Templates | Hooks |
|---|---|---|
| Resource creation from CR fields | ✓ preferred | possible |
| Type-safe spec field access | ✗ | ✓ |
| Complex conditional logic | limited (when:) | ✓ |
| External API calls | ✗ | ✓ |
| Multi-resource with shared state | limited | ✓ |
| `ork generate runtime` needed | ✗ | ✓ |

Most operators use templates for the standard resources (Deployment, Service,
ConfigMap) and hooks only for the parts that genuinely require Go.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
