// pkg/reconciler/run_secrets.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	orksecrets "github.com/ialexeze/orkestra/pkg/orkestra-registry/secrets"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// runSecrets resolves and applies Secret template declarations.
//
// Secrets have two additional behaviours beyond Deployments and Services:
//
// fromSecret — when src.FromSecret is set, the registry copies data from
// an existing Secret in another namespace rather than using static data.
// This is the primary use case: copy a platform secret into every tenant namespace.
//
// toNamespaces — when src.ToNamespaces is non-empty, the registry creates
// one copy of the Secret in each listed namespace. Resolved once, written N times.
//
// reconcile: true — on every reconcile, re-reads the source Secret and syncs
// any changes. This keeps copies up to date when credentials rotate.
func runSecrets(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	resolver *orktmpl.Resolver,
	owner domain.Object,
	srcs []orktypes.SecretTemplateSource,
	update bool,
) error {
	for i, src := range srcs {
		resolved, err := resolver.ResolveSecretTemplate(src)
		if err != nil {
			return fmt.Errorf("secrets[%d]: %w", i, err)
		}

		spec := orksecrets.Resolve(resolved, resolver.OwnerName())

		// toNamespaces — copy to multiple namespaces at once
		if len(resolved.ToNamespaces) > 0 {
			namespaces, err := resolver.ResolveStringSlice(resolved.ToNamespaces)
			if err != nil {
				return fmt.Errorf("secrets[%d].toNamespaces: %w", i, err)
			}

			if update {
				// reconcile: true — re-sync copies with the source Secret
				for _, ns := range namespaces {
					nsSpec := spec
					nsSpec.Namespace = ns
					if err := orksecrets.Update(ctx, kube, owner, nsSpec); err != nil {
						return fmt.Errorf("secrets[%d].update namespace=%s: %w", i, ns, err)
					}
				}
			} else {
				if err := orksecrets.CopyToNamespaces(ctx, kube, owner, spec, namespaces); err != nil {
					return fmt.Errorf("secrets[%d].copyToNamespaces: %w", i, err)
				}
				if src.Reconcile {
					for _, ns := range namespaces {
						nsSpec := spec
						nsSpec.Namespace = ns
						if err := orksecrets.Update(ctx, kube, owner, nsSpec); err != nil {
							return fmt.Errorf("secrets[%d].reconcile namespace=%s: %w", i, ns, err)
						}
					}
				}
			}
			continue
		}

		// Single namespace
		if update {
			if err := orksecrets.Update(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("secrets[%d].update: %w", i, err)
			}
		} else {
			if err := orksecrets.Create(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("secrets[%d].create: %w", i, err)
			}
			if src.Reconcile {
				if err := orksecrets.Update(ctx, kube, owner, spec); err != nil {
					return fmt.Errorf("secrets[%d].reconcile: %w", i, err)
				}
			}
		}
	}
	return nil
}
