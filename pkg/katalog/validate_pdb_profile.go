// PDB Behavior Profile Validation
//
// PDB behavior profiles are named presets that expand into a concrete
// minAvailable or maxUnavailable value at Katalog load time.
//
// Validation enforces:
//
// 1. Known profile names:
//    Allowed: zero-downtime, rolling, relaxed.
//
// 2. Profile-only usage:
//    behavior.profile cannot appear alongside explicit minAvailable or maxUnavailable.
//
// 3. Template expressions:
//    Profile values containing "{{" are skipped at load time.

package katalog

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/profiles"
)

func (k *Katalog) validatePDBBehaviorProfiles() error {
	for crdName, crd := range k.enabledCRDs {
		for _, e := range crd.CollectPDBProfileEntries() {
			if isTemplateExpr(e.Profile) {
				continue
			}
			if !k.isUserPDBProfile(e.Profile) && !profiles.IsValidPDBProfile(e.Profile) {
				return fmt.Errorf(
					"crd %q: PDB %q (phase %s) has unknown behavior.profile %q — "+
						"allowed: zero-downtime, rolling, relaxed",
					crdName, e.ResourceName, e.Phase, e.Profile,
				)
			}
			if e.Mixed {
				return fmt.Errorf(
					"crd %q: PDB %q (phase %s) declares both behavior.profile (%q) and "+
						"explicit minAvailable/maxUnavailable — use one or the other, not both",
					crdName, e.ResourceName, e.Phase, e.Profile,
				)
			}
		}
	}
	return nil
}
