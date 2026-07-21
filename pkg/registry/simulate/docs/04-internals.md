# 04 — How it works

`simulate.Run` wires the production reconciler to a fake Kubernetes cluster backed by an in-memory object tracker. No network calls are made (unless `--skip-external` is not set and `external:` blocks hit real endpoints).

## Fake cluster

`FakeKubeclient` wraps `k8s.io/client-go/kubernetes/fake` and `k8s.io/client-go/dynamic/fake`. A `PrependReactor` intercepts every typed clientset operation before the default object-tracker reactor handles it, recording verb + resource + name as an `Op`.

`PatchStatus`, `PatchLabels`, `PatchAnnotations`, and `PatchFinalizers` are custom methods that record ops directly and persist the mutation to the in-memory object so the reconciler's idempotency guards see the update on the next cycle.

## CR seeding and typed objects

The CR file is parsed as multi-document YAML. Each document is matched to a CRD by `kind`. For typed CRDs (those with `apiTypes.location` set and `ObjectRegistry` populated), the CR is converted from `*unstructured.Unstructured` to the concrete Go type via JSON round-trip before being added to the indexer. This allows constructor type-assertions (`raw.(*MyType)`) and hook `BindToObjectHooks` closures to succeed.

The CR is pre-populated with Orkestra's managed labels and annotations before indexer insertion so the reconciler's idempotency guards skip those patches on every cycle.

## Cross-CRD observation

All CRs from the multi-doc file that do NOT belong to the CRD being simulated are seeded as peers. `simulate.Run` builds a `*kordinator.ResourceKatalog` with one fake `SharedIndexInformer` per peer CRD and passes it as `katalogRegistry` to `NewGenericReconciler`. When the reconciler hits a `cross:` declaration, `readCross()` looks up the peer informer by CRD name and reads from its indexer — zero API calls, same as production.

## Hook and constructor wiring

When the custom operator binary is used (`make registry && make build`):
- `HookRegistry[gvk]` is populated → hook binder is passed to `NewGenericReconciler` → hook fires each cycle
- `ReconcilerRegistry[gvk]` is populated → constructor is called with `fakeKube`, `informer`, and `event.Discard()` → constructor reconcile loop runs against the fake cluster

The event recorder passed to constructors is `event.Discard()` — a silent no-op. All `Eventf` calls are discarded without panicking; no nil guards are needed in constructor code.

## Deployment readiness

After a Deployment is created in a cycle, `MarkDeploymentReady` advances its status (`AvailableReplicas`, `ReadyReplicas`, `Replicas`) so state machines that wait on replica counts can progress on the next cycle.

## Steady state

Detected when two consecutive cycles produce an identical `verb/resource/name` sequence. `Result.Steady` is set and `Result.SteadyAt` records the cycle number. The simulation always runs all requested cycles — it does not break early.

## Result structure

```go
type RunOptions struct {
    SkipExternal bool                                  // stub external: HTTP calls
    Peers        map[string]*unstructured.Unstructured // sibling CRs for cross: observation (keyed by lowercase kind)
}

type Result struct {
    Cycles   []CycleResult
    Steady   bool
    SteadyAt int
    Notes    []string // informational messages (inactive blocks, etc.)
    AllOps   []Op     // every op across all cycles, for --debug-ops
}

type CycleResult struct {
    Cycle int
    Ops   []Op
    Error error
}

type Op struct {
    Cycle     int
    Verb      string // "create", "update", "delete", "get", "patch"
    Resource  string // "deployments", "statefulsets", "jobs", "status", etc.
    Namespace string
    Name      string
    At        time.Time
}
```
