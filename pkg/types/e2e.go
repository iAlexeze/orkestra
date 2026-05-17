package types

// E2E is the top-level document type for declarative end-to-end tests.
// Committed alongside the katalog, it drives `ork e2e` — the same command
// that runs locally, in CI, and inside the GitHub Action.
type E2E struct {
	APIVersion string  `yaml:"apiVersion"`
	Kind       string  `yaml:"kind"`
	Metadata   E2EMeta `yaml:"metadata"`
	Spec       E2ESpec `yaml:"spec"`
}

type E2EMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

type E2ESpec struct {
	// Core operator spec — the three files that define every Orkestra operator.

	// Katalog is the path to the katalog.yaml file.
	Katalog string `yaml:"katalog,omitempty"`
	// CRD is the path to the CRD YAML file for this operator.
	// Applied before the bundle and before Orkestra starts.
	CRD string `yaml:"crd,omitempty"`
	// CR is the path to the CR YAML file to apply during the test.
	CR string `yaml:"cr,omitempty"`

	// Init uses an example pack — for Orkestra's own CI.
	Init *E2EInit `yaml:"init,omitempty"`

	// Cluster controls which cluster to use.
	Cluster E2ECluster `yaml:"cluster"`

	// Setup lists YAML files to apply before Orkestra starts.
	// Use this for external dependencies: namespaces, secrets, additional CRDs.
	// Applied in order with kubectl apply -f, after spec.crd.
	Setup []string `yaml:"setup,omitempty"`

	// Expect is the list of expectations to check after each lifecycle event.
	Expect []E2EExpectation `yaml:"expect"`
}

// E2EInit selects a built-in example pack as the test source.
type E2EInit struct {
	Pack    string `yaml:"pack"`
	Example string `yaml:"example"`
}

// E2ECluster controls cluster creation and selection.
type E2ECluster struct {
	// Provider is the cluster provider — currently only "kind" is supported.
	Provider string `yaml:"provider"` // default: "kind"
	// Name is the kind cluster name. Default: "ork-e2e".
	Name string `yaml:"name"`
	// Reuse controls whether an existing cluster is reused or recreated.
	// false (default) — delete and recreate for a clean state.
	// true — reuse existing cluster if it exists.
	Reuse bool `yaml:"reuse"`
}

// E2EExpectation is one named assertion block.
type E2EExpectation struct {
	// Name is printed in the results table.
	Name string `yaml:"name"`
	// After triggers the expectation — "cr-applied" or "cr-deleted".
	After string `yaml:"after"`
	// Timeout is the maximum time to wait for the expectation to pass.
	Timeout string `yaml:"timeout"` // e.g. "60s"

	// Resources is a unified list of resource checks across any kind.
	Resources []E2EResourceCheck `yaml:"resources,omitempty"`

	// Commands are shell commands checked in the same polling loop as resources.
	Commands []E2ECommand `yaml:"commands,omitempty"`
}

// E2EResourceCheck asserts the state of any Kubernetes resource.
// Set kind to: Deployment, Service, Secret, ConfigMap.
type E2EResourceCheck struct {
	// Kind is the resource kind: Deployment, Service, Secret, ConfigMap.
	Kind string `yaml:"kind"`
	// Name is the resource name. Empty means any resource of this kind in the namespace.
	Name string `yaml:"name,omitempty"`
	// Namespace to look in. Empty means "default".
	Namespace string `yaml:"namespace,omitempty"`
	// Count asserts the exact number of matching resources. nil means "at least 1".
	// Set to 0 to assert none exist (cleanup check).
	Count *int `yaml:"count,omitempty"`
	// Ready asserts at least one available replica (Deployments only).
	Ready bool `yaml:"ready,omitempty"`
}

// E2ECommand runs a shell command and asserts its result.
type E2ECommand struct {
	// Run is a shell command string executed via sh -c.
	Run string `yaml:"run"`
	// ExitCode is the expected exit code. Default 0 (success).
	// Set to non-zero to assert the command must fail — useful for
	// admission webhook rejection tests.
	ExitCode int `yaml:"exitCode,omitempty"`
	// OutputContains asserts the combined stdout+stderr contains this substring.
	OutputContains string `yaml:"outputContains,omitempty"`
}
