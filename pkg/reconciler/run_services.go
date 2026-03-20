// pkg/reconciler/run_services.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	orksvc "github.com/ialexeze/orkestra/pkg/orkestra-registry/services"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// runServices resolves and applies Service template declarations.
// Same update/reconcile: true semantics as runDeployments.
func runServices(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	resolver *orktmpl.Resolver,
	owner domain.Object,
	srcs []orktypes.ServiceTemplateSource,
	update bool,
) error {
	for i, src := range srcs {
		resolved, err := resolver.ResolveServiceTemplate(src)
		if err != nil {
			return fmt.Errorf("services[%d]: %w", i, err)
		}

		spec := orksvc.Resolve(resolved, resolver.OwnerName())

		if update {
			if err := orksvc.Update(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("services[%d].update: %w", i, err)
			}
		} else {
			if err := orksvc.Create(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("services[%d].create: %w", i, err)
			}
			if src.Reconcile {
				if err := orksvc.Update(ctx, kube, owner, spec); err != nil {
					return fmt.Errorf("services[%d].reconcile: %w", i, err)
				}
			}
		}
	}
	return nil
}
