package simulate

import (
	"context"

	"github.com/orkspace/orkestra/domain"
	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// gatedReconciler wraps a domain.Reconciler and evaluates preReconcile
// conditions before delegating. When the gate fires the reconcile is a no-op —
// same behaviour as the kordinator gate in a live cluster.
type gatedReconciler struct {
	inner  domain.Reconciler
	gate   *orktypes.PreReconcileConfig
	getObj func() *unstructured.Unstructured
}

func (g *gatedReconciler) Reconcile(ctx context.Context, key string) error {
	if g.gate == nil {
		return g.inner.Reconcile(ctx, key)
	}
	obj := g.getObj()
	if obj == nil {
		return g.inner.Reconcile(ctx, key)
	}
	resolver, err := orktmpl.NewResolver(ctx, obj)
	if err != nil {
		return g.inner.Reconcile(ctx, key)
	}
	eval := resolver.TemplateEvaluator()

	// Evaluate preReconcile.enqueueGate first — mirrors informer-level drop in live path.
	if g.gate.EnqueueGate.HasConditions() {
		if !orktypes.EvaluateWhen(resolver.Data(), g.gate.EnqueueGate.WhenConditions(), g.gate.EnqueueGate.AnyOfConditions(), eval) {
			return nil // filtered — skip, no error
		}
	}

	// Evaluate preReconcile.when/anyOf — mirrors kordinator gate in live path.
	if g.gate.HasConditions() {
		if !orktypes.EvaluateWhen(resolver.Data(), g.gate.WhenConditions(), g.gate.AnyOfConditions(), eval) {
			return nil // gated — skip, no error
		}
	}

	return g.inner.Reconcile(ctx, key)
}

// wrapWithGate returns r wrapped with a preReconcile gate check if the CRD
// declares any filter or when/anyOf conditions; otherwise returns r unchanged.
func wrapWithGate(r domain.Reconciler, gate *orktypes.PreReconcileConfig, getObj func() *unstructured.Unstructured) domain.Reconciler {
	if gate == nil || (!gate.HasConditions() && !gate.EnqueueGate.HasConditions()) {
		return r
	}
	return &gatedReconciler{inner: r, gate: gate, getObj: getObj}
}
