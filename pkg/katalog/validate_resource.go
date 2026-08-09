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

	"github.com/orkspace/orkestra/pkg/profiles"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func (k *Katalog) validateResourceProfile() error {
	for crdName, crd := range k.enabledCRDs {
		for _, e := range crd.CollectResourceProfileEntries() {
			if orktypes.IsTemplate(e.Profile) {
				continue
			}
			_, userDefined := k.Profiles.LookupResource(e.Profile)
			if !userDefined && !profiles.IsValidResourceProfile(e.Profile) {
				return fmt.Errorf(
					"%s crd %q: %s %q (phase %s) has unknown resources.profile %q — "+
						"allowed: tiny, small, medium, large, burst, steady, compute-heavy, memory-heavy, or a user-defined profile declared in profiles.resources",
					failureMark(), crdName, e.Resource, e.ResourceName, e.Phase, e.Profile,
				)
			}
			if e.Mixed {
				return fmt.Errorf(
					"%s crd %q: %s %q (phase %s) declares both resources.profile (%q) and "+
						"explicit requests/limits — use one or the other, not both",
					failureMark(), crdName, e.Resource, e.ResourceName, e.Phase, e.Profile,
				)
			}
		}
	}
	return nil
}

func (k *Katalog) validateProbeProfiles() error {
	for crdName, crd := range k.enabledCRDs {
		for _, e := range crd.CollectProbeProfileEntries() {
			if orktypes.IsTemplate(e.Profile) {
				continue
			}
			_, userDefined := k.Profiles.LookupProbe(e.Profile)
			if !userDefined && !profiles.IsValidProbeProfile(e.Profile) {
				return fmt.Errorf(
					"%s crd %q: %s probe in %s %q (phase %s) has unknown probe profile %q — "+
						"allowed: fast, standard, patient, slow-start, or a user-defined profile declared in profiles.probes",
					failureMark(), crdName, e.ProbeType, e.Resource, e.ResourceName, e.Phase, e.Profile,
				)
			}
			if e.Mixed {
				return fmt.Errorf(
					"%s crd %q: %s probe in %s %q (phase %s) declares both a profile (%q) and "+
						"explicit timing fields — use one or the other, not both",
					failureMark(), crdName, e.ProbeType, e.Resource, e.ResourceName, e.Phase, e.Profile,
				)
			}
		}
	}
	return nil
}
