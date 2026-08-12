package kordinator

import (
	"context"
	"fmt"

	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
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

// evaluatePreReconcileCheck evaluates the gate conditions declared in
// reconcile.when / reconcile.anyOf for the given CR object.
//
// Returns (true, reason) when the item should be discarded — the reconciler
// must not be called. Returns (false, "") when conditions pass.
//
// The resolver mirrors the full enrichment chain of GenericReconciler.reconcileCore:
// WithProfiles, WithUserNotes, WithRequest (serve intent). The one omission is
// WithUniquenessChecker — it issues a live API server call and violates the
// gate's no-API-call constraint.
func (k *Kontroller) evaluatePreReconcileCheck(
	ctx context.Context,
	obj *unstructured.Unstructured,
	entry RegistryEntry,
	rc *orktypes.PreReconcileConfig,
) (gated bool, reason string) {
	if obj == nil {
		return false, ""
	}

	resolver, err := orktmpl.NewResolver(ctx, obj)
	if err != nil {
		return false, ""
	}

	if k.kat != nil {
		if !k.kat.Profiles.IsEmpty() {
			resolver = resolver.WithProfiles(k.kat.Profiles)
		}
		if !k.kat.Notes.IsEmpty() {
			resolver = resolver.WithUserNotes(k.kat.Notes)
		}
	}

	// Inject serve intent as .request.<field> when the CR was submitted via
	// the Gateway — same enrichment the reconciler applies at line 314 of generic.go.
	if intent := orktypes.ServeIntentFromObject(resolver.Data()); intent != nil {
		resolver = resolver.WithRequest(intent)
	}

	eval := resolver.TemplateEvaluator()

	if !orktypes.EvaluateWhen(resolver.Data(), rc.WhenConditions(), rc.AnyOfConditions(), eval) {
		return true, gateReason(rc, resolver)
	}

	return false, ""
}

// gateReason builds a human-readable reason string for the first failing condition.
func gateReason(rc *orktypes.PreReconcileConfig, resolver *orktmpl.Resolver) string {
	eval := resolver.TemplateEvaluator()
	for _, cond := range rc.WhenConditions() {
		if !orktypes.EvaluateWhen(resolver.Data(), []orktypes.Condition{cond}, nil, eval) {
			val, _ := resolver.Resolve(cond.Field)
			return fmt.Sprintf("when: %q = %q, want %q", cond.Field, val, cond.Equals)
		}
	}
	return "anyOf: no condition satisfied"
}

// resolveInformerKey extracts the cache key for a CR from a cache.SharedIndexInformer.
// Uses the same MetaNamespaceKeyFunc the workqueue uses when enqueuing events.
func resolveInformerKey(obj interface{}) (string, error) {
	return cache.MetaNamespaceKeyFunc(obj)
}
