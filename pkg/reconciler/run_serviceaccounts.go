// pkg/reconciler/run_serviceaccounts.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	orksa "github.com/ialexeze/orkestra/pkg/orkestra-registry/serviceaccounts"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// runServiceAccounts resolves and applies ServiceAccount template declarations.
//
// ServiceAccounts are create-only — there is nothing meaningful to update
// on a ServiceAccount after creation (no spec fields that drift). They are
// therefore always idempotent creates regardless of whether this is called
// from onCreate or onReconcile.
//
// Owner references ensure cleanup when the CR is deleted.
func runServiceAccounts(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	resolver *orktmpl.Resolver,
	owner domain.Object,
	srcs []orktypes.ServiceAccountTemplateSource,
) error {
	for i, src := range srcs {
		resolved, err := resolver.ResolveServiceAccountTemplate(src)
		if err != nil {
			return fmt.Errorf("serviceaccounts[%d]: %w", i, err)
		}

		spec := orksa.Resolve(resolved, resolver.OwnerName())

		// Always create — ServiceAccounts have no meaningful drift to correct
		if err := orksa.Create(ctx, kube, owner, spec); err != nil {
			return fmt.Errorf("serviceaccounts[%d].create: %w", i, err)
		}
	}
	return nil
}
