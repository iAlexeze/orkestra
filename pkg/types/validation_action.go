// pkg/types/validation_action.go
package orktypes

// ValidationAction declares what happens when a validation rule fails.
//
// ── deny ──────────────────────────────────────────────────────────────────────
// Blocks reconciliation. The resource spec violates policy and must be corrected
// before Orkestra will create or update child resources. A Warning Kubernetes
// event is recorded. The workqueue retries with backoff. Use for CRDs where
// the spec is under user control and can be corrected.
//
// ── warn ──────────────────────────────────────────────────────────────────────
// Advises without blocking. Reconciliation proceeds. A Warning event is recorded
// and the violation surfaces as an active warning on /katalog/{crd}. Use for
// advisory rules, gradual policy rollouts, or informational governance.
//
// ── cleanup ───────────────────────────────────────────────────────────────────
// Deletes the violating resource. Use when the resource itself should not
// exist — not when its spec is misconfigured. The canonical use case is
// orphaned Pods: a Pod without an owner reference is not misconfigured,
// it is the wrong kind of object entirely. It should be removed.
//
// Cleanup fires before deny and warn. If a resource matches a cleanup rule,
// it is deleted and the reconcile ends — running deny or warn on a resource
// that is being removed serves no purpose.
//
// A Warning event is recorded on the resource before deletion. The metric
// controller_validation_cleanup_total is incremented. The deletion uses the
// background propagation policy — owner-referenced children are also removed.
//
// Cleanup rules support two additional fields:
//
//	gracePeriodSeconds: 30    # optional — default 0 (immediate)
//	dryRun: true              # optional — log and emit event, do not delete
//
// dryRun on a cleanup rule prevents deletion globally for that rule regardless
// of how ork run was invoked. Use this during policy rollout to observe what
// would be cleaned up before enabling live deletion.
//
// Default when action is omitted: deny.
type ValidationAction string

const (
	ValidationActionDeny    ValidationAction = "deny"
	ValidationActionWarn    ValidationAction = "warn"
	ValidationActionCleanup ValidationAction = "cleanup"
)

// EffectiveAction returns the effective action, defaulting to deny.
func EffectiveAction(a ValidationAction) ValidationAction {
	if a == "" {
		return ValidationActionDeny
	}
	return a
}

func (a ValidationAction) IsDeny() bool {
	return EffectiveAction(a) == ValidationActionDeny
}

func (a ValidationAction) IsWarn() bool {
	return EffectiveAction(a) == ValidationActionWarn
}

func (a ValidationAction) IsCleanup() bool {
	return EffectiveAction(a) == ValidationActionCleanup
}
