// pkg/types/external.go
//
// External HTTP call declarations.
//
// The external: block under onReconcile allows declarative HTTP calls
// before resource creation. Results are injected into the resolver context
// and available in subsequent template expressions and when: conditions.
//
// YAML:
//
//	onReconcile:
//	  external:
//	    - name: health-check
//	      url: "{{ .spec.serviceUrl }}/health"
//	      method: GET
//	      expectedStatus: 200
//	      continueOnError: false
//	      timeout: 5s
//
//	    - name: feature-flags
//	      url: "https://flags.internal/api/{{ .metadata.name }}"
//	      method: GET
//	      token: "$FEATURE_FLAG_TOKEN"
//	      continueOnError: true
//
//	  deployments:
//	    - name: "{{ .metadata.name }}"
//	      image: "{{ .spec.image }}"
//	      when:
//	        - field: external.health-check.status
//	          equals: "200"
//	        - field: external.feature-flags.body
//	          contains: "\"enabled\":true"
//
// Calls run sequentially in declaration order before any resource groups.
// Results are immutable once injected — a call cannot see a later call's result.
//
// Security: tokens are resolved via the standard resolver — use environment
// variable references ($TOKEN) or Kubernetes Secret references via the
// KubeReader mechanism. Never put raw secrets in Katalog YAML.
package types

// ExternalCallSpec declares one HTTP call to make before resource reconciliation.
type ExternalCallSpec struct {
	// Name is the identifier used to access the result in template expressions.
	//   name: health-check → {{ .external.health-check.status }}
	Name string `yaml:"name" json:"name"`

	// URL is the endpoint to call. Template expressions are supported.
	//   url: "{{ .spec.serviceUrl }}/health"
	//   url: "https://api.example.com/resources/{{ .metadata.name }}"
	URL string `yaml:"url" json:"url"`

	// Method is the HTTP method. Default: GET.
	Method string `yaml:"method,omitempty" json:"method,omitempty"`

	// Body is the request body for POST/PUT/PATCH requests.
	// Template expressions supported.
	//   body: '{"name": "{{ .metadata.name }}"}'
	Body string `yaml:"body,omitempty" json:"body,omitempty"`

	// Token is a bearer token for Authorization header.
	// Use $ENV_VAR syntax to reference environment variables:
	//   token: "$API_TOKEN"
	Token string `yaml:"token,omitempty" json:"token,omitempty"`

	// Headers are additional HTTP headers to include.
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`

	// Timeout is the maximum duration for this call.
	// Default: 10s. Format: "5s", "1m", "500ms"
	Timeout string `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// ExpectedStatus is the HTTP status code that signals success.
	// Default: any 2xx status.
	// When set to 200, any non-200 response is treated as a failure.
	ExpectedStatus int `yaml:"expectedStatus,omitempty" json:"expectedStatus,omitempty"`

	// ContinueOnError controls whether a failed call halts reconciliation.
	// false (default): a call failure returns an error, halting the reconcile.
	// true:            failure is logged, result has error set, reconcile continues.
	//
	// Use true for optional calls: notifications, metrics, feature flags.
	// Use false (default) for required calls: health checks, dependency readiness.
	ContinueOnError bool `yaml:"continueOnError,omitempty" json:"continueOnError,omitempty"`

	// When conditions gate this call — if conditions fail, the call is skipped.
	// The result is not injected when skipped.
	Conditions []Condition `yaml:"when,omitempty" json:"when,omitempty"`
	AnyOf      []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
}

// ExternalCallResult is the result of one HTTP call, injected into the resolver
// context under .external.<name>.
type ExternalCallResult struct {
	// Status is the HTTP status code as a string ("200", "404", "503").
	// Empty string when the call failed before receiving a response.
	Status string `json:"status" yaml:"status"`

	// Body is the first 4096 bytes of the response body.
	// Truncated to avoid unbounded memory use.
	Body string `json:"body" yaml:"body"`

	// Error is the error message when the call failed.
	// Empty on success.
	Error string `json:"error" yaml:"error"`

	// Called is "true" when the call was made, "false" when skipped (conditions failed).
	Called string `json:"called" yaml:"called"`

	// Additional values for metrics
	StatusCode      int     `json:"statusCode" yaml:"statusCode"`
	DurationSeconds float64 `json:"durationSeconds" yaml:"durationSeconds"`
}
