// pkg/informer/factory.go
package informer

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ialexeze/multi-crd-controller/pkg/config/domain"
	crderror "github.com/ialexeze/multi-crd-controller/pkg/config/pkg/error"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// For creates or returns an informer for the given type
func (f *Factory) For(obj runtime.Object, ctx context.Context) cache.SharedIndexInformer {
	f.mu.Lock()
	defer f.mu.Unlock()

	t := reflect.TypeOf(obj)

	// Return existing informer if already created
	if inf, ok := f.informers[t]; ok {
		return inf
	}

	// Create new informer - but don't start it yet
	inf := cache.NewSharedIndexInformer(
		f.newListWatch(obj),
		obj,
		f.resync,
		cache.Indexers{},
	)

	// Add event handlers
	inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { f.handleEvent(obj) },
		UpdateFunc: func(old, new interface{}) { f.handleEvent(new) },
		DeleteFunc: func(obj interface{}) { f.handleEvent(obj) },
	})

	f.informers[t] = inf

	// If factory is already started, start this informer immediately
	if f.started {
		go inf.Run(ctx.Done())
	}

	return inf
}

// handleEvent safely enqueues events with proper type detection
func (f *Factory) handleEvent(obj interface{}) {
	// Wait for factory to be ready before processing events
	<-f.ready

	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		logger.Error().Msgf("object is not a runtime.Object: %T", obj)
		return
	}

	// Important queuing pattern
	gvk := runtimeObj.GetObjectKind().GroupVersionKind()

	f.queue.Enqueue(obj, gvk.String())
}

// newListWatch returns a new ListWatch for the given type
func (f *Factory) newListWatch(obj runtime.Object) *cache.ListWatch {
	return &cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			// check if context is cancelled
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}

			// Wait for factory to be ready
			<-f.ready

			client, err := f.clientProvider.For(obj)
			if err != nil {
				return nil, fmt.Errorf("failed to get client for %T: %w", obj, err)
			}
			return client.List(ctx, options)
		},
		WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			// check if context is cancelled
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}

			// Wait for factory to be ready
			<-f.ready

			client, err := f.clientProvider.For(obj)
			if err != nil {
				return nil, fmt.Errorf("failed to get client for %T: %w", obj, err)
			}
			return client.Watch(ctx, options)
		},
	}
}

// Start now signals readiness and starts all informers
func (f *Factory) Start(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.started {
		return crderror.ErrFactoryAlreadyStarted
	}

	// First, mark factory as ready so List/Watch can proceed
	close(f.ready)

	// Then start all informers
	logger.Info().Msgf("starting %v informers...", len(f.informers))
	for _, inf := range f.informers {
		if inf == nil {
			continue
		}
		logger.Debug().Msgf("informer type: %T", inf)
		go inf.Run(ctx.Done())
	}

	f.started = true
	logger.Info().Msg("Factory started and ready")
	return nil
}

// WaitForCacheSync waits for all informers to sync
func (f *Factory) WaitForCacheSync(ctx context.Context) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// First wait for factory to be ready
	select {
	case <-f.ready:
		// Factory is ready
	case <-ctx.Done():
		return false
	}

	hasSynced := func() bool {
		for _, inf := range f.informers {
			if inf == nil {
				continue
			}
			if !inf.HasSynced() {
				return false
			}
		}
		return true
	}

	return cache.WaitForCacheSync(ctx.Done(), hasSynced)
}

// IsReady returns true if the factory has been started
func (f *Factory) IsReady() bool {
	select {
	case <-f.ready:
		return true
	default:
		return false
	}
}

// Implement the component part
var _ domain.Component = (*Factory)(nil)

func (f *Factory) Shutdown(ctx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Stop all informers (they'll stop when ctx is done)
	f.started = false
	// Note: We don't close ready again as it's already closed to signal ready
}

func (f *Factory) Name() string {
	return "shared informer factory"
}
