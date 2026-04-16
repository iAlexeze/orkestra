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
}
