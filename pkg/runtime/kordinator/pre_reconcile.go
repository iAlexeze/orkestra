package kordinator

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
)

// objectFromCache retrieves the live CR from the informer cache for the given
// key. Returns nil if the entry has no informer or the key is not found.
func (k *Kontroller) objectFromCache(entry RegistryEntry, key string) *unstructured.Unstructured {
	if entry.Informer == nil {
		return nil
	}
	raw, exists, err := entry.Informer.GetIndexer().GetByKey(key)
	if err != nil || !exists || raw == nil {
		return nil
	}
	obj, ok := raw.(*unstructured.Unstructured)
	if !ok {
		return nil
	}
	return obj
}

// evaluatePreReconcileCheck evaluates the preReconcile.when/anyOf gate for the
// given CR. Returns (true, reason) when gated — reconciler must not be called.
// Returns (false, "") when conditions pass.
//
// Delegates to k.kat.EvaluatePreReconcile which holds the full resolver chain
// (profiles, notes, serve intent) — no duplication of eval logic here.
func (k *Kontroller) evaluatePreReconcileCheck(
	ctx context.Context,
	obj *unstructured.Unstructured,
	crdName string,
) (gated bool, reason string) {
	if k.kat == nil || obj == nil {
		return false, ""
	}
	allowed, reason := k.kat.EvaluatePreReconcile(ctx, crdName, obj, k.kube.Clientset())
	return !allowed, reason
}

// resolveInformerKey extracts the cache key for a CR from a cache.SharedIndexInformer.
// Uses the same MetaNamespaceKeyFunc the workqueue uses when enqueuing events.
func resolveInformerKey(obj interface{}) (string, error) {
	return cache.MetaNamespaceKeyFunc(obj)
}
