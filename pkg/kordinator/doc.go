// Package kordinator is the orchestration heart of every Orkestra operator.
//
// It sits between the informer factory and the reconcilers: it decides when
// each CRD's workers start, in what order, under what conditions, and how they
// recover when the cluster changes beneath them.
//
// # Architecture
//
// The package is built in two layers.
//
// The base layer is Kontroller — the worker manager. It holds the reconciler
// instances, the per-CRD context cancel functions, the WaitGroups, and the
// queue references. It knows how to start and stop workers for a GVK; it does
// not know anything about dependencies.
//
// The coordination layer is DependencyKordinator, which embeds Kontroller and
// adds the dependency graph, the channel-based readiness signals, the
// self-healing retry loop, and the runtime health checker. It is the only
// component that calls Kordinate() — the method that blocks for the operator's
// lifetime and owns the startup → run → shutdown sequence.
//
// # ResourceKatalog
//
// ResourceKatalog is the in-memory registry written once during
// konstructOrkestra and read many times thereafter. Every GVK maps to a
// RegistryEntry that holds the informer, the reconciler factory closure, and
// the CRD configuration. Workers are created by calling
// entry.ReconcilerFactory() — the closure captures the provider registry,
// kube client, and provider stats so they never pass through the kordinator's
// own API.
//
// # CRD lifecycle
//
// The DependencyKordinator drives every CRD through the following states:
//
//	pending  → informer created, workers not yet started
//	started  → workers running, no successful reconcile yet
//	healthy  → at least one successful reconcile completed
//	degraded → consecutive failures exceeded threshold, or CRD missing
//
// The transitions are recorded in CRDHealth using atomic operations — no locks
// in the hot path. State changes from workers (MarkWorkerProcessing,
// MarkWorkerIdle, RecordSuccess, RecordFailure) and from the health checker
// (SetDegraded, SetStarted) are all safe for concurrent callers.
//
// # Dependency channels
//
// CRDs may declare dependencies with two possible conditions:
//
//	dependsOn:
//	  other-crd:
//	    condition: started   # default — workers are running
//	  another-crd:
//	    condition: healthy   # first successful reconcile has completed
//
// Each CRD gets two channels at startup:
//
//	startedCh[gvk]  — closed when startCRDWorkers returns
//	healthyCh[gvk]  — closed by dependencyHealthChecker on first healthy state
//
// dependenciesReady() checks these channels non-blocking using select/default.
// It returns false immediately if any channel is still open — it never blocks.
//
// This is intentional. A dependency with condition: healthy may take minutes
// to satisfy. Blocking the startup loop on that condition would starve every
// CRD that appears later in the topological order, regardless of whether those
// CRDs depend on the slow one. The fix: skip and defer.
//
// # Non-blocking startup and deferred activation
//
// Kordinate() iterates the topological order exactly once. For each CRD:
//   - if dependenciesReady() returns false → skip, the retry loop handles it
//   - if the CRD is missing from the cluster → skip, the retry loop handles it
//   - otherwise → startCRDWorkers, close startedCh[gvk]
//
// The background retryMissingCRDs goroutine runs for the operator's lifetime.
// On each tick it executes four phases in order:
//
//	Phase 1 — Detect runtime disappearances: polls every registered GVK.
//	           When a CRD vanishes, deactivateCRD drains the workers,
//	           marks the CRD degraded, and propagates degradation to any
//	           dependent that requires condition: healthy.
//
//	Phase 2 — Re-activate missing CRDs: for every GVK in the missing map,
//	           checks whether the CRD has appeared in the cluster. If yes,
//	           activateCRD restarts the informer, launches workers, and
//	           closes startedCh[gvk] to unblock waiting dependents.
//
//	Phase 3 — Deferred activation: iterates the topological order looking
//	           for CRDs that were skipped at startup. When dependenciesReady()
//	           finally returns true, activateCRD is called and the CRD joins
//	           the running set. This is what handles condition: healthy.
//
//	Phase 4 — Aggregate health: if all CRDs are started and none are missing,
//	           sets katalog health to ready.
//
// # Self-healing invariants
//
// startedCh is never closed during deactivation. This is load-bearing: if it
// were closed, dependents would believe the deactivated CRD is still ready and
// start processing against a resource that no longer exists. Keeping it open
// means dependents stay in their current state (degraded) and resume normally
// when activateCRD closes the channel again on re-activation.
//
// activateCRD closes startedCh with a select/default guard:
//
//	select {
//	case <-ch: // already closed from a previous activation — no-op
//	default:   close(ch)
//	}
//
// Without this guard, re-activating a CRD that was previously started would
// panic with "close of closed channel".
//
// # Runtime health checker
//
// dependencyHealthChecker runs on a ticker independent of the retry loop. It
// evaluates the health of every active CRD's dependencies and updates the
// DependencyStatus map inside CRDHealth — this feeds the Control Center and
// the /katalog/{crd} endpoint. It also closes healthyCh[gvk] exactly once
// when a CRD transitions to healthy for the first time, which unblocks any
// deferred CRDs waiting for condition: healthy.
//
// # Worker loop
//
// runWorkerForGVK (in worker.go) is the inner loop for every worker goroutine:
//
//	for {
//	    select {
//	    case <-ctx.Done(): return
//	    default:
//	        item, shutdown := wq.Queue.Get()
//	        if shutdown { return }
//	        health.MarkWorkerProcessing(workerID)
//	        func() {
//	            defer wq.Queue.Done(item)
//	            k.processItemForGVK(ctx, gvk, item)
//	        }()
//	        health.MarkWorkerIdle(workerID)
//	    }
//	}
//
// wq.Queue.Done(item) is always called — deferred inside the closure — whether
// the reconcile succeeds or fails. Re-queuing with rate-limit backoff is
// handled inside processItemForGVK on error.
//
// stopCRDWorkers cancels the CRD context and then calls wq.Queue.ShutDown()
// before waiting on the WaitGroup. The shutdown call is required: a worker
// blocking on wq.Queue.Get() waiting for its next item will never observe the
// context cancellation until it is unblocked by the queue shutdown.
//
// # HTTP surface
//
// All three runtime introspection handlers are in crd_health_handers.go.
//
//	BuildCRDHealthHandler  — /katalog/{crd}/health
//	BuildCRDInfoHandler    — /katalog/{crd}
//	BuildKatalogHandler    — /katalog
//
// BuildCRDInfoHandler assembles a full CRD detail response on every request
// from live atomic reads — no caching. When the CRD declares provider blocks,
// it merges static ProviderBlocks metadata (declared kinds) with runtime
// ProviderStats (total calls, errors, error rate) into a providers array.
//
// Resource count is len(inf.GetStore().List()) — the informer's local cache,
// updated in real time by the informer watch loop without any API calls.
package kordinator
