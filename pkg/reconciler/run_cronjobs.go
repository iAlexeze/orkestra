// pkg/reconciler/run_cronjobs.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	orkcron "github.com/ialexeze/orkestra/pkg/orkestra-registry/cronjobs"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// runCronJobs resolves and applies CronJob template declarations.
//
// CronJobs are long-lived scheduled resources — created under onCreate and
// drift-corrected under onReconcile (or reconcile: true).
//
// Common use cases:
//   - Periodic sync jobs (cache warming, data replication)
//   - Scheduled backup jobs
//   - Recurring cleanup or archival tasks
//   - Health check or audit jobs on a schedule
//
// The schedule field supports both static cron expressions and dynamic
// values from the CR spec:
//
//	schedule: "0 * * * *"               static — every hour
//	schedule: "{{ .spec.syncSchedule }}" dynamic — from CR spec
func runCronJobs(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	resolver *orktmpl.Resolver,
	owner domain.Object,
	srcs []orktypes.CronJobTemplateSource,
	update bool,
	guard func(ctx context.Context, obj domain.Object, ns string) bool,
) error {
	for i, src := range srcs {
		// 1. Evaluate conditions BEFORE resolving templates
		conditionPassed := orktypes.EvaluateWhen(resolver.Data(), src.Conditions, src.AnyOf)

		if !conditionPassed {
			if update || src.Reconcile { // ← src.Reconcile here too to show that this resource is continuously managed
				// Condition no longer passes — delete if owned by this CR
				name, _ := resolver.Resolve(src.Name)
				ns, _ := resolver.Resolve(src.Namespace)
				if ns == "" {
					ns = owner.GetNamespace()
				}
				if err := orkcron.DeleteIfOwned(ctx, kube, owner, name, ns); err != nil {
					return fmt.Errorf("cronJobs[%d]: conditional cleanup: %w", i, err)
				}
			}
			logger.FromContext(ctx).Debug().
				Str("resource", "CronJob").
				Int("index", i).
				Msg("conditions not met — skipping resource")

			continue
		}

		// 2. Resolve template expressions
		resolved, err := resolver.ResolveCronJobTemplate(src)
		if err != nil {
			return fmt.Errorf("cronjobs[%d]: %w", i, err)
		}

		// 3. Build registry spec and apply
		spec := orkcron.Resolve(resolved, resolver.OwnerName())

		if update {
			if err := orkcron.Update(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("cronjobs[%d].update: %w", i, err)
			}
		} else {
			if err := orkcron.Create(ctx, kube, owner, spec); err != nil {
				return fmt.Errorf("cronjobs[%d].create: %w", i, err)
			}
			if src.Reconcile {
				if err := orkcron.Update(ctx, kube, owner, spec); err != nil {
					return fmt.Errorf("cronjobs[%d].reconcile: %w", i, err)
				}
			}
		}
	}
	return nil
}
