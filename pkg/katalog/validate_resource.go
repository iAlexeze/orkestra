// Resource Profile Validation
//
// Resource profiles are computed presets. When a profile is declared under
// resources.profile, it must be the *only* resource input. Profiles expand
// into a complete ResourceRequirements struct during katalog enrichment, so
// mixing them with manual fields creates ambiguity.
//
// Validation uses the same visitor pattern as probe profile validation:
// CollectResourceProfileEntries iterates every hook phase and resource,
// surfacing entries where resources.profile is non-empty.
//
// Validation enforces:
//
// 1. Profile-only usage:
//    resources.profile cannot appear alongside requests or limits.
//
// 2. Known profile names:
//    Allowed: tiny, small, medium, large, burst, steady, compute-heavy,
//    memory-heavy.
//
// 3. Template expressions:
//    Profile values containing "{{" are skipped at load time and validated
//    at reconcile time instead.

package katalog

import (
	"fmt"
	"strings"
)

// isTemplateExpr reports whether s contains a Go template expression.
// Template expressions are resolved at runtime and should not be validated statically.
func isTemplateExpr(s string) bool {
	return strings.Contains(s, "{{")
}

// validateResourceProfile ensures that resources.profile is used correctly
// across all template resources in every hook phase.
// Uses the visitor pattern via CollectResourceProfileEntries.
func (k *Katalog) validateResourceProfile() error {
	for crdName, crd := range k.enabledCRDs {
		for _, e := range crd.CollectResourceProfileEntries() {
			if isTemplateExpr(e.Profile) {
				continue // resolved at runtime, skip static check
			}
			if !isValidResourceProfile(e.Profile) {
				return fmt.Errorf(
					"crd %q: %s %q (phase %s) has unknown resources.profile %q — "+
						"allowed: tiny, small, medium, large, burst, steady, compute-heavy, memory-heavy",
					crdName, e.Resource, e.ResourceName, e.Phase, e.Profile,
				)
			}
			if e.Mixed {
				return fmt.Errorf(
					"crd %q: %s %q (phase %s) declares both resources.profile (%q) and "+
						"explicit requests/limits — use one or the other, not both",
					crdName, e.Resource, e.ResourceName, e.Phase, e.Profile,
				)
			}
		}
	}
	return nil
}

// isValidResourceProfile returns true if the profile name is one of the supported presets.
func isValidResourceProfile(p string) bool {
	switch strings.ToLower(p) {
	case string(ResourceTiny),
		string(ResourceSmall),
		string(ResourceMedium),
		string(ResourceLarge),
		string(ResourceBurst),
		string(ResourceSteady),
		string(ResourceComputeHeavy),
		string(ResourceMemoryHeavy):
		return true
	default:
		return false
	}
}

// validateProbeProfiles checks that all probe profile names declared across all
// CRD hooks are recognized values. Unknown profiles fail fast at load time before
// any reconcile loop runs — same guarantee as resource profile validation.
func (k *Katalog) validateProbeProfiles() error {
	for crdName, crd := range k.enabledCRDs {
		for _, e := range crd.CollectProbeProfileEntries() {
			if isTemplateExpr(e.Profile) {
				continue // resolved at runtime, skip static check
			}
			if !isValidProbeProfile(e.Profile) {
				return fmt.Errorf(
					"crd %q: %s probe in %s %q (phase %s) has unknown probe profile %q — "+
						"allowed: fast, standard, patient, slow-start",
					crdName, e.ProbeType, e.Resource, e.ResourceName, e.Phase, e.Profile,
				)
			}
			if e.Mixed {
				return fmt.Errorf(
					"crd %q: %s probe in %s %q (phase %s) declares both a profile (%q) and "+
						"explicit timing fields — use one or the other, not both",
					crdName, e.ProbeType, e.Resource, e.ResourceName, e.Phase, e.Profile,
				)
			}
		}
	}
	return nil
}
