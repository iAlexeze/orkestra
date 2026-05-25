// pkg/types/types_hook_templates.go
package types

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
	Deployments              []DeploymentTemplateSource     `yaml:"deployments,omitempty" json:"deployments,omitempty" validate:"omitempty"`
	ReplicaSets              []ReplicaSetTemplateSource     `yaml:"replicaSets,omitempty" json:"replicaSets,omitempty" validate:"omitempty"`
	Services                 []ServiceTemplateSource        `yaml:"services,omitempty" json:"services,omitempty" validate:"omitempty"`
	Pods                     []PodTemplateSource            `yaml:"pods,omitempty" json:"pods,omitempty" validate:"omitempty"`
	Jobs                     []JobTemplateSource            `yaml:"jobs,omitempty" json:"jobs,omitempty" validate:"omitempty"`
	CronJobs                 []CronJobTemplateSource        `yaml:"cronJobs,omitempty" json:"cronJobs,omitempty" validate:"omitempty"`
	Secrets                  []SecretTemplateSource         `yaml:"secrets,omitempty" json:"secrets,omitempty" validate:"omitempty"`
	ConfigMaps               []ConfigMapTemplateSource      `yaml:"configMaps,omitempty" json:"configMaps,omitempty" validate:"omitempty"`
	ServiceAccounts          []ServiceAccountTemplateSource `yaml:"serviceAccounts,omitempty" json:"serviceAccounts,omitempty" validate:"omitempty"`
	StatefulSets             []StatefulSetTemplateSource    `yaml:"statefulSets,omitempty" json:"statefulSets,omitempty" validate:"omitempty"`
	Ingresses                []IngressTemplateSource        `yaml:"ingresses,omitempty" json:"ingresses,omitempty" validate:"omitempty"`
	PersistentVolumes        []PVTemplateSource             `yaml:"persistentVolumes,omitempty" json:"persistentVolumes,omitempty" validate:"omitempty"`
	PersistentVolumeClaims   []PVCTemplateSource            `yaml:"persistentVolumeClaims,omitempty" json:"persistentVolumeClaims,omitempty" validate:"omitempty"`
	HorizontalPodAutoscalers []HPATemplateSource            `yaml:"hpa,omitempty" json:"hpa,omitempty" validate:"omitempty"`
	PodDisruptionBudgets     []PDBTemplateSource            `yaml:"pdb,omitempty" json:"pdb,omitempty" validate:"omitempty"`
	Namespaces               []NamespaceTemplateSource      `yaml:"namespaces,omitempty" json:"namespaces,omitempty" validate:"omitempty"`
	Roles                    []RoleTemplateSource           `yaml:"roles,omitempty" json:"roles,omitempty" validate:"omitempty"`
	RoleBindings             []RoleBindingTemplateSource    `yaml:"roleBindings,omitempty" json:"roleBindings,omitempty" validate:"omitempty"`
	CustomResource           []CustomResourceTemplateSource `yaml:"custom,omitempty" json:"custom,omitempty" validate:"omitempty"`

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
	Volumes                     []PlaceholderSource `yaml:"volumes,omitempty" json:"volumes,omitempty" validate:"omitempty"`
	VolumeMounts                []PlaceholderSource `yaml:"volumeMounts,omitempty" json:"volumeMounts,omitempty" validate:"omitempty"`
	ClusterRoles                []PlaceholderSource `yaml:"clusterRoles,omitempty" json:"clusterRoles,omitempty" validate:"omitempty"`
	ClusterRoleBindings         []PlaceholderSource `yaml:"clusterRoleBindings,omitempty" json:"clusterRoleBindings,omitempty" validate:"omitempty"`
	ServiceMonitors             []PlaceholderSource `yaml:"serviceMonitors,omitempty" json:"serviceMonitors,omitempty" validate:"omitempty"`
	PodSecurityPolicies         []PlaceholderSource `yaml:"podSecurityPolicies,omitempty" json:"podSecurityPolicies,omitempty" validate:"omitempty"`
	PriorityClasses             []PlaceholderSource `yaml:"priorityClasses,omitempty" json:"priorityClasses,omitempty" validate:"omitempty"`
	LimitRanges                 []PlaceholderSource `yaml:"limitRanges,omitempty" json:"limitRanges,omitempty" validate:"omitempty"`
	ResourceQuotas              []PlaceholderSource `yaml:"resourceQuotas,omitempty" json:"resourceQuotas,omitempty" validate:"omitempty"`
	RuntimeClasses              []PlaceholderSource `yaml:"runtimeClasses,omitempty" json:"runtimeClasses,omitempty" validate:"omitempty"`
	PriorityLevelConfigurations []PlaceholderSource `yaml:"priorityLevelConfigurations,omitempty" json:"priorityLevelConfigurations,omitempty" validate:"omitempty"`
	PodTemplates                []PlaceholderSource `yaml:"podTemplates,omitempty" json:"podTemplates,omitempty" validate:"omitempty"`
	DaemonSets                  []PlaceholderSource `yaml:"daemonSets,omitempty" json:"daemonSets,omitempty" validate:"omitempty"`
	NetworkPolicies             []PlaceholderSource `yaml:"networkPolicies,omitempty" json:"networkPolicies,omitempty" validate:"omitempty"`

	// Storage
	StorageClasses   []PlaceholderSource `yaml:"storageClasses,omitempty" json:"storageClasses,omitempty" validate:"omitempty"`
	StorageLocations []PlaceholderSource `yaml:"storageLocations,omitempty" json:"storageLocations,omitempty" validate:"omitempty"`
	StoragePools     []PlaceholderSource `yaml:"storagePools,omitempty" json:"storagePools,omitempty" validate:"omitempty"`
	StorageBackups   []PlaceholderSource `yaml:"storageBackups,omitempty" json:"storageBackups,omitempty" validate:"omitempty"`
	StorageSnapshots []PlaceholderSource `yaml:"storageSnapshots,omitempty" json:"storageSnapshots,omitempty" validate:"omitempty"`
	StorageVolumes   []PlaceholderSource `yaml:"storageVolumes,omitempty" json:"storageVolumes,omitempty" validate:"omitempty"`
}

// Placeholder for resources yet to be added to orkestra internal registry
// pkg/orkestra-registry
type PlaceholderSource struct{}
