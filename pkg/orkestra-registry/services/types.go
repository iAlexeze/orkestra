// pkg/orkestra-registry/services/types.go
package services

// ResolvedServiceSpec is the fully resolved Service specification.
// Produced by resolving template expressions and merging static values.
// Passed directly to Create, Update, and Delete.
type ResolvedServiceSpec struct {
	// Name — Service name. Required.
	Name string

	// Namespace — target namespace. Required.
	Namespace string

	// Type — Kubernetes Service type: ClusterIP, NodePort, LoadBalancer.
	// Default: ClusterIP.
	Type string

	// Port — Service port.
	Port int32

	// TargetPort — container port the Service routes to.
	TargetPort int32

	// Labels — applied to Service metadata.
	// Orkestra always adds: managed-by=orkestra, orkestra-owner=<cr-name>
	Labels map[string]string

	// Selector —> service selector to route traffic to pods.
	Selector map[string]string
}
