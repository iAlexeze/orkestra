// Rolling Update Profile Validation
//
// Rolling update profiles are named presets that expand into MaxSurge and
// MaxUnavailable values on a Deployment's rolling update strategy at Katalog load time.
//
// Validation enforces:
//
// 1. Known profile names:
//    Allowed: safe, fast, blue-green.
//
// 2. Profile-only usage:
//    rollingUpdate.profile cannot appear alongside explicit maxSurge or maxUnavailable.
//
// 3. Template expressions:
//    Profile values containing "{{" are skipped at load time.

package katalog

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/profiles"
)

func (k *Katalog) validateRollingUpdateProfiles() error {
	for crdName, crd := range k.enabledCRDs {
		for _, e := range crd.CollectRollingUpdateProfileEntries() {
			if isTemplateExpr(e.Profile) {
				continue
			}
			if !k.isUserRollingUpdateProfile(e.Profile) && !profiles.IsValidRollingUpdateProfile(e.Profile) {
				return fmt.Errorf(
					"crd %q: Deployment %q (phase %s) has unknown rollingUpdate.profile %q — "+
						"allowed: safe, fast, blue-green",
					crdName, e.ResourceName, e.Phase, e.Profile,
				)
			}
			if e.Mixed {
				return fmt.Errorf(
					"crd %q: Deployment %q (phase %s) declares both rollingUpdate.profile (%q) and "+
						"explicit maxSurge/maxUnavailable — use one or the other, not both",
					crdName, e.ResourceName, e.Phase, e.Profile,
				)
			}
		}
	}
	return nil
}
