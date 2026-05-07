// pkg/orkestra-registry/statefulsets/types.go
package statefulsets

import orktypes "github.com/orkspace/orkestra/pkg/types"

// ResolvedStatefulSetSpec is the fully resolved StatefulSet specification.
type ResolvedStatefulSetSpec struct {
	Name        string
	Namespace   string
	Image       string
	Replicas    int32
	Port        int32
	ServiceName string

	// Storage — set when StorageClass is declared.
	StorageClass string
	StorageSize  string
	MountPath    string

	Labels      map[string]string
	Annotations map[string]string

	Env       map[string]orktypes.EnvVarSource
	EnvFrom   []orktypes.EnvFromSource
	Resources *orktypes.ResourceRequirements

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string 

	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use
	// for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	ImagePullSecrets []string

	VolumeClaimRetentionPolicy VolumeClaimRetentionPolicy
}

// VolumeClaimRetentionPolicy describes the policy used for PVCs
// created from the StatefulSet VolumeClaimTemplates.
type VolumeClaimRetentionPolicy struct {
	// WhenDeleted specifies what happens to PVCs created from StatefulSet
	// VolumeClaimTemplates when the StatefulSet is deleted. The default policy
	// of `Retain` causes PVCs to not be affected by StatefulSet deletion. The
	// `Delete` policy causes those PVCs to be deleted.
	WhenDeleted string 
	// WhenScaled specifies what happens to PVCs created from StatefulSet
	// VolumeClaimTemplates when the StatefulSet is scaled down. The default
	// policy of `Retain` causes PVCs to not be affected by a scaledown. The
	// `Delete` policy causes the associated PVCs for any excess pods above
	// the replica count to be deleted.
	WhenScaled string 
}
