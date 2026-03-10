package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ialexeze/multi-crd-controller/pkg/config/domain"
	"github.com/ialexeze/multi-crd-controller/pkg/config/initialize"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/event"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/informer"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/kubeclient"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/queue"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/utils"
)

var _ domain.Component = (*Controller)(nil)

type Controller struct {
	kube            *kubeclient.Kubeclient
	informerFactory *informer.Factory
	event           *event.Event
	registry        *ResourceRegistry
	wq              *queue.Workqueue
	defaultWorkers  int
	started         map[string]bool
	cancelFuncs     map[string]context.CancelFunc
	wgs             map[string]*sync.WaitGroup
	mu              sync.RWMutex
	reconcilers     map[string]domain.Reconciler
	crds            []initialize.CRDEntry
}

func NewController(
	kube *kubeclient.Kubeclient,
	informerFactory *informer.Factory,
	registry *ResourceRegistry,
	event *event.Event,
	wq *queue.Workqueue,
	defaultWorkers int,
) *Controller {
	c := &Controller{
		kube:            kube,
		informerFactory: informerFactory,
		registry:        registry,
		event:           event,
		wq:              wq,
		defaultWorkers:  defaultWorkers,
		started:         make(map[string]bool),
		cancelFuncs:     make(map[string]context.CancelFunc),
		wgs:             make(map[string]*sync.WaitGroup),
		reconcilers:     make(map[string]domain.Reconciler),
	}

	// Load registry entries
	for gvk, entry := range registry.Entries() {
		c.reconcilers[gvk] = entry.Reconciler
		c.crds = append(c.crds, entry.CRD)
	}

	return c
}

func (c *Controller) Start(ctx context.Context) error {
	// CRD check (you may later generalize this per-CRD)
	for _, crd := range c.crds {
		logger.Info().Msgf("checking CRD %s/%s (%s)...", crd.Group, crd.Version, crd.Kind)

		err := utils.RetryBackoff(
			func() error {
				return utils.WaitForCRD(
					c.kube.RestConfig(),
					crd.Group,
					crd.Kind,
					crd.Version,
				)
			},
			5,
			2*time.Second,
		)

		if err != nil {
			return fmt.Errorf("CRD %s/%s (%s) not found: %w",
				crd.Group, crd.Version, crd.Kind, err)
		}

		logger.Info().Msgf("CRD %s/%s (%s) detected", crd.Group, crd.Version, crd.Kind)
	}

	logger.Debug().Msg("waiting for all informer caches to sync...")
	if !c.informerFactory.WaitForCacheSync(ctx) {
		return fmt.Errorf("failed to sync one or more informer caches")
	}
	logger.Info().Msg("all informer caches synced")

	return nil
}

// Deprecated: Now handled by dependency graph
func (c *Controller) RunOrDie(ctx context.Context) {
	// Get all registered GVKs
	gvks := c.registry.ListGVKs()

	// Start per-CRD worker pools
	for _, gvk := range gvks {
		workers := c.registry.GetWorkers(gvk, c.defaultWorkers)

		// Create a cancellable context for this CRD
		crdCtx, cancel := context.WithCancel(ctx)

		c.mu.Lock()
		c.cancelFuncs[gvk] = cancel
		wg := &sync.WaitGroup{}
		c.wgs[gvk] = wg
		c.started[gvk] = true
		c.mu.Unlock()

		// Start workers for this CRD
		logger.Info().Msgf("starting %d workers for %s", workers, gvk)
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				c.runWorkerForGVK(crdCtx, gvk, workerID)
			}(i)
		}
	}

	// BLOCK until leadership is lost
	<-ctx.Done()

	logger.Info().Msg("leadership lost — draining workers...")

	// Stop accepting new items
	c.wq.Shutdown(ctx)

	// Cancel all CRD contexts and wait for their workers
	c.mu.RLock()
	for gvk, cancel := range c.cancelFuncs {
		logger.Info().Msgf("cancelling workers for %s", gvk)
		cancel()
	}
	c.mu.RUnlock()

	// Wait for all CRD worker pools to finish
	c.mu.RLock()
	for gvk, wg := range c.wgs {
		logger.Info().Msgf("waiting for %s workers to drain...", gvk)
		wg.Wait()
	}
	c.mu.RUnlock()

	logger.Info().Msg("controller drained and stopped")
}

// Shutdown gracefully stops the Controller
func (c *Controller) Shutdown(ctx context.Context) {
	logger.Info().Msg("shutting down Controller")
	c.wq.Shutdown(ctx)
}

// Controller name
func (c *Controller) Name() string {
	return "smart controller"
}
