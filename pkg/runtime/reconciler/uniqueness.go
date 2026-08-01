// pkg/runtime/reconciler/uniqueness.go
package reconciler

import (
	"context"
	"fmt"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// dynamicClientProvider is the minimal capability liveUniquenessChecker
// needs — narrower than kubeclient.KubeClient (which GenericReconciler's
// r.kube satisfies structurally) so tests can supply a fake dynamic client
// without implementing the full interface (Clientset, Mapper, RestConfig, …).
type dynamicClientProvider interface {
	DynamicClient() dynamic.Interface
}

// liveUniquenessChecker implements orktypes.UniquenessChecker via a live List
// call against the API server for the CRD currently being reconciled. A live
// call rather than the informer cache is deliberate — uniqueness is exactly
// the kind of check where a stale cache could let two concurrent duplicates
// both pass.
type liveUniquenessChecker struct {
	ctx        context.Context
	kube       dynamicClientProvider
	gvr        schema.GroupVersionResource
	namespaced bool
}

// newUniquenessChecker builds the checker injected into every reconcile via
// template.Resolver.WithUniquenessChecker, so operator: unique has live CRD
// access in both validation.rules and when:/anyOf:.
func newUniquenessChecker(ctx context.Context, kube dynamicClientProvider, gvr schema.GroupVersionResource, namespaced bool) orktypes.UniquenessChecker {
	return &liveUniquenessChecker{ctx: ctx, kube: kube, gvr: gvr, namespaced: namespaced}
}

// IsUnique lists every existing instance of the CRD and reports whether none
// of them (other than the CR under evaluation itself) has field == value.
// Uniqueness is checked across all namespaces for namespaced CRDs — a field
// like spec.domain is typically meant to be globally unique, not merely
// unique per namespace.
func (u *liveUniquenessChecker) IsUnique(field, value, selfNamespace, selfName string) (bool, error) {
	namespaceable := u.kube.DynamicClient().Resource(u.gvr)

	var resource dynamic.ResourceInterface = namespaceable
	if u.namespaced {
		resource = namespaceable.Namespace(metav1.NamespaceAll)
	}

	list, err := resource.List(u.ctx, metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("listing %s for uniqueness check: %w", u.gvr.Resource, err)
	}

	for _, item := range list.Items {
		if item.GetNamespace() == selfNamespace && item.GetName() == selfName {
			continue // never a duplicate of its own stored value
		}
		val, found := orktypes.ResolveScalarField(item.Object, field)
		if found && val == value {
			return false, nil
		}
	}
	return true, nil
}
