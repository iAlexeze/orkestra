# 03 — Startup and dependency channels

`DependencyKordinator.Kordinate()` is the method that blocks for the operator's lifetime. It owns the full startup → run → shutdown sequence.

## The dependency graph

Before `Kordinate()` runs, `konstructRuntime` builds a `DependencyGraph` from the `dependsOn` fields in the Katalog. Kahn's algorithm produces a topological order — alphabetical tie-breaking within the same depth tier makes the order deterministic across restarts.

```yaml
spec:
  crds:
    website:
      dependsOn: {}           # no dependencies — starts first

    route-record:
      dependsOn:
        website:
          condition: started  # starts after website workers are running

    ssl-cert:
      dependsOn:
        website:
          condition: healthy  # starts after website has reconciled at least one CR
```

A cycle in the graph is a fatal error at startup.

## Dependency channels

Every CRD gets two channels created at the beginning of `Kordinate()`:

```go
startedCh[gvk]  // closed when startCRDWorkers returns for this GVK
healthyCh[gvk]  // closed by dependencyHealthChecker on first healthy state
```

These channels are the primitive on which all dependency gating rests. A closed channel means the condition is satisfied. An open channel means it is not.

`dependenciesReady()` checks both channel types with `select/default` — non-blocking:

```go
func (k *DependencyKordinator) dependenciesReady(crd types.CRDEntry, nameToGVK map[string]string) bool {
    for depName, depCond := range crd.DependsOn {
        depGVK := nameToGVK[depName]
        switch strings.ToLower(depCond.Condition) {
        case "healthy":
            select {
            case <-k.healthyCh[depGVK]: // closed → satisfied
            default:
                return false            // still open → not ready
            }
        default: // started
            select {
            case <-k.startedCh[depGVK]: // closed → satisfied
            default:
                return false
            }
        }
    }
    return true
}
```

If any channel is open, the function returns `false` immediately. It never sleeps or retries inside the call.

## The startup loop

```go
for _, name := range startupOrder {
    crd := node.CRD
    gvk := crd.GroupVersionKind.String()

    if !k.dependenciesReady(crd, nameToGVK) {
        logger.Info().Str("crd", name).Msg("dependencies not ready — deferring activation")
        continue  // ← skip, do not block
    }

    if k.informerFactory.IsMissing(gvk) {
        logger.Debug().Str("crd", name).Msg("CRD missing from cluster — deferring")
        continue  // ← skip, retry loop will handle it
    }

    k.startCRDWorkers(ctx, gvk, workers)
    close(k.startedCh[gvk])  // ← unblocks dependents with condition: started
}
```

## What startCRDWorkers does

`startCRDWorkers` is called with the baseline worker count. It sets up the full
runtime for one CRD:

```
1. rec := entry.ReconcilerFactory()
   — creates the reconciler (GenericReconciler or custom)

2. Inject the per-CRD queue (QueueInjector interface)
   — so SetQueueDepthLimit and the resync goroutine can reach the right queue

3. [if autoscale: declared]
   a. Inject spawnWorker callback (workerSpawner interface)
      — ResizeWorkers calls this to start new goroutines on scale-up
   b. Register AutoMetrics with GlobalCrossMetricsRegistry under entry.CRD.Name
      — enables cross.<crd>.metrics.* conditions from other operatorboxes
   c. Set workerInfoFn on CRDHealth (workerInfoProvider interface)
      — /katalog/{crd} calls this on every request for a live WorkerInfo snapshot
   d. go runner.RunAutoscaler(crdCtx)  (autoscalerRunner interface)
      — evaluation loop: conditions → apply/restore override
   e. go rl.StartResyncLoop(crdCtx)    (resyncLoopStarter interface)
      — re-enqueue goroutine: idles at 0 interval, fires at do.resync when active

4. [if rollback: declared]
   Inject rollback notifier callbacks (rollbackNotifierSetter interface)
   — onTrigger: increments rollbackTotal, sets rollbackActive in CRDHealth
   — onClear:   clears rollbackActive in CRDHealth

5. Start baseline.workers goroutines
   — each runs runWorkerForGVK until crdCtx is cancelled or queue shutdown
   — additional goroutines are started lazily by ResizeWorkers on scale-up
```

All interface checks are duck-typed against the reconciler using local interface
definitions in kordinator to avoid an import cycle — the `reconciler` package
already imports `kordinator`. The interfaces are: `QueueInjector`,
`workerSpawner`, `autoMetricsExporter`, `workerInfoProvider`,
`autoscalerRunner`, `resyncLoopStarter`, `rollbackNotifierSetter`.

The loop iterates the topological order exactly once. It skips any CRD whose dependencies are not yet satisfied — it does not block, sleep, or retry. The background retry loop (see [04 — Self-healing](04-self-healing.md)) handles activation of deferred CRDs.

## Why the loop must never block

Consider this graph:

```
A (no deps)
B depends on A:started
C depends on A:healthy    ← may take minutes
D depends on A:started
```

Topological order within the second tier is alphabetical: B, C, D.

If the loop blocked on C waiting for A to become healthy, D would never start — even though D only requires A:started, which is already satisfied. Every CRD alphabetically after C would be frozen.

The non-blocking design means C is skipped, D starts immediately, and the retry loop activates C as soon as the `dependencyHealthChecker` closes `healthyCh[A]`.

## After the loop

Once the loop completes:

```go
if onlineCRDs == totalCRDs {
    k.orkHealth.SetKatalogReady()
} else {
    k.orkHealth.SetKatalogDegraded()
}

<-ctx.Done()  // blocks until leadership is lost
```

Any CRD that did not start in the initial loop is handled by the retry loop while the operator runs. When the context is cancelled, shutdown begins.

## Shutdown

```go
shutdownOrder := k.depGraph.ShutdownOrder()  // reverse of topological order
for _, name := range shutdownOrder {
    gvk := k.depGraph.GetNode(name).CRD.GroupVersionKind.String()
    k.stopCRDWorkers(gvk)
}
```

Shutdown reverses the startup order so dependents drain before the CRDs they depend on. `stopCRDWorkers` cancels the CRD context, shuts down the queue, and waits for workers to exit (see [05 — Workers and drain](05-workers.md)).

---

**Next →** [04 — Self-healing](04-self-healing.md)
