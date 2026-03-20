// pkg/orkestra-registry/deployments/types.go
package deployments

import orktypes "github.com/ialexeze/orkestra/pkg/types"

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

	// Resources — CPU and memory requests/limits. nil means no limits set.
	Resources *orktypes.ResourceRequirements
}
