// pkg/kordinator/dependency_kordinator.go
/*
╔═══════════════════════════════════════════════════════════════════════════════╗
║                    CRD Lifecycle Management Flow                               ║
╚═══════════════════════════════════════════════════════════════════════════════╝

The retryMissingCRDs goroutine runs forever, handling both activation and deactivation
of CRDs. This is the self‑healing core of Orkestra.

─────────────────────────────────────────────────────────────────────────────────
SCENARIO 1: CRD MISSING AT STARTUP
─────────────────────────────────────────────────────────────────────────────────

Startup:
  - CRD A is missing from the cluster
  - Kordinate loop processes A → startedCh["A"] remains open
  - Kordinate continues (does NOT block) because missing CRDs are skipped
  - Retry loop starts in background

Retry loop (every PostStartRetryInterval):
  - Phase 1: checks missing map
    - finds A is missing
    - calls utils.WaitForCRD() → false
    - remains in missing map
  - Phase 2: checks running CRDs (none)
  - Phase 3: allReady? false

Later:
  - User applies CRD A to cluster
  - Retry loop runs again
  - Phase 1: utils.WaitForCRD() → true
  - activateCRD(A) is called:
    - starts informer (entry.Informer.Run)
    - starts workers
    - closes startedCh["A"] ← UNBLOCKS dependents in Kordinate
  - Phase 3: allCRDsPresent() may still be false if other CRDs missing

Result: CRD A becomes operational without restarting Orkestra.

─────────────────────────────────────────────────────────────────────────────────
SCENARIO 2: CRD DELETED AFTER STARTUP
─────────────────────────────────────────────────────────────────────────────────

Running state:
  - CRD A is active (workers running, informer watching)
  - Retry loop runs periodically

User deletes CRD A:
  - Informer reflector starts logging "failed to list... the server could not find..."
  - Retry loop runs:
    - Phase 1: missing map (A not there)
    - Phase 2: checks running CRDs
      - calls crdExists(A) → false
      - deactivateCRD(A) is called:
        - stopCRDWorkers(A) → stops workers, drains queue
        - removes from started map
        - marks as missing in informerFactory
        - health.SetStarted(false)
        - DOES NOT close startedCh[A]
  - Phase 3: allReady becomes false

Result: CRD A becomes degraded, workers stop, but dependents continue (degraded).
No more reflector errors.

─────────────────────────────────────────────────────────────────────────────────
SCENARIO 3: CRD REAPPEARS AFTER BEING DELETED
─────────────────────────────────────────────────────────────────────────────────

After deactivation:
  - CRD A is in missing map
  - Retry loop runs:
    - Phase 1: missing map contains A
    - utils.WaitForCRD() → true
    - activateCRD(A) is called (same as scenario 1)
  - Workers restart, informer starts
  - startedCh[A] is closed again (it was never closed during deactivation,
    but we create a new channel? No — startedCh persists. Actually we don't close it
    during deactivation, so it remains open. We need to be careful: during activation,
    we should close it regardless of whether it was closed before.)

    In activateCRD, we already handle this with select/default:
        select {
        case <-ch:
            // already closed
        default:
            close(ch)
        }

Result: CRD A becomes operational again without restarting Orkestra.

─────────────────────────────────────────────────────────────────────────────────
SCENARIO 4: DEPENDENCY CHAIN WITH DELAYED ACTIVATION
─────────────────────────────────────────────────────────────────────────────────

StartupOrder: [A, B, C] where B depends on A, C depends on B

Initial state:
  - A: missing
  - B: present
  - C: present

Kordinate loop:
  - A → missing → continue (startedCh["A"] open)
  - B → waits on startedCh["A"] → BLOCKS here (main goroutine blocked)
  - C → never reached (blocked at B)

Retry loop:
  - Phase 1: missing map contains A
  - utils.WaitForCRD(A) → true
  - activateCRD(A):
    - starts workers
    - closes startedCh["A"] ← UNBLOCKS Kordinate loop

Kordinate loop continues:
  - B → startedCh["A"] closed → starts workers → closes startedCh["B"]
  - C → startedCh["B"] closed → starts workers → closes startedCh["C"]

Result: Full dependency chain resolves dynamically as CRDs appear.

─────────────────────────────────────────────────────────────────────────────────
SCENARIO 5: DEPENDENCY CHAIN WITH DELETION IN THE MIDDLE
─────────────────────────────────────────────────────────────────────────────────

Running state:
  - A, B, C all active and healthy
  - D depends on C

User deletes CRD C:
  - Retry loop detects C missing
  - deactivateCRD(C):
    - stops workers
    - marks missing
    - DOES NOT close startedCh[C] (it's already closed from startup)

Result:
  - C is degraded, workers stopped
  - D continues running (degraded) because its dependency C is not ready
  - Health endpoints show C as degraded, D as degraded

Later, C is recreated:
  - activateCRD(C):
    - starts workers
    - attempts to close startedCh[C] (already closed, safe)
  - D's health becomes healthy again automatically

─────────────────────────────────────────────────────────────────────────────────
KEY DESIGN DECISIONS
─────────────────────────────────────────────────────────────────────────────────

1. startedCh is NEVER closed during deactivation.
   Reason: Dependents should continue running (degraded) rather than block.
   If we closed the channel, dependents would think the dependency is ready
   when it's actually missing.

2. retry loop runs FOREVER, not just at startup.
   Reason: CRDs can be deleted at any time. We need continuous monitoring.

3. activateCRD closes startedCh safely using select/default.
   Reason: startedCh may already be closed from initial startup or previous activation.

4. deactivateCRD does NOT remove from startedCh map.
   Reason: The channel is still needed for future activations.
   The channel is never closed, so it remains in the map.

5. health.SetStarted(false) on deactivation.
   Reason: Allows health endpoint to show the CRD as not started.

╚═══════════════════════════════════════════════════════════════════════════════╝
*/
package kordinator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/event"
	"github.com/ialexeze/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/ialexeze/orkestra/pkg/informer"
	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/queue"
)

// DependencyKordinator extends the base Kontroller with dependency‑aware startup.
// It ensures CRDs start in topological order and shut down in reverse order.
type DependencyKordinator struct {
	*Kontroller

	depGraph       *katalog.DependencyGraph
	defaultWorkers int
	startedAt      time.Time
	queueReg       *queue.QueueRegistry
	drainTimeout   time.Duration

	// Orkestra and katalog health
	anyOnline atomic.Bool
	allOnline atomic.Bool
	orkHealth *OrkestraHealth

	// startedCh[gvk] is closed when a CRD has fully started its workers.
	startedCh map[string]chan struct{}

	// healthyCh[gvk] is closed after the CRD handles first reconciliation.
	healthyCh map[string]chan struct{}
}

// NewDependencyKordinator constructs a dependency‑aware kordinator.
// It embeds the base Kontroller wll also handling dependencies in the right order
func NewDependencyKordinator(
	kube *kubeclient.Kubeclient,
	factory *informer.Factory,
	katalog *ResourceKatalog,
	events *event.Event,
	hs domain.Health,
	queueRegistry *queue.QueueRegistry,
	defaultWorkqueue *queue.Workqueue,
	crdHealthMap map[string]*CRDHealth,
	orkHealth *OrkestraHealth,
	defaultWorkers int,
	depGraph *katalog.DependencyGraph,
	drainTimeout time.Duration,
) *DependencyKordinator {

	kord := &DependencyKordinator{
		Kontroller: NewKontroller(
			kube, factory, katalog,
			events, hs, crdHealthMap, queueRegistry,
			defaultWorkqueue, defaultWorkers,
		),
		orkHealth:      orkHealth,
		depGraph:       depGraph,
		defaultWorkers: defaultWorkers,
		queueReg:       queueRegistry,
		drainTimeout:   drainTimeout,
		startedCh:      make(map[string]chan struct{}),
		healthyCh:      make(map[string]chan struct{}),
	}

	kord.anyOnline.Store(false)
	return kord
}

// Kordinate starts CRDs in dependency order and blocks until leadership is lost.
// When leadership ends, it shuts down CRDs in reverse dependency order.
func (k *DependencyKordinator) Kordinate(ctx context.Context) {
	logger.Info().Str("component", k.Name()).Msg("starting")
	k.startedAt = time.Now()

	// Mark as ready immediately - the kordinator can serve requests
	k.hs.SetReady()
	k.orkHealth.SetOrkReady()

	// Track allOnline
	k.allOnline.Store(false)
	var totalCRDs, onlineCRDs int

	// Startup order
	startupOrder := k.depGraph.StartupOrder()
	logger.Info().Str("order", strings.Join(startupOrder, " → ")).Msg("startup order")

	totalCRDs = len(startupOrder)

	// Build name → GVK mapping
	nameToGVK := make(map[string]string)
	for _, name := range startupOrder {
		node := k.depGraph.GetNode(name)
		if node == nil {
			continue
		}
		nameToGVK[name] = node.CRD.GroupVersionKind.String()
	}

	// Create started + healthy channels for all CRDs
	for _, name := range startupOrder {
		node := k.depGraph.GetNode(name)
		if node == nil {
			continue
		}
		gvk := node.CRD.GroupVersionKind.String()
		k.startedCh[gvk] = make(chan struct{})
		k.healthyCh[gvk] = make(chan struct{})
	}

	// START RETRY LOOP ONCE, BEFORE ANY BLOCKING
	go k.retryMissingCRDs(ctx)

	// Start dependency health checker (runs until ctx is cancelled)
	go k.dependencyHealthChecker(ctx)

	// Periodically check dependency health if there are dependencies
	// Process CRDs in dependency order
	for _, name := range startupOrder {
		node := k.depGraph.GetNode(name)
		if node == nil {
			continue
		}
		crd := node.CRD
		gvk := crd.GroupVersionKind.String()

		if len(crd.DependsOn) > 0 {
			logger.Info().
				Str("crd", name).
				Str("depends_on", strings.Join(crd.GetDependencies(), ", ")).
				Msg("starting CRD")
		} else {
			logger.Info().Str("crd", name).Msg("starting CRD")
		}

		// Wait for dependencies using correct condition
		for depName, depCond := range crd.DependsOn {
			depGVK, ok := nameToGVK[depName]
			if !ok {
				logger.Error().Str("crd", name).Str("dependency", depName).Msg("dependency not found in dependency graph")
				continue
			}

			logger.Debug().Str("crd", name).Str("dependency", depName).Str("gvk", depGVK).Msg("waiting for dependency")

			switch strings.ToLower(depCond.Condition) {
			case string(types.DependencyConditionHealthy):
				select {
				case <-k.healthyCh[depGVK]:
					logger.Debug().Str("crd", name).Str("dependency", depName).Msg("dependency healthy")
				case <-ctx.Done():
					return
				}
			default: // started
				select {
				case <-k.startedCh[depGVK]:
					logger.Debug().Str("crd", name).Str("dependency", depName).Msg("dependency started")
				case <-ctx.Done():
					return
				}
			}
		}

		// Check if CRD exists
		if k.informerFactory.IsMissing(gvk) {
			logger.Debug().Str("crd", name).Str("gvk", gvk).Msg("CRD missing — workers not started, waiting for retry")
			// DO NOT close startedCh or healthyCh — dependents must block
			continue
		}

		// CRD exists — start workers
		workers := k.katalog.GetWorkers(gvk, k.defaultWorkers)
		logger.Info().Str("gvk", gvk).Int("workers", workers).Msg("starting workers")
		k.startCRDWorkers(ctx, gvk, workers)

		// Update health
		k.crdHealthMap[gvk].queueReg = k.queueReg

		// Signal dependents: STARTED ONLY
		close(k.startedCh[gvk])
		logger.Info().Str("crd", name).Str("gvk", gvk).Int("workers", workers).Msg("workers started")

		// DO NOT close healthyCh here.
		// healthyCh will be closed by the health checker when the CRD becomes healthy.

		k.anyOnline.Store(true)
		onlineCRDs++
	}

	// Mark controller started
	k.startedKtrl.Store(true)
	if k.anyOnline.Load() {
		logger.Info().Str("component", k.Name()).Int("crds_online", len(startupOrder)).Msg("started")
	} else {
		logger.Warn().Str("component", k.Name()).Msg("started — all CRDs missing, waiting for retry loop")
	}

	// Compute final katalog health
	if onlineCRDs == totalCRDs {
		k.allOnline.Store(true)
		k.orkHealth.SetKatalogReady()
	} else {
		k.allOnline.Store(false)
		k.orkHealth.SetKatalogDegraded()
	}

	// Block until leadership lost
	<-ctx.Done()
	logger.Info().Msg("leadership lost — beginning dependency-aware shutdown")
	k.hs.Unhealthy()

	// Shut down CRDs
	shutdownOrder := k.depGraph.ShutdownOrder()
	logger.Info().Str("order", strings.Join(shutdownOrder, " → ")).Msg("shutdown order")
	for _, name := range shutdownOrder {
		logger.Info().Str("crd", name).Msg("shutting down CRD")
		gvk := k.depGraph.GetNode(name).CRD.GroupVersionKind.String()
		k.stopCRDWorkers(gvk)
	}

	logger.Info().Str("component", k.Name()).Msg("drained and stopped")
}

// startCRDWorkers starts a worker pool for a specific CRD and is invoked in dependency order.
func (k *DependencyKordinator) startCRDWorkers(ctx context.Context, gvk string, workers int) {
	entry, ok := k.katalog.Get(gvk)
	if !ok {
		logger.Fatal().Str("gvk", gvk).Msg("no katalog entry found")
		return
	}

	crdCtx, cancel := context.WithCancel(ctx)
	rec := entry.ReconcilerFactory()

	k.mu.Lock()
	k.reconcilers[gvk] = rec
	k.cancelFuncs[gvk] = cancel
	wg := &sync.WaitGroup{}
	k.wgs[gvk] = wg
	k.crdHealthMap[gvk].SetStarted()

	k.crdHealthMap[gvk].SetTotalWorkers(int32(workers))
	k.crdHealthMap[gvk].gvk = gvk // Set GVK for metrics
	k.started[gvk] = true
	k.total[gvk]++
	k.mu.Unlock()

	// Initialize all workers as idle (not processing)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		workerID := fmt.Sprintf("%s-worker-%d", gvk, i)
		workerID = strings.ReplaceAll(workerID, ",", "")
		workerID = strings.ReplaceAll(workerID, " ", "-")

		// Mark as idle initially (not processing)
		k.crdHealthMap[gvk].workerStates.Store(workerID, WorkerStateIdle)

		go func(id string) {
			defer wg.Done()
			k.runWorkerForGVK(crdCtx, gvk, id)
		}(workerID)
	}
}

// stopCRDWorkers cancels the CRD context and waits for all workers to drain.
func (k *DependencyKordinator) stopCRDWorkers(gvk string) {
	k.mu.RLock()
	cancel, okCancel := k.cancelFuncs[gvk]
	wg, okWG := k.wgs[gvk]
	k.mu.RUnlock()

	// Step 1: signal workers to stop accepting new work
	if okCancel {
		cancel()
	}

	// Step 2: shut down the queue — this unblocks any worker
	// blocked on queue.GetWithContext() waiting for the next item.
	// Without this, workers that finished their reconcile and
	// are waiting for work will never exit.
	if wq, ok := k.queueReg.For(gvk); ok {
		wq.Queue.ShutDown()
	}

	if !okWG {
		return
	}

	// Reset worker counts after shutdown
	if health, ok := k.crdHealthMap[gvk]; ok {
		health.ResetWorkerCounts()
		health.workerStates.Range(func(key, value interface{}) bool {
			health.workerStates.Store(key, WorkerStateStopped)
			return true
		})
	}

	// Step 4: wait for workers to drain — with a timeout.
	// The timeout is the safety net for stuck reconciles, not
	// an execution budget for normal shutdown.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info().Str("gvk", gvk).Msg("workers drained cleanly")
	case <-time.After(k.drainTimeout):
		logger.Warn().Str("gvk", gvk).
			Dur("timeout", k.drainTimeout).
			Msg("drain timeout exceeded — workers may still be running. " +
				"Consider increasing SHUTDOWN_TIMEOUT if reconciles call slow external APIs.")
	}
}

// Name returns the name of the dependency kordinator
func (k *DependencyKordinator) Name() string {
	return "orkestra dependency kordinator"
}

// NameToCRD returns the CRD for a given name
func (k *DependencyKordinator) NameToCRD(name string) types.CRDEntry {
	return k.depGraph.GetNode(name).CRD
}

// NameToGVK returns the GVK fr a giben name
func (k *DependencyKordinator) NameToGVK(name string) schema.GroupVersionKind {
	return k.depGraph.GetNode(name).CRD.GroupVersionKind
}

// GVKToCRD returns the CRD entry for a given gvk
func (k *DependencyKordinator) GVKToCRD(gvk schema.GroupVersionKind) types.CRDEntry {
	entry, ok := k.katalog.Get(gvk.String())
	if !ok {
		return types.CRDEntry{}
	}
	return entry.CRD
}

// NameToGVKMap returns a map of names to gvk string
func (k *DependencyKordinator) NameToGVKMap() map[string]string {
	nameToGVK := make(map[string]string)
	for _, name := range k.depGraph.StartupOrder() {
		node := k.depGraph.GetNode(name)
		if node != nil {
			nameToGVK[name] = node.CRD.GroupVersionKind.String()
		}
	}
	return nameToGVK
}
