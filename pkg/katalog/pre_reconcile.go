package katalog

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/domain"
	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// EvaluatePreReconcile evaluates preReconcile.when/anyOf for the named CRD.
// Returns (true, "") when conditions pass and the reconciler should run.
// Returns (false, reason) when gated — reconciler must not be called.
//
// Uses the full resolver chain: profiles, notes, serve intent — mirrors
// the enrichment in GenericReconciler.reconcileCore.
func (k *Katalog) EvaluatePreReconcile(ctx context.Context, crdName string, obj *unstructured.Unstructured) (allowed bool, reason string) {
	if obj == nil {
		return true, ""
	}
	entry, ok := k.CRDEntry(crdName)
	if !ok {
		return true, ""
	}
	rc := entry.PreReconcileCheck()
	if !rc.HasConditions() {
		return true, ""
	}

	resolver, err := orktmpl.NewResolver(ctx, obj)
	if err != nil {
		return true, ""
	}
	if !k.Profiles.IsEmpty() {
		resolver = resolver.WithProfiles(k.Profiles)
	}
	if !k.Notes.IsEmpty() {
		resolver = resolver.WithUserNotes(k.Notes)
	}
	if intent := orktypes.ServeIntentFromObject(resolver.Data()); intent != nil {
		resolver = resolver.WithRequest(intent)
	}

	eval := resolver.TemplateEvaluator()
	if !orktypes.EvaluateConditions(resolver.Data(), rc.WhenConditions(), rc.AnyOfConditions(), eval) {
		return false, preReconcileGateReason(rc, resolver)
	}
	return true, ""
}

// EvaluateEnqueueFilter evaluates preReconcile.enqueueGate.when/anyOf for the named CRD.
// Returns true when the object should be enqueued, false when it should be dropped.
// Accepts domain.Object so it works for both dynamic and typed CRDs — the template
// resolver handles the conversion via objectToMap internally.
func (k *Katalog) EvaluateEnqueueFilter(ctx context.Context, crdName string, obj domain.Object) bool {
	if obj == nil {
		return true
	}
	entry, ok := k.CRDEntry(crdName)
	if !ok {
		return true
	}
	g := entry.PreReconcileCheck().EnqueueGate
	if !g.HasConditions() {
		return true
	}

	resolver, err := orktmpl.NewResolver(ctx, obj)
	if err != nil {
		return true
	}
	if !k.Profiles.IsEmpty() {
		resolver = resolver.WithProfiles(k.Profiles)
	}
	if !k.Notes.IsEmpty() {
		resolver = resolver.WithUserNotes(k.Notes)
	}
	if intent := orktypes.ServeIntentFromObject(resolver.Data()); intent != nil {
		resolver = resolver.WithRequest(intent)
	}

	eval := resolver.TemplateEvaluator()
	return orktypes.EvaluateConditions(resolver.Data(), g.WhenConditions(), g.AnyOfConditions(), eval)
}

// preReconcileGateReason returns a human-readable description of why the gate fired.
func preReconcileGateReason(rc *orktypes.PreReconcileConfig, resolver *orktmpl.Resolver) string {
	eval := resolver.TemplateEvaluator()
	for _, cond := range rc.WhenConditions() {
		if !orktypes.EvaluateConditions(resolver.Data(), []orktypes.Condition{cond}, nil, eval) {
			val, _ := resolver.Resolve(cond.Field)
			return fmt.Sprintf("when: %q = %q, want %q", cond.Field, val, cond.Equals)
		}
	}
	return "anyOf: no condition satisfied"
}
