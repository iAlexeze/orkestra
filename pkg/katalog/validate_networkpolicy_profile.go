// NetworkPolicy Profile Validation
//
// NetworkPolicy profiles are named presets that expand into a complete set of
// ingress/egress rules and policy types at reconcile time.
//
// Validation enforces:
//
// 1. Known profile names:
//    Allowed: deny-all, deny-all-ingress, deny-all-egress,
//             allow-same-namespace, allow-dns-egress.
//
// 2. Profile-only usage:
//    profile cannot appear alongside explicit ingress, egress, or policyTypes
//    fields — profiles are atomic presets.
//
// 3. Template expressions:
//    Profile values containing "{{" are skipped at load time and validated
//    at reconcile time instead.

package katalog

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/profiles"
)

func (k *Katalog) validateNetworkPolicyProfiles() error {
	for crdName, crd := range k.enabledCRDs {
		for _, e := range crd.CollectNetworkPolicyProfileEntries() {
			if isTemplateExpr(e.Profile) {
				continue
			}
			if !k.isUserNetworkPolicyProfile(e.Profile) && !profiles.IsValidNetworkPolicyProfile(e.Profile) {
				return fmt.Errorf(
					"crd %q: networkPolicy %q (phase %s) has unknown profile %q — "+
						"allowed: deny-all, deny-all-ingress, deny-all-egress, allow-same-namespace, allow-dns-egress",
					crdName, e.ResourceName, e.Phase, e.Profile,
				)
			}
			if e.Mixed {
				return fmt.Errorf(
					"crd %q: networkPolicy %q (phase %s) declares both profile (%q) and "+
						"explicit ingress/egress/policyTypes — use one or the other, not both",
					crdName, e.ResourceName, e.Phase, e.Profile,
				)
			}
		}
	}
	return nil
}
