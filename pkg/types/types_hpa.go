// pkg/types/types_hpa.go
package types

// ── HorizontalPodAutoscaler ───────────────────────────────────────────────────

type ScaleTargetRef struct {
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`
	Kind       string `yaml:"kind" json:"kind"`
	Name       string `yaml:"name" json:"name"`
}

// HPATemplateSource declares one HorizontalPodAutoscaler to be managed by Orkestra.
//
// Example:
//
//	onReconcile:
//	  hpa:
//	    - name: "{{ .metadata.name }}-hpa"
//	      deploymentRef: "{{ .metadata.name }}"
//	      minReplicas: "{{ .spec.minReplicas }}"
//	      maxReplicas: "{{ .spec.maxReplicas }}"
//	      targetCPUUtilizationPercentage: "80"
//	      forEach:
//	        field: spec.services
//	        as: item
type HPATemplateSource struct {
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Name — HPA resource name. Default: "{{ .metadata.name }}-hpa"
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Namespace — target namespace. Default: CR namespace.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// ScaleTargetRef — the target workload this HPA scales.
	// Supports Deployment, ReplicaSet, StatefulSet, or any scalable resource.
	// Supports template expressions: "{{ .metadata.name }}"
	ScaleTargetRef ScaleTargetRef `yaml:"scaleTargetRef,omitempty" json:"scaleTargetRef,omitempty"`

	// MinReplicas — minimum replica count as a string. Supports template expressions.
	MinReplicas string `yaml:"minReplicas,omitempty" json:"minReplicas,omitempty"`

	// MaxReplicas — maximum replica count as a string. Supports template expressions.
	MaxReplicas string `yaml:"maxReplicas,omitempty" json:"maxReplicas,omitempty"`

	// TargetCPUUtilizationPercentage — CPU utilization target (0-100). Supports templates.
	// When behavior.profile is set and this field is empty, the profile provides the default.
	TargetCPUUtilizationPercentage string `yaml:"targetCPUUtilizationPercentage,omitempty" json:"targetCPUUtilizationPercentage,omitempty"`

	// Behavior — scale-up and scale-down behavior configuration.
	// Set profile for a complete preset, or configure scaleUp/scaleDown explicitly.
	// Profile and explicit fields are mutually exclusive.
	Behavior *HPABehavior `yaml:"behavior,omitempty" json:"behavior,omitempty"`

	// Labels applied to HPA metadata. Values support template expressions.
	Labels Labels `yaml:"labels,omitempty" json:"labels,omitempty"`

	Reconcile  bool         `yaml:"reconcile,omitempty" json:"reconcile,omitempty"`
	Conditions []Condition  `yaml:"when,omitempty" json:"when,omitempty"`
	AnyOf      []Condition  `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	ForEach    *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string `json:"sleep,omitempty" yaml:"sleep,omitempty"`
}
