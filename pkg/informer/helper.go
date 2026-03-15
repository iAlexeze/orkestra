package informer

import (
	"context"
	"fmt"
	"strings"

	"github.com/ialexeze/orkestra/domain"
	crderror "github.com/ialexeze/orkestra/pkg/error"
	"github.com/ialexeze/orkestra/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// handleEvent resolves the GVK from the scheme and routes the event
// to the correct per-CRD queue. Falls back to the default queue if
// no per-CRD queue is registered for this GVK.
func (f *Factory) handleEvent(obj interface{}) {
	// Block until factory is ready — List/Watch have started
	<-f.ready

	gvk, err := f.gvkFromObject(obj)
	if err != nil {
		return
	}

	gvkStr := gvk.String()

	// Route to per-CRD queue if registered, otherwise fall back to default
	wq, ok := f.queueRegistry.For(gvkStr)
	if !ok {
		logger.Warn().
			Str("gvk", gvkStr).
			Msg("no per-CRD queue registered — falling back to default queue")
		f.defaultWq.Enqueue(obj, gvkStr)
		return // ← return here — do not also enqueue below
	}

	wq.Enqueue(obj, gvkStr)
}

// newListWatch returns a ListWatch for the given object type.
// Both List and Watch block on f.ready so they never run before Start().
func (f *Factory) newListWatch(obj runtime.Object) *cache.ListWatch {
	return &cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			<-f.ready

			client, err := f.clientProvider.For(obj)
			if err != nil {
				return nil, fmt.Errorf("list: failed to get client for %T: %w", obj, err)
			}
			return client.List(ctx, options)
		},
		WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			<-f.ready

			client, err := f.clientProvider.For(obj)
			if err != nil {
				return nil, fmt.Errorf("watch: failed to get client for %T: %w", obj, err)
			}
			return client.Watch(ctx, options)
		},
	}
}

// Start signals readiness and starts all registered informers.
// Must be called exactly once — enforced by ErrFactoryAlreadyStarted.
func (f *Factory) Start(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.started {
		return crderror.ErrFactoryAlreadyStarted
	}

	// Signal readiness first — unblocks any List/Watch already waiting
	close(f.ready)

	logger.Info().Msgf("starting %d informers...", len(f.informers))

	for t, entry := range f.informers {
		if entry == nil || entry.informer == nil {
			logger.Warn().Msgf("nil informer entry for type %s — skipping", t)
			continue
		}
		// Each informer gets its own correct name — no shared opts field
		logger.Debug().
			Str("name", entry.name).
			Str("type", t.String()).
			Dur("resync", entry.resync).
			Msg("starting informer")
		go entry.informer.Run(ctx.Done())
	}

	f.started = true
	logger.Info().Msg("informer factory started and ready")
	return nil
}

// WaitForCacheSync blocks until all informer caches have synced or ctx is done.
func (f *Factory) WaitForCacheSync(ctx context.Context) bool {
	// Wait for factory to be ready — safe to call before Start() returns
	select {
	case <-f.ready:
	case <-ctx.Done():
		return false
	}

	f.mu.RLock()
	// Build hasSynced funcs slice — one per informer
	syncFuncs := make([]cache.InformerSynced, 0, len(f.informers))
	for _, entry := range f.informers {
		if entry != nil && entry.informer != nil {
			syncFuncs = append(syncFuncs, entry.informer.HasSynced)
		}
	}
	f.mu.RUnlock()

	return cache.WaitForCacheSync(ctx.Done(), syncFuncs...)
}

// Status returns a summary of all informers and their sync state.
// Used by the health server /katalog endpoint.
func (f *Factory) Status() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.informers))
	for _, entry := range f.informers {
		if entry != nil {
			synced := "not synced"
			if entry.informer.HasSynced() {
				synced = "synced"
			}
			names = append(names, fmt.Sprintf("%s(%s)", entry.name, synced))
		}
	}
	return strings.Join(names, ", ")
}

// IsReady returns true if the factory has been started.
func (f *Factory) IsReady() bool {
	select {
	case <-f.ready:
		return true
	default:
		return false
	}
}

func (f *Factory) gvkFromObject(obj interface{}) (*schema.GroupVersionKind, error) {
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		logger.Error().Msgf("object is not a runtime.Object: %T", obj)
		return nil, fmt.Errorf("object is not a runtime.Object: %T", obj)
	}

	// Resolve GVK from scheme — cached objects have TypeMeta stripped,
	// GetObjectKind().GroupVersionKind() returns empty.
	gvks, _, err := f.scheme.ObjectKinds(runtimeObj)
	if err != nil || len(gvks) == 0 {
		logger.Error().Err(err).Msgf("failed to resolve GVK for %T — event dropped", obj)
		return nil, fmt.Errorf("failed to resolve GVK for %T — event dropped", obj)
	}

	return &gvks[0], nil
}

// ── Komponent ─────────────────────────────────────────────────────────────────

var _ domain.Komponent = (*Factory)(nil)

func (f *Factory) Started() bool { return f.started }

func (f *Factory) Shutdown(ctx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Informers stop when their ctx.Done() channel closes — handled by caller.
	// We just mark the factory as stopped.
	f.started = false
}

func (f *Factory) Name() string { return "informer factory" }
