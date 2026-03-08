// +groupName=platform.ialexeze.io
package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:generate=true
type ManagedNamespaceSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Team string `json:"team"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default={"cpu":"4","memory":"8Gi"}
	Quota ResourceQuotaSpec `json:"quota,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	NetworkPolicy bool `json:"networkPolicy,omitempty"`
}

// +kubebuilder:object:generate=true
type ManagedNamespaceStatus struct {
	// Standard condition slice — same pattern kubectl uses for Deployment
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	Phase      string             `json:"phase,omitempty"`
}

// Object-level markers — generate CRD, enable status subresource, define kubectl columns
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=mn
// +kubebuilder:printcolumn:name="Team",type="string",JSONPath=".spec.team"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ManagedNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ManagedNamespaceSpec   `json:"spec,omitempty"`
	Status            ManagedNamespaceStatus `json:"status,omitempty"`
}

// ManagedNamespaceList contains a list of ManagedNamespace
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
type ManagedNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedNamespace `json:"items"`
}

// +kubebuilder:object:generate=true
type ResourceQuotaSpec struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}
