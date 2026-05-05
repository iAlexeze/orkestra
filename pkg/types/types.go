// pkg/orktypes/types.go
package types

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/orkspace/orkestra/domain"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ── Registries ────────────────────────────────────────────────────────────────
// Package-level registries — one set per Orkestra instance.
// Populated by RegisterRuntimeObjects() in zz_generated_runtime_registry.go,
// which is produced by `ork generate registry --katalog <path>`.
// Keyed by schema.GroupVersionKind. Set during Katalog validation.
//
// User code never reads or writes these directly. Orkestra reads them during
// Katalog validation via addRuntimeObjects(), addHooks(), and addReconcilers().

var ObjectRegistry = map[schema.GroupVersionKind]func() runtime.Object{}
var ListRegistry = map[schema.GroupVersionKind]func() runtime.Object{}
var HookRegistry = map[schema.GroupVersionKind]func() domain.AnyReconcileHooks{}
var ReconcilerRegistry = map[schema.GroupVersionKind]NewReconcilerFunc{}

// SchemeAdderFns holds AddToScheme functions collected from generated init()
// calls. Each generated zz_generated_runtime_registry.go appends to this slice
// in its init(); NewSchemeRegistry drains it via RegisterTypedScheme.
// This is the bridge between the user's generated package and the Orkestra
// internal pkg/runtime stub — no explicit call needed beyond the blank import.
var SchemeAdderFns []func(*runtime.Scheme) error

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
type DependencyCondtion string

const (
	CRDModeTyped   CRDMode = "typed"
	CRDModeDynamic CRDMode = "dynamic"

	DependencyConditionStarted DependencyCondtion = "started"
	DependencyConditionHealthy DependencyCondtion = "healthy"

	// Future
	DependencyCondtionPending   DependencyCondtion = "pending"
	DependencyCondtionReady     DependencyCondtion = "ready"
	DependencyConditionDegraded DependencyCondtion = "degraded"
	DependencyConditionDeleted  DependencyCondtion = "deleted"
)

func (m CRDMode) String() string {
	return string(m)
}

// ── APITypes ──────────────────────────────────────────────────────────────────
// Mirrors the apiTypes block in crd-katalog.yaml.
// ork generate reads this block to emit ObjectRegistry + ListRegistry entries
// and the RegisterScheme() function.

type APITypes struct {
	// Object — Go type name for a single CR instance. Required for typed mode.
	// Used by ork generate to emit ObjectRegistry entries.
	// e.g. "Project" → func() runtime.Object { return &projv1.Project{} }
	Object string `yaml:"object" json:"object,omitempty" validate:"omitempty"`

	// List — Go type name for the CR list. Required for typed mode.
	// Used by ork generate to emit ListRegistry entries.
	// e.g. "ProjectList" → func() runtime.Object { return &projv1.ProjectList{} }
	List string `yaml:"objectList" json:"objectList,omitempty" validate:"omitempty"`

	// Alias — Go import alias for the API types package. Optional.
	// Auto-derived from the last two segments of Location if not set.
	// e.g. "projv1" → import projv1 "github.com/.../project/v1alpha1"
	Alias string `yaml:"alias" json:"alias,omitempty" validate:"omitempty"`

	// Group — Kubernetes API group. Required in all modes.
	// e.g. "platform.orkestra.io"
	Group string `yaml:"group" json:"group" validate:"required,hostname_rfc1123"`

	// Version — API version. Required in all modes.
	// e.g. "v1alpha1"
	Version string `yaml:"version" json:"version" validate:"required"`

	// Kind — resource Kind. Required in all modes.
	// e.g. "Platform"
	Kind string `yaml:"kind" json:"kind" validate:"required"`

	// Plural — lowercase plural resource name. Required in all modes.
	// Used for REST client URL construction.
	// e.g. "projects"
	Plural string `yaml:"plural" json:"plural" validate:"required"`

	// APIPath — REST API path prefix. Default: /apis.
	// Override to /api only for core Kubernetes types (Pod, ConfigMap, etc.)
	// Almost always leave this empty — Orkestra defaults it to /apis.
	APIPath string `yaml:"apiPath" json:"apiPath,omitempty" validate:"omitempty"`

	// Location — fully qualified Go import path for the API types package.
	// Required for typed mode. Used by ork generate for import statements
	// and scheme registration in RegisterScheme().
	// Not needed for dynamic mode — omit entirely.
	// e.g. "github.com/orkspace/orkestra/api/types/project/v1alpha1"
	Location string `yaml:"location" json:"location,omitempty" validate:"omitempty"`
}

// ── Queue ─────────────────────────────────────────────────────────────────────

type Queue struct {
	// true: — uses the shared default workqueue instead of a per-CRD queue.
	// Suitable for low-volume CRDs where queue isolation is not required.

	// Default: false — each CRD gets its own isolated workqueue.
	Default *bool `yaml:"default" json:"default,omitempty"`

	// MaxQueueDepth — maximum number of items in the per-CRD queue.
	// New items are rejected when the queue is full.
	// 0 → uses Orkestra-level default set by MAX_QUEUE_DEPTH env var.
	MaxQueueDepth int `yaml:"maxQueueDepth" json:"maxQueueDepth,omitempty" validate:"omitempty,gte=0"`

	// DegradeThreshold — number of consecutive reconcile failures before the
	// CRD health state transitions from healthy to degraded.
	// 0 → uses Orkestra-level default.
	DegradeThreshold int `yaml:"degradeThreshold" json:"degradeThreshold,omitempty" validate:"omitempty,gte=0"`
}

// ── Shared resource value types ───────────────────────────────────────────────

// ResourceLabel is a single key-value label or annotation pair.
// When used in hook template declarations, values support Go text/template
// expressions evaluated against the live CR at reconcile time.
// e.g. {key: "app", value: "{{ .metadata.name }}"}
type ResourceLabel struct {
	Key   string `yaml:"key" json:"key" validate:"required"`
	Value string `yaml:"value" json:"value" validate:"required"`
}

func (l ResourceLabel) String() string {
	return fmt.Sprintf("%s=%s", l.Key, l.Value)
}

type ResourceSelector []ResourceLabel

// Stringifier
func (s ResourceSelector) String() string {
	if len(s) == 0 {
		return ""
	}

	parts := make([]string, 0, len(s))
	for _, lbl := range s {
		parts = append(parts, lbl.String())
	}

	return strings.Join(parts, ",")
}

// Selector map
type SelectorMap map[string]string

// Stringifier
func (m SelectorMap) String() string {
	if len(m) == 0 {
		return ""
	}
	parts := make([]string, 0, len(m))
	for k, v := range m {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

// ResourceRequirements mirrors Kubernetes resource requests and limits.
// Values are static Kubernetes quantity strings — template expressions
// are not supported here.
// e.g. requests: {cpu: "100m", memory: "128Mi"}
type ResourceRequirements struct {
	Requests map[string]string `yaml:"requests" json:"requests,omitempty" validate:"omitempty"`
	Limits   map[string]string `yaml:"limits" json:"limits,omitempty" validate:"omitempty"`
}

// EnvVarSource represents a single environment variable value source.
// Only one of Value, SecretKeyRef, or ConfigMapKeyRef should be set.
// Values are static strings — template expressions are not supported.
type EnvVarSource struct {
	Value           string           `yaml:"value,omitempty" json:"value,omitempty"`
	SecretKeyRef    *SecretKeyRef    `yaml:"secretKeyRef,omitempty" json:"secretKeyRef,omitempty"`
	ConfigMapKeyRef *ConfigMapKeyRef `yaml:"configMapKeyRef,omitempty" json:"configMapKeyRef,omitempty"`
}

type EnvFromSource struct {
	ConfigMapRef string `yaml:"configMapRef,omitempty" json:"configMapRef,omitempty"`
	SecretRef    string `yaml:"secretRef,omitempty" json:"secretRef,omitempty"`
}

// SecretKeyRef selects a key from a Kubernetes Secret.
// Both Name and Key are required.
type SecretKeyRef struct {
	Name string `yaml:"name" json:"name"`
	Key  string `yaml:"key" json:"key"`
}

// ConfigMapKeyRef selects a key from a Kubernetes ConfigMap.
// Both Name and Key are required.
type ConfigMapKeyRef struct {
	Name string `yaml:"name" json:"name"`
	Key  string `yaml:"key" json:"key"`
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
//		  resourceProfile: burst		# Use orkestra's burst resource configuration
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
	Version string `yaml:"version" json:"version,omitempty" validate:"omitempty"`

	// Name — Deployment and primary container name.
	// Supports template expressions.
	// Default when omitted: "{{ .metadata.name }}-deployment"
	Name string `yaml:"name" json:"name,omitempty" validate:"omitempty"`

	// Image — container image. Required (must be declared here or resolvable from CR).
	// Static:  "nginx:1.25"
	// Dynamic: "{{ .spec.image }}"
	Image string `yaml:"image" json:"image" validate:"omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use
	// for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	ImagePullSecrets []string `yaml:"imagePullSecrets" json:"imagePullSecrets" validate:"omitempty"`

	// Replicas — number of pod replicas as a string.
	// Static:  "3"
	// Dynamic: "{{ .spec.replicas }}"
	// Default: "1"
	Replicas string `yaml:"replicas" json:"replicas,omitempty" validate:"omitempty"`

	// Port — primary container port as a string.
	// Static:  "8080"
	// Dynamic: "{{ .spec.port }}"
	// Omit to expose no port.
	Port string `yaml:"port" json:"port,omitempty" validate:"omitempty"`

	// Namespace — target namespace for the Deployment.
	// Default when omitted: "{{ .metadata.namespace }}" (same namespace as the CR).
	Namespace string `yaml:"namespace" json:"namespace,omitempty" validate:"omitempty"`

	// Labels — applied to the Deployment ObjectMeta and the pod template.
	// Label values support template expressions.
	// Orkestra always adds: managed-by=orkestra, orkestra-owner=<cr-name>
	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty" validate:"omitempty"`

	// Annotations — applied to the Deployment ObjectMeta only.
	// Annotation values support template expressions.
	Annotations []ResourceLabel `yaml:"annotations" json:"annotations,omitempty" validate:"omitempty"`

	// Resources — CPU and memory requests/limits for the primary container.
	// Values are static Kubernetes quantity strings.
	// Template expressions are not supported in resource quantities.
	Resources *ResourceRequirements `yaml:"resources" json:"resources,omitempty" validate:"omitempty"`

	// ResourceProfile — semantic CPU/memory preset for the primary container.
	// Profiles expand into a complete ResourceRequirements struct during katalog
	// enrichment. Allowed values: tiny, small, medium, large, burst, steady,
	// compute-heavy, memory-heavy.
	//
	// When resourceProfile is set, manual resources.requests/limits must not be
	// provided. Profiles are atomic and fully define resource behavior.
	ResourceProfile string `yaml:"resourceProfile" json:"resourceProfile,omitempty" validate:"omitempty"`

	// Env — environment variables for the primary container.
	// Keys are env var names. Values support template expressions.
	// Example:
	//   env:
	//     REGION: "{{ .item }}"
	//     DB_HOST: "{{ .cross.db.status.endpoint }}"
	//
	// If omitted, no environment variables are added.
	// Env map[string]string `yaml:"env" json:"env,omitempty" validate:"omitempty"`
	Env map[string]EnvVarSource `yaml:"env" json:"env,omitempty"`

	EnvFrom []EnvFromSource `yaml:"envFrom,omitempty" json:"envFrom,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `yaml:"nodeSelector,omitempty" json:"nodeSelector,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string `yaml:"serviceAccountName,omitempty" json:"serviceAccountName,omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile" json:"reconcile,omitempty" validate:"omitempty"`

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

	Conditions []Condition `yaml:"when,omitempty" json:"when,omitempty"`
	// ForEach declares dynamic expansion over a list field.
	// When set, one source declaration becomes N declarations — one per list element.
	// .item and .<as> are available in template expressions within this declaration.
	ForEach *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// AnyOf holds OR conditions — at least one must pass for this resource to be created.
	// Works alongside the existing Conditions (when:) field which uses AND semantics.
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`

	// WorkingDirectory sets the container's working directory (container.WorkingDir).
	// Useful for Git-backed pipelines where build/test commands must run inside
	// a checked-out repository path.
	WorkingDirectory string `yaml:"workingDirectory,omitempty" json:"workingDirectory,omitempty"`
}

// ── ReplicaSet ────────────────────────────────────────────────────────────────

// ReplicaSetTemplateSource declares one ReplicaSet to be managed by Orkestra.
//
// Declare under onCreate to create the ReplicaSet on first reconcile.
// Declare under onReconcile to apply drift correction on every reconcile.
// Declare under both to get idempotent creation and drift correction together.
//
// Minimal example — static values only:
//
//	onCreate:
//	  replicasets:
//	    - image: nginx:1.25
//	      replicas: "3"
//	      port: "8080"
//
// Full example — dynamic values from the CR:
//
//	onCreate:
//	  replicasets:
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
type ReplicaSetTemplateSource struct {
	// Version — OrkestraRegistry implementation version to use. Omit for latest.
	Version string `yaml:"version" json:"version,omitempty" validate:"omitempty"`

	// Name — ReplicaSet and primary container name.
	// Supports template expressions.
	// Default when omitted: "{{ .metadata.name }}-replicaset"
	Name string `yaml:"name" json:"name,omitempty" validate:"omitempty"`

	// Image — container image. Required (must be declared here or resolvable from CR).
	// Static:  "nginx:1.25"
	// Dynamic: "{{ .spec.image }}"
	Image string `yaml:"image" json:"image" validate:"omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use
	// for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	ImagePullSecrets []string `yaml:"imagePullSecrets" json:"imagePullSecrets" validate:"omitempty"`

	// Replicas — number of pod replicas as a string.
	// Static:  "3"
	// Dynamic: "{{ .spec.replicas }}"
	// Default: "1"
	Replicas string `yaml:"replicas" json:"replicas,omitempty" validate:"omitempty"`

	// Port — primary container port as a string.
	// Static:  "8080"
	// Dynamic: "{{ .spec.port }}"
	// Omit to expose no port.
	Port string `yaml:"port" json:"port,omitempty" validate:"omitempty"`

	// Namespace — target namespace for the ReplicaSet.
	// Default when omitted: "{{ .metadata.namespace }}" (same namespace as the CR).
	Namespace string `yaml:"namespace" json:"namespace,omitempty" validate:"omitempty"`

	// Labels — applied to the ReplicaSet ObjectMeta and the pod template.
	// Label values support template expressions.
	// Orkestra always adds: managed-by=orkestra, orkestra-owner=<cr-name>
	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty" validate:"omitempty"`

	// Annotations — applied to the ReplicaSet ObjectMeta only.
	// Annotation values support template expressions.
	Annotations []ResourceLabel `yaml:"annotations" json:"annotations,omitempty" validate:"omitempty"`

	// Resources — CPU and memory requests/limits for the primary container.
	// Values are static Kubernetes quantity strings.
	// Template expressions are not supported in resource quantities.
	Resources *ResourceRequirements `yaml:"resources" json:"resources,omitempty" validate:"omitempty"`

	// ResourceProfile — semantic CPU/memory preset for the primary container.
	// Profiles expand into a complete ResourceRequirements struct during katalog
	// enrichment. Allowed values: tiny, small, medium, large, burst, steady,
	// compute-heavy, memory-heavy.
	//
	// When resourceProfile is set, manual resources.requests/limits must not be
	// provided. Profiles are atomic and fully define resource behavior.
	ResourceProfile string `yaml:"resourceProfile" json:"resourceProfile,omitempty" validate:"omitempty"`

	// Env — environment variables for the primary container.
	// Keys are env var names. Values support template expressions.
	Env map[string]EnvVarSource `yaml:"env" json:"env,omitempty"`

	EnvFrom []EnvFromSource `yaml:"envFrom,omitempty" json:"envFrom,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `yaml:"nodeSelector,omitempty" json:"nodeSelector,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string `yaml:"serviceAccountName,omitempty" json:"serviceAccountName,omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile" json:"reconcile,omitempty" validate:"omitempty"`

	// Conditions (when) — AND semantics.
	Conditions []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// ForEach declares dynamic expansion over a list field.
	ForEach *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// AnyOf holds OR conditions — at least one must pass for this resource.
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`

	// WorkingDirectory sets the container's working directory (container.WorkingDir).
	WorkingDirectory string `yaml:"workingDirectory,omitempty" json:"workingDirectory,omitempty"`
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
	Version string `yaml:"version" json:"version,omitempty" validate:"omitempty"`

	// Name — Service name.
	// Default when omitted: "{{ .metadata.name }}-svc"
	Name string `yaml:"name" json:"name,omitempty" validate:"omitempty"`

	// Type — Kubernetes Service type.
	// Accepted values: ClusterIP, NodePort, LoadBalancer.
	// Default: ClusterIP.
	Type string `yaml:"type" json:"type,omitempty" validate:"omitempty"`

	// Headless — when true, the Service is created without a clusterIP (clusterIP: None).
	// Used primarily for StatefulSets to enable stable network identities and per‑pod DNS:
	//   <podname>.<service>.<namespace>.svc.cluster.local
	// Set this to true when the Service is meant to back a StatefulSet or provide
	// direct pod‑to‑pod addressing rather than load‑balanced traffic.
	Headless bool `yaml:"headless" json:"headless,omitempty" validate:"omitempty"`

	// Protocol defines network protocols supported for things like container ports.
	// "TCP", "UDP", "SCTP"
	Protocol string `yaml:"protocol" json:"protocol,omitempty" validate:"omitempty"`

	// Port — Service port as a string.
	// Static: "80" or Dynamic: "{{ .spec.servicePort }}"
	Port string `yaml:"port" json:"port" validate:"omitempty"`

	// TargetPort — container port the Service routes traffic to.
	// Static: "8080" or Dynamic: "{{ .spec.containerPort }}"
	TargetPort string `yaml:"targetPort" json:"targetPort,omitempty" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" json:"namespace,omitempty" validate:"omitempty"`

	// Labels — applied to Service metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty" validate:"omitempty"`

	// Selector filters which pods this service will route traffic to
	// Useful in forEach situations where the labels would likely be the same
	Selector SelectorMap `yaml:"selector" json:"selector,omitempty" validate:"omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile" json:"reconcile,omitempty" validate:"omitempty"`

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

	Conditions []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// ForEach declares dynamic expansion over a list field.
	// When set, one source declaration becomes N declarations — one per list element.
	// .item and .<as> are available in template expressions within this declaration.
	ForEach *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// AnyOf holds OR conditions — at least one must pass for this resource to be created.
	// Works alongside the existing Conditions (when:) field which uses AND semantics.
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
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
	Version string `yaml:"version" json:"version,omitempty" validate:"omitempty"`

	// Name — Pod name.
	// Default when omitted: "{{ .metadata.name }}-pod"
	Name string `yaml:"name" json:"name,omitempty" validate:"omitempty"`

	// Image — container image. Required.
	// Static: "busybox:1.35" or Dynamic: "{{ .spec.image }}"
	Image string `yaml:"image" json:"image" validate:"omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use
	// for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	ImagePullSecrets []string `yaml:"imagePullSecrets" json:"imagePullSecrets" validate:"omitempty"`

	// Port — container port as a string.
	// Static: "8080" or Dynamic: "{{ .spec.port }}"
	Port string `yaml:"port" json:"port,omitempty" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" json:"namespace,omitempty" validate:"omitempty"`

	// Labels — applied to Pod metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty" validate:"omitempty"`

	// Annotations — applied to Pod metadata. Values support template expressions.
	Annotations []ResourceLabel `yaml:"annotations" json:"annotations,omitempty" validate:"omitempty"`

	// Resources — static CPU and memory requests/limits.
	Resources *ResourceRequirements `yaml:"resources" json:"resources,omitempty" validate:"omitempty"`

	// ResourceProfile — semantic CPU/memory preset for the primary container.
	// Profiles expand into a complete ResourceRequirements struct during katalog
	// enrichment. Allowed values: tiny, small, medium, large, burst, steady,
	// compute-heavy, memory-heavy.
	//
	// When resourceProfile is set, manual resources.requests/limits must not be
	// provided. Profiles are atomic and fully define resource behavior.
	ResourceProfile string `yaml:"resourceProfile" json:"resourceProfile,omitempty" validate:"omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `yaml:"nodeSelector,omitempty" json:"nodeSelector,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string `yaml:"serviceAccountName,omitempty" json:"serviceAccountName,omitempty"`

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

	Conditions []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// ForEach declares dynamic expansion over a list field.
	// When set, one source declaration becomes N declarations — one per list element.
	// .item and .<as> are available in template expressions within this declaration.
	ForEach *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// AnyOf holds OR conditions — at least one must pass for this resource to be created.
	// Works alongside the existing Conditions (when:) field which uses AND semantics.
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
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
	Version string `yaml:"version" json:"version,omitempty" validate:"omitempty"`

	// Name — Job name.
	// Default when omitted: "{{ .metadata.name }}-job"
	Name string `yaml:"name" json:"name,omitempty" validate:"omitempty"`

	// Image — container image. Required.
	Image string `yaml:"image" json:"image" validate:"omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use
	// for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	ImagePullSecrets []string `yaml:"imagePullSecrets" json:"imagePullSecrets" validate:"omitempty"`

	// Command — container entrypoint command.
	// Each element is resolved independently — template expressions are supported per element.
	// e.g. ["sh", "-c", "echo cleaning up {{ .metadata.name }}"]
	Command []string `yaml:"command" json:"command,omitempty" validate:"omitempty"`

	// Args — arguments passed to the container command.
	// Each element supports template expressions independently.
	Args []string `yaml:"args" json:"args,omitempty" validate:"omitempty"`

	// BackoffLimit — number of Pod restart attempts before the Job is marked Failed.
	// Default: 3.
	BackoffLimit int `yaml:"backoffLimit" json:"backoffLimit,omitempty" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" json:"namespace,omitempty" validate:"omitempty"`

	// Labels — applied to Job metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty" validate:"omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `yaml:"nodeSelector,omitempty" json:"nodeSelector,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string `yaml:"serviceAccountName,omitempty" json:"serviceAccountName,omitempty"`

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

	Conditions []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile" json:"reconcile,omitempty" validate:"omitempty"`

	// ForEach declares dynamic expansion over a list field.
	// When set, one source declaration becomes N declarations — one per list element.
	// .item and .<as> are available in template expressions within this declaration.
	ForEach *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// AnyOf holds OR conditions — at least one must pass for this resource to be created.
	// Works alongside the existing Conditions (when:) field which uses AND semantics.
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`

	// WorkingDirectory sets the container's working directory (container.WorkingDir).
	// Useful for Git-backed pipelines where build/test commands must run inside
	// a checked-out repository path.
	WorkingDirectory string `yaml:"workingDirectory,omitempty" json:"workingDirectory,omitempty"`
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
	Version string `yaml:"version" json:"version,omitempty" validate:"omitempty"`

	// Name — CronJob name.
	// Default when omitted: "{{ .metadata.name }}-cronjob"
	Name string `yaml:"name" json:"name,omitempty" validate:"omitempty"`

	// Schedule — cron schedule expression. Required.
	// Static: "0 * * * *" (every hour)
	// Dynamic: "{{ .spec.schedule }}"
	Schedule string `yaml:"schedule" json:"schedule" validate:"required"`

	// Image — container image. Required.
	Image string `yaml:"image" json:"image" validate:"omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use
	// for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	ImagePullSecrets []string `yaml:"imagePullSecrets" json:"imagePullSecrets" validate:"omitempty"`

	// Command — container entrypoint. Each element supports template expressions.
	Command []string `yaml:"command" json:"command,omitempty" validate:"omitempty"`

	// Args — container arguments. Each element supports template expressions.
	Args []string `yaml:"args" json:"args,omitempty" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" json:"namespace,omitempty" validate:"omitempty"`

	// Labels — applied to CronJob metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty" validate:"omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `yaml:"nodeSelector,omitempty" json:"nodeSelector,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string `yaml:"serviceAccountName,omitempty" json:"serviceAccountName,omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile" json:"reconcile,omitempty" validate:"omitempty"`

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

	Conditions []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	Suspend                    string `yaml:"suspend,omitempty" json:"suspend,omitempty"`
	SuccessfulJobsHistoryLimit string `yaml:"successfulJobsHistoryLimit,omitempty" json:"successfulJobsHistoryLimit,omitempty"`
	FailedJobsHistoryLimit     string `yaml:"failedJobsHistoryLimit,omitempty" json:"failedJobsHistoryLimit,omitempty"`
	ConcurrencyPolicy          string `yaml:"concurrencyPolicy,omitempty" json:"concurrencyPolicy,omitempty"`
	StartingDeadlineSeconds    string `yaml:"startingDeadlineSeconds,omitempty" json:"startingDeadlineSeconds,omitempty"`

	// ForEach declares dynamic expansion over a list field.
	// When set, one source declaration becomes N declarations — one per list element.
	// .item and .<as> are available in template expressions within this declaration.
	ForEach *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// AnyOf holds OR conditions — at least one must pass for this resource to be created.
	// Works alongside the existing Conditions (when:) field which uses AND semantics.
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`

	// WorkingDirectory sets the container's working directory (container.WorkingDir).
	// Useful for Git-backed pipelines where build/test commands must run inside
	// a checked-out repository path.
	WorkingDirectory string `yaml:"workingDirectory,omitempty" json:"workingDirectory,omitempty"`
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
	Version string `yaml:"version" json:"version,omitempty" validate:"omitempty"`

	// Name — ConfigMap name.
	// Default when omitted: "{{ .metadata.name }}-config"
	Name string `yaml:"name" json:"name,omitempty" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" json:"namespace,omitempty" validate:"omitempty"`

	// ToNamespaces - a list of target namespaces
	// Default when omitted: "{{ .metadata.namespace }}"
	ToNamespaces []string `yaml:"toNamespaces" json:"toNamespaces,omitempty" validate:"omitempty"`

	// Data — static key-value configuration entries.
	// Values are plain strings — template expressions are not supported here.
	Data map[string]string `yaml:"data" json:"data,omitempty" validate:"omitempty"`

	// Labels — applied to ConfigMap metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty" validate:"omitempty"`
	// FromConfigMap — name of an existing ConfigMap to copy data from.
	// Orkestra reads this at reconcile time — copies stay in sync with the source.
	FromConfigMap string `yaml:"fromConfigMap" json:"fromConfigMap,omitempty" validate:"omitempty"`

	// FromNamespace — namespace where FromConfigMap lives.
	// Default: same namespace as the CR.
	FromNamespace string `yaml:"fromNamespace" json:"fromNamespace,omitempty" validate:"omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile" json:"reconcile,omitempty" validate:"omitempty"`

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

	Conditions []Condition `yaml:"when,omitempty" json:"when,omitempty"`
	// ForEach declares dynamic expansion over a list field.
	// When set, one source declaration becomes N declarations — one per list element.
	// .item and .<as> are available in template expressions within this declaration.
	ForEach *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// AnyOf holds OR conditions — at least one must pass for this resource to be created.
	// Works alongside the existing Conditions (when:) field which uses AND semantics.
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
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
	Version string `yaml:"version" json:"version,omitempty" validate:"omitempty"`

	// Name — Secret name.
	// Default when omitted: "{{ .metadata.name }}-secret"
	Name string `yaml:"name" json:"name,omitempty" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" json:"namespace,omitempty" validate:"omitempty"`

	// Type — Kubernetes Secret type.
	// Default: "Opaque"
	Type string `yaml:"type" json:"type,omitempty" validate:"omitempty"`

	// Data — static key-value entries.
	// Values are plain strings — template expressions are not supported here.
	// If you need templated or dynamic values, use a custom Go hook.
	Data map[string]string `yaml:"data" json:"data,omitempty" validate:"omitempty"`

	// Labels — applied to Secret metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty" validate:"omitempty"`

	// Annotations — applied to Secret metadata.
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`

	// FromSecret — name of an existing Secret to copy data from.
	// Orkestra reads this at reconcile time — copies stay in sync with the source.
	FromSecret string `yaml:"fromSecret" json:"fromSecret,omitempty" validate:"omitempty"`

	// FromNamespace — namespace where FromSecret lives.
	// Default: same namespace as the CR.
	FromNamespace string `yaml:"fromNamespace" json:"fromNamespace,omitempty" validate:"omitempty"`

	// ToNamespaces - a list of target namespaces
	// Default when omitted: "{{ .metadata.namespace }}"
	ToNamespaces []string `yaml:"toNamespaces" json:"toNamespaces,omitempty" validate:"omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile" json:"reconcile,omitempty" validate:"omitempty"`

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

	Conditions []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// Once controls idempotent secret generation.
	// true  — evaluate templates and create only when the Secret does not exist.
	//         Use with random notes (randomAlphanumeric, randomHex, randomBase64).
	// false — standard create/update behavior (default).
	Once bool `yaml:"once,omitempty" json:"once,omitempty"`

	// ForEach declares dynamic expansion (same as other resource types)
	ForEach *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// AnyOf holds OR conditions (same as other resource types)
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`

	// RotateAfter declares a time-based rotation threshold.
	// When set alongside once: true, the Secret is recreated when its age
	// exceeds this duration. The creation time is tracked via the annotation:
	//   orkestra.orkspace.io/generated-at: "2026-04-06T08:00:00Z"
	//
	// Supported formats: 30s, 5m, 12h, 90d, 1y
	// Days (d) and years (y) are extensions beyond Go's standard duration format.
	//
	// Example:
	//   secrets:
	//     - name: "{{ .metadata.name }}-credentials"
	//       once: true
	//       rotateAfter: 90d
	//       data:
	//         password: "{{ randomAlphanumeric 32 }}"
	RotateAfter string `yaml:"rotateAfter,omitempty" json:"rotateAfter,omitempty"`

	// TLS declares self-signed CA and server certificate generation.
	// When set, the data: block is ignored — the Secret is created as type
	// kubernetes.io/tls with fields: tls.crt, tls.key, ca.crt
	//
	// Default Secret name when name is empty: "orkestra-tls"
	// Default validFor when empty: same as rotateAfter, or "1y"
	//
	// Example:
	//   secrets:
	//     - name: "{{ .metadata.name }}-tls"
	//       once: true
	//       rotateAfter: 1y
	//       tls:
	//         commonName: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc"
	//         dnsNames:
	//           - "{{ .metadata.name }}"
	//           - "{{ .metadata.name }}.{{ .metadata.namespace }}"
	//           - "{{ .metadata.name }}.{{ .metadata.namespace }}.svc"
	//           - "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
	//         validFor: 1y
	TLS *TLSSpec `yaml:"tls,omitempty" json:"tls,omitempty"`
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
	Version string `yaml:"version" json:"version,omitempty" validate:"omitempty"`

	// Name — ServiceAccount name.
	// Default when omitted: "{{ .metadata.name }}-sa"
	Name string `yaml:"name" json:"name,omitempty" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" json:"namespace,omitempty" validate:"omitempty"`

	// Labels — applied to ServiceAccount metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty" validate:"omitempty"`

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

	Conditions []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile" json:"reconcile,omitempty" validate:"omitempty"`

	// ForEach declares dynamic expansion over a list field.
	// When set, one source declaration becomes N declarations — one per list element.
	// .item and .<as> are available in template expressions within this declaration.
	ForEach *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// AnyOf holds OR conditions — at least one must pass for this resource to be created.
	// Works alongside the existing Conditions (when:) field which uses AND semantics.
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
}

// ── Namespace ────────────────────────────────────────────────────────────

// NamespaceTemplateSource declares one Namespace to be managed by Orkestra.
//
// Example:
//
//	onCreate:
//	  namespaces:
//	    - name: "{{ .metadata.name }}-sa"
//	      labels:
//	        - key: app
//	          value: "{{ .metadata.name }}"
type NamespaceTemplateSource struct {
	// Version — OrkestraRegistry implementation version. Omit for latest.
	Version string `yaml:"version" json:"version,omitempty" validate:"omitempty"`

	// Name — ServiceAccount name.
	// Default when omitted: "{{ .metadata.name }}-sa"
	Name string `yaml:"name" json:"name,omitempty" validate:"omitempty"`

	Finalizers []string `yaml:"finalizers" json:"finalizers,omitempty" validate:"omitempty"`

	// Labels — applied to ServiceAccount metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty" validate:"omitempty"`

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

	Conditions []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile" json:"reconcile,omitempty" validate:"omitempty"`

	// ForEach declares dynamic expansion over a list field.
	// When set, one source declaration becomes N declarations — one per list element.
	// .item and .<as> are available in template expressions within this declaration.
	ForEach *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// AnyOf holds OR conditions — at least one must pass for this resource to be created.
	// Works alongside the existing Conditions (when:) field which uses AND semantics.
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
}

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
	Version string `yaml:"version" json:"version,omitempty"`

	// Name — Ingress resource name. Default: "{{ .metadata.name }}-ingress"
	Name string `yaml:"name" json:"name,omitempty"`

	// Namespace — target namespace. Default: CR namespace.
	Namespace string `yaml:"namespace" json:"namespace,omitempty"`

	// Host — virtual host name for the Ingress rule.
	Host string `yaml:"host" json:"host,omitempty"`

	// ServiceName — backend Service name this Ingress routes to.
	ServiceName string `yaml:"serviceName" json:"serviceName,omitempty"`

	// ServicePort — backend Service port as a string. Supports template expressions.
	ServicePort string `yaml:"servicePort" json:"servicePort,omitempty"`

	// Path — HTTP path prefix. Default: "/"
	Path string `yaml:"path" json:"path,omitempty"`

	// PathType — Kubernetes IngressPathType: Prefix, Exact, ImplementationSpecific.
	// Default: Prefix.
	PathType string `yaml:"pathType" json:"pathType,omitempty"`

	// IngressClass — Ingress class name (nginx, traefik, etc.). Optional.
	IngressClass string `yaml:"ingressClass" json:"ingressClass,omitempty"`

	// Labels applied to Ingress metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty"`

	// Annotations applied to Ingress metadata. Values support template expressions.
	Annotations []ResourceLabel `yaml:"annotations" json:"annotations,omitempty"`

	// TLS — optional TLS configuration. When tls.enabled is true, Orkestra
	// generates a self-signed TLS Secret before creating the Ingress.
	TLS *IngressTLSSpec `yaml:"tls" json:"tls,omitempty"`

	Reconcile  bool         `yaml:"reconcile" json:"reconcile,omitempty"`
	Conditions []Condition  `yaml:"when,omitempty" json:"when,omitempty"`
	AnyOf      []Condition  `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	ForEach    *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`
}

// IngressTLSSpec configures TLS for an Ingress resource.
// When Enabled is true, Orkestra generates a kubernetes.io/tls Secret before
// the Ingress is applied so the Ingress can reference it immediately.
type IngressTLSSpec struct {
	// Enabled — when true, create a TLS secret and populate ingress.spec.tls.
	Enabled bool `yaml:"enabled" json:"enabled,omitempty"`

	// SecretName — name of the TLS secret. Supports template expressions.
	// Default: "{{ .metadata.name }}-tls"
	SecretName string `yaml:"secretName" json:"secretName,omitempty"`

	// Hosts — list of hostnames to include in the TLS certificate SANs.
	// Each element supports template expressions.
	Hosts []string `yaml:"hosts" json:"hosts,omitempty"`

	// ValidFor — certificate validity duration (e.g. "1y", "90d"). Default: "1y".
	ValidFor string `yaml:"validFor" json:"validFor,omitempty"`
}

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
	Version string `yaml:"version" json:"version,omitempty"`

	// Name — HPA resource name. Default: "{{ .metadata.name }}-hpa"
	Name string `yaml:"name" json:"name,omitempty"`

	// Namespace — target namespace. Default: CR namespace.
	Namespace string `yaml:"namespace" json:"namespace,omitempty"`

	// ScaleTargetRef — the target workload this HPA scales.
	// Supports Deployment, ReplicaSet, StatefulSet, or any scalable resource.
	// Supports template expressions: "{{ .metadata.name }}"
	ScaleTargetRef ScaleTargetRef `yaml:"scaleTargetRef" json:"scaleTargetRef,omitempty"`

	// MinReplicas — minimum replica count as a string. Supports template expressions.
	MinReplicas string `yaml:"minReplicas" json:"minReplicas,omitempty"`

	// MaxReplicas — maximum replica count as a string. Supports template expressions.
	MaxReplicas string `yaml:"maxReplicas" json:"maxReplicas,omitempty"`

	// TargetCPUUtilizationPercentage — CPU utilization target (0-100). Supports templates.
	TargetCPUUtilizationPercentage string `yaml:"targetCPUUtilizationPercentage" json:"targetCPUUtilizationPercentage,omitempty"`

	// Labels applied to HPA metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty"`

	Reconcile  bool         `yaml:"reconcile" json:"reconcile,omitempty"`
	Conditions []Condition  `yaml:"when,omitempty" json:"when,omitempty"`
	AnyOf      []Condition  `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	ForEach    *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`
}

// ── PodDisruptionBudget ───────────────────────────────────────────────────────

// PDBTemplateSource declares one PodDisruptionBudget to be managed by Orkestra.
//
// Example:
//
//	onReconcile:
//	  pdb:
//	    - name: "{{ .metadata.name }}-pdb"
//	      minAvailable: "1"
//	      selector:
//	        app: "{{ .metadata.name }}"
//	      forEach:
//	        field: spec.services
//	        as: item
type PDBTemplateSource struct {
	Version string `yaml:"version" json:"version,omitempty"`

	// Name — PDB resource name. Default: "{{ .metadata.name }}-pdb"
	Name string `yaml:"name" json:"name,omitempty"`

	// Namespace — target namespace. Default: CR namespace.
	Namespace string `yaml:"namespace" json:"namespace,omitempty"`

	// Selector — label selector identifying the pods this PDB protects.
	// Keys are static; values support template expressions.
	Selector SelectorMap `yaml:"selector" json:"selector,omitempty"`

	// MinAvailable — minimum number of pods that must remain available.
	// Accepts integer strings ("1") or percentage strings ("50%").
	// Mutually exclusive with MaxUnavailable.
	MinAvailable string `yaml:"minAvailable" json:"minAvailable,omitempty"`

	// MaxUnavailable — maximum number of pods that may be unavailable.
	// Accepts integer strings ("1") or percentage strings ("25%").
	// Mutually exclusive with MinAvailable.
	MaxUnavailable string `yaml:"maxUnavailable" json:"maxUnavailable,omitempty"`

	// Labels applied to PDB metadata. Values support template expressions.
	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty"`

	Reconcile  bool         `yaml:"reconcile" json:"reconcile,omitempty"`
	Conditions []Condition  `yaml:"when,omitempty" json:"when,omitempty"`
	AnyOf      []Condition  `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	ForEach    *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`
}

// StatefulSetTemplateSource declares one StatefulSet to be managed by Orkestra.
type StatefulSetTemplateSource struct {
	Version string `yaml:"version" json:"version,omitempty"`

	// Name — StatefulSet name. Default: "{{ .metadata.name }}".
	Name string `yaml:"name" json:"name,omitempty"`

	// Namespace — target namespace. Default: CR namespace.
	Namespace string `yaml:"namespace" json:"namespace,omitempty"`

	// Image — container image. Required.
	Image string `yaml:"image" json:"image" validate:"omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use
	// for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	ImagePullSecrets []string `yaml:"imagePullSecrets" json:"imagePullSecrets" validate:"omitempty"`

	// Tag — image tag. Default: "latest".
	Tag string `yaml:"tag" json:"tag,omitempty"`

	// Replicas — number of pod replicas. Default: "1".
	Replicas string `yaml:"replicas" json:"replicas,omitempty"`

	// Port — container port. "0" or empty means no port exposed.
	Port string `yaml:"port" json:"port,omitempty"`

	// ServiceName — name of the headless Service governing the StatefulSet.
	// Default: same as Name.
	ServiceName string `yaml:"serviceName" json:"serviceName,omitempty"`

	// StorageClass — storage class for auto-generated VolumeClaimTemplates.
	StorageClass string `yaml:"storageClass" json:"storageClass,omitempty"`

	// StorageSize — size of each volume claim (e.g. "10Gi"). Required when StorageClass is set.
	StorageSize string `yaml:"storageSize" json:"storageSize,omitempty"`

	// MountPath — mount path for the storage volume inside the container. Default: "/data".
	MountPath string `yaml:"mountPath" json:"mountPath,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `yaml:"nodeSelector,omitempty" json:"nodeSelector,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string `yaml:"serviceAccountName,omitempty" json:"serviceAccountName,omitempty"`

	Labels      []ResourceLabel         `yaml:"labels" json:"labels,omitempty"`
	Annotations []ResourceLabel         `yaml:"annotations" json:"annotations,omitempty"`
	Env         map[string]EnvVarSource `yaml:"env" json:"env,omitempty"`
	EnvFrom     []EnvFromSource         `yaml:"envFrom,omitempty" json:"envFrom,omitempty"`

	// Resources — CPU and memory requests/limits for the primary container.
	// Values are static Kubernetes quantity strings.
	// Template expressions are not supported in resource quantities.
	Resources *ResourceRequirements `yaml:"resources" json:"resources,omitempty" validate:"omitempty"`

	// ResourceProfile — semantic CPU/memory preset for the primary container.
	// Profiles expand into a complete ResourceRequirements struct during katalog
	// enrichment. Allowed values: tiny, small, medium, large, burst, steady,
	// compute-heavy, memory-heavy.
	//
	// When resourceProfile is set, manual resources.requests/limits must not be
	// provided. Profiles are atomic and fully define resource behavior.
	ResourceProfile string `yaml:"resourceProfile" json:"resourceProfile,omitempty" validate:"omitempty"`

	Reconcile  bool         `yaml:"reconcile" json:"reconcile,omitempty"`
	Conditions []Condition  `yaml:"when,omitempty" json:"when,omitempty"`
	AnyOf      []Condition  `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	ForEach    *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`
}

// PVCTemplateSource declares one PersistentVolumeClaim to be managed by Orkestra.
type PVCTemplateSource struct {
	Version string `yaml:"version" json:"version,omitempty"`

	// Name — PVC name. Required.
	Name string `yaml:"name" json:"name,omitempty"`

	// Namespace — target namespace. Default: CR namespace.
	Namespace string `yaml:"namespace" json:"namespace,omitempty"`

	// StorageClassName — storage class to use. Empty means cluster default.
	StorageClassName string `yaml:"storageClassName" json:"storageClassName,omitempty"`

	// AccessModes — access modes. Default: ["ReadWriteOnce"].
	// Supports: ReadWriteOnce, ReadOnlyMany, ReadWriteMany, ReadWriteOncePod.
	AccessModes []string `yaml:"accessModes" json:"accessModes,omitempty"`

	// Storage — requested storage size (e.g. "10Gi"). Required.
	Storage string `yaml:"storage" json:"storage,omitempty" validate:"required"`

	// VolumeMode — Filesystem or Block. Default: Filesystem.
	VolumeMode string `yaml:"volumeMode" json:"volumeMode,omitempty"`

	// VolumeName — bind to a specific PV by name.
	VolumeName string `yaml:"volumeName" json:"volumeName,omitempty"`

	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty"`

	Reconcile  bool         `yaml:"reconcile" json:"reconcile,omitempty"`
	Conditions []Condition  `yaml:"when,omitempty" json:"when,omitempty"`
	AnyOf      []Condition  `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	ForEach    *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`
}

// PVTemplateSource declares one PersistentVolume to be managed by Orkestra.
// PersistentVolumes are cluster-scoped — Namespace is ignored.
type PVTemplateSource struct {
	Version string `yaml:"version" json:"version,omitempty"`

	// Name — PV name. Required.
	Name string `yaml:"name" json:"name,omitempty"`

	// StorageClassName — storage class name.
	StorageClassName string `yaml:"storageClassName" json:"storageClassName,omitempty"`

	// Capacity — storage capacity (e.g. "10Gi"). Required.
	Capacity string `yaml:"capacity" json:"capacity,omitempty" validate:"required"`

	// AccessModes — access modes. Default: ["ReadWriteOnce"].
	AccessModes []string `yaml:"accessModes" json:"accessModes,omitempty"`

	// ReclaimPolicy — Retain, Delete, or Recycle. Default: Retain.
	ReclaimPolicy string `yaml:"reclaimPolicy" json:"reclaimPolicy,omitempty"`

	// HostPath — host path for HostPath volume type. Used for local/dev PVs.
	HostPath string `yaml:"hostPath" json:"hostPath,omitempty"`

	// CSI driver fields for cloud/CSI volumes.
	CSIDriver       string `yaml:"csiDriver" json:"csiDriver,omitempty"`
	CSIVolumeHandle string `yaml:"csiVolumeHandle" json:"csiVolumeHandle,omitempty"`

	Labels []ResourceLabel `yaml:"labels" json:"labels,omitempty"`

	Reconcile  bool         `yaml:"reconcile" json:"reconcile,omitempty"`
	Conditions []Condition  `yaml:"when,omitempty" json:"when,omitempty"`
	AnyOf      []Condition  `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	ForEach    *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`
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
	Deployments              []DeploymentTemplateSource     `yaml:"deployments" json:"deployments,omitempty" validate:"omitempty"`
	ReplicaSets              []ReplicaSetTemplateSource     `yaml:"replicaSets" json:"replicaSets,omitempty" validate:"omitempty"`
	Services                 []ServiceTemplateSource        `yaml:"services" json:"services,omitempty" validate:"omitempty"`
	Pods                     []PodTemplateSource            `yaml:"pods" json:"pods,omitempty" validate:"omitempty"`
	Jobs                     []JobTemplateSource            `yaml:"jobs" json:"jobs,omitempty" validate:"omitempty"`
	CronJobs                 []CronJobTemplateSource        `yaml:"cronJobs" json:"cronJobs,omitempty" validate:"omitempty"`
	Secrets                  []SecretTemplateSource         `yaml:"secrets" json:"secrets,omitempty" validate:"omitempty"`
	ConfigMaps               []ConfigMapTemplateSource      `yaml:"configMaps" json:"configMaps,omitempty" validate:"omitempty"`
	ServiceAccounts          []ServiceAccountTemplateSource `yaml:"serviceAccounts" json:"serviceAccounts,omitempty" validate:"omitempty"`
	StatefulSets             []StatefulSetTemplateSource    `yaml:"statefulSets" json:"statefulSets,omitempty" validate:"omitempty"`
	Ingresses                []IngressTemplateSource        `yaml:"ingresses" json:"ingresses,omitempty" validate:"omitempty"`
	PersistentVolumes        []PVTemplateSource             `yaml:"persistentVolumes" json:"persistentVolumes,omitempty" validate:"omitempty"`
	PersistentVolumeClaims   []PVCTemplateSource            `yaml:"persistentVolumeClaims" json:"persistentVolumeClaims,omitempty" validate:"omitempty"`
	HorizontalPodAutoscalers []HPATemplateSource            `yaml:"hpa" json:"hpa,omitempty" validate:"omitempty"`
	PodDisruptionBudgets     []PDBTemplateSource            `yaml:"pdb" json:"pdb,omitempty" validate:"omitempty"`
	Namespaces               []NamespaceTemplateSource      `yaml:"namespaces" json:"namespaces,omitempty" validate:"omitempty"`

	// External declares HTTP calls to make before resource creation.
	// Results available as .external.<n>.status, .body, .error
	External []ExternalCallSpec `yaml:"external,omitempty" json:"external,omitempty"`

	// Git declares optional Git-backed reconcile behaviour for this CRD.
	//
	// When configured, Orkestra:
	//   - Maintains a local working copy of the repository.
	//   - Periodically checks the target branch for new commits.
	//   - Enqueues reconciles for all CRs of this type when the branch tip changes.
	//
	// This enables declarative, in-cluster CI/CD pipelines where Git acts
	// as the source of pipeline logic and the CRs provide parameters.
	//
	// When omitted, reconcile behaviour is unchanged and no Git traffic
	// is generated for this CRD.
	Git *GitHookSpec `yaml:"git,omitempty" json:"git,omitempty"`

	// Docker declares optional Docker-backed reconcile behaviour for this CRD.
	//
	// When configured
	//	- Builds and optionally pushes a docker image
	Docker *DockerHookSpec `yaml:"docker,omitempty" json:"docker,omitempty"`

	// Ordered controls whether deletion happens sequentially with verification.
	// true  — delete groups in order, verify each is gone before proceeding
	// false — delete all resources via owner references (default, parallel)
	Ordered bool `yaml:"ordered,omitempty" json:"ordered,omitempty"`

	// Groups declares sequential deletion stages for ordered deletes.
	// Each element is a full HookTemplates block whose resources are deleted
	// as a unit. Orkestra deletes stage N, waits until all resources are gone,
	// then deletes stage N+1. Omit when Ordered is false.
	// When Ordered is true and Groups is empty, the flat resource fields above
	// (Jobs, Deployments, …) are treated as a single implicit group.
	Groups []HookTemplates `yaml:"groups,omitempty" json:"groups,omitempty"`

	// Timeout is the maximum time to wait for each deletion group to complete.
	// Defaults to 5m when Ordered is true. Ignored when Ordered is false.
	Timeout *Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// TODO with placeholer
	Volumes                     []PlaceholderSource         `yaml:"volumes" json:"volumes,omitempty" validate:"omitempty"`
	VolumeMounts                []PlaceholderSource         `yaml:"volumeMounts" json:"volumeMounts,omitempty" validate:"omitempty"`
	Roles                       []RoleTemplateSource        `yaml:"roles" json:"roles,omitempty" validate:"omitempty"`
	RoleBindings                []RoleBindingTemplateSource `yaml:"roleBindings" json:"roleBindings,omitempty" validate:"omitempty"`
	ClusterRoles                []PlaceholderSource         `yaml:"clusterRoles" json:"clusterRoles,omitempty" validate:"omitempty"`
	ClusterRoleBindings         []PlaceholderSource         `yaml:"clusterRoleBindings" json:"clusterRoleBindings,omitempty" validate:"omitempty"`
	ServiceMonitors             []PlaceholderSource         `yaml:"serviceMonitors" json:"serviceMonitors,omitempty" validate:"omitempty"`
	PodSecurityPolicies         []PlaceholderSource         `yaml:"podSecurityPolicies" json:"podSecurityPolicies,omitempty" validate:"omitempty"`
	PriorityClasses             []PlaceholderSource         `yaml:"priorityClasses" json:"priorityClasses,omitempty" validate:"omitempty"`
	LimitRanges                 []PlaceholderSource         `yaml:"limitRanges" json:"limitRanges,omitempty" validate:"omitempty"`
	ResourceQuotas              []PlaceholderSource         `yaml:"resourceQuotas" json:"resourceQuotas,omitempty" validate:"omitempty"`
	RuntimeClasses              []PlaceholderSource         `yaml:"runtimeClasses" json:"runtimeClasses,omitempty" validate:"omitempty"`
	PriorityLevelConfigurations []PlaceholderSource         `yaml:"priorityLevelConfigurations" json:"priorityLevelConfigurations,omitempty" validate:"omitempty"`
	PodTemplates                []PlaceholderSource         `yaml:"podTemplates" json:"podTemplates,omitempty" validate:"omitempty"`
	DaemonSets                  []PlaceholderSource         `yaml:"daemonSets" json:"daemonSets,omitempty" validate:"omitempty"`
	NetworkPolicies             []PlaceholderSource         `yaml:"networkPolicies" json:"networkPolicies,omitempty" validate:"omitempty"`

	// Storage
	StorageClasses   []PlaceholderSource `yaml:"storageClasses" json:"storageClasses,omitempty" validate:"omitempty"`
	StorageLocations []PlaceholderSource `yaml:"storageLocations" json:"storageLocations,omitempty" validate:"omitempty"`
	StoragePools     []PlaceholderSource `yaml:"storagePools" json:"storagePools,omitempty" validate:"omitempty"`
	StorageBackups   []PlaceholderSource `yaml:"storageBackups" json:"storageBackups,omitempty" validate:"omitempty"`
	StorageSnapshots []PlaceholderSource `yaml:"storageSnapshots" json:"storageSnapshots,omitempty" validate:"omitempty"`
	StorageVolumes   []PlaceholderSource `yaml:"storageVolumes" json:"storageVolumes,omitempty" validate:"omitempty"`
}

// ── Role / RoleBinding ────────────────────────────────────────────────────────

// PolicyRuleSpec declares one RBAC policy rule.
// String values within slices support template expressions.
type PolicyRuleSpec struct {
	APIGroups     []string `yaml:"apiGroups" json:"apiGroups,omitempty"`
	Resources     []string `yaml:"resources" json:"resources,omitempty"`
	Verbs         []string `yaml:"verbs" json:"verbs,omitempty"`
	ResourceNames []string `yaml:"resourceNames" json:"resourceNames,omitempty"`
}

// SubjectSpec declares one RBAC subject for a RoleBinding.
// Name and Namespace support template expressions.
type SubjectSpec struct {
	Kind      string `yaml:"kind" json:"kind,omitempty"`
	Name      string `yaml:"name" json:"name,omitempty"`
	Namespace string `yaml:"namespace" json:"namespace,omitempty"`
}

// RoleRefSpec names the Role (or ClusterRole) being bound.
// Name supports template expressions. Kind defaults to "Role".
type RoleRefSpec struct {
	Name string `yaml:"name" json:"name,omitempty"`
	Kind string `yaml:"kind" json:"kind,omitempty"` // Role | ClusterRole; defaults to Role
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
	Version    string           `yaml:"version" json:"version,omitempty"`
	Name       string           `yaml:"name" json:"name,omitempty"`
	Namespace  string           `yaml:"namespace" json:"namespace,omitempty"`
	Labels     []ResourceLabel  `yaml:"labels" json:"labels,omitempty"`
	Rules      []PolicyRuleSpec `yaml:"rules" json:"rules,omitempty"`
	Conditions []Condition      `yaml:"when,omitempty" json:"when,omitempty"`
	Reconcile  bool             `yaml:"reconcile" json:"reconcile,omitempty"`
	ForEach    *ForEachSpec     `yaml:"forEach,omitempty" json:"forEach,omitempty"`
	AnyOf      []Condition      `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
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
	Version    string          `yaml:"version" json:"version,omitempty"`
	Name       string          `yaml:"name" json:"name,omitempty"`
	Namespace  string          `yaml:"namespace" json:"namespace,omitempty"`
	Labels     []ResourceLabel `yaml:"labels" json:"labels,omitempty"`
	RoleRef    RoleRefSpec     `yaml:"roleRef" json:"roleRef,omitempty"`
	Subjects   []SubjectSpec   `yaml:"subjects" json:"subjects,omitempty"`
	Conditions []Condition     `yaml:"when,omitempty" json:"when,omitempty"`
	Reconcile  bool            `yaml:"reconcile" json:"reconcile,omitempty"`
	ForEach    *ForEachSpec    `yaml:"forEach,omitempty" json:"forEach,omitempty"`
	AnyOf      []Condition     `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
}

// Placeholder for resources yet to be added to orkestra internal registry
// pkg/orkestra-registry
type PlaceholderSource struct{}

// ── OperatorBoxConfig ──────────────────────────────────────────────────────────

type OperatorBoxConfig struct {
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
	Default *bool `yaml:"default" json:"default,omitempty" validate:"omitempty"`

	// Finalizers — per-CRD finalizer list. Overrides the Katalog-level finalizer.
	// Applied by GenericReconciler when a CR is first created.
	// Stripped one-by-one before delete to unblock Kubernetes garbage collection.
	// If empty, falls back to the Katalog-level finalizer declaration.
	Finalizers []string `yaml:"finalizers" json:"finalizers,omitempty" validate:"omitempty"`

	// ── YAML mode declarations ────────────────────────────────────────────────
	// These fields declare where Go functions live in your codebase or in remote modules.
	// ork generate reads them and emits HookRegistry / ReconcilerRegistry entries.
	// Katalog validation reads them to wire HookFactory and Constructor at startup.

	// Hooks — declares a Go hook function for Default: true CRDs in typed or dynamic mode.
	// The function at Location.Function must match: func() domain.AnyReconcileHooks
	// Use this when you want full Go control over reconcile logic.
	// For declarative resource management without Go code, use OnCreate/OnReconcile/OnDelete.
	// Only one of Hooks or OnCreate/OnReconcile/OnDelete should be used — not both.
	Hooks *HookDeclaration `yaml:"hooks" json:"hooks,omitempty" validate:"omitempty"`

	// ConstructorDecl — declares a custom reconciler constructor for Default: false CRDs.
	// The function at Location.Function must match: NewReconcilerFunc
	// Required when Default: false in YAML mode.
	ConstructorDecl *ConstructorDeclaration `yaml:"constructor" json:"constructor,omitempty" validate:"omitempty"`

	// ── Declarative hook templates ────────────────────────────────────────────
	// Only valid when Default: true and mode: dynamic.
	// ork generate reads these declarations and emits complete hook implementations
	// in __generated_runtime_hooks.go that call OrkestraRegistry resource functions
	// with resolved field values. No Go code required from the user.
	// Registered automatically in HookRegistry at startup via generated init().

	// OnCreate — resources to create when the CR is first reconciled.
	OnCreate *HookTemplates `yaml:"onCreate" json:"onCreate,omitempty" validate:"omitempty"`

	// OnReconcile — drift correction resources applied on every reconcile.
	// Omit if onCreate alone is sufficient.
	OnReconcile *HookTemplates `yaml:"onReconcile" json:"onReconcile,omitempty" validate:"omitempty"`

	// OnDelete — cleanup resources applied before finalizer removal.
	// Omit for resources covered by owner reference cascade deletion.
	OnDelete *HookTemplates `yaml:"onDelete" json:"onDelete,omitempty" validate:"omitempty"`

	// HookFactory — called once at startCRDWorkers time to produce typed hooks.
	// nil → GenericReconciler runs with no user hooks.
	//       Finalizers, events, and metrics are still handled automatically.
	HookFactory func() domain.AnyReconcileHooks `yaml:"-" json:"-"`

	// Constructor — called once at startCRDWorkers time to build a custom reconciler.
	// Must not be nil when Default: false — enforced by Katalog validation at startup.
	Constructor NewReconcilerFunc `yaml:"-" json:"-"`

	// Status declares how Orkestra manages the CR's /status subresource.
	// nil (default): Layer 1 only — standard Ready condition after every reconcile.
	// non-nil: Layer 1 + Layer 2 declarative fields from Status.Fields.
	Status *StatusConfig `yaml:"status,omitempty" json:"status,omitempty"`

	// ProviderBlocks holds the parsed provider declarations from the Katalog.
	// Populated during Katalog loading via ParseProviderBlocks.
	// Not a YAML field — parsed from RawProviders after unmarshal.
	ProviderBlocks []ProviderBlock `yaml:"-" json:"-"`

	// RawProviders is the raw YAML map, populated during unmarshal.
	// Converted to ProviderBlocks in the Katalog loading step.
	RawProviders map[string][]map[string]interface{} `yaml:"providers,omitempty" json:"providers,omitempty"`

	// Imports declares Motif imports for this operatorBox.
	// Each import references a Motif by OCI reference, file path, or short name,
	// and binds its inputs via with:. Resources from imported Motifs are merged
	// into OnReconcile at Katalog load time.
	// Required inputs not provided in with: are a validation error.
	Imports []MotifImport `yaml:"imports,omitempty" json:"imports,omitempty"`

	// Cross declares cross-CRD observations.
	// Read before any resource groups — results available as .cross.<as>.status.*
	Cross []CrossCRDDeclaration `yaml:"cross,omitempty" json:"cross,omitempty"`

	// Autoscale declares runtime autoscale behavior for this operatorbox.
	// When declared, the autoscaler evaluates conditions on a ticker and applies
	// or restores worker/queue/resync overrides automatically.
	// nil → no autoscaling; CRD runs with its declared static worker count.
	Autoscale *AutoscaleSpec `yaml:"autoscale,omitempty" json:"autoscale,omitempty"`

	// Rollback declares failure-recovery behavior for this operatorbox.
	// When declared, Orkestra tracks consecutive reconcile failures and re-applies
	// the last known good spec when the trigger threshold is crossed.
	// nil → no rollback; failures are retried indefinitely.
	Rollback *RollbackBlock `yaml:"rollback,omitempty" json:"rollback,omitempty"`

	// RollBackOnError is a zero-config rollback shorthand.
	//
	// When true, Orkestra automatically rolls back to the previous spec whenever
	// the default trigger threshold is reached (3 consecutive failures), without
	// requiring a separate rollback: block.
	//
	// The rollback templates are derived from all resource declarations that have
	// reconcile: true in onCreate: and onReconcile:. Those same templates are
	// re-applied using the previous spec as the base context — .spec.* resolves
	// to the previous spec values, so no .previous.spec.* references are needed.
	//
	// Resources with once: true are excluded — they are never regenerated by rollback.
	//
	// Combine with an explicit rollback.trigger to adjust the threshold without
	// redeclaring the rollback templates:
	//
	//	operatorBox:
	//	  rollBackOnError: true
	//	  rollback:
	//	    trigger:
	//	      consecutiveFailures: 5
	//	      withinDuration: 10m
	//
	// Declare rollback.onRollback alongside rollBackOnError: true to override the
	// derived templates for specific resources while keeping the shorthand trigger.
	RollBackOnError bool `yaml:"rollBackOnError,omitempty" json:"rollBackOnError,omitempty"`

	// When is an optional list of conditions that must all pass before
	// this field is written. If absent or empty, the field is always written.
	//
	// All conditions are AND-ed together.
	// To express OR logic, declare multiple StatusField entries for the same path.
	//
	// Conditions are evaluated against the full CR object map — the same
	// map available to template expressions. This means .status.phase,
	// .spec.image, .children.job.status.succeeded are all accessible.
	When []Condition `yaml:"when,omitempty"`

	AnyOf []Condition `yaml:"anyOf,omitempty"`
}

// HookDeclaration declares where a Go hook function lives.
// Read by ork generate to emit HookRegistry entries in zz_generated_runtime_registry.go.
// The declared function must match the signature: func() domain.AnyReconcileHooks
type HookDeclaration struct {
	// Location — fully qualified Go import path. Local or remote module.
	// e.g. "github.com/myorg/hooks" or "github.com/orkspace/orkestra/pkg/reconciler/hooks"
	Location string `yaml:"location" json:"location" validate:"required"`

	// Function — exported function name at Location that returns hooks.
	// e.g. "ProjectHooks"
	Function string `yaml:"function" json:"function" validate:"required"`

	// Alias — Go import alias. Optional, auto-derived from Location if omitted.
	// e.g. "projecthooks"
	Alias string `yaml:"alias" json:"alias,omitempty" validate:"omitempty"`
}

// ConstructorDeclaration declares where a custom reconciler constructor lives.
// Read by ork generate to emit ReconcilerRegistry entries.
// The declared function must match: NewReconcilerFunc
type ConstructorDeclaration struct {
	// Location — fully qualified Go import path. Local or remote module.
	Location string `yaml:"location" json:"location" validate:"required"`

	// Function — exported constructor function name at Location.
	// e.g. "NewManagedNamespaceReconciler"
	Function string `yaml:"function" json:"function" validate:"required"`

	// Alias — Go import alias. Optional, auto-derived from Location if omitted.
	Alias string `yaml:"alias" json:"alias,omitempty" validate:"omitempty"`
}

// ── DependsOn types ───────────────────────────────────────────────────────────

// DependsOnCondition is the value in the dependsOn map.
// Condition values: "started" (workers running) or "healthy" (running + consecutive failures = 0).
type DependsOnCondition struct {
	Condition string `yaml:"condition" json:"condition"`
}

// UnmarshalYAML handles Format 2 (scalar) and Format 3 (map) for a single dependency value.
//
//	database: healthy          ← Format 2: scalar
//	database:                  ← Format 3: map
//	  condition: healthy
func (d *DependsOnCondition) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.MappingNode {
		type plain DependsOnCondition
		return value.Decode((*plain)(d))
	}
	if value.Kind == yaml.ScalarNode {
		d.Condition = value.Value
		return nil
	}
	return fmt.Errorf("dependsOn value must be a string or map, got kind %v", value.Kind)
}

// DependsOnMap is the internal representation of all dependsOn formats.
// All three YAML formats unmarshal into this type.
type DependsOnMap map[string]DependsOnCondition

// UnmarshalYAML handles all three dependsOn formats:
//
//	Format 1 — list (condition defaults to "started"):
//	  dependsOn:
//	    - database
//
//	Format 2 — key-value map (condition explicit):
//	  dependsOn:
//	    database: healthy
//
//	Format 3 — full map:
//	  dependsOn:
//	    database:
//	      condition: healthy
func (m *DependsOnMap) UnmarshalYAML(value *yaml.Node) error {
	*m = make(DependsOnMap)

	// Format 1: sequence (list of names) → condition = "started"
	if value.Kind == yaml.SequenceNode {
		for _, item := range value.Content {
			if item.Kind == yaml.ScalarNode {
				(*m)[item.Value] = DependsOnCondition{Condition: "started"}
			}
		}
		return nil
	}

	// Format 2 + 3: mapping node
	if value.Kind == yaml.MappingNode {
		for i := 0; i < len(value.Content)-1; i += 2 {
			key := value.Content[i].Value
			val := value.Content[i+1]

			var cond DependsOnCondition
			if err := val.Decode(&cond); err != nil {
				return fmt.Errorf("dependsOn[%s]: %w", key, err)
			}
			if cond.Condition == "" {
				cond.Condition = string(DependencyConditionStarted)
			}
			(*m)[key] = cond
		}
		return nil
	}

	return fmt.Errorf("dependsOn must be a list or map")
}

// Names returns the dependency names in sorted order.
func (m DependsOnMap) Names() []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ConditionHealthy returns true if the dependency condition is healthy
func (m DependsOnMap) ConditionHealthy(name string) bool {
	cond, ok := m[name]
	return ok && cond.Condition == string(DependencyConditionHealthy)
}

// ConditionStarted returns true if the dependency condition is started
func (m DependsOnMap) ConditionStarted(name string) bool {
	cond, ok := m[name]
	return ok && cond.Condition == string(DependencyConditionStarted)
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
	// Injected from the map key during loading — never set from YAML.
	Name string `yaml:"-" json:"name" validate:"required,hostname_rfc1123"`

	// KatalogName — unique identifier for the the katalog in the runtime
	KatalogName string `yaml:"-" json:"katalogName,omitempty"`

	// Enabled — include this CRD in the runtime. false = skipped entirely.
	// WARNING: only set to false after stripping Orkestra finalizers from all
	// live CRs — disabled CRDs with live finalizers will cause stuck objects.
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Critical — if true, Orkestra marks the entire controller as degraded when
	// this CRD's health state transitions to degraded.
	// Use for CRDs that are fundamental to the platform's correctness.
	// Critical *bool `yaml:"critical" json:"critical,omitempty"`

	// Description — human-readable description. Shown in /katalog API responses.
	Description string `yaml:"description,omitempty" json:"description,omitempty" validate:"omitempty"`

	// Mode — see CRDMode for full documentation.
	// Auto-detected when omitted based on whether apiTypes.location is set.
	Mode CRDMode `yaml:"mode,omitempty" json:"mode,omitempty" validate:"omitempty,oneof=typed dynamic"`

	// ── API Types ─────────────────────────────────────────────────────────────
	// See APITypes for full field documentation.
	APITypes APITypes `yaml:"apiTypes" json:"apiTypes" validate:"required"`

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
	TypedModeObject       runtime.Object        `yaml:"-" json:"-"`
	ListTypedModeObject   runtime.Object        `yaml:"-" json:"-"`
	DynamicModeObject     func() runtime.Object `yaml:"-" json:"-"`
	ListDynamicModeObject func() runtime.Object `yaml:"-" json:"-"`

	// Scheme — AddToScheme function generated by controller-gen for this API type.
	// Required for typed mode so the REST client can decode API server responses.
	// Not needed for dynamic mode — the dynamic client bypasses scheme decoding.
	// Set in BuildKatalogFromGo() for Go mode. Handled by RegisterScheme() for YAML mode.
	Scheme func(s *runtime.Scheme) error `yaml:"-" json:"-"`

	// ── Computed GVK/GVR ─────────────────────────────────────────────────────
	// Set by setGroupVersionKind() during Katalog validation.
	// Derived from APITypes fields. Never set manually.
	GroupVersion         *schema.GroupVersion        `yaml:"-" json:"-"`
	GroupVersionKind     schema.GroupVersionKind     `yaml:"-" json:"-"`
	GroupVersionResource schema.GroupVersionResource `yaml:"-" json:"-"`

	// ── Scope ─────────────────────────────────────────────────────────────────

	// Namespaced — true if this CRD is namespace-scoped, false if cluster-scoped.
	// Default is true
	Namespaced *bool `yaml:"namespaced,omitempty" json:"namespaced,omitempty"`

	// Namespace — target namespace for namespace-scoped CRDs.
	// Informer watches this namespace only. Empty = all namespaces.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty" validate:"omitempty"`

	// ── Runtime behaviour ─────────────────────────────────────────────────────

	// Workers — number of concurrent reconcile workers for this CRD.
	// Higher values increase throughput but also increase API server load.
	// 0 → uses Orkestra-level default (DEFAULT_WORKERS env var).
	Workers int `yaml:"workers,omitempty" json:"workers,omitempty" validate:"omitempty,gte=1,lte=50"`

	// WorkersActive — records number of active concurrent reconcile workers for this CRD.
	WorkersActive int `yaml:"workersActive,omitempty" json:"workersActive,omitempty" validate:"omitempty,gte=1,lte=50"`

	// Resync — full re-list interval for the informer cache.
	// Triggers a reconcile for every cached object at this interval.
	// 0 → uses Orkestra-level default (DEFAULT_RESYNC env var).
	Resync time.Duration `yaml:"resync,omitempty" json:"resync,omitempty" validate:"omitempty"`

	// DependsOn — names of other CRDs that must reach a condition before this one starts.
	// Orkestra resolves the dependency graph and starts CRDs in topological order.
	// Cycle detection runs at validation time — cycles fail fast with a clear error.
	// Supports three YAML formats (list, key-value, full map) — see DependsOnMap.
	DependsOn DependsOnMap `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`

	// ── OperatorBox ────────────────────────────────────────────────────
	OperatorBox OperatorBoxConfig `yaml:"operatorBox,omitempty" json:"operatorBox,omitempty"`

	// ── Queue ────────────────────────────────────────────────────
	Queue Queue `yaml:"queue,omitempty" json:"queue,omitempty"`

	// Labels           []ResourceLabel  `yaml:"labels,omitempty" json:"labels,omitempty" validate:"omitempty"`
	// LabelSelector filters which resources this CRD entry reconciles.
	// Only resources whose labels match ALL declared key-value pairs are watched.
	// Required for built-in types (ConfigMap, Pod, etc.) — without a selector,
	// Orkestra would reconcile every instance in the cluster.
	// For custom CRDs this is optional — can narrow scope within a CRD.
	LabelSelector SelectorMap `yaml:"labelSelector,omitempty"`

	// FieldSelector filters which resources this CRD entry reconciles.
	// Only resources whose *fields* match ALL declared key-value expressions
	// are listed or watched. Field selectors operate on the server side and
	// support exact-match comparisons on well-known metadata paths
	// (e.g. "metadata.name", "metadata.namespace").
	//
	// Unlike label selectors, field selectors cannot match arbitrary user-defined
	// keys — only fields exposed by the Kubernetes API server. They are evaluated
	// before any client-side filtering, reducing load on the informer pipeline.
	//
	// Common use cases:
	//   - Restricting reconciliation to a specific namespace:
	//       {key: "metadata.namespace", value: "default"}
	//   - Targeting a single object by name:
	//       {key: "metadata.name", value: "my-config"}
	//
	// Field selectors are optional for all types. When omitted, Orkestra will
	// watch all objects permitted by LabelSelector and namespace restrictions.
	FieldSelector SelectorMap `yaml:"fieldSelector,omitempty"`

	// IsBuiltIn is set to true when this CRD entry was enriched from the
	// built-in Kubernetes resource registry. Used for ork validate output
	// and informational logging only — does not affect runtime behavior.
	IsBuiltIn bool `yaml:"-" json:"-"` // never serialized — runtime state only

	// IgnoreStatusPatch reports whether or not to patch the status of this CRD
	IgnoreStatusPatch bool `yaml:"ignoreStatusPatch,omitempty" json:"ignoreStatusPatch,omitempty"`

	// IgnoreObservedGeneration reports whether or not to ignore the observedGeneration field for this CRD.
	IgnoreObservedGeneration bool `yaml:"ignoreObservedGeneration,omitempty" json:"ignoreObservedGeneration,omitempty"`

	// IsStatusless reports whether this CRD has no meaningful readiness semantics.
	// These resources become "Ready" immediately upon creation.
	IsStatusless bool `yaml:"-" json:"IsStatusless,omitempty"`

	// BuiltInGroup is the display name of the API group for built-in resources.
	// "core" for resources in the core group (empty string group).
	// Only set when IsBuiltIn is true.
	BuiltInGroup string `yaml:"-" json:"-"` // never serialized

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
	EnrichmentOutcome EnrichmentOutcome `yaml:"-" json:"-"` // never serialized

	// Endpoints defines which operator HTTP endpoints are enabled for this CRD.
	Endpoints EndpointsConfig `yaml:"endpoints,omitempty" json:"endpoints,omitempty"`

	// Restricted Namespaces
	RestrictedNamespaces RestrictedNamespaces `yaml:"restrictedNamespaces,omitempty" json:"restrictedNamespaces,omitempty"`

	// Allowed Namespaces
	AllowedNamespaces AllowedNamespaces `yaml:"allowedNamespaces,omitempty" json:"allowedNamespaces,omitempty"`

	// Conversion is useful for handling multi-version crd
	Conversion *CRDConversion `yaml:"conversion,omitempty" json:"conversion,omitempty"`

	// Validation is a list of rules
	Validation *ValidationConfig `yaml:"validation,omitempty" json:"validation,omitempty"`

	// Mutation is a list of rules
	Mutation *MutationConfig `yaml:"mutation,omitempty" json:"mutation,omitempty"`

	// Webhooks controls per-CRD admission webhook behaviour.
	// Only meaningful when ENABLE_ADMISSION_WEBHOOK=true.
	// By default, any CRD with Validation or Mutation rules is included
	// in the corresponding webhook configuration automatically.
	// Set validation: false or mutation: false to opt a specific CRD out of
	// admission-time interception while keeping its reconcile-time enforcement.
	Webhooks AdmissionWebhookConfig `yaml:"webhooks,omitempty" json:"webhooks,omitempty"`

	// Normalize Spec fields before rendering
	Normalize *NormalizeConfig `yaml:"normalize,omitempty"`

	// NotificationEnabled returns whether this CRD belongs to katalog with notification access
	NotificationEnabled *bool `yaml:"-" json:"-"`

	// RemoveFinalizers -> testing
	RemoveFinalizers bool `yaml:"removeFinalizers,omitempty" json:"removeFinalizers,omitempty"`
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
	Enabled *bool `yaml:"enabled" json:"enabled,omitempty"`

	// Health controls whether the /health endpoint is served.
	Health *bool `yaml:"health" json:"health,omitempty"`

	// Info controls whether the /info endpoint is served.
	Info *bool `yaml:"info" json:"info,omitempty"`
}
