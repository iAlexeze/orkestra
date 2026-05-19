package generate

const (
	// TypeRegistryPackage is the fixed output directory for all generated typeregistry files.
	// Both the registry and declarative hook implementations are written here.
	// The Orkestra runtime imports this package directly.
	TypeRegistryPackage = "pkg/typeregistry"

	// RegistryFile is the generated file containing:
	//   - ObjectRegistry
	//   - ListRegistry
	//   - HookRegistry
	//   - ReconcilerRegistry
	//   - RegisterScheme()
	//
	// It is regenerated on every `ork generate registry` invocation.
	RegistryFile = "zz_generated_typeregistry.go"

	// DocsDir is the output directory for generated Markdown documentation.
	// Includes per-CRD docs, an index, and dependency graph documentation.
	DocsDir = "_generated/docs"

	// DashDir is the output directory for generated Grafana dashboards.
	// Each CRD receives a dashboard JSON file with metrics panels.
	DashDir = "_generated/dashboards"
)
