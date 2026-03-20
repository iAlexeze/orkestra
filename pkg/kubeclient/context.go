// pkg/kubeclient/context.go
package kubeclient

import "context"

// contextKey is an unexported type for context keys in this package.
// Using a dedicated type prevents collisions with keys from other packages.
type contextKey string

// ContextKey is the key used to store and retrieve a *Kubeclient from a context.
// Used by OrkestraRegistry hook implementations to access the kube client
// without changing the domain.ReconcileHooks function signatures.
//
// Usage — inject before calling hooks:
//
//	ctx = kubeclient.WithKubeclient(ctx, kube)
//
// Usage — retrieve in generated hooks:
//
//	kube, ok := ctx.Value(kubeclient.ContextKey).(*kubeclient.Kubeclient)
//	if !ok || kube == nil {
//	    return fmt.Errorf("kubeclient not found in context")
//	}
const ContextKey contextKey = "orkestra-kubeclient"

// WithKubeclient returns a new context with the Kubeclient stored under ContextKey.
// Example usage:
// This is called in GenericReconciler.Reconcile before invoking hook functions.
// To allow hooks retrieve the kubeclient
func WithKubeclient(ctx context.Context, kube *Kubeclient) context.Context {
	return context.WithValue(ctx, ContextKey, kube)
}

// FromContext retrieves the Kubeclient from a context.
// Returns (nil, false) if not present.
func FromContext(ctx context.Context) (*Kubeclient, bool) {
	kube, ok := ctx.Value(ContextKey).(*Kubeclient)
	return kube, ok && kube != nil
}
