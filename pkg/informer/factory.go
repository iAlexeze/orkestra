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

	return f.getOrCreate(gvk, f.newListWatch(obj), obj, ctx, opts)
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

	// 1. Return existing informer
	if entry, ok := f.informers[key]; ok {
		logger.Debug().Msgf("informer for %s already exists — reusing", opts.Name)
		return entry.Informer
	}

	// 2. Check CRD existence BEFORE creating informer
	logger.Info().Msgf("checking CRD (%s)...", gvk.String())
	if !f.crdExists(gvk) {
		logger.Warn().
			Str("gvk", key).
			Msg("CRD not installed — skipping informer creation until it appears")

		entry := &InformerEntry{
			Informer: nil,
			Name:     opts.Name,
			Resync:   opts.Resync,
			Missing:  true,
			GVK:      gvk,
		}

		f.informers[key] = entry
		f.missing[key] = entry
		return nil
	}

	// 3. Resolve resync
	resync := opts.Resync
	if resync == 0 {
		resync = f.defaultResync
	}

	// 4. Create informer
	inf := cache.NewSharedIndexInformer(
		lw,
		obj,
		resync,
		cache.Indexers{},
	)

	inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { f.handleEvent(obj) },
		UpdateFunc: func(_, newObj interface{}) { f.handleEvent(newObj) },
		DeleteFunc: func(obj interface{}) { f.handleEvent(obj) },
	})

	// 5. Register informer
	entry := &InformerEntry{
		Informer: inf,
		Name:     opts.Name,
		Resync:   resync,
		Missing:  false,
		GVK:      gvk,
	}

	f.informers[key] = entry
	delete(f.missing, key)

	// 6. Start immediately if factory already running
	if f.started.Load() {
		go inf.Run(ctx.Done())
	}

	logger.Info().Msgf("informer for %s created", opts.Name)
	return inf
}
