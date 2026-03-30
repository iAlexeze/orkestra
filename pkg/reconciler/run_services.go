// pkg/reconciler/run_services.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
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
		// 1. Evaluate conditions BEFORE resolving templates
		conditionPassed := evaluateConditions(owner, src.Conditions)

		if !conditionPassed {
			if update || src.Reconcile { // ← src.Reconcile here too to show that this resource is continuously managed
				// Condition no longer passes — delete if owned by this CR
				name, _ := resolver.Resolve(src.Name)
				ns, _ := resolver.Resolve(src.Namespace)
				if ns == "" {
					ns = owner.GetNamespace()
				}
				if err := orksvc.DeleteIfOwned(ctx, kube, owner, name, ns); err != nil {
					return fmt.Errorf("services[%d]: conditional cleanup: %w", i, err)
				}
			}
			logger.FromContext(ctx).Debug().
				Str("resource", "ConfigMap").
				Int("index", i).
				Msg("conditions not met — skipping resource")

			continue
		}

		// 2. Resolve template expressions
		resolved, err := resolver.ResolveServiceTemplate(src)
		if err != nil {
			return fmt.Errorf("services[%d]: %w", i, err)
		}

		// 3. Build registry spec and apply
		spec := orksvc.Resolve(resolved, resolver.OwnerName())

		if update {
			if err := orksvc.Update(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("services[%d].update: %w", i, err)
			}
		} else {
			if err := orksvc.Create(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("services[%d].create: %w", i, err)
			}

			// reconcile: true
			if src.Reconcile {
				if err := orksvc.Update(ctx, kube, owner, spec); err != nil {
					return fmt.Errorf("services[%d].reconcile: %w", i, err)
				}
			}
		}
	}
	return nil
}
