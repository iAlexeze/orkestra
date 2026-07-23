// LimitRange Profile Validation
//
// LimitRange profiles are user-defined named presets that expand into a
// list of LimitRangeItems at reconcile time. There are no built-in presets —
// every limitRange profile must be declared in the katalog or an imported motif.
//
// Validation enforces:
//
// 1. Known profile names:
//    Profile must appear in profiles.limitRanges or an imported motif.
//
// 2. Profile-only usage:
//    profile cannot appear alongside an explicit limits list.
//
// 3. Template expressions:
//    Profile values containing "{{" are skipped at load time.

package katalog

import (
	"fmt"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func (k *Katalog) validateLimitRangeProfiles() error {
	for crdName, crd := range k.enabledCRDs {
		for _, e := range crd.CollectLimitRangeProfileEntries() {
			if orktypes.IsTemplate(e.Profile) {
				continue
			}
			if !k.isUserLimitRangeProfile(e.Profile) {
				return fmt.Errorf(
					"crd %q: LimitRange %q (phase %s) has unknown profile %q — "+
						"define it in profiles.limitRanges",
					crdName, e.ResourceName, e.Phase, e.Profile,
				)
			}
			if e.Mixed {
				return fmt.Errorf(
					"crd %q: LimitRange %q (phase %s) declares both profile (%q) and "+
						"explicit limits — use one or the other, not both",
					crdName, e.ResourceName, e.Phase, e.Profile,
				)
			}
		}
	}
	return nil
}
