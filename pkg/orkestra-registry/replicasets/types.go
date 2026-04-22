// pkg/orkestra-registry/replicasets/types.go
package replicasets

import orktypes "github.com/orkspace/orkestra/pkg/types"

// ResolvedReplicaSetSpec is the fully resolved ReplicaSet specification.
// Produced by resolving template expressions and merging static values.
// Passed directly to Create, Update, and Delete.
type ResolvedReplicaSetSpec struct {
	// Name — resolved ReplicaSet name. Required.
	Name string

	// Image — container image. Required.
	Image string

	// Replicas — number of pod replicas. Default: 1.
	Replicas int32

	// Port — container port. 0 means no port exposed.
	Port int32

	// Namespace — target namespace. Required.
	Namespace string

	// Labels — applied to the ReplicaSet and pod template.
	// Orkestra always adds: managed-by=orkestra, orkestra-owner=<cr-name>
	Labels map[string]string

	// Annotations — applied to the ReplicaSet.
	Annotations map[string]string

	// Env — environment variables.
	Env     map[string]orktypes.EnvVarSource
	EnvFrom []orktypes.EnvFromSource

	// Resources — CPU and memory requests/limits. nil means no limits set.
	Resources *orktypes.ResourceRequirements
}
