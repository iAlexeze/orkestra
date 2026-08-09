// HPA Behavior Profile Validation
//
// HPA behavior profiles are named presets that expand into a complete
// HorizontalPodAutoscalerBehavior block (scaleUp, scaleDown policies and
// stabilization windows) plus a suggested CPU utilization target.
//
// Validation enforces:
//
// 1. Known profile names:
//    Allowed: web, api, latency-sensitive, batch, cost-optimized.
//
// 2. Profile-only usage:
//    behavior.profile cannot appear alongside explicit scaleUp or scaleDown
//    fields — profiles are atomic presets.
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

func (k *Katalog) validateHPABehaviorProfiles() error {
	for crdName, crd := range k.enabledCRDs {
		for _, e := range crd.CollectHPAProfileEntries() {
			if orktypes.IsTemplate(e.Profile) {
				continue
			}
			if !k.isUserHPAProfile(e.Profile) && !profiles.IsValidHPAProfile(e.Profile) {
				return fmt.Errorf(
					"%s crd %q: HPA %q (phase %s) has unknown behavior.profile %q — "+
						"allowed: web, api, latency-sensitive, batch, cost-optimized",
					failureMark(), crdName, e.ResourceName, e.Phase, e.Profile,
				)
			}
			if e.Mixed {
				return fmt.Errorf(
					"%s crd %q: HPA %q (phase %s) declares both behavior.profile (%q) and "+
						"explicit scaleUp/scaleDown — use one or the other, not both",
					failureMark(), crdName, e.ResourceName, e.Phase, e.Profile,
				)
			}
		}
	}
	return nil
}
