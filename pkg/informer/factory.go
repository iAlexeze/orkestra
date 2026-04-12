// pkg/informer/factory.go
package informer

import (
	"context"

	"github.com/ialexeze/orkestra/pkg/logger"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// For creates or returns a SharedIndexInformer for the given object type.
// Uses the client provider to build the ListerWatcher via newListWatch.
// Each type gets exactly one informer — subsequent calls return the cached one.
// The informer is not started here — Start() starts all registered informers.
func (f *Factory) For(obj runtime.Object, ctx context.Context, opts Options) cache.SharedIndexInformer {
	f.mu.Lock()
	defer f.mu.Unlock()

	gvk, err := f.gvkFromObject(obj)
	if err != nil {
		return nil
	}

	return f.getOrCreate(gvk, f.newListWatch(obj, opts.LabelSelector, opts.FieldSelector), obj, ctx, opts)
}

// ForListerWatcher creates or returns a SharedIndexInformer using an explicit
// ListerWatcher. Used for unstructured CRDs where the dynamic client provides
// the ListerWatcher directly, bypassing the typed client provider entirely.
// The scheme is never consulted — no conversion errors for unstructured types.
func (f *Factory) ForListerWatcher(lw cache.ListerWatcher, obj runtime.Object, ctx context.Context, opts Options) cache.SharedIndexInformer {
	f.mu.Lock()
	defer f.mu.Unlock()

	gvk, err := f.gvkFromObject(obj)
	if err != nil {
		return nil
	}
	return f.getOrCreate(gvk, lw, obj, ctx, opts)
}

// getOrCreate is the shared core — returns an existing informer or creates one.
// Called by both For and ForListerWatcher. Must be called with f.mu held.
func (f *Factory) getOrCreate(
	gvk *schema.GroupVersionKind,
	lw cache.ListerWatcher,
	obj runtime.Object,
	ctx context.Context,
	opts Options,
) cache.SharedIndexInformer {
	key := gvk.String()

	// Return existing informer
	if entry, ok := f.informers[key]; ok {
		return entry.Informer
	}

	// ALWAYS create the informer (even for missing CRDs)
	resync := opts.Resync
	if resync == 0 {
		resync = f.defaultResync
	}

	inf := cache.NewSharedIndexInformer(lw, obj, resync, cache.Indexers{})

	inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { f.handleEvent(obj) },
		UpdateFunc: func(_, newObj interface{}) { f.handleEvent(newObj) },
		DeleteFunc: func(obj interface{}) { f.handleEvent(obj) },
	})

	// Check if CRD exists
	crdExists := f.crdExists(gvk)

	entry := &InformerEntry{
		Informer: inf,
		Name:     opts.Name,
		Resync:   resync,
		Missing:  !crdExists,
		GVK:      gvk,
	}

	f.informers[key] = entry

	if crdExists {
		delete(f.missing, key)
		if f.started.Load() {
			go inf.Run(ctx.Done())
		}
		logger.Info().Str("name", opts.Name).Str("gvk", gvk.String()).Msg("informer created and started")
	} else {
		f.missing[key] = entry
		logger.Warn().Str("name", opts.Name).Str("gvk", gvk.String()).Msg("CRD missing — informer created but not started")
	}

	return inf
}
