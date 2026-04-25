package informer

import (
	"context"
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/domain"
	orkerror "github.com/orkspace/orkestra/pkg/error"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
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

	// ── Tier 2: Pre-enqueue namespace filter ─────────────────────────────
	// Check namespace restriction BEFORE the item enters the queue.
	// Items that fail this check are dropped — they do no work and create
	// no queue pressure. The reconciler check (Tier 3) remains as a safety
	// net for race conditions during startup.
	namespace := extractNamespace(obj)
	if !f.namespaceAllowed(gvkStr, namespace) {
		logger.Debug().
			Str("gvk", gvkStr).
			Str("namespace", namespace).
			Msg("informer: event dropped — namespace not allowed")
		return
	}

	// Route to per-CRD queue if registered, otherwise fall back to default
	wq, ok := f.queueRegistry.For(gvkStr)
	if !ok {
		logger.Warn().
			Str("gvk", gvkStr).
			Msg("no per-CRD queue registered — falling back to default queue")
		f.defaultWq.Enqueue(obj, gvkStr)
		return
	}

	wq.Enqueue(obj, gvkStr)
}

// newListWatch returns a ListWatch for the given object type.
// Both List and Watch block on f.ready so they never run before Start().
// When opts.Namespace is set (Tier 1 single-namespace filter), the ListerWatcher
// is scoped to that namespace — the informer never sees events from other namespaces.
func (f *Factory) newListWatch(obj runtime.Object, opts Options) *cache.ListWatch {
	return &cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			<-f.ready

			if opts.LabelSelector != "" {
				utils.Merge(&options.LabelSelector, opts.LabelSelector, ",")
			}
			if opts.FieldSelector != "" {
				utils.Merge(&options.FieldSelector, opts.FieldSelector, ",")
			}

			client, err := f.clientProvider.For(obj)
			if err != nil {
				return nil, fmt.Errorf("list: failed to get client for %T: %w", obj, err)
			}
			if opts.Namespace != "" {
				return client.ListInNamespace(ctx, opts.Namespace, options)
			}
			return client.List(ctx, options)
		},
		WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			<-f.ready

			if opts.LabelSelector != "" {
				utils.Merge(&options.LabelSelector, opts.LabelSelector, ",")
			}
			if opts.FieldSelector != "" {
				utils.Merge(&options.FieldSelector, opts.FieldSelector, ",")
			}

			client, err := f.clientProvider.For(obj)
			if err != nil {
				return nil, fmt.Errorf("watch: failed to get client for %T: %w", obj, err)
			}
			if opts.Namespace != "" {
				return client.WatchInNamespace(ctx, opts.Namespace, options)
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

	if f.started.Load() {
		return orkerror.ErrFactoryAlreadyStarted
	}

	// Signal readiness first — unblocks any List/Watch already waiting
	close(f.ready)

	logger.Info().Int("count", len(f.informers)).Msg("starting informers")

	for t, entry := range f.informers {
		if entry == nil || entry.Informer == nil || entry.Missing {
			logger.Warn().Str("gvk", t).Msg("informer entry nil or missing — skipping")
			entry.WasNeverStarted = true
			continue
		}
		// Each informer gets its own correct name
		logger.Debug().
			Str("name", entry.Name).
			Str("type", t).
			Dur("resync", entry.Resync).
			Msg("starting informer")
		go entry.Informer.Run(ctx.Done())
	}

	f.started.Store(true)
	logger.Info().Msg("informer factory started and ready")
	return nil
}

// WaitForCacheSync blocks until all informer caches have synced or ctx is done.
// It only waits on informers that were successfully created and started.
// Informers for missing CRDs (where ListWatch failed) should never be added
// to the factory, so they are naturally excluded here.
func (f *Factory) WaitForCacheSync(ctx context.Context) bool {
	// Wait for factory to be marked ready (Start() has been called).
	select {
	case <-f.ready:
	case <-ctx.Done():
		return false
	}

	f.mu.RLock()
	syncFuncs := make([]cache.InformerSynced, 0, len(f.informers))

	for _, entry := range f.informers {
		if entry == nil || entry.Missing {
			entry.WasNeverStarted = true
			logger.Warn().
				Str("name", entry.Name).
				Msg("skipping cache sync — CRD missing or not registered")
			continue
		}
		syncFuncs = append(syncFuncs, entry.Informer.HasSynced)
		logger.Debug().
			Str("name", entry.Name).
			Bool("synced", entry.Informer.HasSynced()).
			Int("count", len(syncFuncs)).
			Msg("informer cache sync state")
	}

	f.mu.RUnlock()

	// If there are no informers at all, treat as trivially synced.
	if len(syncFuncs) == 0 {
		logger.Warn().Msg("no informers registered — treating cache sync as successful")
		return true
	}

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
			if entry.Informer.HasSynced() {
				synced = "synced"
			}
			names = append(names, fmt.Sprintf("%s(%s)", entry.Name, synced))
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
		logger.Error().Str("type", fmt.Sprintf("%T", obj)).Msg("object is not a runtime.Object — event dropped")
		return nil, fmt.Errorf("object is not a runtime.Object: %T", obj)
	}

	// Resolve GVK from scheme — cached objects have TypeMeta stripped,
	// GetObjectKind().GroupVersionKind() returns empty.
	gvks, _, err := f.scheme.ObjectKinds(runtimeObj)
	if err != nil || len(gvks) == 0 {
		logger.Error().Err(err).Str("type", fmt.Sprintf("%T", obj)).Msg("failed to resolve GVK — event dropped")
		return nil, fmt.Errorf("failed to resolve GVK for %T — event dropped", obj)
	}

	return &gvks[0], nil
}

// Check if a CRD exists
func (f *Factory) crdExists(gvk *schema.GroupVersionKind) bool {
	disco, err := discovery.NewDiscoveryClientForConfig(f.restConfig)
	if err != nil {
		return false
	}
	resources, err := disco.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return false
	}
	for _, r := range resources.APIResources {
		if r.Kind == gvk.Kind {
			return true
		}
	}
	return false
}

// Registered CRDs
func (f *Factory) Registered() map[string]*InformerEntry {
	f.mu.RLock()
	defer f.mu.RUnlock()

	out := make(map[string]*InformerEntry, len(f.informers))
	for k, v := range f.informers {
		out[k] = v
	}
	return out
}

// Missing CRDs on startup
func (f *Factory) Missing() map[string]*InformerEntry {
	f.mu.RLock()
	defer f.mu.RUnlock()

	out := make(map[string]*InformerEntry, len(f.missing))
	for k, v := range f.missing {
		out[k] = v
	}
	return out
}

// SetMissing CRDs on startup
func (f *Factory) SetMissingOnStartup(missing map[string]*InformerEntry) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.missing = missing
}

// RemoveMissing CRDs in between runs
func (f *Factory) RemoveMissing(gvkStr string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.missing, gvkStr)
}

// IsMissing CRDs on startup
func (f *Factory) IsMissing(key string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, ok := f.missing[key]
	return ok
}

// ── Komponent ─────────────────────────────────────────────────────────────────

var _ domain.Komponent = (*Factory)(nil)

func (f *Factory) Started() bool { return f.started.Load() }

func (f *Factory) Shutdown(ctx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Informers stop when their ctx.Done() channel closes — handled by caller.
	// We just mark the factory as stopped.
	f.started.Store(false)
}

func (f *Factory) Name() string { return "informer factory" }
