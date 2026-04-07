// pkg/reconciler/run_jobs.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	orkjobs "github.com/ialexeze/orkestra/pkg/orkestra-registry/jobs"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// runJobs resolves and applies Job template declarations.
//
// Jobs are fire-and-forget — they run once to completion and are not updated
// after creation. They are therefore always idempotent creates.
//
// Jobs appear almost exclusively under onDelete for cleanup tasks that must
// complete before Orkestra removes finalizers:
//   - Draining message queues before a consumer CR is deleted
//   - Archiving state to external storage
//   - Notifying external systems of deletion
//   - Running database migrations before removing a schema CR
//
// Jobs can also appear under onCreate for one-time provisioning tasks.
//
// Owner references are NOT set on onDelete Jobs because the owner CR is
// being deleted — the Job must complete independently after the CR is gone.
func runJobs(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	resolver *orktmpl.Resolver,
	owner domain.Object,
	srcs []orktypes.JobTemplateSource,
) error {
	for i, src := range srcs {
		// 1. Evaluate conditions BEFORE resolving templates
		conditionPassed := orktypes.EvaluateWhen(resolver.Data(), src.Conditions, src.AnyOf)

		if !conditionPassed {
			logger.FromContext(ctx).Debug().
				Str("resource", "Job").
				Int("index", i).
				Msg("conditions not met — skipping resource")

			continue
		}

		// 2. Resolve template expressions
		resolved, err := resolver.ResolveJobTemplate(src)
		if err != nil {
			return fmt.Errorf("jobs[%d]: %w", i, err)
		}

		// 3. Build registry spec and apply
		spec := orkjobs.Resolve(resolved, resolved.BackoffLimit, resolver.OwnerName())

		// Jobs are always creates — no update semantics
		if err := orkjobs.Create(ctx, kube, owner, spec); err != nil {
			return fmt.Errorf("jobs[%d].create: %w", i, err)
		}
	}
	return nil
}
