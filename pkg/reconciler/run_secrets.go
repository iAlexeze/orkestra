// pkg/reconciler/run_secrets.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
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
		// 1. Evaluate conditions BEFORE resolving templates
		conditionPassed := evaluateConditions(owner, src.Conditions)

		if !conditionPassed {
			if update {
				// Condition no longer passes — delete if owned by this CR
				name, _ := resolver.Resolve(src.Name)
				ns, _ := resolver.Resolve(src.Namespace)
				if ns == "" {
					ns = owner.GetNamespace()
				}
				if err := orksecrets.DeleteIfOwned(ctx, kube, owner, name, ns); err != nil {
					return fmt.Errorf("secrets[%d]: conditional cleanup: %w", i, err)
				}
			}
			logger.FromContext(ctx).Debug().
				Str("resource", "Secret").
				Int("index", i).
				Msg("conditions not met — skipping resource")

			continue
		}

		// 2. Resolve template expressions
		resolved, err := resolver.ResolveSecretTemplate(src)
		if err != nil {
			return fmt.Errorf("secrets[%d]: %w", i, err)
		}

		// 3. Build registry spec and apply
		spec := orksecrets.Resolve(resolved, resolver.OwnerName())

		// toNamespaces — copy to multiple namespaces at once
		// toNamespaces — copy to multiple namespaces at once
		if len(resolved.ToNamespaces) > 0 {
			// Use Update (sync-aware) when either:
			//   update=true  → called from onReconcile block
			//   src.Reconcile → declared reconcile: true in onCreate
			// Use CopyToNamespaces (create-only) only for pure onCreate with no reconcile flag.
			shouldSync := update || src.Reconcile

			if shouldSync {
				for _, ns := range resolved.ToNamespaces {
					nsSpec := spec
					nsSpec.Namespace = ns
					if err := orksecrets.Update(ctx, kube, owner, nsSpec); err != nil {
						return fmt.Errorf("secrets[%d].sync namespace=%s: %w", i, ns, err)
					}
				}
			} else {
				if err := orksecrets.CopyToNamespaces(ctx, kube, owner, spec, resolved.ToNamespaces); err != nil {
					return fmt.Errorf("secrets[%d].copyToNamespaces: %w", i, err)
				}
			}
			continue
		} // Single namespace
		if update {
			if err := orksecrets.Update(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("secrets[%d].update: %w", i, err)
			}
		} else {
			if err := orksecrets.Create(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("secrets[%d].create: %w", i, err)
			}

			// reconcile: true
			if src.Reconcile {
				if err := orksecrets.Update(ctx, kube, owner, spec); err != nil {
					return fmt.Errorf("secrets[%d].reconcile: %w", i, err)
				}
			}
		}
	}
	return nil
}
