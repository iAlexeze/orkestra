// pkg/kontroller/dependency_kontroller.go
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
  - RunOrDie loop processes A → readyCh["A"] remains open
  - RunOrDie continues (does NOT block) because missing CRDs are skipped
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
    - closes readyCh["A"] ← UNBLOCKS dependents in RunOrDie
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
        - DOES NOT close readyCh[A]
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
  - readyCh[A] is closed again (it was never closed during deactivation,
    but we create a new channel? No — readyCh persists. Actually we don't close it
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

RunOrDie loop:
  - A → missing → continue (readyCh["A"] open)
  - B → waits on readyCh["A"] → BLOCKS here (main goroutine blocked)
  - C → never reached (blocked at B)

Retry loop:
  - Phase 1: missing map contains A
  - utils.WaitForCRD(A) → true
  - activateCRD(A):
    - starts workers
    - closes readyCh["A"] ← UNBLOCKS RunOrDie loop

RunOrDie loop continues:
  - B → readyCh["A"] closed → starts workers → closes readyCh["B"]
  - C → readyCh["B"] closed → starts workers → closes readyCh["C"]

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
    - DOES NOT close readyCh[C] (it's already closed from startup)

Result:
  - C is degraded, workers stopped
  - D continues running (degraded) because its dependency C is not ready
  - Health endpoints show C as degraded, D as degraded

Later, C is recreated:
  - activateCRD(C):
    - starts workers
    - attempts to close readyCh[C] (already closed, safe)
  - D's health becomes healthy again automatically

─────────────────────────────────────────────────────────────────────────────────
KEY DESIGN DECISIONS
─────────────────────────────────────────────────────────────────────────────────

1. readyCh is NEVER closed during deactivation.
   Reason: Dependents should continue running (degraded) rather than block.
   If we closed the channel, dependents would think the dependency is ready
   when it's actually missing.

2. retry loop runs FOREVER, not just at startup.
   Reason: CRDs can be deleted at any time. We need continuous monitoring.

3. activateCRD closes readyCh safely using select/default.
   Reason: readyCh may already be closed from initial startup or previous activation.

4. deactivateCRD does NOT remove from readyCh map.
   Reason: The channel is still needed for future activations.
   The channel is never closed, so it remains in the map.

5. health.SetStarted(false) on deactivation.
   Reason: Allows health endpoint to show the CRD as not started.

╚═══════════════════════════════════════════════════════════════════════════════╝
*/
package kontroller

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/event"

	"github.com/ialexeze/orkestra/pkg/informer"
	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/metrics"
	"github.com/ialexeze/orkestra/pkg/queue"
)

// DependencyKontroller extends the base Kontroller with dependency‑aware startup.
// It ensures CRDs start in topological order and shut down in reverse order.
type DependencyKontroller struct {
	*Kontroller

	depGraph       *katalog.DependencyGraph
	defaultWorkers int
	startedAt      time.Time
	queueReg       *queue.QueueRegistry

	// readyCh[gvk] is closed when a CRD has fully started its workers.
	readyCh map[string]chan struct{}
}

// NewDependencyKontroller constructs a dependency‑aware Kontroller.
// It embeds the base Kontroller so all worker logic, queue handling,
// and reconciler dispatching remain unchanged.
func NewDependencyKontroller(
	kube *kubeclient.Kubeclient,
	factory *informer.Factory,
	katalog *ResourceKatalog,
	events *event.Event,
	hs domain.Health,
	queueRegistry *queue.QueueRegistry,
	defaultWorkqueue *queue.Workqueue,
	crdHealthMap map[string]*CRDHealth,
	defaultWorkers int,
	depGraph *katalog.DependencyGraph,
) *DependencyKontroller {

	return &DependencyKontroller{
		Kontroller:     NewKontroller(kube, factory, katalog, events, hs, crdHealthMap, queueRegistry, defaultWorkqueue, defaultWorkers),
		depGraph:       depGraph,
		defaultWorkers: defaultWorkers,
		queueReg:       queueRegistry,
		readyCh:        make(map[string]chan struct{}),
	}
}

// RunOrDie starts CRDs in dependency order and blocks until leadership is lost.
// When leadership ends, it shuts down CRDs in reverse dependency order.
func (k *DependencyKontroller) RunOrDie(ctx context.Context) {
	logger.Info().Msgf("%s is starting...", k.Name())
	k.startedAt = time.Now()

	startupOrder := k.depGraph.StartupOrder()
	logger.Info().Msgf("startupOrder: %v", strings.Join(startupOrder, " → "))

	// Build name → GVK mapping
	nameToGVK := make(map[string]string)
	for _, name := range startupOrder {
		node := k.depGraph.GetNode(name)
		if node == nil {
			continue
		}
		nameToGVK[name] = node.CRD.GroupVersionKind.String()
	}

	// Create ready channels for all CRDs
	for _, name := range startupOrder {
		node := k.depGraph.GetNode(name)
		if node == nil {
			continue
		}
		gvk := node.CRD.GroupVersionKind.String()
		k.readyCh[gvk] = make(chan struct{})
	}

	// START RETRY LOOP ONCE, BEFORE ANY BLOCKING
	go k.retryMissingCRDs(ctx)

	anyOnline := false

	// Process CRDs in dependency order
	for _, name := range startupOrder {
		node := k.depGraph.GetNode(name)
		if node == nil {
			continue
		}
		crd := node.CRD
		gvk := crd.GroupVersionKind.String()

		if len(crd.DependsOn) > 0 {
			logger.Info().Msgf("starting %s (depends on: %s)", name, strings.Join(crd.GetDependencies(), ", "))
		} else {
			logger.Info().Msgf("starting %s", name)
		}

		// Wait for dependencies
		for _, depName := range crd.DependsOn {
			depGVK, ok := nameToGVK[depName]
			if !ok {
				logger.Error().Msgf("%s depends on %s, but %s not found in dependency graph", name, depName, depName)
				continue
			}

			logger.Debug().Msgf("%s waiting for dependency %q (%s)", name, depName, depGVK)
			select {
			case <-k.readyCh[depGVK]:
				logger.Debug().Msgf("%s: dependency %q ready", name, depName)
			case <-ctx.Done():
				return
			}
		}

		// Check if CRD exists
		if k.informerFactory.IsMissing(gvk) {
			logger.Debug().Msgf("%s is missing — readyCh remains open, workers not started", name)
			// DO NOT close readyCh — dependents will block until this CRD appears
			continue
		}

		// CRD exists — start workers
		workers := k.katalog.GetWorkers(gvk, k.defaultWorkers)
		logger.Info().Msgf("starting %d workers for %s", workers, gvk)
		k.startCRDWorkers(ctx, gvk, workers)

		// Update health and metrics
		k.crdHealthMap[gvk].SetWorkersActive(workers)
		k.crdHealthMap[gvk].queueReg = k.queueReg
		metrics.WorkersActive.WithLabelValues(gvk).Set(float64(workers))

		// Signal dependents
		close(k.readyCh[gvk])
		logger.Info().Msgf("%s workers started and ready", name)

		anyOnline = true
	}

	// Mark controller started
	k.startedKtrl.Store(true)
	if anyOnline {
		k.hs.SetReady()
		logger.Info().Msgf("%s started — %d CRD(s) online", k.Name(), len(startupOrder))
	} else {
		logger.Warn().Msgf("%s started — all CRDs missing, waiting for retry loop", k.Name())
	}

	// Block until leadership lost
	<-ctx.Done()
	logger.Info().Msg("leadership lost — beginning dependency-aware shutdown")
	k.hs.Unhealthy()

	shutdownOrder := k.depGraph.ShutdownOrder()
	logger.Info().Msgf("shutdownOrder: %v", strings.Join(shutdownOrder, " → "))
	for _, name := range shutdownOrder {
		logger.Info().Msgf("shutting down %s", name)
		gvk := k.depGraph.GetNode(name).CRD.GroupVersionKind.String()
		k.stopCRDWorkers(gvk)
	}

	logger.Info().Msgf("%s drained and stopped", k.Name())
}

// startCRDWorkers starts a worker pool for a specific CRD and is invoked in dependency order.
func (k *DependencyKontroller) startCRDWorkers(ctx context.Context, gvk string, workers int) {
	// Build reconciler once here — kube and ev are started by now
	entry, ok := k.katalog.Get(gvk)
	if !ok {
		logger.Fatal().Str("gvk", gvk).Msg("no katalog entry found")
		return
	}

	// CRD context
	crdCtx, cancel := context.WithCancel(ctx)

	rec := entry.ReconcilerFactory() // ← initialize reconciler factory for all CRDs once

	k.mu.Lock()

	k.reconcilers[gvk] = rec // ← reconciler stored here

	// cancel func for each
	k.cancelFuncs[gvk] = cancel

	// wait group for each
	wg := &sync.WaitGroup{}
	k.wgs[gvk] = wg

	// compute started for each
	k.crdHealthMap[gvk].SetStarted() // ← health map
	k.started[gvk] = true            // ← state map
	k.total[gvk]++

	k.mu.Unlock()

	for i := 0; i < workers; i++ {
		wg.Add(1) // ← one crd at a time
		go func(workerID string) {
			defer wg.Done()
			k.runWorkerForGVK(crdCtx, gvk, workerID)
		}(uuid.New().String()) // ← for tracing
	}
}

// stopCRDWorkers cancels the CRD context and waits for all workers to drain.
func (k *DependencyKontroller) stopCRDWorkers(gvk string) {
	k.mu.RLock()
	cancel, okCancel := k.cancelFuncs[gvk]
	wg, okWG := k.wgs[gvk]
	k.mu.RUnlock()

	// logger.Debug().Msgf("stopCRDWorkers: looking for %s", gvk)
	// logger.Debug().Msgf("  okCancel=%v, okWG=%v", okCancel, okWG)

	if okCancel {
		logger.Info().Msgf("cancelling workers for %s", gvk)
		cancel()
	} else {
		logger.Warn().Msgf("no cancel function found for %s", gvk)
	}

	if okWG {
		logger.Info().Msgf("waiting for workers for %s to drain", gvk)
		wg.Wait()
		logger.Info().Msgf("workers for %s drained", gvk)
	} else {
		logger.Warn().Msgf("no wait group found for %s", gvk)
	}

	logger.Info().Msgf("workers for %s stopped", gvk)
}

// Name returns the name of the dependency kontroller
func (k *DependencyKontroller) Name() string {
	return "orkestra dependency kontroller"
}
