// pkg/kubeclient/context.go
package kubeclient

import "context"

// contextKey is an unexported type for context keys in this package.
type contextKey string

// ContextKey is the key used to store and retrieve a KubeClient from a context.
//
// Usage — inject before calling hooks:
//
//	ctx = kubeclient.WithKubeclient(ctx, kube)
//
// Usage — retrieve in hook/registry functions:
//
//	kube, ok := kubeclient.FromContext(ctx)
const ContextKey contextKey = "orkestra-kubeclient"

// WithKubeclient returns a new context with the KubeClient stored under ContextKey.
// Called in GenericReconciler.Reconcile before invoking hook and registry functions.
func WithKubeclient(ctx context.Context, kube KubeClient) context.Context {
	return context.WithValue(ctx, ContextKey, kube)
}

// FromContext retrieves the KubeClient from a context.
// Returns (nil, false) if not present.
func FromContext(ctx context.Context) (KubeClient, bool) {
	kube, ok := ctx.Value(ContextKey).(KubeClient)
	return kube, ok && kube != nil
}
