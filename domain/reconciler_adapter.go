package domain

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcilerFrom wraps a sigs.k8s.io/controller-runtime/pkg/reconcile.Reconciler
// as a domain.Reconciler so it can be returned from a constructor function without
// any changes to the reconciler body.
//
// Orkestra calls Reconcile(ctx, key) where key is "namespace/name". The adapter
// splits the key and builds a reconcile.Request, then discards the returned
// ctrl.Result — Orkestra's operatorBox owns requeue scheduling via its own
// rate-limiting queue. Return an error to trigger a rate-limited retry as normal.
//
// Usage:
//
//	func NewMyReconciler(kube kubeclient.Interface) domain.Reconciler {
//	    return domain.ReconcilerFrom(&MyReconciler{
//	        client: kubeclient.ToClient(kube),
//	    })
//	}
func ReconcilerFrom(r reconcile.Reconciler) Reconciler {
	return &ctrlReconcilerAdapter{r: r}
}

type ctrlReconcilerAdapter struct {
	r reconcile.Reconciler
}

var _ Reconciler = (*ctrlReconcilerAdapter)(nil)

func (a *ctrlReconcilerAdapter) Reconcile(ctx context.Context, key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	// ctrl.Result is intentionally discarded — Orkestra manages requeue.
	_, err = a.r.Reconcile(ctx, reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: ns, Name: name},
	})
	return err
}
