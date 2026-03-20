// pkg/reconciler/run_configmaps.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
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
		resolved, err := resolver.ResolveConfigMapTemplate(src)
		if err != nil {
			return fmt.Errorf("configmaps[%d]: %w", i, err)
		}

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
