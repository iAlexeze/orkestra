package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ialexeze/multi-crd-controller/pkg/config/domain"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/event"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/informer"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/kubeclient"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/queue"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/registry"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/utils"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ domain.Component = (*Controller)(nil)

type Controller struct {
	kube            *kubeclient.Kubeclient
	informerFactory *informer.Factory
	event           *event.Event
	registry        *ResourceRegistry
	wq              *queue.Workqueue
	wg              sync.WaitGroup
	workers         int
	reconcilers     map[string]domain.Reconciler
	crds            []registry.CRDInfo
}

func NewControllerManager(
	kube *kubeclient.Kubeclient,
	informerFactory *informer.Factory,
	registry *ResourceRegistry,
	event *event.Event,
	wq *queue.Workqueue,
	workers int,
) *Controller {
	c := &Controller{
		kube:            kube,
		informerFactory: informerFactory,
		registry:        registry,
		event:           event,
		wq:              wq,
		workers:         workers,
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

func (c *Controller) RunOrDie(ctx context.Context) {
	logger.Info().Msgf("starting %d workers", c.workers)

	// Start workers
	for i := 0; i < c.workers; i++ {
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			wait.UntilWithContext(
				ctx,
				func(ctx context.Context) {
					c.runWorker(ctx)
				}, time.Second)
		}()
	}

	// BLOCK until leadership is lost
	<-ctx.Done()

	logger.Info().Msg("leadership lost — draining workers...")

	// Stop accepting new items
	c.wq.Shutdown(ctx)

	// Wait for all workers to finish
	c.wg.Wait()

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
