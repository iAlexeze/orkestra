// pkg/orktypes/types.go
package orktypes

import (
	"time"

	"github.com/ialexeze/orkestra/domain"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ── Registries ────────────────────────────────────────────────────────────────
// Package-level registries — one set per Orkestra instance.
// Populated by RegisterRuntimeObjects() in zz_generated_runtime_registry.go,
// which is produced by `ork generate runtime --katalog <path>`.
// Keyed by schema.GroupVersionKind. Set during Katalog validation.
//
// User code never reads or writes these directly. Orkestra reads them during
// Katalog validation via addRuntimeObjects(), addHooks(), and addReconcilers().

var ObjectRegistry = map[schema.GroupVersionKind]func() runtime.Object{}
var ListRegistry = map[schema.GroupVersionKind]func() runtime.Object{}
var HookRegistry = map[schema.GroupVersionKind]func() domain.AnyReconcileHooks{}
var ReconcilerRegistry = map[schema.GroupVersionKind]NewReconcilerFunc{}

// ── CRDMode ────────────────────────────────────────────────────────────

// CRDMode controls how the GenericReconciler handles CR objects at runtime.
//
// typed
//
//	Requires compiled API types via apiTypes.location.
//	Objects are decoded into concrete Go structs by the REST client.
//	Full type safety and generated DeepCopy methods.
//	Required when Go hooks reference typed fields directly.
//	Use when you have generated API types and want compile-time guarantees.
//
// dynamic
//
//	No compiled types needed. Objects are map[string]interface{} at runtime.
//	Works with any CRD without code generation or controller-gen.
//	Required for declarative hook templates (onCreate, onReconcile, onDelete)
//	because field values are resolved at reconcile time via Go text/template
//	expressions against the live CR object map.
//	Use when you want zero-code operator behavior from the Katalog alone.
//
// Auto-detection when mode is omitted:
//
//	apiTypes.location is set   → typed       (compiled types available)
//	apiTypes.location is empty → dynamic (no compiled types)
//
// Override auto-detection by setting mode explicitly:
//
//	crd:
//	 - name: websites
//	   mode: dynamic   		# force dynamic even if location is set
//	   mode: typed          # force typed even if location is empty
type CRDMode string

const (
	CRDModeTyped   CRDMode = "typed"
	CRDModeDynamic CRDMode = "dynamic"
)

// ── APITypes ──────────────────────────────────────────────────────────────────
// Mirrors the apiTypes block in crd-katalog.yaml.
// ork generate reads this block to emit ObjectRegistry + ListRegistry entries
// and the RegisterScheme() function.

type APITypes struct {
	// Object — Go type name for a single CR instance. Required for typed mode.
	// Used by ork generate to emit ObjectRegistry entries.
	// e.g. "Project" → func() runtime.Object { return &projv1.Project{} }
	Object string `yaml:"object" validate:"omitempty"`

	// List — Go type name for the CR list. Required for typed mode.
	// Used by ork generate to emit ListRegistry entries.
	// e.g. "ProjectList" → func() runtime.Object { return &projv1.ProjectList{} }
	List string `yaml:"list" validate:"omitempty"`

	// Alias — Go import alias for the API types package. Optional.
	// Auto-derived from the last two segments of Location if not set.
	// e.g. "projv1" → import projv1 "github.com/.../project/v1alpha1"
	Alias string `yaml:"alias" validate:"omitempty"`

	// Group — Kubernetes API group. Required in all modes.
	// e.g. "platform.orkestra.io"
	Group string `yaml:"group" validate:"required,hostname_rfc1123"`

	// Version — API version. Required in all modes.
	// e.g. "v1alpha1"
	Version string `yaml:"version" validate:"required"`

	// Kind — resource Kind. Required in all modes.
	// e.g. "Project"
	Kind string `yaml:"kind" validate:"required"`

	// Plural — lowercase plural resource name. Required in all modes.
	// Used for REST client URL construction.
	// e.g. "projects"
	Plural string `yaml:"plural" validate:"required"`

	// APIPath — REST API path prefix. Default: /apis.
	// Override to /api only for core Kubernetes types (Pod, ConfigMap, etc.)
	// Almost always leave this empty — Orkestra defaults it to /apis.
	APIPath string `yaml:"apiPath" validate:"omitempty"`

	// Location — fully qualified Go import path for the API types package.
	// Required for typed mode. Used by ork generate for import statements
	// and scheme registration in RegisterScheme().
	// Not needed for dynamic mode — omit entirely.
	// e.g. "github.com/ialexeze/orkestra/api/types/project/v1alpha1"
	Location string `yaml:"location" validate:"omitempty"`
}

// ── Queue ─────────────────────────────────────────────────────────────────────

type Queue struct {
	// Default: true — uses the shared default workqueue instead of a per-CRD queue.
	// Suitable for low-volume CRDs where queue isolation is not required.
	// Default: false — each CRD gets its own isolated workqueue.
	Default *bool `yaml:"default"`

	// MaxQueueDepth — maximum number of items in the per-CRD queue.
	// New items are rejected when the queue is full.
	// 0 → uses Orkestra-level default set by MAX_QUEUE_DEPTH env var.
	MaxQueueDepth int `yaml:"maxQueueDepth" validate:"omitempty,gte=0"`

	// DegradeThreshold — number of consecutive reconcile failures before the
	// CRD health state transitions from healthy to degraded.
	// 0 → uses Orkestra-level default.
	DegradeThreshold int `yaml:"degradeThreshold" validate:"omitempty,gte=0"`
}

// ── Shared resource value types ───────────────────────────────────────────────

// ResourceLabel is a single key-value label or annotation pair.
// When used in hook template declarations, values support Go text/template
// expressions evaluated against the live CR at reconcile time.
// e.g. {key: "app", value: "{{ .metadata.name }}"}
type ResourceLabel struct {
	Key   string `yaml:"key"   validate:"required"`
	Value string `yaml:"value" validate:"required"`
}

// ResourceRequirements mirrors Kubernetes resource requests and limits.
// Values are static Kubernetes quantity strings — template expressions
// are not supported here.
// e.g. requests: {cpu: "100m", memory: "128Mi"}
type ResourceRequirements struct {
	Requests map[string]string `yaml:"requests" validate:"omitempty"`
	Limits   map[string]string `yaml:"limits"   validate:"omitempty"`
}

// ── Hook template source types — flat format ────────────────────────
//
// All template source types use a single flat field layout.
//
// Instead, Orkestra uses:
//   Any string field containing "{{" → treated as a Go text/template expression.
//                                       Evaluated against the live CR at reconcile time.
//   Any string field without "{{"    → treated as a static value. Used as-is.
//
// This means the same field can hold either a static value or a CR field reference
// without any additional YAML structure:
//
//   image: "nginx:latest"                 static — same for every CR of this type
//   image: "{{ .spec.image }}"            dynamic — resolved from CR spec at reconcile time
//   name:  "{{ .metadata.name }}-app"     dynamic — CR name with a static suffix
//   port:  "8080"                         static integer string
//   port:  "{{ .spec.port }}"             dynamic integer string
//
// Template context is the full CR object as map[string]interface{}:
//   .metadata.name        CR name
//   .metadata.namespace   CR namespace
//   .metadata.labels      CR labels map
//   .spec.*               any spec field (dynamic mode only — full spec accessible)
//   .status.*             any status field
//
// All resources created by hook templates receive owner references pointing to the CR.
// This means cascade deletion is automatic — child resources are garbage collected
// when the CR is deleted without requiring explicit onDelete declarations for most cases.
//
// version field — optional OrkestraRegistry implementation version to pin.
//   Omit → uses the latest implementation shipped with this Orkestra version.
//   Set  → pins to a specific OrkestraRegistry release tag for stability.
//   e.g. version: v1.2.0

// ── Deployment ────────────────────────────────────────────────────────────────

// DeploymentTemplateSource declares one Deployment to be managed by Orkestra.
//
// Declare under onCreate to create the Deployment on first reconcile.
// This automatically creates onReconcile
// Declare under onReconcile to apply drift correction on every reconcile.
// Declare under both to get idempotent creation and drift correction together.
// Or simply declare under onCreate
//
// Minimal example — static values only:
//
//	onCreate:
//	  deployments:
//	    - image: nginx:1.25
//	      replicas: "3"
//	      port: "8080"
//
// Full example — dynamic values from the CR:
//
//	onCreate:
//	  deployments:
//	    - name: "{{ .metadata.name }}-app"
//	      image: "{{ .spec.image }}"
//	      replicas: "{{ .spec.replicas }}"
//	      port: "{{ .spec.port }}"
//	      namespace: "{{ .metadata.namespace }}"
//	      labels:
//	        - key: app
//	          value: "{{ .metadata.name }}"
//	        - key: managed-by
//	          value: orkestra
//	      resources:
//	        requests:
//	          cpu: 100m
//	          memory: 128Mi
//	        limits:
//	          cpu: 500m
//	          memory: 512Mi
type DeploymentTemplateSource struct {
	// Version — OrkestraRegistry implementation version to use. Omit for latest.
	Version string `yaml:"version" validate:"omitempty"`

	// Name — Deployment and primary container name.
	// Supports template expressions.
	// Default when omitted: "{{ .metadata.name }}-deployment"
	Name string `yaml:"name" validate:"omitempty"`

	// Image — container image. Required (must be declared here or resolvable from CR).
	// Static:  "nginx:1.25"
	// Dynamic: "{{ .spec.image }}"
	Image string `yaml:"image" validate:"omitempty"`

	// Replicas — number of pod replicas as a string.
	// Static:  "3"
	// Dynamic: "{{ .spec.replicas }}"
	// Default: "1"
	Replicas string `yaml:"replicas" validate:"omitempty"`

	// Port — primary container port as a string.
	// Static:  "8080"
	// Dynamic: "{{ .spec.port }}"
	// Omit to expose no port.
	Port string `yaml:"port" validate:"omitempty"`

	// Namespace — target namespace for the Deployment.
	// Default when omitted: "{{ .metadata.namespace }}" (same namespace as the CR).
	Namespace string `yaml:"namespace" validate:"omitempty"`

	// Labels — applied to the Deployment ObjectMeta and the pod template.
	// Label values support template expressions.
	// Orkestra always adds: managed-by=orkestra, orkestra-owner=<cr-name>
	Labels []ResourceLabel `yaml:"labels" validate:"omitempty"`

	// Annotations — applied to the Deployment ObjectMeta only.
	// Annotation values support template expressions.
	Annotations []ResourceLabel `yaml:"annotations" validate:"omitempty"`

	// Resources — CPU and memory requests/limits for the primary container.
	// Values are static Kubernetes quantity strings.
	// Template expressions are not supported in resource quantities.
	Resources *ResourceRequirements `yaml:"resources" validate:"omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile" validate:"omitempty"`

	// Conditions declares the set of runtime predicates that must all evaluate to
	// true for this resource template to be applied during reconciliation.
	//
	// Each condition inspects a field on the live Custom Resource using dot-notation
	// (e.g. "spec.enabled", "metadata.labels.tier") and compares it against a value
	// using the chosen operator. All conditions in the list are AND‑ed together.
	//
	// If any condition fails, the resource is skipped for that reconcile cycle.
	// This is not an error — it simply means “do not create/update this resource
	// right now”. This enables expressive, data‑driven orchestration such as:
	//
	//   when:
	//     - field: spec.exposePublicly
	//       equals: "true"
	//     - field: spec.environment
	//       prefix: "prod"
	//
	// Conditions allow templates to be selectively activated based on the CR’s
	// state, enabling dynamic topologies, feature flags, environment‑specific
	// behavior, and conditional provisioning without writing Go code.

	Conditions []Condition `yaml:"when,omitempty"`
}

// ── Service ───────────────────────────────────────────────────────────────────

// ServiceTemplateSource declares one Service to be managed by Orkestra.
//
// Example:
//
//	onCreate:
//	  services:
//	    - name: "{{ .metadata.name }}-svc"
//	      type: ClusterIP
//	      port: "80"
//	      targetPort: "8080"
//	      namespace: "{{ .metadata.namespace }}"
//	      labels:
//	        - key: app
//	          value: "{{ .metadata.name }}"
type ServiceTemplateSource struct {
	// Version — OrkestraRegistry implementation version. Omit for latest.
	Version string `yaml:"version" validate:"omitempty"`

	// Name — Service name.
	// Default when omitted: "{{ .metadata.name }}-svc"
	Name string `yaml:"name" validate:"omitempty"`

	// Type — Kubernetes Service type.
	// Accepted values: ClusterIP, NodePort, LoadBalancer.
	// Default: ClusterIP.
	Type string `yaml:"type" validate:"omitempty"`

	// Port — Service port as a string.
	// Static: "80" or Dynamic: "{{ .spec.servicePort }}"
	Port string `yaml:"port" validate:"omitempty"`

	// TargetPort — container port the Service routes traffic to.
	// Static: "8080" or Dynamic: "{{ .spec.containerPort }}"
	TargetPort string `yaml:"targetPort" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" validate:"omitempty"`

	// Labels — applied to Service metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" validate:"omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile" validate:"omitempty"`

	// Conditions declares the set of runtime predicates that must all evaluate to
	// true for this resource template to be applied during reconciliation.
	//
	// Each condition inspects a field on the live Custom Resource using dot-notation
	// (e.g. "spec.enabled", "metadata.labels.tier") and compares it against a value
	// using the chosen operator. All conditions in the list are AND‑ed together.
	//
	// If any condition fails, the resource is skipped for that reconcile cycle.
	// This is not an error — it simply means “do not create/update this resource
	// right now”. This enables expressive, data‑driven orchestration such as:
	//
	//   when:
	//     - field: spec.exposePublicly
	//       equals: "true"
	//     - field: spec.environment
	//       prefix: "prod"
	//
	// Conditions allow templates to be selectively activated based on the CR’s
	// state, enabling dynamic topologies, feature flags, environment‑specific
	// behavior, and conditional provisioning without writing Go code.

	Conditions []Condition `yaml:"when,omitempty"`
}

// ── Pod ───────────────────────────────────────────────────────────────────────

// PodTemplateSource declares one Pod to be managed by Orkestra.
//
// Prefer DeploymentTemplateSource for long-running workloads.
// Deployments manage Pod restarts, rolling updates, and replica sets automatically.
// Use PodTemplateSource only when you need direct, single-instance Pod control.
//
// Example:
//
//	onCreate:
//	  pods:
//	    - name: "{{ .metadata.name }}-worker"
//	      image: "{{ .spec.workerImage }}"
//	      port: "9090"
type PodTemplateSource struct {
	// Version — OrkestraRegistry implementation version. Omit for latest.
	Version string `yaml:"version" validate:"omitempty"`

	// Name — Pod name.
	// Default when omitted: "{{ .metadata.name }}-pod"
	Name string `yaml:"name" validate:"omitempty"`

	// Image — container image. Required.
	// Static: "busybox:1.35" or Dynamic: "{{ .spec.image }}"
	Image string `yaml:"image" validate:"omitempty"`

	// Port — container port as a string.
	// Static: "8080" or Dynamic: "{{ .spec.port }}"
	Port string `yaml:"port" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" validate:"omitempty"`

	// Labels — applied to Pod metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" validate:"omitempty"`

	// Annotations — applied to Pod metadata. Values support template expressions.
	Annotations []ResourceLabel `yaml:"annotations" validate:"omitempty"`

	// Resources — static CPU and memory requests/limits.
	Resources *ResourceRequirements `yaml:"resources" validate:"omitempty"`

	// Conditions declares the set of runtime predicates that must all evaluate to
	// true for this resource template to be applied during reconciliation.
	//
	// Each condition inspects a field on the live Custom Resource using dot-notation
	// (e.g. "spec.enabled", "metadata.labels.tier") and compares it against a value
	// using the chosen operator. All conditions in the list are AND‑ed together.
	//
	// If any condition fails, the resource is skipped for that reconcile cycle.
	// This is not an error — it simply means “do not create/update this resource
	// right now”. This enables expressive, data‑driven orchestration such as:
	//
	//   when:
	//     - field: spec.exposePublicly
	//       equals: "true"
	//     - field: spec.environment
	//       prefix: "prod"
	//
	// Conditions allow templates to be selectively activated based on the CR’s
	// state, enabling dynamic topologies, feature flags, environment‑specific
	// behavior, and conditional provisioning without writing Go code.

	Conditions []Condition `yaml:"when,omitempty"`
}

// ── Job ───────────────────────────────────────────────────────────────────────

// JobTemplateSource declares one Job to be run by Orkestra.
//
// Most commonly used under onDelete for cleanup tasks that must complete
// before Orkestra removes finalizers from the CR:
//   - Draining queues or buffers
//   - Archiving state to external storage
//   - Notifying external systems of deletion
//   - Running database migrations before removing a CRD instance
//
// Can also be used under onCreate for one-time provisioning tasks.
//
// Example (cleanup on delete):
//
//	onDelete:
//	  jobs:
//	    - name: "{{ .metadata.name }}-cleanup"
//	      image: busybox
//	      command: ["sh", "-c", "echo cleaning up {{ .metadata.name }}"]
//	      backoffLimit: 3
type JobTemplateSource struct {
	// Version — OrkestraRegistry implementation version. Omit for latest.
	Version string `yaml:"version" validate:"omitempty"`

	// Name — Job name.
	// Default when omitted: "{{ .metadata.name }}-job"
	Name string `yaml:"name" validate:"omitempty"`

	// Image — container image. Required.
	Image string `yaml:"image" validate:"omitempty"`

	// Command — container entrypoint command.
	// Each element is resolved independently — template expressions are supported per element.
	// e.g. ["sh", "-c", "echo cleaning up {{ .metadata.name }}"]
	Command []string `yaml:"command" validate:"omitempty"`

	// Args — arguments passed to the container command.
	// Each element supports template expressions independently.
	Args []string `yaml:"args" validate:"omitempty"`

	// BackoffLimit — number of Pod restart attempts before the Job is marked Failed.
	// Default: 3.
	BackoffLimit int `yaml:"backoffLimit" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" validate:"omitempty"`

	// Labels — applied to Job metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" validate:"omitempty"`

	// Conditions declares the set of runtime predicates that must all evaluate to
	// true for this resource template to be applied during reconciliation.
	//
	// Each condition inspects a field on the live Custom Resource using dot-notation
	// (e.g. "spec.enabled", "metadata.labels.tier") and compares it against a value
	// using the chosen operator. All conditions in the list are AND‑ed together.
	//
	// If any condition fails, the resource is skipped for that reconcile cycle.
	// This is not an error — it simply means “do not create/update this resource
	// right now”. This enables expressive, data‑driven orchestration such as:
	//
	//   when:
	//     - field: spec.exposePublicly
	//       equals: "true"
	//     - field: spec.environment
	//       prefix: "prod"
	//
	// Conditions allow templates to be selectively activated based on the CR’s
	// state, enabling dynamic topologies, feature flags, environment‑specific
	// behavior, and conditional provisioning without writing Go code.

	Conditions []Condition `yaml:"when,omitempty"`
}

// ── CronJob ───────────────────────────────────────────────────────────────────

// CronJobTemplateSource declares one CronJob to be managed by Orkestra.
//
// Example:
//
//	onCreate:
//	  cronJobs:
//	    - name: "{{ .metadata.name }}-sync"
//	      schedule: "{{ .spec.syncSchedule }}"
//	      image: "{{ .spec.syncImage }}"
//	      command: ["sh", "-c", "sync.sh"]
type CronJobTemplateSource struct {
	// Version — OrkestraRegistry implementation version. Omit for latest.
	Version string `yaml:"version" validate:"omitempty"`

	// Name — CronJob name.
	// Default when omitted: "{{ .metadata.name }}-cronjob"
	Name string `yaml:"name" validate:"omitempty"`

	// Schedule — cron schedule expression. Required.
	// Static: "0 * * * *" (every hour)
	// Dynamic: "{{ .spec.schedule }}"
	Schedule string `yaml:"schedule" validate:"required"`

	// Image — container image. Required.
	Image string `yaml:"image" validate:"omitempty"`

	// Command — container entrypoint. Each element supports template expressions.
	Command []string `yaml:"command" validate:"omitempty"`

	// Args — container arguments. Each element supports template expressions.
	Args []string `yaml:"args" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" validate:"omitempty"`

	// Labels — applied to CronJob metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" validate:"omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile" validate:"omitempty"`

	// Conditions declares the set of runtime predicates that must all evaluate to
	// true for this resource template to be applied during reconciliation.
	//
	// Each condition inspects a field on the live Custom Resource using dot-notation
	// (e.g. "spec.enabled", "metadata.labels.tier") and compares it against a value
	// using the chosen operator. All conditions in the list are AND‑ed together.
	//
	// If any condition fails, the resource is skipped for that reconcile cycle.
	// This is not an error — it simply means “do not create/update this resource
	// right now”. This enables expressive, data‑driven orchestration such as:
	//
	//   when:
	//     - field: spec.exposePublicly
	//       equals: "true"
	//     - field: spec.environment
	//       prefix: "prod"
	//
	// Conditions allow templates to be selectively activated based on the CR’s
	// state, enabling dynamic topologies, feature flags, environment‑specific
	// behavior, and conditional provisioning without writing Go code.

	Conditions []Condition `yaml:"when,omitempty"`
}

// ── ConfigMap ─────────────────────────────────────────────────────────────────

// ConfigMapTemplateSource declares one ConfigMap to be managed by Orkestra.
//
// ConfigMap data values are static — template expressions are not evaluated
// in ConfigMap data entries. For dynamic configuration, use a custom Go hook.
//
// Example:
//
//	onCreate:
//	  configMaps:
//	    - name: "{{ .metadata.name }}-config"
//	      data:
//	        LOG_LEVEL: info
//	        MAX_CONNECTIONS: "100"
type ConfigMapTemplateSource struct {
	// Version — OrkestraRegistry implementation version. Omit for latest.
	Version string `yaml:"version" validate:"omitempty"`

	// Name — ConfigMap name.
	// Default when omitted: "{{ .metadata.name }}-config"
	Name string `yaml:"name" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" validate:"omitempty"`

	// ToNamespaces - a list of target namespaces
	// Default when omitted: "{{ .metadata.namespace }}"
	ToNamespaces []string `yaml:"toNamespaces" validate:"omitempty"`

	// Data — static key-value configuration entries.
	// Values are plain strings — template expressions are not supported here.
	Data map[string]string `yaml:"data" validate:"omitempty"`

	// Labels — applied to ConfigMap metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" validate:"omitempty"`
	// FromConfigMap — name of an existing ConfigMap to copy data from.
	// Orkestra reads this at reconcile time — copies stay in sync with the source.
	FromConfigMap string `yaml:"fromConfigMap" validate:"omitempty"`

	// FromNamespace — namespace where FromConfigMap lives.
	// Default: same namespace as the CR.
	FromNamespace string `yaml:"fromNamespace" validate:"omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile" validate:"omitempty"`

	// Conditions declares the set of runtime predicates that must all evaluate to
	// true for this resource template to be applied during reconciliation.
	//
	// Each condition inspects a field on the live Custom Resource using dot-notation
	// (e.g. "spec.enabled", "metadata.labels.tier") and compares it against a value
	// using the chosen operator. All conditions in the list are AND‑ed together.
	//
	// If any condition fails, the resource is skipped for that reconcile cycle.
	// This is not an error — it simply means “do not create/update this resource
	// right now”. This enables expressive, data‑driven orchestration such as:
	//
	//   when:
	//     - field: spec.exposePublicly
	//       equals: "true"
	//     - field: spec.environment
	//       prefix: "prod"
	//
	// Conditions allow templates to be selectively activated based on the CR’s
	// state, enabling dynamic topologies, feature flags, environment‑specific
	// behavior, and conditional provisioning without writing Go code.

	Conditions []Condition `yaml:"when,omitempty"`
}

// ── Secret ─────────────────────────────────────────────────────────────────────

// SecretTemplateSource declares one Secret to be managed by Orkestra.
//
// Secret data values are static — template expressions are not evaluated
// in Secret data entries. For dynamic configuration, use a custom Go hook.
//
// Example:
//
//	onCreate:
//	  secrets:
//	    - name: "{{ .metadata.name }}-credentials"
//	      type: Opaque
//	      data:
//	        USERNAME: admin
//	        PASSWORD: "supersecret"
//
// You may also copy from an existing Secret using FromSecret.
type SecretTemplateSource struct {
	// Version — OrkestraRegistry implementation version. Omit for latest.
	Version string `yaml:"version" validate:"omitempty"`

	// Name — Secret name.
	// Default when omitted: "{{ .metadata.name }}-secret"
	Name string `yaml:"name" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" validate:"omitempty"`

	// Type — Kubernetes Secret type.
	// Default: "Opaque"
	Type string `yaml:"type" validate:"omitempty"`

	// Data — static key-value entries.
	// Values are plain strings — template expressions are not supported here.
	// If you need templated or dynamic values, use a custom Go hook.
	Data map[string]string `yaml:"data" validate:"omitempty"`

	// Labels — applied to Secret metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" validate:"omitempty"`

	// FromSecret — name of an existing Secret to copy data from.
	// Orkestra reads this at reconcile time — copies stay in sync with the source.
	FromSecret string `yaml:"fromSecret" validate:"omitempty"`

	// FromNamespace — namespace where FromSecret lives.
	// Default: same namespace as the CR.
	FromNamespace string `yaml:"fromNamespace" validate:"omitempty"`

	// ToNamespaces - a list of target namespaces
	// Default when omitted: "{{ .metadata.namespace }}"
	ToNamespaces []string `yaml:"toNamespaces" validate:"omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile" validate:"omitempty"`

	// Conditions declares the set of runtime predicates that must all evaluate to
	// true for this resource template to be applied during reconciliation.
	//
	// Each condition inspects a field on the live Custom Resource using dot-notation
	// (e.g. "spec.enabled", "metadata.labels.tier") and compares it against a value
	// using the chosen operator. All conditions in the list are AND‑ed together.
	//
	// If any condition fails, the resource is skipped for that reconcile cycle.
	// This is not an error — it simply means “do not create/update this resource
	// right now”. This enables expressive, data‑driven orchestration such as:
	//
	//   when:
	//     - field: spec.exposePublicly
	//       equals: "true"
	//     - field: spec.environment
	//       prefix: "prod"
	//
	// Conditions allow templates to be selectively activated based on the CR’s
	// state, enabling dynamic topologies, feature flags, environment‑specific
	// behavior, and conditional provisioning without writing Go code.

	Conditions []Condition `yaml:"when,omitempty"`
}

// ── ServiceAccount ────────────────────────────────────────────────────────────

// ServiceAccountTemplateSource declares one ServiceAccount to be managed by Orkestra.
//
// Example:
//
//	onCreate:
//	  serviceAccounts:
//	    - name: "{{ .metadata.name }}-sa"
//	      namespace: "{{ .metadata.namespace }}"
//	      labels:
//	        - key: app
//	          value: "{{ .metadata.name }}"
type ServiceAccountTemplateSource struct {
	// Version — OrkestraRegistry implementation version. Omit for latest.
	Version string `yaml:"version" validate:"omitempty"`

	// Name — ServiceAccount name.
	// Default when omitted: "{{ .metadata.name }}-sa"
	Name string `yaml:"name" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" validate:"omitempty"`

	// Labels — applied to ServiceAccount metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" validate:"omitempty"`

	// Conditions declares the set of runtime predicates that must all evaluate to
	// true for this resource template to be applied during reconciliation.
	//
	// Each condition inspects a field on the live Custom Resource using dot-notation
	// (e.g. "spec.enabled", "metadata.labels.tier") and compares it against a value
	// using the chosen operator. All conditions in the list are AND‑ed together.
	//
	// If any condition fails, the resource is skipped for that reconcile cycle.
	// This is not an error — it simply means “do not create/update this resource
	// right now”. This enables expressive, data‑driven orchestration such as:
	//
	//   when:
	//     - field: spec.exposePublicly
	//       equals: "true"
	//     - field: spec.environment
	//       prefix: "prod"
	//
	// Conditions allow templates to be selectively activated based on the CR’s
	// state, enabling dynamic topologies, feature flags, environment‑specific
	// behavior, and conditional provisioning without writing Go code.

	Conditions []Condition `yaml:"when,omitempty"`
}

// ── HookTemplates ─────────────────────────────────────────────────────────────
// Declares the complete set of resources Orkestra manages at each lifecycle event.
// All resource type slices are optional — omit any type you do not need.
// Resources not declared in HookTemplates are never created, updated, or deleted
// by Orkestra — they are invisible to the reconciler.
//
// All resources created via hook templates receive owner references pointing to
// the CR. This means Kubernetes garbage collection handles deletion automatically
// when the CR is deleted — no onDelete declaration is needed for cleanup in most cases.
//
// Lifecycle events:
//
//	onCreate
//	  Runs on every reconcile. Create calls are idempotent — if the resource
//	  already exists it is skipped without error.
//	  Declare all long-lived child resources here.
//	  Resources are created in the order declared within each type slice.
//
//	onReconcile
//	  Runs on every reconcile, after onCreate.
//	  Use for drift correction — re-applies desired state when child resources
//	  have been manually modified, scaled, or deleted outside of Orkestra.
//	  Omit entirely if onCreate alone is sufficient (no drift correction needed).
//
//	onDelete
//	  Runs when the CR has a DeletionTimestamp set, before Orkestra removes finalizers.
//	  Use only for resources that need explicit cleanup beyond owner references:
//	    - External resources not in Kubernetes (cloud provider APIs, DNS records, etc.)
//	    - Jobs that must complete successfully before the CR can be considered deleted
//	    - Notification or archival tasks that must run before deletion is finalized
type HookTemplates struct {
	Deployments     []DeploymentTemplateSource     `yaml:"deployments"     validate:"omitempty"`
	Services        []ServiceTemplateSource        `yaml:"services"        validate:"omitempty"`
	Pods            []PodTemplateSource            `yaml:"pods"            validate:"omitempty"`
	Jobs            []JobTemplateSource            `yaml:"jobs"            validate:"omitempty"`
	CronJobs        []CronJobTemplateSource        `yaml:"cronJobs"        validate:"omitempty"`
	Secrets         []SecretTemplateSource         `yaml:"secrets"      validate:"omitempty"`
	ConfigMaps      []ConfigMapTemplateSource      `yaml:"configMaps"      validate:"omitempty"`
	ServiceAccounts []ServiceAccountTemplateSource `yaml:"serviceAccounts" validate:"omitempty"`
}

// ── ReconcilerConfig ──────────────────────────────────────────────────────────

type ReconcilerConfig struct {
	// Default controls which reconciler implementation is used for this CRD.
	//
	// true  — GenericReconciler manages the full lifecycle automatically.
	//         Handles: finalizer add/remove, Kubernetes events, metrics, health state.
	//         HookFactory is optional — set for custom business logic.
	//         OnCreate/OnReconcile/OnDelete templates are only valid when Default: true.
	//
	// false — Custom reconciler. The user provides the full reconcile implementation.
	//         Constructor must be declared (in YAML mode) or set directly (Go mode).
	//         GenericReconciler is not used — the user owns the entire lifecycle.
	Default *bool `yaml:"default" validate:"omitempty"`

	// Finalizers — per-CRD finalizer list. Overrides the Katalog-level finalizer.
	// Applied by GenericReconciler when a CR is first created.
	// Stripped one-by-one before delete to unblock Kubernetes garbage collection.
	// If empty, falls back to the Katalog-level finalizer declaration.
	Finalizers []string `yaml:"finalizers" validate:"omitempty"`

	// ── YAML mode declarations ────────────────────────────────────────────────
	// These fields declare where Go functions live in your codebase or in remote modules.
	// ork generate reads them and emits HookRegistry / ReconcilerRegistry entries.
	// Katalog validation reads them to wire HookFactory and Constructor at startup.

	// Hooks — declares a Go hook function for Default: true CRDs in typed or dynamic mode.
	// The function at Location.Function must match: func() domain.AnyReconcileHooks
	// Use this when you want full Go control over reconcile logic.
	// For declarative resource management without Go code, use OnCreate/OnReconcile/OnDelete.
	// Only one of Hooks or OnCreate/OnReconcile/OnDelete should be used — not both.
	Hooks *HookDeclaration `yaml:"hooks" validate:"omitempty"`

	// ConstructorDecl — declares a custom reconciler constructor for Default: false CRDs.
	// The function at Location.Function must match: NewReconcilerFunc
	// Required when Default: false in YAML mode.
	ConstructorDecl *ConstructorDeclaration `yaml:"constructor" validate:"omitempty"`

	// ── Declarative hook templates ────────────────────────────────────────────
	// Only valid when Default: true and mode: dynamic.
	// ork generate reads these declarations and emits complete hook implementations
	// in __generated_runtime_hooks.go that call OrkestraRegistry resource functions
	// with resolved field values. No Go code required from the user.
	// Registered automatically in HookRegistry at startup via generated init().

	// OnCreate — resources to create when the CR is first reconciled.
	OnCreate *HookTemplates `yaml:"onCreate" validate:"omitempty"`

	// OnReconcile — drift correction resources applied on every reconcile.
	// Omit if onCreate alone is sufficient.
	OnReconcile *HookTemplates `yaml:"onReconcile" validate:"omitempty"`

	// OnDelete — cleanup resources applied before finalizer removal.
	// Omit for resources covered by owner reference cascade deletion.
	OnDelete *HookTemplates `yaml:"onDelete" validate:"omitempty"`

	// HookFactory — called once at startCRDWorkers time to produce typed hooks.
	// nil → GenericReconciler runs with no user hooks.
	//       Finalizers, events, and metrics are still handled automatically.
	HookFactory func() domain.AnyReconcileHooks `yaml:"-"`

	// Constructor — called once at startCRDWorkers time to build a custom reconciler.
	// Must not be nil when Default: false — enforced by Katalog validation at startup.
	Constructor NewReconcilerFunc `yaml:"-"`

	// Status declares how Orkestra manages the CR's /status subresource.
	// nil (default): Layer 1 only — standard Ready condition after every reconcile.
	// non-nil: Layer 1 + Layer 2 declarative fields from Status.Fields.
	Status *StatusConfig `yaml:"status,omitempty"`
}

// HookDeclaration declares where a Go hook function lives.
// Read by ork generate to emit HookRegistry entries in zz_generated_runtime_registry.go.
// The declared function must match the signature: func() domain.AnyReconcileHooks
type HookDeclaration struct {
	// Location — fully qualified Go import path. Local or remote module.
	// e.g. "github.com/myorg/hooks" or "github.com/ialexeze/orkestra/pkg/reconciler/hooks"
	Location string `yaml:"location" validate:"required"`

	// Function — exported function name at Location that returns hooks.
	// e.g. "ProjectHooks"
	Function string `yaml:"function" validate:"required"`

	// Alias — Go import alias. Optional, auto-derived from Location if omitted.
	// e.g. "projecthooks"
	Alias string `yaml:"alias" validate:"omitempty"`
}

// ConstructorDeclaration declares where a custom reconciler constructor lives.
// Read by ork generate to emit ReconcilerRegistry entries.
// The declared function must match: NewReconcilerFunc
type ConstructorDeclaration struct {
	// Location — fully qualified Go import path. Local or remote module.
	Location string `yaml:"location" validate:"required"`

	// Function — exported constructor function name at Location.
	// e.g. "NewManagedNamespaceReconciler"
	Function string `yaml:"function" validate:"required"`

	// Alias — Go import alias. Optional, auto-derived from Location if omitted.
	Alias string `yaml:"alias" validate:"omitempty"`
}

// ── CRDEntry ──────────────────────────────────────────────────────────────────
// One entry per CRD in the Katalog.
//
// YAML fields are populated by the YAML parser when running in YAML mode,
// or set directly in BuildKatalogFromGo() when running in Go mode.
//
// Fields tagged yaml:"-" are populated at runtime during Katalog validation
// and wiring — they are never parsed from YAML and never set manually.

type CRDEntry struct {
	// ── Identity ──────────────────────────────────────────────────────────────

	// Name — unique CRD identifier within the Katalog. Must be lowercase.
	// Used for routing, health endpoints (/katalog/{name}), and log context.
	Name string `yaml:"name" validate:"required,hostname_rfc1123"`

	// Enabled — include this CRD in the runtime. false = skipped entirely.
	// WARNING: only set to false after stripping Orkestra finalizers from all
	// live CRs — disabled CRDs with live finalizers will cause stuck objects.
	Enabled *bool `yaml:"enabled"`

	// Critical — if true, Orkestra marks the entire controller as degraded when
	// this CRD's health state transitions to degraded.
	// Use for CRDs that are fundamental to the platform's correctness.
	Critical *bool `yaml:"critical"`

	// Description — human-readable description. Shown in /katalog API responses.
	Description string `yaml:"description" validate:"omitempty"`

	// Mode — see CRDMode for full documentation.
	// Auto-detected when omitted based on whether apiTypes.location is set.
	Mode CRDMode `yaml:"mode" validate:"omitempty,oneof=typed dynamic"`

	// ── API Types ─────────────────────────────────────────────────────────────
	// See APITypes for full field documentation.
	APITypes APITypes `yaml:"apiTypes" validate:"required"`

	// ── Runtime objects ───────────────────────────────────────────────────────
	// Set by addRuntimeObjects() during Katalog validation. Never set from YAML.
	//
	// Typed mode:        DynamicModeObject and ListDynamicModeObject are factory functions
	//                    from ObjectRegistry and ListRegistry respectively.
	//                    TypedModeObject and ListTypedModeObject are set in BuildKatalogFromGo().
	//
	// Dynamic mode: DynamicModeObject and ListDynamicModeObject are factory functions
	//                    that return *unstructured.Unstructured and *unstructured.UnstructuredList.
	//                    These are always set by addRuntimeObjects() — never nil after validation.
	TypedModeObject       runtime.Object        `yaml:"-"`
	ListTypedModeObject   runtime.Object        `yaml:"-"`
	DynamicModeObject     func() runtime.Object `yaml:"-"`
	ListDynamicModeObject func() runtime.Object `yaml:"-"`

	// Scheme — AddToScheme function generated by controller-gen for this API type.
	// Required for typed mode so the REST client can decode API server responses.
	// Not needed for dynamic mode — the dynamic client bypasses scheme decoding.
	// Set in BuildKatalogFromGo() for Go mode. Handled by RegisterScheme() for YAML mode.
	Scheme func(s *runtime.Scheme) error `yaml:"-"`

	// ── Computed GVK/GVR ─────────────────────────────────────────────────────
	// Set by setGroupVersionKind() during Katalog validation.
	// Derived from APITypes fields. Never set manually.
	GroupVersion         *schema.GroupVersion        `yaml:"-"`
	GroupVersionKind     schema.GroupVersionKind     `yaml:"-"`
	GroupVersionResource schema.GroupVersionResource `yaml:"-"`

	// ── Scope ─────────────────────────────────────────────────────────────────

	// Namespaced — true if this CRD is namespace-scoped, false if cluster-scoped.
	// Default is true
	Namespaced *bool `yaml:"namespaced"`

	// Namespace — target namespace for namespace-scoped CRDs.
	// Informer watches this namespace only. Empty = all namespaces.
	Namespace string `yaml:"namespace" validate:"omitempty"`

	// ── Runtime behaviour ─────────────────────────────────────────────────────

	// Workers — number of concurrent reconcile workers for this CRD.
	// Higher values increase throughput but also increase API server load.
	// 0 → uses Orkestra-level default (DEFAULT_WORKERS env var).
	Workers int `yaml:"workers" validate:"omitempty,gte=1,lte=50"`

	// WorkersActive — records number of active concurrent reconcile workers for this CRD.
	WorkersActive int `yaml:"workersActive" validate:"omitempty,gte=1,lte=50"`

	// Resync — full re-list interval for the informer cache.
	// Triggers a reconcile for every cached object at this interval.
	// 0 → uses Orkestra-level default (DEFAULT_RESYNC env var).
	Resync time.Duration `yaml:"resync" validate:"omitempty"`

	// DependsOn — names of other CRDs that must be fully started before this one.
	// Orkestra resolves the dependency graph and starts CRDs in topological order.
	// Cycle detection runs at validation time — cycles fail fast with a clear error.
	DependsOn []string `yaml:"dependsOn"`

	// ── Reconciler + Queue ────────────────────────────────────────────────────
	ReconcilerConfig ReconcilerConfig `yaml:"reconciler"`
	Queue            Queue            `yaml:"queue"`
	Labels           []ResourceLabel  `yaml:"labels" validate:"omitempty"`

	// IsBuiltIn is set to true when this CRD entry was enriched from the
	// built-in Kubernetes resource registry. Used for ork validate output
	// and informational logging only — does not affect runtime behavior.
	IsBuiltIn bool `yaml:"-"` // never serialized — runtime state only

	// BuiltInGroup is the display name of the API group for built-in resources.
	// "core" for resources in the core group (empty string group).
	// Only set when IsBuiltIn is true.
	BuiltInGroup string `yaml:"-"` // never serialized

	// EnrichmentOutcome records the result of the API type enrichment phase.
	// During validation, built‑in Kubernetes kinds (e.g., Pod, Deployment, Secret)
	// are automatically enriched with their full API metadata — group, version,
	// plural, API path, and namespaced scope. This allows users to specify only:
	//
	//	apiTypes:
	//	  kind: Pod
	//
	// and rely on Orkestra to resolve all remaining fields based on the
	// Kubernetes discovery API. Custom resources are enriched using their declared
	// group/version/kind. This field is never serialized and is used internally to
	// report enrichment status and drive downstream runtime behavior.
	EnrichmentOutcome EnrichmentOutcome `yaml:"-"` // never serialized

	// Endpoints defines which operator HTTP endpoints are enabled for this CRD.
	Endpoints EndpointsConfig `yaml:"endpoints"`

	// Restricted Namespaces
	RestrictedNamespaces RestrictedNamespaces `yaml:"restrictedNamespaces,omitempty"`

	// Conversion is useful for handling multi-version crd
	Conversion *CRDConversion `yaml:"conversion,omitempty"`

	// Validation is a list of rules
	Validation *ValidationConfig `yaml:"validation,omitempty"`

	// Mutation is a list of rules
	Mutation *MutationConfig `yaml:"mutation,omitempty"`

	// Webhooks controls per-CRD admission webhook behaviour.
	// Only meaningful when ENABLE_WEBHOOKS=true.
	// By default, any CRD with Validation or Mutation rules is included
	// in the corresponding webhook configuration automatically.
	// Set validation: false or mutation: false to opt a specific CRD out of
	// admission-time interception while keeping its reconcile-time enforcement.
	Webhooks AdmissionWebhookConfig `yaml:"webhooks,omitempty"`
}

type ConversionVersionSpec struct {
	Version string                 `json:"version"`
	Spec    map[string]interface{} `json:"spec"`
}

// EndpointsConfig controls which HTTP endpoints are exposed by the operator.
//
// This allows users to selectively enable/disable endpoints while keeping
// the configuration minimal and declarative.
type EndpointsConfig struct {
	// Enabled if false disables all endpoints for this CRD
	// Default is true
	Enabled *bool `yaml:"enabled"`

	// Health controls whether the /health endpoint is served.
	Health *bool `yaml:"health"`

	// Info controls whether the /info endpoint is served.
	Info *bool `yaml:"info"`
}
