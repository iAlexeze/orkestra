// pkg/informer/factory.go
package informer

import (
	"context"
	"reflect"

	"github.com/ialexeze/orkestra/pkg/logger"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// For creates or returns a SharedIndexInformer for the given object type.
// Uses the client provider to build the ListerWatcher via newListWatch.
// Each type gets exactly one informer — subsequent calls return the cached one.
// The informer is not started here — Start() starts all registered informers.
func (f *Factory) For(obj runtime.Object, ctx context.Context, opts Options) cache.SharedIndexInformer {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.getOrCreate(reflect.TypeOf(obj), f.newListWatch(obj), obj, ctx, opts)
}

// ForListerWatcher creates or returns a SharedIndexInformer using an explicit
// ListerWatcher. Used for unstructured CRDs where the dynamic client provides
// the ListerWatcher directly, bypassing the typed client provider entirely.
// The scheme is never consulted — no conversion errors for unstructured types.
func (f *Factory) ForListerWatcher(lw cache.ListerWatcher, obj runtime.Object, ctx context.Context, opts Options) cache.SharedIndexInformer {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Unstructured objects all have the same reflect.Type — use name as key instead
	key := reflect.TypeOf(obj)
	if opts.Name != "" {
		// Store under a named type derived from the informer name
		// This ensures each unstructured CRD gets its own informer
		key = reflect.TypeOf(struct{ name string }{opts.Name})
	}

	return f.getOrCreate(key, lw, obj, ctx, opts)
}

// getOrCreate is the shared core — returns an existing informer or creates one.
// Called by both For and ForListerWatcher. Must be called with f.mu held.
func (f *Factory) getOrCreate(
	key reflect.Type,
	lw cache.ListerWatcher,
	obj runtime.Object,
	ctx context.Context,
	opts Options,
) cache.SharedIndexInformer {
	// Return existing informer — idempotent
	if entry, ok := f.informers[key]; ok {
		logger.Debug().Msgf("informer for %s already exists — reusing", opts.Name)
		return entry.informer
	}

	// Resolve resync — per-CRD takes priority, fall back to factory default
	resync := opts.Resync
	if resync == 0 {
		logger.Info().Msgf(
			"processing informer for %s with default resync duration: %v",
			opts.Name, f.defaultResync,
		)
		resync = f.defaultResync
	} else {
		logger.Info().Msgf(
			"processing informer for %s with resync duration: %v",
			opts.Name, resync,
		)
	}

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

	f.informers[key] = &informerEntry{
		informer: inf,
		name:     opts.Name,
		resync:   resync,
	}

	// If factory already started (dynamic CRD registration), start immediately
	if f.started {
		go inf.Run(ctx.Done())
	}

	logger.Info().Msgf("informer for %s created", opts.Name)
	return inf
}
