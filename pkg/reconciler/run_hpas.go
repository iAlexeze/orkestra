// pkg/reconciler/run_hpas.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	orkhpa "github.com/orkspace/orkestra/pkg/resources/hpas"
	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// runHPAs resolves and applies HorizontalPodAutoscaler template declarations.
func runHPAs(
	ctx context.Context,
	kube kubeclient.KubeClient,
	resolver *orktmpl.Resolver,
	owner domain.Object,
	srcs []orktypes.HPATemplateSource,
	update bool,
	guard func(ctx context.Context, obj domain.Object, ns string) bool,
) error {
	activeNames := make(map[string]bool, len(srcs))
	for _, s := range srcs {
		if !orktypes.EvaluateWhen(resolver.Data(), s.Conditions, s.AnyOf, resolver.TemplateEvaluator()) {
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
		conditionPassed := orktypes.EvaluateWhen(resolver.Data(), src.Conditions, src.AnyOf, resolver.TemplateEvaluator())

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
					if err := orkhpa.DeleteIfOwned(ctx, kube, owner, name, ns); err != nil {
						return fmt.Errorf("hpas[%d]: conditional cleanup: %w", i, err)
					}
				}
			}
			logger.FromContext(ctx).Debug().
				Str("resource", "HorizontalPodAutoscaler").
				Int("index", i).
				Msg("conditions not met — skipping resource")
			continue
		}

		resolved, err := resolver.ResolveHPATemplate(src)
		if err != nil {
			return fmt.Errorf("hpas[%d]: %w", i, err)
		}

		spec := orkhpa.Resolve(resolved, resolver.OwnerName())

		if update {
			if err := orkhpa.Update(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("hpas[%d].update: %w", i, err)
			}
		} else {
			if err := orkhpa.Create(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("hpas[%d].create: %w", i, err)
			}
			if src.Reconcile {
				if err := orkhpa.Update(ctx, kube, owner, spec); err != nil {
					return fmt.Errorf("hpas[%d].reconcile: %w", i, err)
				}
			}
		}
	}
	return nil
}
