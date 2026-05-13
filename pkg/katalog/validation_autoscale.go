// Autoscaler Profile Validation
//
// Profiles are computed presets. When a profile is declared, it must be the
// *only* autoscale input. Profiles expand into a complete AutoscaleSpec during
// katalog enrichment, so mixing them with manual fields creates ambiguity.
//
// Validation enforces:
//
// 1. Profile-only usage:
//    `autoscale.profile` cannot appear alongside interval, cooldown,
//    conditions, or do overrides.
//
// 2. Known profile names:
//    Allowed: burst, steady, batch, latency-sensitive, cost-optimized.
//
// 3. Valid baseline (checked before expansion):
//    Baseline workers and queueDepth must be > 0.
//
// 4. Expansion safety:
//    The generated AutoscaleSpec must contain a valid interval, cooldown,
//    non-empty conditions, and a non-empty do block.
//
// 5. No cross-operator usage:
//    Profiles are local-metric presets and cannot be combined with cross-
//    operator autoscale conditions.
//
// 6. No partial overrides:
//    Users cannot override parts of a profile (e.g., adding a do block).
//
// Profiles are atomic and predictable. If a profile is set, it fully defines
// autoscale behavior for that operator.

package katalog

import (
	"fmt"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// validateAutoscaleProfile ensures that autoscale.profile is used correctly,
// then expands the named profile into a complete AutoscaleSpec so the runtime
// only ever sees a fully-formed spec (never a bare profile name).
func (k *Katalog) validateAutoscaleProfile() error {
	for name, crd := range k.enabledCRDs {
		spec := crd.OperatorBox.Autoscale

		// No autoscale block → nothing to validate
		if spec == nil {
			continue
		}

		// No profile declared → nothing to validate
		if !crd.HasAutoscaleProfile() {
			continue
		}

		profile := crd.AutoScaleProfile()

		// Rule 1: profile cannot be combined with manual autoscale fields
		if !autoscaleIsProfileOnly(spec) {
			return fmt.Errorf(
				"autoscale.profile %q cannot be combined with manual autoscale fields; "+
					"remove interval/cooldown/conditions/do when using a profile",
				profile,
			)
		}

		// Rule 2: profile must be recognized
		if !isValidAutoscaleProfile(profile) {
			return fmt.Errorf("unknown autoscale profile: %q", profile)
		}

		// Expand the profile into a fully-formed AutoscaleSpec using the CRD's
		// declared workers and queue depth as the baseline.
		baseline := orktypes.AutoscaleBaseline{
			Workers:       crd.Workers,
			MaxQueueDepth: crd.Queue.MaxQueueDepth,
			Resync:        crd.Resync,
		}
		expanded, err := ApplyAutoscalerProfile(profile, baseline)
		if err != nil {
			return fmt.Errorf("autoscale.profile %q expansion failed: %w", profile, err)
		}
		crd.OperatorBox.Autoscale = expanded

		k.enabledCRDs[name] = crd
	}
	return nil
}

// autoscaleIsProfileOnly returns true when the user has declared ONLY a profile
// and no manual autoscale fields.
func autoscaleIsProfileOnly(spec *orktypes.AutoscaleSpec) bool {
	if spec == nil {
		return true
	}

	// Any manual field invalidates profile usage
	if spec.HasCooldownDuration() || spec.HasIntervalDuration() ||
		spec.HasWhenConditions() || spec.HasAnyOfConditions() ||
		spec.HasDoWorkers() || spec.HasDoQueueDepth() ||
		spec.HasDoResync() {
		return false
	}

	return true
}

// isValidAutoscaleProfile returns true if the profile name is one of the supported presets.
func isValidAutoscaleProfile(p string) bool {
	switch p {
	case string(AutoscaleBurst),
		string(AutoscaleSteady),
		string(AutoscaleBatch),
		string(AutoscaleLatencySensitive),
		string(AutoscaleCostOptimized):
		return true
	default:
		return false
	}
}
