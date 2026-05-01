// pkg/orkestra-registry/deployments/types.go
package deployments

import orktypes "github.com/orkspace/orkestra/pkg/types"

// ResolvedDeploymentSpec is the fully resolved Deployment specification.
// Produced by resolving template expressions and merging static values.
// Passed directly to Create, Update, and Delete.
type ResolvedDeploymentSpec struct {
	// Name — resolved Deployment name. Required.
	Name string

	// Image — container image. Required.
	Image string

	// Replicas — number of pod replicas. Default: 1.
	Replicas int32

	// Port — container port. 0 means no port exposed.
	Port int32

	// Namespace — target namespace. Required.
	Namespace string

	// Labels — applied to Deployment and pod template.
	// Orkestra always adds: managed-by=orkestra, orkestra-owner=<cr-name>
	Labels map[string]string

	// Annotations — applied to the Deployment.
	Annotations map[string]string

	// Env — environment variables.
	Env     map[string]orktypes.EnvVarSource
	EnvFrom []orktypes.EnvFromSource

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
