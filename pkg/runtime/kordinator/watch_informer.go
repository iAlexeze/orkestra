// pkg/runtime/kordinator/watch_informer.go
//
// Secondary watch informers for operatorBox.watch entries.
//
// When a CRD declares operatorBox.watch, Orkestra sets up a dynamic informer
// for each listed resource. When a watched resource changes, the handler
// resolves the relevant primary CR key(s) and enqueues them — no Go required
// from the constructor author.
//
// Key resolution order (first match wins):
//  1. keyFrom.label  — the watched object has a label whose value is the primary CR key.
//  2. keyFrom.name   — a fixed primary CR name declared in the watch entry.
//  3. ownerReference — the watched object is owned by a primary CR of this CRD.
//  4. broadcast      — none of the above matched; enqueue all known primary CRs.
//     Right for shared resources (ConfigMap, Secret) that affect every CR equally.
package kordinator

import (
	"context"

	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/runtime/queue"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// startWatchInformers creates one dynamic informer per watch: entry on the CRD.
// Called from startCRDWorkers after the worker pool is started.
// Informers run within crdCtx and stop when the primary CRD stops.
func (k *DependencyKordinator) startWatchInformers(ctx context.Context, crd orktypes.CRDEntry) {
	if !crd.WithWatchEntries() {
		return
	}

	primaryGVK := crd.GVKString()
	wq, ok := k.queueReg.For(primaryGVK)
	if !ok {
		logger.Warn().Str("gvk", primaryGVK).Msg("watch: no queue registered for primary CRD — skipping")
		return
	}

	for _, watchEntry := range crd.WatchEntries() {
		watchEntry := watchEntry

		gvr, ok := k.kat.ResolveGVR(watchEntry.ToManagedResource())
		if !ok {
			logger.Warn().
				Str("apiVersion", watchEntry.APIVersion).
				Str("kind", watchEntry.Kind).
				Str("primary", crd.APITypes.Kind).
				Msg("watch: cannot resolve GVR — entry skipped")
			continue
		}

		lw := k.kube.NewDynamicListerWatcher(watchEntryToCRDInfo(watchEntry, gvr), kubeclient.ListOptions{})
		inf := cache.NewSharedIndexInformer(
			lw,
			&unstructured.Unstructured{},
			0, // no resync — primary CRD resync handles re-queuing
			cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
		)

		// captured so each handler can check HasSynced; events fired during the
		// initial List phase (before sync) are dropped — same as controller-runtime.
		localInf := inf
		_, _ = inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if !localInf.HasSynced() || !watchEntry.WatchesOn(string(orktypes.WatchEventCreate)) {
					return
				}
				k.resolveAndEnqueue(obj, watchEntry, crd, primaryGVK, wq)
			},
			UpdateFunc: func(_, newObj interface{}) {
				if !localInf.HasSynced() || !watchEntry.WatchesOn(string(orktypes.WatchEventUpdate)) {
					return
				}
				k.resolveAndEnqueue(newObj, watchEntry, crd, primaryGVK, wq)
			},
			DeleteFunc: func(obj interface{}) {
				if !localInf.HasSynced() || !watchEntry.WatchesOn(string(orktypes.WatchEventDelete)) {
					return
				}
				if ts, ok := obj.(cache.DeletedFinalStateUnknown); ok {
					obj = ts.Obj
				}
				k.resolveAndEnqueue(obj, watchEntry, crd, primaryGVK, wq)
			},
		})

		go inf.Run(ctx.Done())
		logger.Info().
			Str("primary", crd.APITypes.Kind).
			Str("watched", watchEntry.Kind).
			Str("gvr", gvr.String()).
			Msg("watch: secondary informer started")
	}
}

// resolveAndEnqueue resolves the primary CR key(s) from a watched resource event
// and adds them to the primary CRD's workqueue.
//
// Resolution order:
//  1. keyFrom.label  — label on the watched object carries the key
//  2. keyFrom.name   — fixed named primary CR
//  3. ownerReference — owner of the watched object matches the primary CRD
//  4. broadcast      — no match found; enqueue all known primary CRs
func (k *DependencyKordinator) resolveAndEnqueue(obj interface{}, w orktypes.WatchEntry, crd orktypes.CRDEntry, primaryGVK string, wq *queue.Workqueue) {
	u, ok := watchedToUnstructured(obj)
	if !ok {
		return
	}

	primaryKind := crd.APITypes.Kind

	// 1. keyFrom.label — read key from a label on the watched object.
	if kf := w.KeyFrom; kf != nil && kf.Label != "" {
		key, ok := u.GetLabels()[kf.Label]
		if ok && key != "" {
			wq.EnqueueKey(key, primaryGVK)
			logger.Debug().
				Str("primary", primaryKind).
				Str("key", key).
				Str("label", kf.Label).
				Msg("watch: enqueued via keyFrom.label")
			return
		}
		// Label declared but absent on this object — fall through to broadcast.
	}

	// 2. keyFrom.name — fixed primary CR key regardless of which object changed.
	if kf := w.KeyFrom; kf != nil && kf.Name != "" {
		wq.EnqueueKey(kf.Key(), primaryGVK)
		logger.Debug().
			Str("primary", primaryKind).
			Str("key", kf.Key()).
			Msg("watch: enqueued via keyFrom.name")
		return
	}

	// 3. ownerReference — enqueue the specific primary CR if the watched object
	// is owned by one.
	primaryAPIVersion := crd.APIVersion()
	for _, ref := range u.GetOwnerReferences() {
		if ref.APIVersion == primaryAPIVersion && ref.Kind == primaryKind {
			ns := u.GetNamespace()
			key := ref.Name
			if ns != "" {
				key = ns + "/" + ref.Name
			}
			wq.EnqueueKey(key, primaryGVK)
			logger.Debug().
				Str("primary", primaryKind).
				Str("key", key).
				Str("watched", u.GetKind()).
				Msg("watch: enqueued via ownerReference")
			return
		}
	}

	// 4. Broadcast — no specific match; enqueue all known primary CRs.
	registered := k.informerFactory.Registered()
	entry, ok := registered[primaryGVK]
	if !ok || entry == nil {
		return
	}
	for _, item := range entry.Informer.GetIndexer().List() {
		o, ok := item.(metav1.Object)
		if !ok {
			continue
		}
		key, err := cache.MetaNamespaceKeyFunc(o)
		if err != nil {
			continue
		}
		wq.EnqueueKey(key, primaryGVK)
	}
	logger.Debug().
		Str("primary", primaryKind).
		Str("watched", u.GetKind()).
		Msg("watch: broadcasted to all primary CRs")
}

// watchedToUnstructured unwraps a cache tombstone and asserts to *unstructured.Unstructured.
func watchedToUnstructured(obj interface{}) (*unstructured.Unstructured, bool) {
	if ts, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = ts.Obj
	}
	u, ok := obj.(*unstructured.Unstructured)
	return u, ok
}

// watchEntryToCRDInfo converts a WatchEntry + resolved GVR to a kubeclient.CRDInfo
// for NewDynamicListerWatcher. Namespace is set to the entry's declared namespace;
// Namespaced is true when a namespace is declared (restricts the watch to that
// namespace), false for a cluster-scoped watch (all namespaces).
func watchEntryToCRDInfo(w orktypes.WatchEntry, gvr schema.GroupVersionResource) kubeclient.CRDInfo {
	return kubeclient.CRDInfo{
		Group:      gvr.Group,
		Version:    gvr.Version,
		Plural:     gvr.Resource,
		Namespace:  w.Namespace,
		Namespaced: w.Namespace != "",
	}
}
