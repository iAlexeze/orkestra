package kontroller

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/event"
	"github.com/ialexeze/orkestra/pkg/runtime"

	// "github.com/ialexeze/orkestra/pkg/health"
	"github.com/ialexeze/orkestra/pkg/informer"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/queue"
	"github.com/ialexeze/orkestra/pkg/utils"
)

var _ domain.Komponent = (*Controller)(nil)

// Every map has the same key GVK
type Controller struct {
	kube             *kubeclient.Kubeclient
	informerFactory  *informer.Factory
	event            *event.Event
	katalog          *ResourceKatalog
	queueRegistry    *queue.QueueRegistry
	defaultWorkqueue *queue.Workqueue
	degradeThreshold map[string]int

	hs           domain.Health
	crdHealthMap map[string]*CRDHealth

	defaultWorkers int
	startedKtrl    atomic.Bool
	started        map[string]bool
	cancelFuncs    map[string]context.CancelFunc
	wgs            map[string]*sync.WaitGroup
	mu             sync.RWMutex
	reconcilers    map[string]domain.Reconciler
	crds           []runtime.CRDEntry

	// Error rate
	total  map[string]int
	failed map[string]int
}

func NewKontroller(
	kube *kubeclient.Kubeclient,
	informerFactory *informer.Factory,
	katalog *ResourceKatalog,
	event *event.Event,
	hs domain.Health,
	crdHealthMap map[string]*CRDHealth,
	queueRegistry *queue.QueueRegistry,
	defaultWorkqueue *queue.Workqueue,
	defaultWorkers int,
) *Controller {
	c := &Controller{
		kube:             kube,
		informerFactory:  informerFactory,
		katalog:          katalog,
		event:            event,
		hs:               hs,
		defaultWorkqueue: defaultWorkqueue,
		queueRegistry:    queueRegistry,
		defaultWorkers:   defaultWorkers,
		started:          make(map[string]bool),
		cancelFuncs:      make(map[string]context.CancelFunc),
		total:            make(map[string]int),
		failed:           make(map[string]int),
		wgs:              make(map[string]*sync.WaitGroup),
		reconcilers:      make(map[string]domain.Reconciler),
		crdHealthMap:     crdHealthMap,
		degradeThreshold: make(map[string]int),
	}

	// Load registry entries
	for gvk, entry := range katalog.Entries() {
		c.crds = append(c.crds, entry.CRD)
		c.degradeThreshold[gvk] = entry.CRD.Queue.DegradeThreshold
	}

	return c
}

func (c *Controller) Start(ctx context.Context) error {
	// Run CRD checks in parallel, respecting dependency order
	var wg sync.WaitGroup
	errCh := make(chan error, len(c.crds))

	// readyCh per CRD — same pattern as DependencyKontroller
	readyCh := make(map[string]chan struct{}, len(c.crds))
	for _, crd := range c.crds {
		readyCh[crd.Name] = make(chan struct{})
	}

	for _, crd := range c.crds {
		wg.Add(1)

		go func() {
			defer wg.Done()

			// Wait for all dependencies to be confirmed present first
			for _, dep := range crd.DependsOn {
				select {
				case <-readyCh[dep]:
					// dependency confirmed — proceed
				case <-ctx.Done():
					errCh <- fmt.Errorf("context cancelled waiting for dependency %q", dep)
					return
				}
			}

			logger.Info().Msgf("checking CRD %s/%s (%s)...", crd.APITypes.Group, crd.APITypes.Version, crd.APITypes.Kind)

			err := utils.RetryBackoff(
				func() error {
					return utils.WaitForCRD(
						c.kube.RestConfig(),
						crd.APITypes.Group,
						crd.APITypes.Kind,
						crd.APITypes.Version,
					)
				},
				5,
				2*time.Second,
			)

			if err != nil {
				errCh <- fmt.Errorf("CRD %s/%s (%s) not found: %w",
					crd.APITypes.Group, crd.APITypes.Version, crd.APITypes.Kind, err)
				return
			}

			logger.Info().Msgf("CRD %s/%s (%s) detected", crd.APITypes.Group, crd.APITypes.Version, crd.APITypes.Kind)

			// Signal dependents that this CRD is confirmed
			close(readyCh[crd.Name])
		}()
	}

	// Wait for all checks to complete
	wg.Wait()
	close(errCh)

	// Collect any errors
	var errs []string
	for err := range errCh {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf("CRD checks failed:\n%s", strings.Join(errs, "\n"))
	}

	// All CRDs confirmed — now sync caches
	logger.Debug().Msg("waiting for all informer caches to sync...")
	if !c.informerFactory.WaitForCacheSync(ctx) {
		return fmt.Errorf("failed to sync one or more informer caches")
	}
	logger.Info().Msg("all informer caches synced")

	// Build reconcilers now — kube, ev, and REST clients are all live
	logger.Debug().Msg("building reconcilers...")
	for gvk, entry := range c.katalog.Entries() {
		rec := entry.ReconcilerFactory() // ← safe here, manager has started everything
		c.mu.Lock()
		c.reconcilers[gvk] = rec
		c.mu.Unlock()
		logger.Debug().Str("gvk", gvk).Msg("reconciler built")
	}

	return nil
}

// Set the controller ready
func (c *Controller) SetReady(h domain.Health) {
	h.SetReady()
}

// Set the controller to degraded
func (c *Controller) Degraded(h domain.Health) {
	h.Degraded()
}

// Healthy mark on startup
func (c *Controller) Started() bool { return c.startedKtrl.Load() }

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
