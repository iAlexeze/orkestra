package katalog

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/external"
	orktarget "github.com/orkspace/orkestra/pkg/intent/target"
	orktmpl "github.com/orkspace/orkestra/pkg/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
)

// EvaluatePreReconcile evaluates preReconcile.reconcileGate conditions for the named CRD.
// Returns (true, "") when conditions pass and the reconciler should run.
// Returns (false, reason) when gated — reconciler must not be called.
//
// preReconcile.external runs first (shared enrichment), then reconcileGate.external,
// then conditions are evaluated against the accumulated resolver.
func (k *Katalog) EvaluatePreReconcile(ctx context.Context, gvk string, obj *unstructured.Unstructured, cs kubernetes.Interface, sentinels map[string]string) (allowed bool, reason string) {
	box := k.effectiveBox(obj, gvk)
	if box.Empty() {
		return true, ""
	}

	pr := box.PreReconcile
	if pr.Empty() || !pr.HasReconcileGate() {
		return true, ""
	}

	resolver := k.effectiveResolver(ctx, obj, pr, sentinels)
	if resolver.Empty() {
		return true, ""
	}

	var err error
	if pr.HasPreReconcileExternal() {
		if resolver, err = external.Run(ctx, gvk, resolver, pr.External, cs); err != nil {
			return true, "" // shared pre-reconcile external: always fail-open
		}
	}
	g := pr.ReconcileGate
	if pr.HasReconcileGateExternal() {
		if resolver, err = external.Run(ctx, gvk, resolver, g.External, cs); err != nil {
			if g.FailPolicy == orktypes.FailPolicyClosed {
				return false, "reconcileGate: external evaluation failed (failPolicy: closed)"
			}
			return true, ""
		}
	}

	// Evaluate reconcileGate.sentinels (shorthand) - first match wins
	if g.HasSentinels() {
		return g.SentinelsAllowed(sentinels), ""
	}

	if !orktypes.EvaluateConditions(resolver.Data(), pr.WhenConditions(), pr.OrConditions(), resolver.TemplateEvaluator()) {
		return false, preReconcileGateReason(pr, resolver)
	}
	return true, ""
}

// EvaluateEnqueueFilter evaluates preReconcile.enqueueGate conditions for the named CRD.
// Returns true when the object should be enqueued, false when it should be dropped.
// Accepts domain.Object so it works for both dynamic and typed CRDs.
//
// preReconcile.external runs first (shared enrichment), then enqueueGate.external,
// then conditions are evaluated against the accumulated resolver.
func (k *Katalog) EvaluateEnqueueFilter(ctx context.Context, gvk string, obj domain.Object, cs kubernetes.Interface, sentinels map[string]string) bool {
	box := k.effectiveBox(obj, gvk)
	if box.Empty() {
		return true
	}

	pr := box.PreReconcile
	if pr.Empty() || !pr.HasEnqueueGate() {
		return true
	}

	resolver := k.effectiveResolver(ctx, obj, pr, sentinels)
	if resolver.Empty() {
		return true
	}

	var err error
	if pr.HasPreReconcileExternal() {
		if resolver, err = external.Run(ctx, gvk, resolver, pr.External, cs); err != nil {
			return true // shared pre-reconcile external: always fail-open
		}
	}
	g := pr.EnqueueGate
	if pr.HasEnqueueGateExternal() {
		if resolver, err = external.Run(ctx, gvk, resolver, g.External, cs); err != nil {
			if g.FailPolicy == orktypes.FailPolicyClosed {
				return false
			}
			return true
		}
	}

	// Evaluate enqueueGate.sentinels (shorthand) - first match wins
	if g.HasSentinels() {
		return g.SentinelsAllowed(sentinels)
	}

	return orktypes.EvaluateConditions(resolver.Data(), g.WhenConditions(), g.OrConditions(), resolver.TemplateEvaluator())
}

// EvaluateQueueBehaviourConditions completes the queue behaviour evaluation started by the
// workqueue but delegated to the informer. Evaluates queue.behaviour conditions for the named CRD.
// Returns true when the object should be enqueued, false when it should be dropped.
// Accepts domain.Object so it works for both dynamic and typed CRDs.
//
// Here because it influences 'pre-reconcile' decisions.
func (k *Katalog) EvaluateQueueBehaviourConditions(ctx context.Context, gvk string, obj domain.Object, sentinels map[string]string) bool {
	box := k.effectiveBox(obj, gvk)
	if box.Empty() {
		return true
	}

	rc := box.Reconciler
	q := rc.Queue
	if rc.Empty() || q.Empty() || !q.HasBehaviourCondition() {
		return true
	}

	resolver := k.effectiveResolver(ctx, obj, box.PreReconcile, sentinels)
	if resolver == nil {
		return true
	}

	if q.HasOnLimitConditions() {
		return orktypes.EvaluateConditions(resolver.Data(), q.OnLimitWhen(), q.OnLimitOr(), resolver.TemplateEvaluator())
	}
	if q.HasOnThresholdConditions() {
		return orktypes.EvaluateConditions(resolver.Data(), q.OnThresholdWhen(), q.OnThresholdOr(), resolver.TemplateEvaluator())
	}

	return true
}

// effectiveResolver computes the common resolver used by all evaluators
func (k *Katalog) effectiveResolver(ctx context.Context, obj domain.Object, pr *orktypes.PreReconcileConfig, sentinels map[string]string) *orktmpl.Resolver {
	resolver, err := orktmpl.NewResolver(ctx, obj)
	if err != nil {
		return nil
	}
	if !k.Profiles.Empty() {
		resolver = resolver.WithProfiles(k.UserProfiles())
	}
	if !k.Notes.Empty() {
		resolver = resolver.WithUserNotes(k.UserNotes())
	}
	// .request context
	if intent := orktarget.ResolveIntentFromObject(resolver.Data()); intent != nil {
		resolver = resolver.WithRequest(intent)
	}
	// .metrics context
	if metrics := common.ResolveResourceMetricFromObject(resolver.Data()); metrics != nil {
		resolver = resolver.WithMetrics(metrics)
	}
	// .health context
	if health := common.ResolveResourceHealthFromObject(resolver.Data()); health != nil {
		resolver = resolver.WithHealth(health)
	}
	if len(sentinels) > 0 {
		resolver = resolver.WithSentinels(pr.DeclaredSentinels(), sentinels)
	}
	return resolver
}

// effectiveBox computes the common operatorBox used by all evaluators
func (k *Katalog) effectiveBox(obj domain.Object, gvk string) *orktypes.OperatorBoxConfig {
	if obj == nil {
		return nil
	}
	entry := k.LookupByGVKString(gvk).Entry()
	if entry == nil {
		return nil
	}
	target := orktarget.ResolveTargetFromAnnotations(obj.GetAnnotations())
	return entry.EffectiveOperatorBox(target)
}

// preReconcileGateReason returns a human-readable description of why the gate fired.
func preReconcileGateReason(pr *orktypes.PreReconcileConfig, resolver *orktmpl.Resolver) string {
	eval := resolver.TemplateEvaluator()
	for _, cond := range pr.WhenConditions() {
		if !orktypes.EvaluateConditions(resolver.Data(), []orktypes.Condition{cond}, nil, eval) {
			val, _ := resolver.Resolve(cond.Field)
			return fmt.Sprintf("when: %q = %q, want %q", cond.Field, val, cond.Equals)
		}
	}
	return "or: no condition satisfied"
}
