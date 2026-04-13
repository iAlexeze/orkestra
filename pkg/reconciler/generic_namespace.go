// pkg/reconciler/generic_namespace.go
//
// Namespace guard wiring for GenericReconciler.
//
// RestrictedNamespaces and AllowedNamespaces live on orktypes.CRDEntry
// directly — not on ReconcilerConfig. GenericReconciler stores the full
// CRDEntry as r.crd, so the guard reads r.crd.RestrictedNamespaces and
// r.crd.AllowedNamespaces directly.
//
// namespaceGuardFunc is called once per runResourceGroup invocation and
// returns a closure. run_*.go files call it after resolving the target
// namespace and before condition evaluation.
//
// Fast path: returns nil when CRD has no restrictions — all run_*.go
// guard with "if guard != nil" so nil means zero allocation, zero check.
package reconciler

import (
	"context"

	"github.com/ialexeze/orkestra/domain"
)

// namespaceGuardFunc returns a guard closure pre-bound to this CRD's
// namespace restrictions, or nil when no restrictions are configured.
func (r *GenericReconciler[T]) namespaceGuardFunc(
	ctx context.Context,
	obj domain.Object,
) func(ctx context.Context, obj domain.Object, ns string) bool {
	// Fields are on CRDEntry directly — set during Katalog validation
	restricted := r.crd.RestrictedNamespaces
	allowed := r.crd.AllowedNamespaces

	if len(restricted) == 0 && len(allowed) == 0 {
		return nil // fast path — no restrictions, no closure allocation
	}

	kind := r.crd.APITypes.Kind

	return func(ctx context.Context, obj domain.Object, ns string) bool {
		result := CheckNamespace(ctx, obj, ns, restricted, allowed, kind)
		return result.Allowed
	}
}
