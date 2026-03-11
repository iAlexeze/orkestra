package kontroller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/initialize"
	"github.com/ialexeze/orkestra/pkg/event"
	"github.com/ialexeze/orkestra/pkg/health"
	"github.com/ialexeze/orkestra/pkg/informer"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/queue"
	"github.com/ialexeze/orkestra/pkg/utils"
)

var _ domain.Komponent = (*Controller)(nil)

// Every map has the same key GVK
type Controller struct {
	kube            *kubeclient.Kubeclient
	informerFactory *informer.Factory
	event           *event.Event
	katalog         *ResourceKatalog
	wq              *queue.Workqueue
	hs              *health.HealthServer
	defaultWorkers  int
	healthy         bool
	started         map[string]bool
	cancelFuncs     map[string]context.CancelFunc
	wgs             map[string]*sync.WaitGroup
	mu              sync.RWMutex
	reconcilers     map[string]domain.Reconciler
	crds            []initialize.CRDEntry

	// Error rate
	total         map[string]int
	failed        map[string]int
	maxQueueDepth int
}

func NewController(
	kube *kubeclient.Kubeclient,
	informerFactory *informer.Factory,
	katalog *ResourceKatalog,
	event *event.Event,
	wq *queue.Workqueue,
	hs *health.HealthServer,
	defaultWorkers int,
	maxQueueDepth int,
) *Controller {
	c := &Controller{
		kube:            kube,
		informerFactory: informerFactory,
		katalog:         katalog,
		event:           event,
		wq:              wq,
		hs:              hs,
		defaultWorkers:  defaultWorkers,
		maxQueueDepth:   maxQueueDepth,
		started:         make(map[string]bool),
		cancelFuncs:     make(map[string]context.CancelFunc),
		total:           make(map[string]int),
		failed:          make(map[string]int),
		wgs:             make(map[string]*sync.WaitGroup),
		reconcilers:     make(map[string]domain.Reconciler),
	}

	// Load registry entries
	for gvk, entry := range katalog.Entries() {
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

// Healthy mark on startup
func (c *Controller) Started() bool { return c.healthy }

// Shutdown gracefully stops orkestra
func (c *Controller) Shutdown(ctx context.Context) {}

// Controller name
func (c *Controller) Name() string {
	return "orkestra kontroller"
}

// Errorrate
func (c *Controller) errorRate(gvk string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.total[gvk] == 0 {
		return 0
	}

	return float64(c.failed[gvk] / c.total[gvk])
}
