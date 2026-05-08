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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

// orkestraResourceLabels defines the labels applied to every Orkestra control-plane
// resource (Deployment, Service, ServiceAccount, ClusterRole, ClusterRoleBinding,
// webhook configurations, and the TLS Secret). The deletion-protection label is
// included so the admission webhook's objectSelector matches exactly these
// resources — it fires only for objects already carrying the label.
var orkestraResourceLabels = map[string]string{
	"app.kubernetes.io/name":          "orkestra",
	"app.kubernetes.io/tag":           "orkestra-internal",
	"orkestra.io/deletion-protection": "true",
}

// Label selector shared by all Orkestra-managed Kubernetes resources.
// Narrows the webhook to only the operator's own deployment, service, ingress,
// and admission webhook configurations (validation + mutation).
var orkestraResourceSelector = &metav1.LabelSelector{
	MatchLabels: orkestraResourceLabels,
}

// OrkestraResourceSelector returns the internal label selector for orkestra control plane resources
func OrkestraResourceSelector() *metav1.LabelSelector {
	return orkestraResourceSelector
}

// DeletionProtectionSelector returns a LabelSelector matching only the
// deletion‑protection label.
func DeletionProtectionSelector() *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			DeletionProtectionLabel: "true",
		},
	}
}

// OrkestraBaseLabels returns a copy of the standard Orkestra control-plane labels.
// It can be called without a Konfig instance — useful in generators and CLI commands
// that do not load the full operator configuration.
func OrkestraBaseLabels() map[string]string {
	out := make(map[string]string, len(orkestraResourceLabels))
	for k, v := range orkestraResourceLabels {
		out[k] = v
	}
	return out
}

// OrkestraResourceLabels returns the internal labels for orkestra control plane resources
func OrkestraResourceLabels() map[string]string {
	return orkestraResourceLabels
}
