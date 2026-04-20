# 10 — Custom Constructor

Full reconciler control. The GenericReconciler is replaced entirely by a
constructor that implements `domain.Reconciler` directly. This is the pattern
for state machine operators, for migrating existing controller-runtime operators,
and for the 1% of use cases where the reconcile loop itself must be controlled.

**What you learn:** `reconciler.default: false`, constructor signature, state
machine pattern, finalizer management from Go, status patching from Go, when
constructors are the right tool.

**Builds on:** [09 — Hooks](../09-hooks/)

---

## Hooks vs constructor

| | Hooks | Constructor |
|---|---|---|
| GenericReconciler used | Yes | No |
| Finalizer management | Orkestra handles | You handle |
| Status Layer 1 (Ready condition) | Orkestra writes | You write |
| Status Layer 2 (declared fields) | Orkestra writes | Ignored |
| Kubernetes events | Orkestra emits | You emit |
| Cache reads | Orkestra handles | You handle |
| Use case | Most advanced operators | State machines, migrations |

Hooks are additive — you add logic inside Orkestra's loop. Constructors
replace the loop. Choose hooks unless you genuinely need to control the
loop itself.

---

## What Orkestra still provides with a constructor

Even with `reconciler.default: false`, Orkestra provides:

- Informer watching `demo.orkestra.io/v1alpha1, Kind=Pipeline`
- Workqueue — the constructor's `Reconcile` is called when items are dequeued
- Worker pool — `workers: 5` goroutines call `Reconcile` concurrently
- `safeReconcile` — panics in `Reconcile` are caught, logged, and requeued
- Prometheus metrics — reconcile total, duration, queue depth, error rate
- Per-CRD health tracking — `/katalog/pipeline/health`

The constructor owns everything else.

---

## The state machine

```
Create Pipeline CR
       │
       ▼
   Pending ──────────────────────────────────────────────────────────┐
       │                                                              │
       │  Create Job for step[0]                                     │
       │  status.phase = Running                                      │
       │  status.currentStep = "build"                               │
       ▼                                                              │
   Running                                                           │
       │                                                              │
       │  On each reconcile: check current step Job status            │
       │                                                              │
       ├── Job still running → no-op (resync will re-check)          │
       │                                                              │
       ├── Job failed ──────────────────────────────────────────────►┤
       │                 status.phase = Failed                        │
       │                                                              │
       └── Job succeeded:                                            │
               nextStep exists?                                       │
                    Yes → Create next step Job                        │
                          status.currentStep = "test"                 │
                          (loop back to Running)                      │
                    No  → status.phase = Succeeded                   │
                          status.completionTime = now                 │
                          → Terminal, no more reconciles needed       │
                                                                      │
Failed ◄──────────────────────────────────────────────────────────────┘
       Terminal state — no further reconciles
```

---

## Step 1 — Code generation

```bash
ork generate registry --katalog katalog.yaml --output ./cmd/
```

This registers `*apiv1.Pipeline` in the object registry and
`NewPipelineReconciler` in the reconciler registry.

---

## Step 2 — Build

```bash
go mod tidy
go build ./...
```

---

## Step 3 — Install and run

```bash
kubectl apply -f crd.yaml

# Local development
ork run --katalog katalog.yaml

# In-cluster
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

## Step 5 — Watch the state machine

```bash
# Watch status change in real time
kubectl get pipeline build-and-test -w
```

```
NAME             PHASE     STEP
build-and-test   Pending
build-and-test   Running   build
build-and-test   Running   test
build-and-test   Running   notify
build-and-test   Succeeded
```

Watch Jobs being created and completed:

```bash
kubectl get jobs -w
```

```
NAME                         COMPLETIONS   DURATION
build-and-test-build         0/1           3s
build-and-test-build         1/1           3s
build-and-test-test          0/1           2s
build-and-test-test          1/1           2s
build-and-test-notify        0/1           1s
build-and-test-notify        1/1           1s
```

---

## Step 6 — Check the final status

```bash
kubectl get pipeline build-and-test -o yaml | grep -A10 "status:"
```

```yaml
status:
  phase: Succeeded
  currentStep: notify
  message: "all steps completed"
  startTime: "2026-03-29T10:00:00Z"
  completionTime: "2026-03-29T10:00:06Z"
```

---

## Step 7 — Test failure

Edit the CR to introduce a failing step:

```bash
kubectl patch pipeline build-and-test --type=merge -p '{
  "spec": {
    "steps": [
      {"name": "fail", "command": ["sh", "-c", "exit 1"]}
    ]
  }
}'
```

The state machine drives to `Failed`:

```yaml
status:
  phase: Failed
  message: "step \"fail\" failed"
```

Because `Failed` is a terminal state, no further reconciles run. The Pipeline
stays in `Failed` until you delete it and create a new one — or until your
reconciler adds a retry mechanism.

---

## When to use a constructor

The constructor is the right tool when:

- The reconcile loop drives through **named phases** where each phase has
  distinct behavior — state machines
- You have an **existing controller-runtime reconciler** and want Orkestra's
  runtime infrastructure without rewriting your logic
- The reconcile logic needs to **read other resources** mid-reconcile and
  make decisions based on their state (Job completion, external API status)
- You need **complete control over event emission** and status — the hook
  model's automatic events and status would conflict with your logic

For everything else — hooks are the right tool. Hooks give you type-safe
struct access and full Go expressiveness while Orkestra manages the loop.

---

## Migration from controller-runtime

If you have an existing `Reconcile(ctx, req) (ctrl.Result, error)`:

```go
// Before: controller-runtime
func (r *PipelineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // your existing logic
    return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// After: Orkestra constructor
func (r *PipelineReconciler) Reconcile(ctx context.Context, key string) error {
    // same logic — return error to requeue, nil to complete
    // RequeueAfter equivalent: return a sentinel error with exponential backoff
    return nil
}
```

The key changes:
- `ctrl.Request` → `key string` (same namespace/name, different format)
- `(ctrl.Result, error)` → `error` (requeue by returning an error)
- Remove `ctrl.Manager` setup — Orkestra provides the informer and queue

Register `NewPipelineReconciler` in the Katalog and run `ork generate registry`.
Your existing reconcile logic stays intact.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
