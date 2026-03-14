package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/apimachinery/pkg/runtime/schema"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Application is a definition of an Application resource.
// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true
// +kubebuilder:resource:shortName=app
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Sync",type="string",JSONPath=".status.sync.status"
// +kubebuilder:printcolumn:name="Revision",type="string",JSONPath=".status.sync.revision"
// +kubebuilder:printcolumn:name="Health",type="string",JSONPath=".status.health.status"
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// // GetObjectKind implements [runtime.Object].
// // Subtle: this method shadows the method (TypeMeta).GetObjectKind of Application.TypeMeta.
// func (a *Application) GetObjectKind() schema.ObjectKind {
// 	panic("unimplemented")
// }

// ApplicationSpec defines the desired state of Application
// +kubebuilder:object:generate=true
type ApplicationSpec struct {
	// Source is the Git repository source
	Source ApplicationSource `json:"source"`

	// Destination is the target cluster/namespace
	Destination ApplicationDestination `json:"destination"`

	// Project is the ArgoCD project this app belongs to
	Project string `json:"project,omitempty"`

	// SyncPolicy controls sync behavior
	SyncPolicy *SyncPolicy `json:"syncPolicy,omitempty"`
}

// ApplicationSource defines the source configuration
// +kubebuilder:object:generate=true
type ApplicationSource struct {
	// RepoURL is the Git repository URL
	RepoURL string `json:"repoURL"`

	// Path is the directory path within the repository
	Path string `json:"path"`

	// TargetRevision is the Git revision (branch/tag/commit)
	TargetRevision string `json:"targetRevision"`

	// Helm specific configuration
	Helm *HelmConfig `json:"helm,omitempty"`

	// Kustomize specific configuration
	Kustomize *KustomizeConfig `json:"kustomize,omitempty"`
}

// HelmConfig holds Helm-specific options
// +kubebuilder:object:generate=true
type HelmConfig struct {
	ValueFiles  []string        `json:"valueFiles,omitempty"`
	Parameters  []HelmParameter `json:"parameters,omitempty"`
	ReleaseName string          `json:"releaseName,omitempty"`
	Values      string          `json:"values,omitempty"`
	Version     string          `json:"version,omitempty"`
	SkipCrds    bool            `json:"skipCrds,omitempty"`
}

// HelmParameter represents a Helm parameter
// +kubebuilder:object:generate=true
type HelmParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// KustomizeConfig holds Kustomize-specific options
// +kubebuilder:object:generate=true
type KustomizeConfig struct {
	NamePrefix string   `json:"namePrefix,omitempty"`
	NameSuffix string   `json:"nameSuffix,omitempty"`
	Images     []string `json:"images,omitempty"`
	Version    string   `json:"version,omitempty"`
}

// ApplicationDestination defines the target cluster/namespace
// +kubebuilder:object:generate=true
type ApplicationDestination struct {
	// Server is the API server URL of the target cluster
	Server string `json:"server"`

	// Namespace is the target namespace
	Namespace string `json:"namespace"`

	// Name is the cluster name (alternative to Server)
	Name string `json:"name,omitempty"`
}

// SyncPolicy controls sync behavior
// +kubebuilder:object:generate=true
type SyncPolicy struct {
	// Automated sync options
	Automated *SyncPolicyAutomated `json:"automated,omitempty"`

	// Sync options (e.g., "Validate=false", "Prune=false")
	SyncOptions []string `json:"syncOptions,omitempty"`

	// Retry controls failed sync retry behavior
	Retry *RetryStrategy `json:"retry,omitempty"`
}

// SyncPolicyAutomated controls automated sync behavior
// +kubebuilder:object:generate=true
type SyncPolicyAutomated struct {
	Prune    bool `json:"prune,omitempty"`
	SelfHeal bool `json:"selfHeal,omitempty"`
}

// RetryStrategy controls retry behavior
// +kubebuilder:object:generate=true
type RetryStrategy struct {
	Limit   int      `json:"limit,omitempty"`
	Backoff *Backoff `json:"backoff,omitempty"`
}

// Backoff controls retry backoff
// +kubebuilder:object:generate=true
type Backoff struct {
	Duration    string `json:"duration,omitempty"`
	Factor      int    `json:"factor,omitempty"`
	MaxDuration string `json:"maxDuration,omitempty"`
}

// ApplicationStatus defines the observed state of Application
// +kubebuilder:object:generate=true
// +kubebuilder:subresource:status
type ApplicationStatus struct {
	// Sync status summary
	Sync StatusSync `json:"sync,omitempty"`

	// Health status
	Health StatusHealth `json:"health,omitempty"`

	// History of sync operations
	History []SyncHistory `json:"history,omitempty"`

	// Conditions represent observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase is the current lifecycle phase
	Phase string `json:"phase,omitempty"`
}

// StatusSync contains sync status information
// +kubebuilder:object:generate=true
type StatusSync struct {
	Status    string   `json:"status"` // Synced, OutOfSync, Unknown
	Revision  string   `json:"revision"`
	Revisions []string `json:"revisions,omitempty"`
}

// StatusHealth contains health status information
// +kubebuilder:object:generate=true
type StatusHealth struct {
	Status  string `json:"status"` // Healthy, Degraded, Progressing, Unknown
	Message string `json:"message,omitempty"`
}

// SyncHistory represents a historical sync operation
// +kubebuilder:object:generate=true
type SyncHistory struct {
	Revision        string            `json:"revision"`
	Revisions       []string          `json:"revisions,omitempty"`
	DeployedAt      metav1.Time       `json:"deployedAt"`
	DeployStartedAt metav1.Time       `json:"deployStartedAt,omitempty"`
	Source          ApplicationSource `json:"source"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApplicationList contains a list of Application
// +kubebuilder:object:generate=true
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}
