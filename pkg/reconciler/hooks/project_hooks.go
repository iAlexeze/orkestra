// reconciler/hooks/project_hooks.go
package hooks

import (
	"context"
	"fmt"

	projectv1 "github.com/ialexeze/orkestra/api/types/project/v1alpha1"
	"github.com/ialexeze/orkestra/domain"
)

func ProjectHooks() domain.ReconcileHooks[domain.Object] {
	return domain.ReconcileHooks[domain.Object]{
		OnReconcile: func(ctx context.Context, obj domain.Object) error {
			project, ok := obj.(*projectv1.Project)
			if !ok {
				return fmt.Errorf("expected *Project, got %T", obj)
			}
			// your logic here — project is correctly typed
			_ = project
			return nil
		},
		OnDelete: func(ctx context.Context, obj domain.Object) error {
			project, ok := obj.(*projectv1.Project)
			if !ok {
				return fmt.Errorf("expected *Project, got %T", obj)
			}
			_ = project
			return nil
		},
	}
}
