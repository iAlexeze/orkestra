// pkg/types/types_statefulset.go
package types

// StatefulSetTemplateSource declares one StatefulSet to be managed by Orkestra.
// VolumeClaimTemplateSource declares one PersistentVolumeClaim template for a StatefulSet.
// Each StatefulSet pod receives its own PVC created from this template.
type VolumeClaimTemplateSource struct {
	// Name — PVC template name. Default: "data".
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// StorageClass — storage class to provision from. Required.
	StorageClass string `yaml:"storageClass" json:"storageClass"`

	// StorageSize — requested storage size (e.g. "10Gi"). Required.
	StorageSize string `yaml:"storageSize" json:"storageSize"`

	// MountPath — path inside the container to mount this volume. Default: "/data".
	MountPath string `yaml:"mountPath,omitempty" json:"mountPath,omitempty"`

	// AccessModes — defaults to ["ReadWriteOnce"].
	AccessModes []string `yaml:"accessModes,omitempty" json:"accessModes,omitempty"`
}

type StatefulSetTemplateSource struct {
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Name — StatefulSet name. Default: "{{ .metadata.name }}".
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Namespace — target namespace. Default: CR namespace.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// Image — container image. Required.
	Image string `yaml:"image" json:"image" validate:"omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use
	// for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	ImagePullSecrets []string `yaml:"imagePullSecrets,omitempty" json:"imagePullSecrets,omitempty" validate:"omitempty"`

	// Tag — image tag. Default: "latest".
	Tag string `yaml:"tag,omitempty" json:"tag,omitempty"`

	// Replicas — number of pod replicas. Default: "1".
	Replicas string `yaml:"replicas,omitempty" json:"replicas,omitempty"`

	// Port — container port. "0" or empty means no port exposed.
	Port string `yaml:"port,omitempty" json:"port,omitempty"`

	// Protocol — network protocol for the container port.
	// Accepted values: TCP (default), UDP, SCTP. Omit to use TCP.
	Protocol string `yaml:"protocol,omitempty" json:"protocol,omitempty"`

	// ServiceName — name of the headless Service governing the StatefulSet.
	// Default: same as Name.
	ServiceName string `yaml:"serviceName,omitempty" json:"serviceName,omitempty"`

	// VolumeClaimTemplates — one or more PVC templates; each pod gets its own volume per entry.
	VolumeClaimTemplates []VolumeClaimTemplateSource `yaml:"volumeClaimTemplates,omitempty" json:"volumeClaimTemplates,omitempty"`

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

	Labels      []ResourceLabel `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations []ResourceLabel `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	Env         EnvVarList      `yaml:"env,omitempty" json:"env,omitempty"`
	EnvFrom     *EnvFrom        `yaml:"envFrom,omitempty" json:"envFrom,omitempty"`

	// Resources — CPU and memory requests/limits for the primary container.
	// Set resources.profile for a named preset, or resources.requests/limits for
	// explicit values. Profile and explicit values are mutually exclusive.
	Resources *ResourceRequirements `yaml:"resources,omitempty" json:"resources,omitempty" validate:"omitempty"`

	Reconcile  bool         `yaml:"reconcile,omitempty" json:"reconcile,omitempty"`
	Conditions []Condition  `yaml:"when,omitempty" json:"when,omitempty"`
	AnyOf      []Condition  `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	ForEach    *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// Probes — startup, liveness, and readiness probe configuration.
	Probes *ProbesConfig `yaml:"probes,omitempty" json:"probes,omitempty"`

	// SecurityContext — container-level security settings.
	// Set securityContext.profile for a named preset (baseline, restricted, hardened)
	// or declare individual fields. Profile and explicit fields are mutually exclusive.
	SecurityContext *ContainerSecurityContext `yaml:"securityContext,omitempty" json:"securityContext,omitempty"`

	// PodSecurity — pod-level security settings applied to the pod spec.
	// Set podSecurity.profile for a named preset or declare individual fields.
	PodSecurity *PodSecurityContext `yaml:"podSecurity,omitempty" json:"podSecurity,omitempty"`

	// RollingUpdate — rolling update strategy for this StatefulSet.
	// Set rollingUpdate.profile for a named preset (safe, fast, blue-green),
	// or declare maxSurge/maxUnavailable explicitly.
	RollingUpdate *RollingUpdateBehavior `yaml:"rollingUpdate,omitempty" json:"rollingUpdate,omitempty"`

	// Volumes — pod volumes available for mounting into the container.
	Volumes []VolumeSource `yaml:"volumes,omitempty" json:"volumes,omitempty"`

	// VolumeMounts — mounts for the primary container.
	VolumeMounts []VolumeMount `yaml:"volumeMounts,omitempty" json:"volumeMounts,omitempty"`

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string `json:"sleep,omitempty" yaml:"sleep,omitempty"`
}
