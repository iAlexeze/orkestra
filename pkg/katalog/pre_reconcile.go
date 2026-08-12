package katalog

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/external"
	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
)

// EvaluatePreReconcile evaluates preReconcile.reconcileGate conditions for the named CRD.
// Returns (true, "") when conditions pass and the reconciler should run.
// Returns (false, reason) when gated — reconciler must not be called.
//
// preReconcile.external runs first (shared enrichment), then reconcileGate.external,
// then conditions are evaluated against the accumulated resolver.
func (k *Katalog) EvaluatePreReconcile(ctx context.Context, crdName string, obj *unstructured.Unstructured, cs kubernetes.Interface) (allowed bool, reason string) {
	if obj == nil {
		return true, ""
	}
	entry, ok := k.CRDEntry(crdName)
	if !ok {
		return true, ""
	}
	rc := entry.PreReconcileCheck()
	if !rc.HasReconcileGate() {
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

	gvk := entry.GVKString()

	if rc.HasPreReconcileExternal() {
		if resolver, err = external.Run(ctx, gvk, resolver, rc.External, cs); err != nil {
			return true, ""
		}
	}
	if rc.HasReconcileGateExternal() {
		if resolver, err = external.Run(ctx, gvk, resolver, rc.ReconcileGate.External, cs); err != nil {
			return true, ""
		}
	}

	eval := resolver.TemplateEvaluator()
	if !orktypes.EvaluateConditions(resolver.Data(), rc.WhenConditions(), rc.AnyOfConditions(), eval) {
		return false, preReconcileGateReason(rc, resolver)
	}
	return true, ""
}

// EvaluateEnqueueFilter evaluates preReconcile.enqueueGate conditions for the named CRD.
// Returns true when the object should be enqueued, false when it should be dropped.
// Accepts domain.Object so it works for both dynamic and typed CRDs.
//
// preReconcile.external runs first (shared enrichment), then enqueueGate.external,
// then conditions are evaluated against the accumulated resolver.
func (k *Katalog) EvaluateEnqueueFilter(ctx context.Context, crdName string, obj domain.Object, cs kubernetes.Interface) bool {
	if obj == nil {
		return true
	}
	entry, ok := k.CRDEntry(crdName)
	if !ok {
		return true
	}
	rc := entry.PreReconcileCheck()
	if !rc.HasEnqueueGate() {
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

	gvk := entry.GVKString()

	if rc.HasPreReconcileExternal() {
		if resolver, err = external.Run(ctx, gvk, resolver, rc.External, cs); err != nil {
			return true
		}
	}
	g := rc.EnqueueGate
	if rc.HasEnqueueGateExternal() {
		if resolver, err = external.Run(ctx, gvk, resolver, g.External, cs); err != nil {
			return true
		}
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
