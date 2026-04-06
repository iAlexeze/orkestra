// pkg/reconciler/run_secrets.go
//
// Changes from previous version:
//   - evaluateConditions(owner, ...) → orktypes.EvaluateWhen(resolver.Data(), ...)
//     Uses the exported pkg/types evaluator which supports anyOf: in addition to when:
//   - once: true support added before template resolution
//     Checks Secret existence before evaluating randomAlphanumeric or other notes
//   - anyOf: []Condition field on SecretTemplateSource now evaluated
//     (requires adding AnyOf []Condition yaml:"anyOf,omitempty" to SecretTemplateSource)
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
// Execution per declaration:
//  1. Evaluate when: (allOf) and anyOf: conditions — skip if failing
//  2. once: guard — skip if Secret already exists (for random generation)
//  3. Resolve template expressions (randomAlphanumeric etc. evaluated here)
//  4. Apply via create/update/copyToNamespaces
//
// once: true — idempotent secret generation:
//
//	secrets:
//	  - name: "{{ .metadata.name }}-credentials"
//	    once: true
//	    data:
//	      password: "{{ randomAlphanumeric 32 }}"
//
// The template is only evaluated when the Secret does not exist.
// On every subsequent reconcile the existence check short-circuits before
// template evaluation — randomAlphanumeric is never called again.
//
// WARNING: once: true is incompatible with reconcile: true.
// Both on the same declaration logs a warning and once: is ignored —
// the caller opted into continuous reconciliation.
func runSecrets(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	resolver *orktmpl.Resolver,
	owner domain.Object,
	srcs []orktypes.SecretTemplateSource,
	update bool,
) error {
	for i, src := range srcs {
		// ── Step 1: condition evaluation ─────────────────────────────────────
		// EvaluateWhen handles both when: (AND) and anyOf: (OR).
		// Uses resolver.Data() which has the full CR including .children.* and
		// .external.* — same context as template expressions.
		if !orktypes.EvaluateWhen(resolver.Data(), src.Conditions, src.AnyOf) {
			if update || src.Reconcile {
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
				Msg("conditions not met — skipping secret")
			continue
		}

		// ── Step 2: once: guard ───────────────────────────────────────────────
		// Must run BEFORE template resolution so random notes are not evaluated.
		// Skip when update=true or reconcile=true (incompatible with once:).
		if src.Once {
			if src.Reconcile {
				logOnceReconcileConflict(ctx, src.Name)
			} else if !update {
				// Resolve the name to check existence — static resolution only,
				// no random notes involved in the name itself.
				name, _ := resolver.Resolve(src.Name)
				ns, _ := resolver.Resolve(src.Namespace)
				if ns == "" {
					ns = owner.GetNamespace()
				}
				exists, err := secretExists(ctx, kube, ns, name)
				if err != nil {
					return fmt.Errorf("secrets[%d]: once: existence check: %w", i, err)
				}
				if exists {
					logger.FromContext(ctx).Debug().
						Str("secret", name).
						Str("namespace", ns).
						Msg("once: secret already exists — skipping (random values preserved)")
					continue
				}
				// Does not exist — fall through to create
			}
		}

		// ── Step 3: resolve template expressions ──────────────────────────────
		// randomAlphanumeric, randomHex, randomBase64 are evaluated here.
		// The once: guard above ensures this only runs on first creation.
		resolved, err := resolver.ResolveSecretTemplate(src)
		if err != nil {
			return fmt.Errorf("secrets[%d]: %w", i, err)
		}

		// ── Step 4: apply ─────────────────────────────────────────────────────
		spec := orksecrets.Resolve(resolved, resolver.OwnerName())

		if len(resolved.ToNamespaces) > 0 {
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
		}

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
