// domain/generic.go
package domain

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

// ReconcileHooks contains the user-provided business logic.
// Only implement the hooks you need — all are optional.
type ReconcileHooks[T runtime.Object] struct {
	// OnReconcile is called for every create/update event.
	// obj is already type-asserted and deep-copied.
	// Return an error to requeue with backoff.
	OnReconcile func(ctx context.Context, obj T) error

	// OnDelete is called when the object's DeletionTimestamp is set.
	// Use this to clean up external resources before the finalizer is removed.
	// If nil, deletion is a no-op (finalizer removed automatically).
	OnDelete func(ctx context.Context, obj T) error

	// OnNotFound is called when the object no longer exists in the store.
	// Use this for cleanup that depends on the key but not the object.
	// If nil, not-found is a no-op.
	OnNotFound func(ctx context.Context, key string) error
}

// AnyReconcileHooks is the type-erased form of ReconcileHooks[T].
// GenericReconciler holds the concrete typed version internally.
// This exists only so the Katalog can store hooks without knowing T.
type AnyReconcileHooks interface {
	isHooks() // unexported marker — prevents accidental implementation
}

func (r ReconcileHooks[T]) isHooks() {} // satisfies AnyReconcileHooks
