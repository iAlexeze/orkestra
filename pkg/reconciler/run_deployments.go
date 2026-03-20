// pkg/reconciler/run_deployments.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	orkdeploy "github.com/ialexeze/orkestra/pkg/orkestra-registry/deployments"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
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
) error {
	for i, src := range srcs {
		resolved, err := resolver.ResolveDeploymentTemplate(src)
		if err != nil {
			return fmt.Errorf("deployments[%d]: %w", i, err)
		}

		staticReplicas := 1
		if resolved.Replicas != "" {
			fmt.Sscanf(resolved.Replicas, "%d", &staticReplicas)
		}

		spec := orkdeploy.Resolve(resolved, staticReplicas, resolver.OwnerName())

		if update {
			if err := orkdeploy.Update(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("deployments[%d].update: %w", i, err)
			}
		} else {
			if err := orkdeploy.Create(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("deployments[%d].create: %w", i, err)
			}
			if src.Reconcile {
				if err := orkdeploy.Update(ctx, kube, owner, spec); err != nil {
					return fmt.Errorf("deployments[%d].reconcile: %w", i, err)
				}
			}
		}
	}
	return nil
}
