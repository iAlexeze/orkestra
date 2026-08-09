// ResourceQuota Profile Validation
//
// ResourceQuota profiles are named presets that expand into a complete set of
// hard resource limits at reconcile time.
//
// Validation enforces:
//
// 1. Known profile names:
//    Allowed: small, medium, large, xlarge.
//
// 2. Profile-only usage:
//    profile cannot appear alongside an explicit hard map — profiles are
//    atomic presets.
//
// 3. Template expressions:
//    Profile values containing "{{" are skipped at load time and validated
//    at reconcile time instead.

package katalog

import (
	"fmt"
	orktypes "github.com/orkspace/orkestra/pkg/types"

	"github.com/orkspace/orkestra/pkg/profiles"
)

func (k *Katalog) validateResourceQuotaProfiles() error {
	for crdName, crd := range k.enabledCRDs {
		for _, e := range crd.CollectResourceQuotaProfileEntries() {
			if orktypes.IsTemplate(e.Profile) {
				continue
			}
			if !k.isUserResourceQuotaProfile(e.Profile) && !profiles.IsValidResourceQuotaProfile(e.Profile) {
				return fmt.Errorf(
					"%s crd %q: resourceQuota %q (phase %s) has unknown profile %q — "+
						"allowed: small, medium, large, xlarge",
					failureMark(), crdName, e.ResourceName, e.Phase, e.Profile,
				)
			}
			if e.Mixed {
				return fmt.Errorf(
					"%s crd %q: resourceQuota %q (phase %s) declares both profile (%q) and "+
						"explicit hard limits — use one or the other, not both",
					failureMark(), crdName, e.ResourceName, e.Phase, e.Profile,
				)
			}
		}
	}
	return nil
}
