// pkg/reconciler/run_namespace.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	orkns "github.com/orkspace/orkestra/pkg/orkestra-registry/namespaces"
	orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// deleteOwnedNamespaces explicitly deletes all Namespaces declared across
// onCreate and onReconcile that are owned by this CR.
//
// Kubernetes GC does not handle this automatically: owner references from
// namespace-scoped resources (CRs) to cluster-scoped resources (Namespaces)
// are not honoured by the garbage collector. Explicit deletion is required.
func deleteOwnedNamespaces(
	ctx context.Context,
	kube kubeclient.KubeClient,
	resolver *orktmpl.Resolver,
	obj domain.Object,
	box orktypes.OperatorBoxConfig,
) error {
	// Collect all declared namespace sources across template blocks.
	var srcs []orktypes.NamespaceTemplateSource
	if box.OnCreate != nil {
		srcs = append(srcs, box.OnCreate.Namespaces...)
	}
	if box.OnReconcile != nil {
		srcs = append(srcs, box.OnReconcile.Namespaces...)
	}
	if box.OnDelete != nil {
		srcs = append(srcs, box.OnDelete.Namespaces...)
	}

	for i, src := range srcs {
		name, err := resolver.Resolve(src.Name)
		if err != nil || name == "" {
			continue
		}
		if err := orkns.DeleteIfOwned(ctx, kube, obj, name); err != nil {
			return fmt.Errorf("namespace[%d] %q: %w", i, name, err)
		}
	}
	return nil
}

// runNamespaces resolves and applies Namespace template declarations.
//
// Namespaces are create-only — there is nothing meaningful to update
// on a Namespace after creation (no spec fields that drift). They are
// therefore always idempotent creates regardless of whether this is called
// from onCreate or onReconcile.
//
// Owner references ensure cleanup when the CR is deleted.
func runNamespaces(
	ctx context.Context,
	kube kubeclient.KubeClient,
	resolver *orktmpl.Resolver,
	owner domain.Object,
	srcs []orktypes.NamespaceTemplateSource,
	update bool,
) error {
	activeNames := make(map[string]bool, len(srcs))
	for _, s := range srcs {
		if !orktypes.EvaluateWhen(resolver.Data(), s.Conditions, s.AnyOf) {
			continue
		}
		n, _ := resolver.Resolve(s.Name)
		activeNames["orkestra.io"+"/"+n] = true
	}

	for i, src := range srcs {
		// 1. Evaluate conditions BEFORE resolving templates
		conditionPassed := orktypes.EvaluateWhen(resolver.Data(), src.Conditions, src.AnyOf)

		// Early name resolution — needed for DeleteIfOwned cleanup.
		name, _ := resolver.Resolve(src.Name)

		if !conditionPassed {
			if update || src.Reconcile {
				if !activeNames["orkestra.io"+"/"+name] {
					if err := orkns.DeleteIfOwned(ctx, kube, owner, name); err != nil {
						return fmt.Errorf("namespace[%d]: conditional cleanup: %w", i, err)
					}
				}
			}
			logger.FromContext(ctx).Debug().
				Str("resource", "Namespace").
				Int("index", i).
				Msg("conditions not met — skipping resource")

			continue
		}

		// 2. Resolve template expressions
		resolved, err := resolver.ResolveNamespaceTemplate(src)
		if err != nil {
			return fmt.Errorf("namespace[%d]: %w", i, err)
		}

		// 3. Build registry spec and apply
		spec := orkns.Resolve(resolved, resolver.OwnerName())

		// Always create — Namespaces have no meaningful drift to correct
		if err := orkns.Create(ctx, kube, owner, spec); err != nil {
			return fmt.Errorf("namespace[%d].create: %w", i, err)
		}
	}
	return nil
}
