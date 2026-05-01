// pkg/orkestra-registry/pods/types.go
package pods

import orktypes "github.com/orkspace/orkestra/pkg/types"

// ResolvedPodSpec is the fully resolved Pod specification.
// Produced by merging PodFromCRD (dynamic) and PodFromKatalog (static).
// fromCRD wins over fromKatalog when both declare the same field.
// Passed directly to Create, Update, and Delete.
type ResolvedPodSpec struct {
	// Name — resolved Pod name. Required.
	Name string

	// Image — container image. Required.
	Image string

	// Namespace — target namespace. Required.
	Namespace string

	// Port — container port. 0 means no port exposed.
	Port int

	// Labels — merged labels from both sources.
	// Orkestra always adds: managed-by=orkestra, orkestra-owner=<cr-name>
	Labels map[string]string

	// Annotations — merged annotations from both sources.
	Annotations map[string]string

	// Resources — CPU and memory requests/limits. nil means no limits set.
	Resources *orktypes.ResourceRequirements

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `json:"nodeSelector,omitempty" protobuf:"bytes,7,rep,name=nodeSelector"`

	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty" protobuf:"bytes,8,opt,name=serviceAccountName"`
}
