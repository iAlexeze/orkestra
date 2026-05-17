// pkg/reconciler/run_pdbs.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	orkpdb "github.com/orkspace/orkestra/pkg/orkestra-registry/pdbs"
	orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// runPDBs resolves and applies PodDisruptionBudget template declarations.
func runPDBs(
	ctx context.Context,
	kube kubeclient.KubeClient,
	resolver *orktmpl.Resolver,
	owner domain.Object,
	srcs []orktypes.PDBTemplateSource,
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
		conditionPassed := orktypes.EvaluateWhen(resolver.Data(), src.Conditions, src.AnyOf)

		name, _ := resolver.Resolve(src.Name)
		ns, _ := resolver.Resolve(src.Namespace)
		if ns == "" {
			ns = owner.GetNamespace()
		}

		if guard != nil && !guard(ctx, owner, ns) {
			continue
		}

		if !conditionPassed {
			if update || src.Reconcile {
				if !activeNames[ns+"/"+name] {
					if err := orkpdb.DeleteIfOwned(ctx, kube, owner, name, ns); err != nil {
						return fmt.Errorf("pdbs[%d]: conditional cleanup: %w", i, err)
					}
				}
			}
			logger.FromContext(ctx).Debug().
				Str("resource", "PodDisruptionBudget").
				Int("index", i).
				Msg("conditions not met — skipping resource")
			continue
		}

		resolved, err := resolver.ResolvePDBTemplate(src)
		if err != nil {
			return fmt.Errorf("pdbs[%d]: %w", i, err)
		}

		spec := orkpdb.Resolve(resolved, resolver.OwnerName())

		if update {
			if err := orkpdb.Update(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("pdbs[%d].update: %w", i, err)
			}
		} else {
			if err := orkpdb.Create(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("pdbs[%d].create: %w", i, err)
			}
			if src.Reconcile {
				if err := orkpdb.Update(ctx, kube, owner, spec); err != nil {
					return fmt.Errorf("pdbs[%d].reconcile: %w", i, err)
				}
			}
		}
	}
	return nil
}
