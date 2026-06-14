package types

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// SetupConfig declares prerequisite resources to apply before Orkestra starts.
//
// Shorthand — a plain list of strings is equivalent to setup.apply:
//
//	setup:
//	  - ./prereqs/secret.yaml
//
// Struct form:
//
//	setup:
//	  apply:
//	    - ./prereqs/secret.yaml
//	  helm:
//	    - repo: https://charts.cert-manager.io
//	      chart: cert-manager
//	      version: v1.14.0
//	  wait:
//	    - kind: Deployment
//	      name: cert-manager
//	      namespace: cert-manager
//	      ready: true
//	      timeout: 120s
type SetupConfig struct {
	// Apply is an ordered list of YAML file paths to kubectl-apply.
	// Applied first, before helm installs, after the CRD is installed.
	Apply []string `yaml:"apply,omitempty"`

	// Helm is an ordered list of Helm charts to install before Orkestra starts.
	// Executed as helm upgrade --install — not rendered for Katalog extraction.
	Helm []SetupHelmInstall `yaml:"helm,omitempty"`

	// Wait blocks until all listed resources exist and satisfy conditions.
	// Runs last. If any wait times out, setup fails and the operator does not start.
	Wait []SetupWait `yaml:"wait,omitempty"`
}

// UnmarshalYAML allows setup: to be written as either a plain list of strings
// (backward-compatible shorthand for setup.apply) or a full struct.
func (s *SetupConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.SequenceNode {
		for _, item := range value.Content {
			if item.Kind != yaml.ScalarNode {
				break
			}
			s.Apply = append(s.Apply, item.Value)
		}
		if len(s.Apply) > 0 {
			return nil
		}
	}
	type plain SetupConfig
	return value.Decode((*plain)(s))
}

// SetupHelmInstall installs a Helm chart as a real release into the cluster.
// Unlike HelmSource (which renders charts to extract Katalog documents),
// this runs helm upgrade --install for a prerequisite chart.
type SetupHelmInstall struct {
	// Repo is the Helm repository URL.
	Repo string `yaml:"repo"`
	// Chart is the chart name within the repository.
	Chart string `yaml:"chart"`
	// Release is the Helm release name. Defaults to the chart name when empty.
	Release string `yaml:"release,omitempty"`
	// Namespace for the release. Defaults to "default".
	Namespace string `yaml:"namespace,omitempty"`
	// Version pins the chart version. Leave empty for latest.
	Version string `yaml:"version,omitempty"`
	// ValueFiles is an ordered list of values files (local paths or URLs).
	ValueFiles []string `yaml:"valueFiles,omitempty"`
	// Values are inline key-value overrides, equivalent to helm --set.
	Values map[string]interface{} `yaml:"values,omitempty"`
	// CreateNamespace passes --create-namespace to helm.
	CreateNamespace bool `yaml:"createNamespace,omitempty"`
}

// ReleaseName returns the effective Helm release name.
func (h SetupHelmInstall) ReleaseName() string {
	if h.Release != "" {
		return h.Release
	}
	return h.Chart
}

// EffectiveNamespace returns the effective namespace, defaulting to "default".
func (h SetupHelmInstall) EffectiveNamespace() string {
	if h.Namespace != "" {
		return h.Namespace
	}
	return "default"
}

// Validate returns an error when required fields are missing.
func (h SetupHelmInstall) Validate() error {
	if h.Repo == "" {
		return fmt.Errorf("setup.helm: repo is required")
	}
	if h.Chart == "" {
		return fmt.Errorf("setup.helm: chart is required")
	}
	return nil
}

// SetupWait describes a single resource to wait for before the operator starts.
type SetupWait struct {
	// Kind is the Kubernetes resource kind (e.g. Deployment, Secret, Namespace).
	Kind string `yaml:"kind"`
	// Name is the exact resource name.
	Name string `yaml:"name"`
	// Namespace to look in. Omit for cluster-scoped resources.
	Namespace string `yaml:"namespace,omitempty"`
	// Ready waits for an available/ready condition, not just existence.
	Ready bool `yaml:"ready,omitempty"`
	// Timeout is a Go duration string. Default: "30s".
	Timeout string `yaml:"timeout,omitempty"`
}

// E2E is the top-level document type for declarative end-to-end tests.
// Committed alongside the katalog, it drives `ork e2e` — the same command
// that runs locally, in CI, and inside the GitHub Action.
type E2E struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Metadata   E2EMeta     `yaml:"metadata"`
	Spec       E2ESpec     `yaml:"spec"`
	Imports    []E2EImport `yaml:"imports,omitempty"`
}

// E2EImport references another E2E file to run after this one completes.
// By default imports share the same cluster. Set freshCluster: true to
// provision a new cluster for that import instead.
//
// Shorthand — a plain path string is equivalent to {path: <string>}:
//
//	imports:
//	  - ./auth-e2e.yaml
//	  - ./rbac-e2e.yaml
//	  - path: ./infra-e2e.yaml
//	    freshCluster: true
type E2EImport struct {
	// Path is the path to another E2E spec file (must be kind: E2E).
	Path string `yaml:"path"`
	// FreshCluster provisions a new kind cluster for this import instead of
	// reusing the parent's cluster. Default: false (share parent cluster).
	FreshCluster bool `yaml:"freshCluster,omitempty"`
	// Wait is an optional duration to sleep before this import starts.
	// Useful when the previous test leaves cluster state that needs time to
	// clear — webhook deregistration, namespace termination, cert provisioning.
	// Must be a valid Go duration string (e.g. "10s", "1m30s").
	Wait string `yaml:"wait,omitempty"`
}

// UnmarshalYAML allows imports to be written as a plain string path.
func (i *E2EImport) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		i.Path = value.Value
		return nil
	}
	type plain E2EImport
	return value.Decode((*plain)(i))
}

type E2EMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

type E2ESpec struct {
	// Core operator spec — the three files that define every Orkestra operator.

	// Katalog is the path to the katalog.yaml file.
	// Optional when customOperator is true.
	Katalog string `yaml:"katalog,omitempty"`
	// CRD is the path to the CRD YAML file for this operator.
	// Applied before the bundle and before Orkestra starts.
	CRD string `yaml:"crd,omitempty"`
	// CR is the path to the CR YAML file to apply during the test.
	CR string `yaml:"cr,omitempty"`

	// CustomOperator declares that this test uses its own operator rather than
	// Orkestra's reconcile loop. Bundle generation and Orkestra helm install/uninstall
	// are skipped. Everything else runs unchanged: cluster setup, CRD apply, setup
	// manifests, CR apply, assertions, and cleanup.
	//
	// Use this when your operator is installed via setup.helm or is already present
	// in the cluster. See documentation/reference/schema/04-e2e/05-custom-operator.md.
	CustomOperator bool `yaml:"customOperator,omitempty"`

	// Init uses an example pack — for Orkestra's own CI.
	Init *E2EInit `yaml:"init,omitempty"`

	// Cluster controls which cluster to use.
	Cluster E2ECluster `yaml:"cluster"`

	// Setup declares prerequisite resources to apply before Orkestra starts.
	// Shorthand: a plain list of strings applies each file (backward compatible).
	// Struct form adds helm installs and resource waiting.
	Setup *SetupConfig `yaml:"setup,omitempty"`

	// ValuesFiles is a list of Helm values files passed to the Orkestra chart
	// installation during e2e. Paths are relative to the e2e.yaml file.
	// Use this to configure custom runtime images or any other Helm values
	// without requiring --values flags on the command line — useful when e2e
	// runs automatically during ork push.
	ValuesFiles []string `yaml:"valuesFiles,omitempty"`

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
