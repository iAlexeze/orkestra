// pkg/types/types_ingress.go
package types

// ── Ingress ───────────────────────────────────────────────────────────────────

// IngressTemplateSource declares one Ingress to be managed by Orkestra.
//
// Example:
//
//	onReconcile:
//	  ingresses:
//	    - name: "{{ .metadata.name }}-ingress"
//	      host: "{{ .spec.hostname }}"
//	      serviceName: "{{ .metadata.name }}-svc"
//	      servicePort: "{{ .spec.port }}"
//	      path: /
//	      pathType: Prefix
//	      ingressClass: nginx
//	      tls:
//	        enabled: true
//	        secretName: "{{ .metadata.name }}-tls"
//	        hosts:
//	          - "{{ .spec.hostname }}"
type IngressTemplateSource struct {
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Name — Ingress resource name. Default: "{{ .metadata.name }}-ingress"
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Namespace — target namespace. Default: CR namespace.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// Host — virtual host name for the Ingress rule.
	Host string `yaml:"host,omitempty" json:"host,omitempty"`

	// ServiceName — backend Service name this Ingress routes to.
	ServiceName string `yaml:"serviceName,omitempty" json:"serviceName,omitempty"`

	// ServicePort — backend Service port as a string. Supports template expressions.
	ServicePort string `yaml:"servicePort,omitempty" json:"servicePort,omitempty"`

	// Path — HTTP path prefix. Default: "/"
	Path string `yaml:"path,omitempty" json:"path,omitempty"`

	// PathType — Kubernetes IngressPathType: Prefix, Exact, ImplementationSpecific.
	// Default: Prefix.
	PathType string `yaml:"pathType,omitempty" json:"pathType,omitempty"`

	// IngressClass — Ingress class name (nginx, traefik, etc.). Optional.
	IngressClass string `yaml:"className,omitempty" json:"className,omitempty"`

	// Labels applied to Ingress metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels,omitempty" json:"labels,omitempty"`

	// Annotations applied to Ingress metadata. Values support template expressions.
	Annotations []ResourceLabel `yaml:"annotations,omitempty" json:"annotations,omitempty"`

	// TLS — optional TLS configuration. When tls.enabled is true, Orkestra
	// generates a self-signed TLS Secret before creating the Ingress.
	TLS *IngressTLSSpec `yaml:"tls,omitempty" json:"tls,omitempty"`

	Reconcile  bool         `yaml:"reconcile,omitempty" json:"reconcile,omitempty"`
	Conditions []Condition  `yaml:"when,omitempty" json:"when,omitempty"`
	AnyOf      []Condition  `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	ForEach    *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string `json:"sleep,omitempty" yaml:"sleep,omitempty"`
}

// IngressTLSSpec configures TLS for an Ingress resource.
// When Create is true, Orkestra generates a kubernetes.io/tls Secret before
// the Ingress is applied so the Ingress can reference it immediately.
type IngressTLSSpec struct {
	// Create — when true, create a TLS secret and populate ingress.spec.tls.
	Create bool `yaml:"create,omitempty" json:"create,omitempty"`

	// SecretName — name of the TLS secret. Supports template expressions.
	// Default: "{{ .metadata.name }}-tls"
	SecretName string `yaml:"secretName,omitempty" json:"secretName,omitempty"`

	// Hosts — list of hostnames to include in the TLS certificate SANs.
	// Each element supports template expressions.
	Hosts []string `yaml:"hosts,omitempty" json:"hosts,omitempty"`

	// ValidFor — certificate validity duration (e.g. "1y", "90d"). Default: "1y".
	ValidFor string `yaml:"validFor,omitempty" json:"validFor,omitempty"`

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string `json:"sleep,omitempty" yaml:"sleep,omitempty"`
}
