// pkg/runtime/kordinator/watch_informer.go
//
// Secondary watch informers for operatorBox.watch entries and managed resources.
//
// Two sources produce watch informers:
//
//  1. operatorBox.watch — explicit entries declared by the operator author. Full
//     control: on:, enqueueGate:, keyFrom:, index:.
//
//  2. constructor.resources / hooks.resources — owned resource types. Treated as
//     implicit watch entries: all events, owner-reference key resolution, no index.
//     Mirrors what Owns() does in controller-runtime — cache-backed reads and
//     re-enqueue when an owned resource changes. Explicit watch: entries take
//     priority when the same type appears in both lists.
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
	"strings"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/runtime/queue"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// startWatchInformers creates one dynamic informer per watch: entry on the CRD,
// plus one implicit informer per managed resource (constructor.resources /
// hooks.resources) that is not already covered by an explicit watch: entry.
// Called from startCRDWorkers after the worker pool is started.
// Informers run within crdCtx and stop when the primary CRD stops.
func (k *DependencyKordinator) startWatchInformers(ctx context.Context, crd orktypes.CRDEntry) {
	hasWatch := crd.WithWatchEntries()
	hasResources := crd.WithAnyManagedResources()
	if !hasWatch && !hasResources {
		return
	}

	primaryGVK := crd.GVKString()
	wq, ok := k.queueReg.For(primaryGVK)
	if !ok {
		logger.Warn().Str("gvk", primaryGVK).Msg("watch: no queue registered for primary CRD — skipping")
		return
	}

	// Track GVRs covered by explicit watch: entries so managed resources don't
	// register a second informer for the same type. Explicit watch: takes priority.
	covered := map[string]bool{}
	for _, w := range crd.WatchEntries() {
		gvr, ok := k.kat.ResolveGVR(w.ToManagedResource())
		if ok {
			covered[gvr.String()] = true
		}
	}

	// Explicit watch: entries — full control over events, enqueueGate, keyFrom, index.
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
		k.startOneWatchInformer(ctx, watchEntry, gvr, crd, primaryGVK, wq, true)
	}

	// Implicit watch entries from resources: — all events, owner-reference
	// key resolution only. Explicit watch: entries take priority.
	// broadcastAllowed=false: owned resources with no ownerReference mean nothing to enqueue.
	for _, r := range crd.AllManagedResources() {
		gvr, ok := k.kat.ResolveGVR(r)
		if !ok || covered[gvr.String()] {
			continue
		}
		covered[gvr.String()] = true // deduplicate within the resources list itself
		synth := orktypes.WatchEntry{
			APIVersion: gvr.Group + "/" + gvr.Version,
			Kind:       r.Kind,
		}
		if synth.APIVersion == "/" {
			synth.APIVersion = "v1" // core group
		}
		k.startOneWatchInformer(ctx, synth, gvr, crd, primaryGVK, wq, false)
	}
}

// startOneWatchInformer creates, wires, registers, and starts a single watch
// informer for the given entry + resolved GVR.
func (k *DependencyKordinator) startOneWatchInformer(
	ctx context.Context,
	watchEntry orktypes.WatchEntry,
	gvr schema.GroupVersionResource,
	crd orktypes.CRDEntry,
	primaryGVK string,
	wq *queue.Workqueue,
	broadcastAllowed bool,
) {
	lw := k.kube.NewDynamicListerWatcher(watchEntryToCRDInfo(watchEntry, gvr), kubeclient.ListOptions{})

	// Build indexers: always include namespace; add any user-declared index: entries.
	indexers := cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}
	for _, wi := range watchEntry.Index {
		parts := splitWatchField(wi.Field) // pre-split; capture by value
		indexers[wi.Name] = func(obj interface{}) ([]string, error) {
			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return nil, nil
			}
			val, found, err := unstructured.NestedString(u.Object, parts...)
			if err != nil || !found || val == "" {
				return nil, err
			}
			return []string{val}, nil
		}
	}

	inf := cache.NewSharedIndexInformer(
		lw,
		&unstructured.Unstructured{},
		0, // no resync — primary CRD resync handles re-queuing
		indexers,
	)

	// captured so each handler can check HasSynced; events fired during the
	// initial List phase (before sync) are dropped — same as controller-runtime.
	localInf := inf
	_, _ = inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if !localInf.HasSynced() || !watchEntry.WatchesOn(string(orktypes.WatchEventCreate)) {
				return
			}
			k.resolveAndEnqueue(obj, watchEntry, crd, primaryGVK, wq, broadcastAllowed)
		},
		UpdateFunc: func(_, newObj interface{}) {
			if !localInf.HasSynced() || !watchEntry.WatchesOn(string(orktypes.WatchEventUpdate)) {
				return
			}
			k.resolveAndEnqueue(newObj, watchEntry, crd, primaryGVK, wq, broadcastAllowed)
		},
		DeleteFunc: func(obj interface{}) {
			if !localInf.HasSynced() || !watchEntry.WatchesOn(string(orktypes.WatchEventDelete)) {
				return
			}

			k.resolveAndEnqueue(domain.UnwrapCacheTombstone(obj), watchEntry, crd, primaryGVK, wq, broadcastAllowed)
		},
	})

	// Register in the shared factory so IndexerFor/StoreFor can serve this
	// informer's cache to the kubeclient layer (MatchingFields, cached Get/List).
	watchGVK := schema.GroupVersionKind{Group: gvr.Group, Version: gvr.Version, Kind: watchEntry.Kind}
	k.informerFactory.RegisterInformer(watchGVK, inf)

	go inf.Run(ctx.Done())
	logger.Info().
		Str("primary", crd.APITypes.Kind).
		Str("watched", watchEntry.Kind).
		Str("gvr", gvr.String()).
		Msg("watch: secondary informer started")
}

// resolveAndEnqueue resolves the primary CR key(s) from a watched resource event
// and adds them to the primary CRD's workqueue.
//
// Resolution order:
//  1. keyFrom.label  — label on the watched object carries the key
//  2. keyFrom.name   — fixed named primary CR
//  3. ownerReference — owner of the watched object matches the primary CRD
//  4. broadcast      — no match found; enqueue all known primary CRs
func (k *DependencyKordinator) resolveAndEnqueue(obj interface{}, w orktypes.WatchEntry, crd orktypes.CRDEntry, primaryGVK string, wq *queue.Workqueue, broadcastAllowed bool) {
	u, ok := domain.ToUnstructured(obj)
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
	// Skipped for implicit informers from resources: (broadcastAllowed=false) — an
	// owned resource with no ownerReference has no CR to enqueue.
	if !broadcastAllowed {
		return
	}
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

// splitWatchField converts a dot-separated field path to path segments for
// unstructured.NestedString. Accepts both "spec.owner" and ".spec.owner".
func splitWatchField(field string) []string {
	field = strings.TrimPrefix(field, ".")
	if field == "" {
		return nil
	}
	return strings.Split(field, ".")
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
