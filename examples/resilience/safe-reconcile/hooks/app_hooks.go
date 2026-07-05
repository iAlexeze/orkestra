//go:build ignore

// hooks/app_hooks.go
//
// This hook contains an intentional nil pointer dereference.
//
// obj.Spec.Config is an optional pointer field. The CR in cr-app.yaml does not
// set it — so it is nil at reconcile time. Dereferencing it panics.
//
// Without safeReconcile, this panic would crash the operator process and take
// Monitor and Queue (the two declarative CRDs) down with it. With safeReconcile,
// Orkestra catches the panic, logs the full stack trace, records the failure in
// health state and metrics, and continues processing the next item in the queue.
// Monitor and Queue keep reconciling without interruption.
//
// This is the isolation boundary Orkestra guarantees.
package hooks

import (
	"context"

	"github.com/orkspace/orkestra/domain"
	appv1 "github.com/orkspace/safe-reconcile-demo/api/v1alpha1"
)

// AppHooks returns the hook implementation for the App CRD.
func AppHooks() domain.AnyReconcileHooks {
	return domain.ReconcileHooks[*appv1.App]{
		OnReconcile: onAppReconcile,
	}
}

// onAppReconcile runs on every reconcile cycle for an App CR.
//
// BUG (intentional): obj.Spec.Config is nil because the CR does not set it.
// This dereference panics. safeReconcile catches it — the operator keeps running.
func onAppReconcile(_ context.Context, obj *appv1.App) error {
	// obj.Spec.Config is nil — the CR omits the optional config field.
	// A nil pointer dereference panics here.
	_ = obj.Spec.Config.Endpoint
	return nil
}
