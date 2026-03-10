// pkg/controller/dependency_controller.go
package controller

import (
	"context"
	"sync"

	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/event"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/informer"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/kubeclient"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/queue"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/registry"
)

// DependencyController extends the base Controller with dependency‑aware startup.
// It ensures CRDs start in topological order and shut down in reverse order.
type DependencyController struct {
	*Controller

	depGraph       *registry.DependencyGraph
	defaultWorkers int

	// readyCh[name] is closed when a CRD has fully started its workers.
	readyCh map[string]chan struct{}
}

// NewDependencyController constructs a dependency‑aware controller.
// It embeds the base Controller so all worker logic, queue handling,
// and reconciler dispatching remain unchanged.
func NewDependencyController(
	kube *kubeclient.Kubeclient,
	factory *informer.Factory,
	registry *ResourceRegistry,
	events *event.Event,
	wq *queue.Workqueue,
	defaultWorkers int,
	depGraph *registry.DependencyGraph,
) *DependencyController {

	return &DependencyController{
		Controller:     NewController(kube, factory, registry, events, wq, defaultWorkers),
		depGraph:       depGraph,
		defaultWorkers: defaultWorkers,
		readyCh:        make(map[string]chan struct{}),
	}
}

// RunOrDie starts CRDs in dependency order and blocks until leadership is lost.
// When leadership ends, it shuts down CRDs in reverse dependency order.
func (c *DependencyController) RunOrDie(ctx context.Context) {
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

		logger.Info().Msgf("starting %s (depends on: %v)", name, crd.DependsOn)

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

		// 3c. Signal that this CRD is ready
		close(c.readyCh[name])
	}

	// 4. Block until leadership is lost
	<-ctx.Done()
	logger.Info().Msg("leadership lost — beginning dependency‑aware shutdown")

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
func (c *DependencyController) startCRDWorkers(ctx context.Context, gvk string, workers int) {
	crdCtx, cancel := context.WithCancel(ctx)

	c.mu.Lock()
	c.cancelFuncs[gvk] = cancel
	wg := &sync.WaitGroup{}
	c.wgs[gvk] = wg
	c.started[gvk] = true
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
func (c *DependencyController) stopCRDWorkers(name string) {
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
