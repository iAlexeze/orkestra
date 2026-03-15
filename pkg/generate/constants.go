package generate

const (
	// RuntimePackage is the fixed output directory for all generated runtime files.
	// Both the registry and declarative hook implementations are written here.
	// The Orkestra runtime imports this package directly.
	RuntimePackage = "pkg/runtime"

	// RegistryFile is the generated file containing:
	//   - ObjectRegistry
	//   - ListRegistry
	//   - HookRegistry
	//   - ReconcilerRegistry
	//   - RegisterScheme()
	//
	// It is regenerated on every `ork generate registry` invocation.
	RegistryFile = "generated_runtime_registry.go"

	// HooksFile is the generated file containing declarative hook implementations
	// for unstructured CRDs (onCreate, onReconcile, onDelete).
	// Only emitted when at least one CRD declares template-based hooks.
	HooksFile = "generated_runtime_hooks.go"

	// ExamplesDir is the output directory for generated example manifests.
	// Each CRD receives a minimal example YAML manifest for onboarding and testing.
	ExamplesDir = "_runtime_examples"

	// DocsDir is the output directory for generated Markdown documentation.
	// Includes per-CRD docs, an index, and dependency graph documentation.
	DocsDir = "_runtime_docs"

	// TestDir is the output directory for generated test scaffolding.
	// Contains unit test stubs and integration test templates for each CRD.
	TestDir = "_runtime_test"

	// DashDir is the output directory for generated Grafana dashboards.
	// Each CRD receives a dashboard JSON file with metrics panels.
	DashDir = "_runtime_dashboards"
)
