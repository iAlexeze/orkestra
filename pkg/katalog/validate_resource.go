// Resource Profile Validation
//
// Resource profiles are computed presets. When a profile is declared, it must be
// the *only* resource input. Profiles expand into a complete
// ResourceRequirements struct during katalog enrichment, so mixing them with
// manual fields creates ambiguity.
//
// Validation enforces:
//
// 1. Profile-only usage:
//    `resources.profile` cannot appear alongside requests or limits.
//
// 2. Known profile names:
//    Allowed: tiny, small, medium, large, burst, steady, compute-heavy,
//    memory-heavy.
//
// 3. No partial overrides:
//    Users cannot override parts of a profile (e.g., adding requests.cpu).
//
// 4. Expansion safety:
//    The generated ResourceRequirements must contain valid requests and limits.
//
// 5. No cross-field mixing:
//    Profiles cannot be combined with container-level overrides.
//
// Profiles are atomic and predictable. If a profile is set, it fully defines
// resource behavior for that operator.

package katalog

import (
	"fmt"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// validateResourceProfile ensures that resources.profile is used correctly.
// Runs before profile expansion. Profiles must be the only resource input.
func (k *Katalog) validateResourceProfile() error {
	for name, crd := range k.enabledCRDs {
		// No resources block → nothing to validate
		if !crd.HasAnyHooks() {
			continue
		}

		// No Deployments, Replicasets, StatefulSet → nothing to validate
		if !crd.NeedsResourceDecl() {
			continue
		}

		// No profile declared → nothing to validate
		if !crd.HasResourceProfile() {
			continue
		}

		spec := crd.ResourceDecl()
		profile := crd.ResourceProfile()

		// Rule 1: profile cannot be combined with manual resource fields
		if !resourceIsProfileOnly(spec) {
			return fmt.Errorf(
				"resources.profile %q cannot be combined with manual resource fields; "+
					"remove requests/limits when using a profile",
				profile,
			)
		}

		// Rule 2: profile must be recognized
		if !isValidResourceProfile(profile) {
			return fmt.Errorf("unknown resource profile: %q", profile)
		}

		// Save updated CRD
		k.enabledCRDs[name] = crd
	}
	return nil
}

// resourceIsProfileOnly returns true when the user has declared ONLY a profile
// and no manual resource fields.
func resourceIsProfileOnly(spec *orktypes.ResourceRequirements) bool {
	if spec == nil {
		return true
	}

	// Any manual field invalidates profile usage
	if len(spec.Requests) > 0 || len(spec.Limits) > 0 {
		return false
	}

	return true
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
