// Package labels defines all label, annotation, and finalizer constants used by
// the Orkestra control plane. These identifiers form the contract between the
// runtime, admission webhooks, generators, CLI tooling, and developer-created
// workloads.
//
// Nothing in this package performs logic — it only provides:
//   - stable label keys
//   - stable annotation keys
//   - stable finalizer keys
//   - helpers for constructing label sets
//   - selectors used by the admission webhooks
//
// This package is intentionally dependency‑free and safe to import from any
// layer of the system (runtime, CLI, generators, komposers, motifs, etc.).
package labels

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

//
// ────────────────────────────────────────────────────────────────────────────────
//   Deletion Protection
// ────────────────────────────────────────────────────────────────────────────────
//

// DeletionProtectionLabel marks resources that must not be deleted. Any object
// carrying:
//
//	orkestra.io/deletion-protection=true
//
// will be matched by the deletion‑protection admission webhook. This protects
// both Orkestra control‑plane resources and developer‑opt‑in resources.
const (
	DeletionProtectionLabel = "orkestra.io/deletion-protection"
	// DeletionProtectionValue is the label value that enables protection.
	DeletionProtectionValue = "true"

	// StrictModeExemptKey is the label that exempts a resource from strict‑mode
	// enforcement. When present with value "true", the resource's deletion‑protection
	// label can be removed even if strictMode is enabled.
	StrictModeExemptKey   = "orkestra.io/strict-mode-exempt"
	StrictModeExemptValue = "true"
)

// WithDeletionProtection returns a copy of m with the deletion‑protection label
// set to "true". The input map is never modified.
func WithDeletionProtection(m map[string]string) map[string]string {
	out := make(map[string]string, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	out[DeletionProtectionLabel] = DeletionProtectionValue
	return out
}

// DeletionProtectionSelector returns a LabelSelector matching only the
// deletion‑protection label. Used when constructing webhook configurations.
func DeletionProtectionSelector() *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			DeletionProtectionLabel: DeletionProtectionValue,
		},
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
//   Orkestra Control‑Plane Labels
// ────────────────────────────────────────────────────────────────────────────────
//
// These labels are applied to every Orkestra control‑plane resource (Deployment,
// Service, ServiceAccount, ClusterRole, ClusterRoleBinding, webhook configs,
// TLS Secret). They allow the runtime and admission webhooks to identify the
// operator’s own components.
//

var orkestraResourceLabels = map[string]string{
	"app.kubernetes.io/name": "orkestra",
	"app.kubernetes.io/tag":  "orkestra-internal",
	DeletionProtectionLabel:  DeletionProtectionValue,
}

// OrkestraBaseLabels returns a copy of the standard Orkestra control‑plane
// labels. Useful for generators and CLI commands that do not load Konfig.
func OrkestraBaseLabels() map[string]string {
	out := make(map[string]string, len(orkestraResourceLabels))
	for k, v := range orkestraResourceLabels {
		out[k] = v
	}
	return out
}

// OrkestraResourceLabels returns the internal label map used for control‑plane
// resources. Callers must treat the returned map as read‑only.
func OrkestraResourceLabels() map[string]string {
	return orkestraResourceLabels
}

// OrkestraResourceSelector matches exactly the Orkestra control‑plane resources.
// This is used by the admission webhook for mutation/validation of operator
// components (not for deletion protection — that uses DeletionProtectionSelector).
var orkestraResourceSelector = &metav1.LabelSelector{
	MatchLabels: orkestraResourceLabels,
}

// OrkestraResourceSelector returns the selector for Orkestra control‑plane
// resources.
func OrkestraResourceSelector() *metav1.LabelSelector {
	return orkestraResourceSelector
}

//
// ────────────────────────────────────────────────────────────────────────────────
//   Ownership & Management Labels
// ────────────────────────────────────────────────────────────────────────────────
//

// Managed marks resources that Orkestra actively manages.
const (
	ManagedKey = "orkestra.orkspace.io/managed"

	// ManagedValue is always "true".
	ManagedValue = "true"

	// OrkestraOwner identifies which CR owns a generated resource. Used by
	// reconcile loops to determine whether a resource should be updated or deleted.
	OrkestraOwner = "orkestra-owner"

	// LabelCreatedBy identifies the creator of a resource.
	LabelCreatedBy = "app.kubernetes.io/createdBy"

	// CreatedByOrkDoctor marks resources created by orkdoctor. These are excluded
	// from cleanup logic even if ownership matches.
	CreatedByOrkDoctor = "orkdoctor"

	//
	// ────────────────────────────────────────────────────────────────────────────────
	//   Annotations
	// ────────────────────────────────────────────────────────────────────────────────
	//

	// AnnotationManagedBy identifies which Orkestra operator instance manages a CR.
	AnnotationManagedBy = "orkestra.orkspace.io/managed-by"
	// AnnotationManagedSince records when Orkestra first took ownership of a CR.
	AnnotationManagedSince = "orkestra.orkspace.io/managed-since"

	//
	// ────────────────────────────────────────────────────────────────────────────────
	//   Finalizers
	// ────────────────────────────────────────────────────────────────────────────────
	//

	// FinalizerOrkestra ensures cleanup runs before a CR is removed.
	FinalizerOrkestra = "orkestra.orkspace.io/finalizer"

	// CleanupFinalizer ensures cluster-scoped GC runs before a CR is removed.
	CleanupFinalizer = "orkestra.orkspace.io/cleanup"

	//
	// ────────────────────────────────────────────────────────────────────────────────
	//   Serve Provenance Annotations
	// ────────────────────────────────────────────────────────────────────────────────
	//
	// Written by the gateway on every CR it produces. Callers never set these —
	// the gateway owns them. All three are read by the provenance notes
	// (getServeTarget, getServeAlias, getServeSource) exposed via the FuncMap.

	// AnnotationServeTarget records the serve target used by the caller.
	// Example: "smartapp"
	AnnotationServeTarget = "orkestra.orkspace.io/serve-target"

	// AnnotationServeAlias records the alias used by the caller, when the
	// request arrived through a named alias rather than the primary target.
	// Empty when the primary target was used directly.
	// Example: "public", "internal", "v2"
	AnnotationServeAlias = "orkestra.orkspace.io/serve-alias"

	// AnnotationServeSource identifies the delivery mechanism.
	// Empty for direct Gateway API calls. Set by webhook handlers.
	// Values: "github", "gitlab", "slack", "pagerduty", "generic"
	AnnotationServeSource = "orkestra.orkspace.io/serve-source"

	// AnnotationServeIntent stores the raw intent payload submitted by the caller
	// as a JSON-encoded string. Injected by the Gateway API in target mode so the
	// admission webhook can make it available as .request in validation.rules —
	// enabling intent-level gates that fire on the caller's vocabulary before any
	// field translation has occurred.
	AnnotationServeIntent = "orkestra.orkspace.io/serve-intent"
)
