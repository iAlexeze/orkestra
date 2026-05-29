// pkg/types/types_rbac.go
package types

// ── Role / RoleBinding ────────────────────────────────────────────────────────

// PolicyRuleSpec declares one RBAC policy rule.
// String values within slices support template expressions.
type PolicyRuleSpec struct {
	APIGroups     []string `yaml:"apiGroups,omitempty" json:"apiGroups,omitempty"`
	Resources     []string `yaml:"resources,omitempty" json:"resources,omitempty"`
	Verbs         []string `yaml:"verbs,omitempty" json:"verbs,omitempty"`
	ResourceNames []string `yaml:"resourceNames,omitempty" json:"resourceNames,omitempty"`
}

// SubjectSpec declares one RBAC subject for a RoleBinding.
// Name and Namespace support template expressions.
type SubjectSpec struct {
	Kind      string `yaml:"kind,omitempty" json:"kind,omitempty"`
	Name      string `yaml:"name,omitempty" json:"name,omitempty"`
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
}

// RoleRefSpec names the Role (or ClusterRole) being bound.
// Name supports template expressions. Kind defaults to "Role".
type RoleRefSpec struct {
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty"` // Role | ClusterRole; defaults to Role
}

// RoleTemplateSource declares one namespaced Role to be managed by Orkestra.
//
// Example:
//
//	onCreate:
//	  roles:
//	    - name: "{{ .metadata.name }}-role"
//	      namespace: "{{ .metadata.name }}-ns"
//	      rules:
//	        - apiGroups: ["apps"]
//	          resources: ["deployments"]
//	          verbs: ["get", "list", "watch", "update", "patch"]
//	          resourceNames: ["{{ .metadata.name }}"]
type RoleTemplateSource struct {
	Version    string           `yaml:"version,omitempty" json:"version,omitempty"`
	Name       string           `yaml:"name,omitempty" json:"name,omitempty"`
	Namespace  string           `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Labels     []ResourceLabel  `yaml:"labels,omitempty" json:"labels,omitempty"`
	Rules      []PolicyRuleSpec `yaml:"rules,omitempty" json:"rules,omitempty"`
	Conditions []Condition      `yaml:"when,omitempty" json:"when,omitempty"`
	Reconcile  bool             `yaml:"reconcile,omitempty" json:"reconcile,omitempty"`
	ForEach    *ForEachSpec     `yaml:"forEach,omitempty" json:"forEach,omitempty"`
	AnyOf      []Condition      `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string `json:"sleep,omitempty" yaml:"sleep,omitempty"`
}

// RoleBindingTemplateSource declares one RoleBinding to be managed by Orkestra.
//
// Example:
//
//	onCreate:
//	  roleBindings:
//	    - name: "{{ .metadata.name }}-rolebinding"
//	      namespace: "{{ .metadata.name }}-ns"
//	      roleRef:
//	        name: "{{ .metadata.name }}-role"
//	      subjects:
//	        - kind: ServiceAccount
//	          name: "{{ .metadata.name }}-sa"
//	          namespace: "{{ .metadata.name }}-ns"
type RoleBindingTemplateSource struct {
	Version    string          `yaml:"version,omitempty" json:"version,omitempty"`
	Name       string          `yaml:"name,omitempty" json:"name,omitempty"`
	Namespace  string          `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Labels     []ResourceLabel `yaml:"labels,omitempty" json:"labels,omitempty"`
	RoleRef    RoleRefSpec     `yaml:"roleRef,omitempty" json:"roleRef,omitempty"`
	Subjects   []SubjectSpec   `yaml:"subjects,omitempty" json:"subjects,omitempty"`
	Conditions []Condition     `yaml:"when,omitempty" json:"when,omitempty"`
	Reconcile  bool            `yaml:"reconcile,omitempty" json:"reconcile,omitempty"`
	ForEach    *ForEachSpec    `yaml:"forEach,omitempty" json:"forEach,omitempty"`
	AnyOf      []Condition     `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string `json:"sleep,omitempty" yaml:"sleep,omitempty"`
}
