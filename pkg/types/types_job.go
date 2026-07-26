// pkg/types/types_job.go
package types

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
	Version string `yaml:"version,omitempty" json:"version,omitempty" validate:"omitempty"`

	// Name — Job name.
	// Default when omitted: "{{ .metadata.name }}-job"
	Name string `yaml:"name,omitempty" json:"name,omitempty" validate:"omitempty"`

	// Image — container image. Required.
	Image string `yaml:"image" json:"image" validate:"omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use
	// for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	ImagePullSecrets []string `yaml:"imagePullSecrets,omitempty" json:"imagePullSecrets,omitempty" validate:"omitempty"`

	// Command — container entrypoint command.
	// Each element is resolved independently — template expressions are supported per element.
	// e.g. ["sh", "-c", "echo cleaning up {{ .metadata.name }}"]
	Command []string `yaml:"command,omitempty" json:"command,omitempty" validate:"omitempty"`

	// Args — arguments passed to the container command.
	// Each element supports template expressions independently.
	Args []string `yaml:"args,omitempty" json:"args,omitempty" validate:"omitempty"`

	// BackoffLimit — number of Pod restart attempts before the Job is marked Failed.
	// Default: 3.
	BackoffLimit int `yaml:"backoffLimit,omitempty" json:"backoffLimit,omitempty" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty" validate:"omitempty"`

	// Labels — applied to Job metadata. Values support template expressions.
	Labels Labels `yaml:"labels,omitempty" json:"labels,omitempty" validate:"omitempty"`

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
	// This is not an error — it simply means "do not create/update this resource
	// right now". This enables expressive, data‑driven orchestration such as:
	//
	//   when:
	//     - field: spec.exposePublicly
	//       equals: "true"
	//     - field: spec.environment
	//       prefix: "prod"
	//
	// Conditions allow templates to be selectively activated based on the CR's
	// state, enabling dynamic topologies, feature flags, environment‑specific
	// behavior, and conditional provisioning without writing Go code.

	Conditions []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// Reconcile: true — also apply this declaration as drift correction on every
	// reconcile. Equivalent to declaring the same entry under both onCreate and
	// onReconcile. When false (default), only runs on onCreate (idempotent create).
	Reconcile bool `yaml:"reconcile,omitempty" json:"reconcile,omitempty" validate:"omitempty"`

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

	// Resources — CPU and memory requests/limits for the container.
	// Set resources.profile for a named preset, or resources.requests/limits for
	// explicit values. Profile and explicit values are mutually exclusive.
	Resources *ResourceRequirements `yaml:"resources,omitempty" json:"resources,omitempty" validate:"omitempty"`

	// SecurityContext — container-level security settings.
	// Set securityContext.profile for a named preset (baseline, restricted, hardened)
	// or declare individual fields. Profile and explicit fields are mutually exclusive.
	SecurityContext *ContainerSecurityContext `yaml:"securityContext,omitempty" json:"securityContext,omitempty"`

	// PodSecurity — pod-level security settings applied to the pod spec.
	// Set podSecurity.profile for a named preset or declare individual fields.
	PodSecurity *PodSecurityContext `yaml:"podSecurity,omitempty" json:"podSecurity,omitempty"`

	// Volumes — pod volumes available for mounting into the container.
	Volumes []VolumeSource `yaml:"volumes,omitempty" json:"volumes,omitempty"`

	// VolumeMounts — mounts for the primary container.
	VolumeMounts []VolumeMount `yaml:"volumeMounts,omitempty" json:"volumeMounts,omitempty"`

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string `json:"sleep,omitempty" yaml:"sleep,omitempty"`
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
	Version string `yaml:"version,omitempty" json:"version,omitempty" validate:"omitempty"`

	// Name — CronJob name.
	// Default when omitted: "{{ .metadata.name }}-cronjob"
	Name string `yaml:"name,omitempty" json:"name,omitempty" validate:"omitempty"`

	// Schedule — cron schedule expression. Required.
	// Static: "0 * * * *" (every hour)
	// Dynamic: "{{ .spec.schedule }}"
	Schedule string `yaml:"schedule" json:"schedule" validate:"required"`

	// Image — container image. Required.
	Image string `yaml:"image" json:"image" validate:"omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use
	// for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	ImagePullSecrets []string `yaml:"imagePullSecrets,omitempty" json:"imagePullSecrets,omitempty" validate:"omitempty"`

	// Command — container entrypoint. Each element supports template expressions.
	Command []string `yaml:"command,omitempty" json:"command,omitempty" validate:"omitempty"`

	// Args — container arguments. Each element supports template expressions.
	Args []string `yaml:"args,omitempty" json:"args,omitempty" validate:"omitempty"`

	// Namespace — target namespace.
	// Default when omitted: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty" validate:"omitempty"`

	// Labels — applied to CronJob metadata. Values support template expressions.
	Labels Labels `yaml:"labels,omitempty" json:"labels,omitempty" validate:"omitempty"`

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
	Reconcile bool `yaml:"reconcile,omitempty" json:"reconcile,omitempty" validate:"omitempty"`

	// Conditions declares the set of runtime predicates that must all evaluate to
	// true for this resource template to be applied during reconciliation.
	//
	// Each condition inspects a field on the live Custom Resource using dot-notation
	// (e.g. "spec.enabled", "metadata.labels.tier") and compares it against a value
	// using the chosen operator. All conditions in the list are AND‑ed together.
	//
	// If any condition fails, the resource is skipped for that reconcile cycle.
	// This is not an error — it simply means "do not create/update this resource
	// right now". This enables expressive, data‑driven orchestration such as:
	//
	//   when:
	//     - field: spec.exposePublicly
	//       equals: "true"
	//     - field: spec.environment
	//       prefix: "prod"
	//
	// Conditions allow templates to be selectively activated based on the CR's
	// state, enabling dynamic topologies, feature flags, environment‑specific
	// behavior, and conditional provisioning without writing Go code.

	Conditions []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	Suspend                    string `yaml:"suspend,omitempty" json:"suspend,omitempty"`
	SuccessfulJobsHistoryLimit string `yaml:"successfulJobsHistoryLimit,omitempty" json:"successfulJobsHistoryLimit,omitempty"`
	FailedJobsHistoryLimit     string `yaml:"failedJobsHistoryLimit,omitempty" json:"failedJobsHistoryLimit,omitempty"`
	ConcurrencyPolicy          string `yaml:"concurrencyPolicy,omitempty" json:"concurrencyPolicy,omitempty"`
	StartingDeadlineSeconds    string `yaml:"startingDeadlineSeconds,omitempty" json:"startingDeadlineSeconds,omitempty"`

	// Resources — CPU and memory requests/limits for the container.
	// Set resources.profile for a named preset, or resources.requests/limits for
	// explicit values. Profile and explicit values are mutually exclusive.
	Resources *ResourceRequirements `yaml:"resources,omitempty" json:"resources,omitempty" validate:"omitempty"`

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

	// SecurityContext — container-level security settings.
	// Set securityContext.profile for a named preset (baseline, restricted, hardened)
	// or declare individual fields. Profile and explicit fields are mutually exclusive.
	SecurityContext *ContainerSecurityContext `yaml:"securityContext,omitempty" json:"securityContext,omitempty"`

	// PodSecurity — pod-level security settings applied to the pod spec.
	// Set podSecurity.profile for a named preset or declare individual fields.
	PodSecurity *PodSecurityContext `yaml:"podSecurity,omitempty" json:"podSecurity,omitempty"`

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string `json:"sleep,omitempty" yaml:"sleep,omitempty"`
}
