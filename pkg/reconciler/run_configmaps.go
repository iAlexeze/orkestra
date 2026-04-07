// pkg/reconciler/run_configmaps.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	orkcm "github.com/ialexeze/orkestra/pkg/orkestra-registry/configmaps"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// runConfigMaps resolves and applies ConfigMap template declarations.
//
// ConfigMaps support the same fromConfigMap and toNamespaces patterns as Secrets:
//
// fromConfigMap — copies data from an existing ConfigMap, with declared
// data entries overriding matching keys from the source. This is the
// "base config + environment override" pattern.
//
// toNamespaces — creates one copy in each listed namespace.
//
// reconcile: true — re-reads the source ConfigMap on every reconcile
// and syncs any changes. When logLevel changes in the CR spec, the
// ConfigMap in every namespace updates automatically.
func runConfigMaps(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	resolver *orktmpl.Resolver,
	owner domain.Object,
	srcs []orktypes.ConfigMapTemplateSource,
	update bool,
) error {
	for i, src := range srcs {
		// 1. Evaluate conditions BEFORE resolving templates
		conditionPassed := orktypes.EvaluateWhen(resolver.Data(), src.Conditions, src.AnyOf)

		if !conditionPassed {
			if update || src.Reconcile { // ← src.Reconcile here too to show that this resource is continuously managed
				// If conditions change, it should also affect it
				// Condition no longer passes — delete if owned by this CR
				name, _ := resolver.Resolve(src.Name)
				ns, _ := resolver.Resolve(src.Namespace)
				if ns == "" {
					ns = owner.GetNamespace()
				}
				if err := orkcm.DeleteIfOwned(ctx, kube, owner, name, ns); err != nil {
					return fmt.Errorf("configMaps[%d]: conditional cleanup: %w", i, err)
				}
			}
			logger.FromContext(ctx).Debug().
				Str("resource", "ConfigMap").
				Int("index", i).
				Msg("conditions not met — skipping resource")

			continue
		}

		// 2. Resolve template expressions
		resolved, err := resolver.ResolveConfigMapTemplate(src)
		if err != nil {
			return fmt.Errorf("configmaps[%d]: %w", i, err)
		}

		// 3. Build registry spec and apply
		spec := orkcm.Resolve(resolved, resolver.OwnerName())

		// toNamespaces — copy to multiple namespaces at once
		if len(resolved.ToNamespaces) > 0 {
			namespaces, err := resolver.ResolveStringSlice(resolved.ToNamespaces)
			if err != nil {
				return fmt.Errorf("configmaps[%d].toNamespaces: %w", i, err)
			}

			if update {
				for _, ns := range namespaces {
					nsSpec := spec
					nsSpec.Namespace = ns
					if err := orkcm.Update(ctx, kube, owner, nsSpec); err != nil {
						return fmt.Errorf("configmaps[%d].update namespace=%s: %w", i, ns, err)
					}
				}
			} else {
				if err := orkcm.CopyToNamespaces(ctx, kube, owner, spec, namespaces); err != nil {
					return fmt.Errorf("configmaps[%d].copyToNamespaces: %w", i, err)
				}
				if src.Reconcile {
					for _, ns := range namespaces {
						nsSpec := spec
						nsSpec.Namespace = ns
						if err := orkcm.Update(ctx, kube, owner, nsSpec); err != nil {
							return fmt.Errorf("configmaps[%d].reconcile namespace=%s: %w", i, ns, err)
						}
					}
				}
			}
			continue
		}

		// Single namespace
		if update {
			if err := orkcm.Update(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("configmaps[%d].update: %w", i, err)
			}
		} else {
			if err := orkcm.Create(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("configmaps[%d].create: %w", i, err)
			}
			if src.Reconcile {
				if err := orkcm.Update(ctx, kube, owner, spec); err != nil {
					return fmt.Errorf("configmaps[%d].reconcile: %w", i, err)
				}
			}
		}
	}
	return nil
}
