package katalog

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/utils"
)

//
// NamespaceProtectionGVRs returns the list of GVRs for CRDs that declare
// allowedNamespaces or restrictedNamespaces. Only these CRDs are intercepted
// by the namespace-protection webhook.
//
// When running outside the cluster (e.g. `ork run`), the webhook cannot be
// reached, so no rules are returned.
func (k *Katalog) NamespaceProtectionGVRs() []GVREntry {
    if !k.IsNamespaceProtectionEnabled() {
        return nil
    }
    if !utils.IsRunningInCluster() {
        return nil
    }

    var out []GVREntry

    for _, crd := range k.enabledCRDs {
        // Skip built-ins — they are not CRDs and do not have namespace rules.
        if crd.IsBuiltIn {
            continue
        }

        // Skip CRDs with no namespace restrictions declared.
        if !crd.HasNamespaceRules() {
            continue
        }

        // Compose the GVR from the CRD's API types.
        out = append(out, GVREntry{
            Key:        fmt.Sprintf("%s/%s/%s", crd.APITypes.Group, crd.APITypes.Version, crd.APITypes.Plural),
            Group:      crd.APITypes.Group,
            Version:    crd.APITypes.Version,
            Resource:   crd.APITypes.Plural,
            Operations: []string{"CREATE", "UPDATE"},
        })
    }

    return out
}
