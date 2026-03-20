// pkg/kontroller/dependency_kontroller.go
package kontroller

import (
	"context"
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
	logger.Info().Msgf("%s is starting...", k.Name())
	k.startedAt = time.Now()

	startupOrder := k.depGraph.StartupOrder()

	// Create ready channels for all CRDs upfront
	for _, name := range startupOrder {
		k.readyCh[name] = make(chan struct{})
	}

	anyOnline := false

	// Process CRDs in dependency order
	for _, name := range startupOrder {
		node := k.depGraph.GetNode(name)
		crd := node.CRD
		gvk := crd.GroupVersionKind.String()

		// Wait for hard dependencies — blocks until dep is online or ctx cancelled
		for _, dep := range crd.DependsOn {
			select {
			case <-k.readyCh[dep]:
				logger.Debug().Msgf("%s: dependency %q ready", name, dep)
			case <-ctx.Done():
				return
			}
		}

		if k.informerFactory.IsMissing(gvk) {
			logger.Warn().Msgf("CRD %s is missing — workers not started, retry loop will activate it", gvk)
			// DO NOT close readyCh[name] — retry loop closes it when CRD appears
			// DO NOT call SetReady here — nothing is online yet for this CRD
			continue
		}

		// CRD exists — start workers
		workers := k.katalog.GetWorkers(gvk, k.defaultWorkers)
		logger.Info().Msgf("starting %d workers for %s", workers, gvk)
		k.startCRDWorkers(ctx, gvk, workers)
		metrics.WorkersActive.WithLabelValues(gvk).Set(float64(workers))

		// Signal dependents — this CRD is ready
		close(k.readyCh[name])
		logger.Info().Msgf("%s workers started and ready", name)

		anyOnline = true
	}

	// ── Start retry loop ONCE after the startup sequence ─────────────────────
	// Handles all missing CRDs together — no races, no duplicate goroutines.
	// Also handles CRDs that get deleted and recreated after startup.
	go k.retryMissingCRDs(ctx)

	// ── Mark controller started ───────────────────────────────────────────────
	// Ready only if at least one CRD came online.
	// If everything is missing, we're started but not ready — retry loop will flip this.
	k.startedKtrl.Store(true)
	if anyOnline {
		k.hs.SetReady()
		logger.Info().Msgf("%s started — %d CRD(s) online, retry loop active for missing",
			k.Name(), len(startupOrder))
	} else {
		logger.Warn().Msgf("%s started — all CRDs missing, waiting for retry loop", k.Name())
	}

	// Block until leadership lost
	<-ctx.Done()
	logger.Info().Msg("leadership lost — beginning dependency-aware shutdown")
	k.hs.Unhealthy()

	shutdownOrder := k.depGraph.ShutdownOrder()
	for _, name := range shutdownOrder {
		logger.Info().Msgf("shutting down %s", name)
		k.stopCRDWorkers(name)
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
