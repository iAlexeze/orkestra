// pkg/reconciler/run_deployments.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	orkdeploy "github.com/orkspace/orkestra/pkg/orkestra-registry/deployments"
	orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// runDeployments resolves and applies Deployment template declarations.
//
// update=false  onCreate path  — idempotent Create
// update=true   onReconcile path — Update for drift correction
//
// reconcile: true on an onCreate entry means also call Update on that
// same reconcile loop — the shorthand for "create it and keep it in sync"
// without a separate onReconcile declaration.
func runDeployments(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	resolver *orktmpl.Resolver,
	owner domain.Object,
	srcs []orktypes.DeploymentTemplateSource,
	update bool,
	guard func(ctx context.Context, obj domain.Object, ns string) bool,
) error {
	activeNames := make(map[string]bool, len(srcs))
	for _, s := range srcs {
		if !orktypes.EvaluateWhen(resolver.Data(), s.Conditions, s.AnyOf) {
			continue
		}
		n, _ := resolver.Resolve(s.Name)
		nsp, _ := resolver.Resolve(s.Namespace)
		if nsp == "" {
			nsp = owner.GetNamespace()
		}
		activeNames[nsp+"/"+n] = true
	}

	for i, src := range srcs {

		// 1. Evaluate conditions BEFORE resolving templates
		conditionPassed := orktypes.EvaluateWhen(resolver.Data(), src.Conditions, src.AnyOf)

		// Early name/ns resolution — needed for guard check and DeleteIfOwned cleanup.
		// ResolveDeploymentTemplate resolves these again internally — intentional, cheap.
		name, _ := resolver.Resolve(src.Name)
		ns, _ := resolver.Resolve(src.Namespace)
		if ns == "" {
			ns = owner.GetNamespace()
		}

		// ── Namespace guard ───────────────────────────────────────────────────
		if guard != nil && !guard(ctx, owner, ns) {
			continue // skipped — CheckNamespace already logged the reason
		}

		if !conditionPassed {
			if update || src.Reconcile {
				if !activeNames[ns+"/"+name] {
					if err := orkdeploy.DeleteIfOwned(ctx, kube, owner, name, ns); err != nil {
						return fmt.Errorf("deployments[%d]: conditional cleanup: %w", i, err)
					}
				}
			}
			logger.FromContext(ctx).Debug().
				Str("resource", "Deployment").
				Int("index", i).
				Msg("conditions not met — skipping resource")

			continue
		}

		// 2. Resolve template expressions
		resolved, err := resolver.ResolveDeploymentTemplate(src)
		if err != nil {
			return fmt.Errorf("deployments[%d]: %w", i, err)
		}

		staticReplicas := 1
		if resolved.Replicas != "" {
			fmt.Sscanf(resolved.Replicas, "%d", &staticReplicas)
		}

		// 3. Build registry spec and apply
		spec := orkdeploy.Resolve(resolved, staticReplicas, resolver.OwnerName())

		if update {
			if err := orkdeploy.Update(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("deployments[%d].update: %w", i, err)
			}
		} else {
			if err := orkdeploy.Create(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("deployments[%d].create: %w", i, err)
			}

			// reconcile: true
			if src.Reconcile {
				if err := orkdeploy.Update(ctx, kube, owner, spec); err != nil {
					return fmt.Errorf("deployments[%d].reconcile: %w", i, err)
				}
			}
		}
	}
	return nil
}
