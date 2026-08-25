package domain

import "context"

type Komponent interface {

	// Start() starts the komponents
	Start(context.Context) error

	// Shutdown() shuts down the komponent gracefully
	Shutdown(context.Context)

	// Name() returns the name of the komponent
	Name() string

	// Started() is set when manager starts a komponent
	Started() bool
}

type Reconciler interface {
	// Reconcile handles the actual business logic for a resource.
	// Return a non-zero Result.RequeueAfter to schedule a precise re-enqueue
	// after a successful reconcile. Return an error to trigger retryBackoff.
	Reconcile(ctx context.Context, req Request) (Result, error)
}
