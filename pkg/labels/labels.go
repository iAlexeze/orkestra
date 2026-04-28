// Package labels defines the label constants and helpers used by the Orkestra
// control plane to identify and protect its own Kubernetes resources.
//
// The deletion-protection label marks resources (Deployments, Services, Secrets,
// webhook configurations) that the Orkestra admission webhook should refuse to
// delete when a delete request arrives. This prevents accidental teardown of the
// operator itself.
//
// Usage:
//
//	import orklabels "github.com/orkspace/orkestra/pkg/labels"
//
//	// Apply deletion-protection on top of any existing label set:
//	labels := orklabels.WithDeletionProtection(kfg.OrkestraResourceLabels())
//
//	// Check the constant directly when building label selectors:
//	selector := metav1.LabelSelector{
//	    MatchLabels: map[string]string{orklabels.DeletionProtectionLabel: "true"},
//	}
package labels

// DeletionProtectionLabel is the label applied to all Orkestra-managed control-plane
// resources (Deployment, Service, Secrets, webhook configs) so that the deletion-protection
// admission webhook can identify and protect them from accidental deletion.
const DeletionProtectionLabel = "orkestra.io/deletion-protection"

// WithDeletionProtection returns a copy of m with DeletionProtectionLabel set to "true".
// The input map is never modified.
func WithDeletionProtection(m map[string]string) map[string]string {
	out := make(map[string]string, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	out[DeletionProtectionLabel] = "true"
	return out
}
