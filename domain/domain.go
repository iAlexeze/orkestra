package domain

import "context"

type Komponent interface {

	// Start() starts the compnent
	Start(context.Context) error

	// Shutdown() shuts down the component gracefully
	Shutdown(context.Context)

	// Name() returns the name of the component
	Name() string

	// Started() is set when manager starts a component
	Started() bool
}

type Reconciler interface {
	// Reconcile handles the actual business logic for a resource
	Reconcile(ctx context.Context, key string) error
}
