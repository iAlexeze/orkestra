// reconciler/hooks/project_hooks.go
package hooks

import (
	"context"

	projectv1 "github.com/ialexeze/orkestra/api/types/project/v1alpha1"
	"github.com/ialexeze/orkestra/domain"
)

func ProjectHooks() domain.ReconcileHooks[*projectv1.Project] {
	return domain.ReconcileHooks[*projectv1.Project]{

		OnReconcile: func(ctx context.Context, obj *projectv1.Project) error {
			// custom logic — ensure a ConfigMap exists, call an external API, etc.
			return nil
		},

		OnDelete: func(ctx context.Context, obj *projectv1.Project) error {
			// cleanup before finalizer is removed
			return nil
		},

		// OnNotFound left nil — default no-op is fine
	}
}
