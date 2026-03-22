// pkg/reconciler/run_cleanup.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/metrics"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CleanupResult holds the outcome of evaluating cleanup rules for one resource.
type CleanupResult struct {
	// ShouldDelete — true when a cleanup rule fired and DryRun is false.
	// When true, the caller must delete the resource and end the reconcile.
	ShouldDelete bool

	// DryRunMatch — true when a cleanup rule fired but DryRun was true.
	// The resource was not deleted. The violation is logged and metered.
	DryRunMatch bool

	// MatchedRule — the first cleanup rule that matched (cleanup short-circuits).
	MatchedRule *orktypes.ValidationRule

	// Message — the user-defined message from the matched rule.
	Message string
}

// RunCleanupRules evaluates all cleanup-action validation rules for a resource.
//
// Cleanup rules are evaluated before deny and warn rules. The first matching
// cleanup rule short-circuits evaluation — there is no point evaluating further
// rules on a resource that is about to be deleted.
//
// Returns a CleanupResult. If ShouldDelete is true, the caller deletes the
// resource and terminates the reconcile. No deny or warn rules run.
//
// If DryRunMatch is true, the caller emits an event and records the metric
// but does not delete. Reconciliation continues to deny and warn rules so
// all violations are surfaced during dry-run rollout.
func RunCleanupRules(
	obj domain.Object,
	cfg *orktypes.ValidationConfig,
	crdName string,
) *CleanupResult {
	if cfg == nil {
		return &CleanupResult{}
	}

	u, ok := toUnstructured(obj)
	if !ok {
		// Typed objects cannot use dot-notation cleanup rules.
		return &CleanupResult{}
	}

	for i := range cfg.Rules {
		rule := &cfg.Rules[i]
		if !rule.Action.IsCleanup() {
			continue
		}

		violation := evaluateValidationRule(u, *rule)
		if violation == nil {
			continue // rule passed — resource is compliant
		}

		// Cleanup rule matched
		dryRunLabel := "false"
		if rule.DryRun {
			dryRunLabel = "true"
		}

		metrics.CleanupTotal.WithLabelValues(crdName, rule.Field, ruleType(*rule), dryRunLabel).Inc()

		logger.Info().
			Str("crd", crdName).
			Str("name", obj.GetName()).
			Str("namespace", obj.GetNamespace()).
			Str("field", rule.Field).
			Str("got", violation.Value).
			Bool("dryRun", rule.DryRun).
			Msg("cleanup rule matched — resource will be deleted")

		return &CleanupResult{
			ShouldDelete: !rule.DryRun,
			DryRunMatch:  rule.DryRun,
			MatchedRule:  rule,
			Message:      rule.Message,
		}
	}

	return &CleanupResult{}
}

// ExecuteCleanup deletes the resource identified by obj.
//
// Uses background deletion propagation — owner-referenced children
// (pods managed by a ReplicaSet, for example) are also removed.
//
// A Warning Kubernetes event is emitted before deletion so there is a
// record of why the resource was removed. This is visible in:
//
//	kubectl describe <kind> <name>
//	kubectl get events --field-selector involvedObject.name=<name>
func ExecuteCleanup(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	obj domain.Object,
	gvr schema.GroupVersionResource,
	rule *orktypes.ValidationRule,
	crdName string,
) error {
	name := obj.GetName()
	ns := obj.GetNamespace()

	gracePeriod := int64(0)
	if rule.GracePeriodSeconds != nil {
		gracePeriod = *rule.GracePeriodSeconds
	}

	propagation := metav1.DeletePropagationBackground
	deleteOpts := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
		PropagationPolicy:  &propagation,
	}

	logger.Info().
		Str("crd", crdName).
		Str("name", name).
		Str("namespace", ns).
		Str("field", rule.Field).
		Str("message", rule.Message).
		Int64("gracePeriodSeconds", gracePeriod).
		Msg("cleanup: deleting resource")

	var err error
	if ns != "" {
		err = kube.Dynamic().Resource(gvr).Namespace(ns).Delete(
			ctx, name, deleteOpts,
		)
	} else {
		err = kube.Dynamic().Resource(gvr).Delete(
			ctx, name, deleteOpts,
		)
	}

	if err != nil {
		return fmt.Errorf("cleanup: deleting %s/%s: %w", ns, name, err)
	}

	logger.Info().
		Str("crd", crdName).
		Str("name", name).
		Str("namespace", ns).
		Msg("cleanup: resource deleted")

	return nil
}

// CleanupEventMessage returns the Kubernetes event message for a cleanup action.
// Recorded on the resource before deletion so operators can understand why
// it was removed.
func CleanupEventMessage(rule *orktypes.ValidationRule, got string) string {
	suffix := ""
	if rule.DryRun {
		suffix = " [dry-run: resource was NOT deleted]"
	}
	return fmt.Sprintf(
		"Cleanup rule matched: field %q — %s (got: %q)%s",
		rule.Field, rule.Message, got, suffix,
	)
}
