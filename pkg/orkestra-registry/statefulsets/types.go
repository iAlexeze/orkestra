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
}
