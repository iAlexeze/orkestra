// pkg/kontroller/dependency_kontroller.go
package kontroller

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/event"

	// "github.com/ialexeze/orkestra/pkg/health"
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

	// readyCh[name] is closed when a CRD has fully started its workers.
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
		readyCh:        make(map[string]chan struct{}),
	}
}

// RunOrDie starts CRDs in dependency order and blocks until leadership is lost.
// When leadership ends, it shuts down CRDs in reverse dependency order.
func (k *DependencyKontroller) RunOrDie(ctx context.Context) {
	logger.Info().Msgf("%s starting in %s mode...", k.Name(), k.depGraph.GetMode())

	k.startedAt = time.Now()

	// ── 1. Compute topological startup order (A → B → C) ─────────────────
	// This ensures dependencies start before dependents.
	startupOrder := k.depGraph.StartupOrder()

	// ── 2. Create a "ready" channel for each CRD ────────────────────────
	// These channels signal when a CRD has fully started its workers.
	// Dependents block on these channels until they're closed.
	for _, name := range startupOrder {
		k.readyCh[name] = make(chan struct{})
	}

	// ── 3. Start CRDs in dependency order ───────────────────────────────
	for _, name := range startupOrder {
		node := k.depGraph.GetNode(name)
		crd := node.CRD
		gvk := crd.GroupVersionKind.String()

		// ── 3a. Wait for all dependencies to signal readiness ───────────
		// For each dependency, block on its ready channel.
		// If a dependency is missing, its channel never closes → we block forever.
		// This is intentional – the retry loop will close it when the CRD appears.
		for _, dep := range crd.DependsOn {
			select {
			case <-k.readyCh[dep]:
				logger.Debug().Msgf("%s dependency %s is ready", name, dep)
			case <-ctx.Done():
				return
			}
		}

		// Log startup intent
		if len(crd.DependsOn) > 0 {
			logger.Info().Msgf("starting %s (depends on: %s)...", name, strings.Join(crd.DependsOn, ", "))
		} else {
			logger.Info().Msgf("starting %s...", name)
		}

		// ── 3b. Start workers for this CRD if available ─────────────────
		// If the CRD exists in the cluster, start its worker pool.
		// If it's missing, we don't start workers – the retry loop will handle it.
		if !k.informerFactory.IsMissing(gvk) {
			workers := k.katalog.GetWorkers(gvk, k.defaultWorkers)

			logger.Info().Msgf("starting %d workers for %s", workers, gvk)

			k.startCRDWorkers(ctx, gvk, workers)

			// ── 3c. Send metrics to prometheus for this CRD ─────────────
			metrics.WorkersActive.WithLabelValues(gvk).Set(float64(workers))

			// ── 3d. Signal that this CRD is ready ───────────────────────
			// Closing the channel unblocks any dependents waiting on this CRD.
			close(k.readyCh[name])
			logger.Info().Msgf("%s workers started", name)

			// ── 4. Mark controller as ready ─────────────────────────────────────
			// At this point, all CRDs that existed at startup have started workers.
			// Missing CRDs are being retried in the background.
			k.startedKtrl.Store(true)
			k.hs.SetReady()
			logger.Info().Msgf("%s started – retry loop active for missing CRDs", k.Name())
		} else {
			// ── 5a. CRD is missing – log and move on ────────────────────
			k.startedKtrl.Store(true)
			k.hs.SetReady()
			logger.Info().Msgf("%s started – retry loop active for missing CRDs", k.Name())
			// Workers are NOT started. The retry loop will activate this CRD
			// later when it appears in the cluster.
			logger.Warn().Msgf("CRD %s is missing – workers not started (will retry)", gvk)

			// DO NOT close readyCh[name] here – that would falsely signal readiness.
			// The retry loop will close it when the CRD actually appears.

			// ── 5b. Start background retry loop for missing CRDs ─────────────────
			// This runs once, regardless of whether any CRDs were missing.
			// It will keep running until context cancellation, handling:
			//   - CRDs that were missing at startup
			//   - CRDs that get deleted and recreated later
			//   - New CRDs added after Orkestra started
			go k.retryMissingCRDs(ctx)
		}
	}

	// ── 6. Block until leadership is lost ───────────────────────────────
	// This goroutine stays here until leader election is lost or SIGTERM.
	<-ctx.Done()
	logger.Info().Msg("leadership lost — beginning dependency‑aware shutdown")

	// Mark as unhealthy – stop routing traffic to this instance
	k.hs.Unhealthy()

	// ── 7. Shutdown CRDs in reverse dependency order ────────────────────
	// Reverse of startup order ensures dependents shut down before their dependencies.
	shutdownOrder := k.depGraph.ShutdownOrder()
	for _, name := range shutdownOrder {
		logger.Info().Msgf("shutting down %s", name)
		k.stopCRDWorkers(name)
	}

	logger.Info().Msgf("%s drained and stopped", k.Name())
}

// startCRDWorkers starts a worker pool for a specific CRD.
// It mirrors the logic in Kontroller.RunOrDie but is invoked in dependency order.
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
func (k *DependencyKontroller) stopCRDWorkers(name string) {
	gvk := k.depGraph.GetNode(name).CRD.GroupVersionKind.String()

	k.mu.RLock()
	cancel, okCancel := k.cancelFuncs[gvk]
	wg, okWG := k.wgs[gvk]
	k.mu.RUnlock()

	// cancel if seen
	if okCancel {
		cancel()
	}

	// wait if seen
	if okWG {
		wg.Wait()
	}
}

// Name returns the name of the dependency kontroller
func (k *DependencyKontroller) Name() string {
	return "orkestra dependency kontroller"
}
