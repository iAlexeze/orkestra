// pkg/kontroller/dependency_kontroller.go
package kontroller

import (
	"context"
	"strings"
	"sync"

	"github.com/ialexeze/orkestra/pkg/event"
	"github.com/ialexeze/orkestra/pkg/health"
	"github.com/ialexeze/orkestra/pkg/informer"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/metrics"
	"github.com/ialexeze/orkestra/pkg/queue"
	"github.com/ialexeze/orkestra/pkg/registry"
)

// DependencyKontroller extends the base Controller with dependency‑aware startup.
// It ensures CRDs start in topological order and shut down in reverse order.
type DependencyKontroller struct {
	*Controller

	depGraph       *registry.DependencyGraph
	defaultWorkers int
	bannKfg           *BannerKonfig


	// readyCh[name] is closed when a CRD has fully started its workers.
	readyCh map[string]chan struct{}
}

// NewDependencyKontroller constructs a dependency‑aware controller.
// It embeds the base Kontroller so all worker logic, queue handling,
// and reconciler dispatching remain unchanged.
func NewDependencyKontroller(
	kube *kubeclient.Kubeclient,
	factory *informer.Factory,
	registry *ResourceRegistry,
	events *event.Event,
	wq *queue.Workqueue,
	hs *health.HealthServer,
	defaultWorkers int,
	maxQueueDepth int,
	depGraph *registry.DependencyGraph,
	bannKfg *BannerKonfig,
) *DependencyKontroller {

	return &DependencyKontroller{
		Controller:     NewController(kube, factory, registry, events, wq, hs, defaultWorkers, maxQueueDepth),
		depGraph:       depGraph,
		defaultWorkers: defaultWorkers,
		bannKfg:           bannKfg,
		readyCh:        make(map[string]chan struct{}),
	}
}

// RunOrDie starts CRDs in dependency order and blocks until leadership is lost.
// When leadership ends, it shuts down CRDs in reverse dependency order.
func (c *DependencyKontroller) RunOrDie(ctx context.Context) {
	logger.Info().Msgf("dependency controller starting in %s mode...", c.depGraph.GetMode())

	// 1. Compute topological startup order (A → B → C)
	startupOrder := c.depGraph.StartupOrder()

	// 2. Create a "ready" channel for each CRD
	for _, name := range startupOrder {
		c.readyCh[name] = make(chan struct{})
	}

	// 3. Start CRDs in dependency order
	for _, name := range startupOrder {
		node := c.depGraph.GetNode(name)
		crd := node.CRD
		gvk := crd.GroupVersionKind.String()

		if len(crd.DependsOn) > 0 {
			logger.Info().Msgf("starting %s (depends on: %s)", name, strings.Join(crd.DependsOn, ", "))
		} else {
			logger.Info().Msgf("starting %s", name)
		}

		// 3a. Wait for all dependencies to signal readiness
		for _, dep := range crd.DependsOn {
			select {
			case <-c.readyCh[dep]:
				logger.Debug().Msgf("%s dependency %s is ready", name, dep)
			case <-ctx.Done():
				return
			}
		}

		// 3b. Start workers for this CRD
		workers := c.registry.GetWorkers(gvk, c.defaultWorkers)
		logger.Info().Msgf("starting %d workers for %s", workers, gvk)

		c.startCRDWorkers(ctx, gvk, workers)

		// 3c Send metrics to prometheus
		metrics.WorkersActive.WithLabelValues(gvk).Set(float64(workers))

		// 3d. Signal that this CRD is ready
		close(c.readyCh[name])
	}

	// Mark as started
	c.healthy = true
	c.hs.SetReady()

	c.bannKfg.Komponents = append(c.bannKfg.Komponents, c)

	// Print banner
	c.printBanner(c.bannKfg)

	// 4. Block until leadership is lost
	<-ctx.Done()
	logger.Info().Msg("leadership lost — beginning dependency‑aware shutdown")

	// Mark as degraded
	c.healthy = false
	c.hs.Degraded()

	// 5. Shutdown CRDs in reverse dependency order
	shutdownOrder := c.depGraph.ShutdownOrder()
	for _, name := range shutdownOrder {
		logger.Info().Msgf("shutting down %s", name)
		c.stopCRDWorkers(name)
	}

	logger.Info().Msg("dependency controller drained and stopped")
}

// startCRDWorkers starts a worker pool for a specific CRD.
// It mirrors the logic in Controller.RunOrDie but is invoked in dependency order.
func (c *DependencyKontroller) startCRDWorkers(ctx context.Context, gvk string, workers int) {
	crdCtx, cancel := context.WithCancel(ctx)

	c.mu.Lock()
	c.cancelFuncs[gvk] = cancel
	wg := &sync.WaitGroup{}
	c.wgs[gvk] = wg
	c.started[gvk] = true
	c.total[gvk]++
	c.mu.Unlock()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			c.runWorkerForGVK(crdCtx, gvk, workerID)
		}(i)
	}
}

// stopCRDWorkers cancels the CRD context and waits for all workers to drain.
func (c *DependencyKontroller) stopCRDWorkers(name string) {
	gvk := c.depGraph.GetNode(name).CRD.GroupVersionKind.String()

	c.mu.RLock()
	cancel, okCancel := c.cancelFuncs[gvk]
	wg, okWG := c.wgs[gvk]
	c.mu.RUnlock()

	if okCancel {
		cancel()
	}
	if okWG {
		wg.Wait()
	}
}
