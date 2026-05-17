# 04 — How it works

`simulate.Run` wires the production `GenericReconciler` to a fake Kubernetes cluster backed by an in-memory object tracker. No network calls are made.

## Fake cluster

`FakeKubeclient` wraps `k8s.io/client-go/kubernetes/fake` and `k8s.io/client-go/dynamic/fake`. A `PrependReactor` intercepts every clientset operation before the default object-tracker reactor handles it, recording verb + resource + name as an `Op`.

`PatchStatus`, `PatchLabels`, `PatchAnnotations`, and `PatchFinalizers` are custom methods that record ops directly and persist the mutation to the in-memory object so the reconciler's idempotency guards see the update on the next cycle.

## CR seeding

The CR is pre-populated with Orkestra's managed labels and annotations (`orklabels.Managed`, `AnnotationManagedBy`, `AnnotationManagedSince`) before being added to the indexer. Without this, the reconciler would patch those fields every cycle, adding noise to every cycle's op list.

## Deployment readiness

After a Deployment is created, `MarkDeploymentReady` advances its status (`AvailableReplicas`, `ReadyReplicas`, `Replicas`) so state machines that wait for `AvailableReplicas > 0` can progress on the next cycle.

## Steady state

Detected when two consecutive cycles produce an identical `verb/resource/name` sequence. `Result.Steady` is set and `Result.SteadyAt` records the cycle number. The simulation always runs all requested cycles — it does not break early.

## Result structure

```go
type Result struct {
    Cycles   []CycleResult
    Steady   bool
    SteadyAt int
}

type CycleResult struct {
    Cycle int
    Ops   []Op
    Error error
}

type Op struct {
    Cycle     int
    Verb      string // "create", "update", "delete", "get", "patch"
    Resource  string // "deployments", "services", etc.
    Namespace string
    Name      string
    At        time.Time
}
```
