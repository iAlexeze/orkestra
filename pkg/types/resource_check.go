package types

// ResourceCheck defines how to detect whether a CRD uses a given resource type.
type ResourceCheck struct {
	Get func(crd CRDEntry) bool
}

// usesTemplates safely checks a HookTemplates pointer and returns true
// if the selected slice contains at least one template.
func usesTemplates[T any](tpl *HookTemplates, sel func(*HookTemplates) []T) bool {
	if tpl == nil {
		return false
	}
	return len(sel(tpl)) > 0
}

// ResourceChecks maps resource keys → detection logic.
// Each entry checks OnCreate, OnReconcile, and OnDelete safely.
var ResourceChecks = map[string]ResourceCheck{

	// ───────────────────────────────────────────────
	// Core Kubernetes Workloads
	// ───────────────────────────────────────────────

	"deployments": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []DeploymentTemplateSource { return t.Deployments }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []DeploymentTemplateSource { return t.Deployments }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []DeploymentTemplateSource { return t.Deployments })
		},
	},

	"statefulsets": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []StatefulSetTemplateSource { return t.StatefulSets }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []StatefulSetTemplateSource { return t.StatefulSets }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []StatefulSetTemplateSource { return t.StatefulSets })
		},
	},

	"daemonsets": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []PlaceholderSource { return t.DaemonSets }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []PlaceholderSource { return t.DaemonSets }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []PlaceholderSource { return t.DaemonSets })
		},
	},

	"jobs": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []JobTemplateSource { return t.Jobs }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []JobTemplateSource { return t.Jobs }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []JobTemplateSource { return t.Jobs })
		},
	},

	"cronjobs": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []CronJobTemplateSource { return t.CronJobs }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []CronJobTemplateSource { return t.CronJobs }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []CronJobTemplateSource { return t.CronJobs })
		},
	},

	// ───────────────────────────────────────────────
	// Networking & Services
	// ───────────────────────────────────────────────

	"services": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []ServiceTemplateSource { return t.Services }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []ServiceTemplateSource { return t.Services }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []ServiceTemplateSource { return t.Services })
		},
	},

	"ingresses": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []IngressTemplateSource { return t.Ingresses }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []IngressTemplateSource { return t.Ingresses }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []IngressTemplateSource { return t.Ingresses })
		},
	},

	"networkpolicies": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []PlaceholderSource { return t.NetworkPolicies }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []PlaceholderSource { return t.NetworkPolicies }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []PlaceholderSource { return t.NetworkPolicies })
		},
	},

	// ───────────────────────────────────────────────
	// Configuration & Secrets
	// ───────────────────────────────────────────────

	"configmaps": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []ConfigMapTemplateSource { return t.ConfigMaps }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []ConfigMapTemplateSource { return t.ConfigMaps }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []ConfigMapTemplateSource { return t.ConfigMaps })
		},
	},

	"secrets": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []SecretTemplateSource { return t.Secrets }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []SecretTemplateSource { return t.Secrets }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []SecretTemplateSource { return t.Secrets })
		},
	},

	// ───────────────────────────────────────────────
	// Storage
	// ───────────────────────────────────────────────

	"persistentvolumes": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []PVTemplateSource { return t.PersistentVolumes }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []PVTemplateSource { return t.PersistentVolumes }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []PVTemplateSource { return t.PersistentVolumes })
		},
	},

	"persistentvolumeclaims": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []PVCTemplateSource { return t.PersistentVolumeClaims }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []PVCTemplateSource { return t.PersistentVolumeClaims }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []PVCTemplateSource { return t.PersistentVolumeClaims })
		},
	},

	"volumes": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []PlaceholderSource { return t.Volumes }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []PlaceholderSource { return t.Volumes }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []PlaceholderSource { return t.Volumes })
		},
	},

	"volumemounts": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []PlaceholderSource { return t.VolumeMounts }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []PlaceholderSource { return t.VolumeMounts }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []PlaceholderSource { return t.VolumeMounts })
		},
	},

	// ───────────────────────────────────────────────
	// RBAC Resources
	// ───────────────────────────────────────────────

	"namespaces": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []NamespaceTemplateSource { return t.Namespaces }) ||
				usesTemplates(rc.OnReconcile, func(ht *HookTemplates) []NamespaceTemplateSource { return ht.Namespaces }) ||
				usesTemplates(rc.OnDelete, func(ht *HookTemplates) []NamespaceTemplateSource { return ht.Namespaces })
		},
	},

	"serviceaccounts": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []ServiceAccountTemplateSource { return t.ServiceAccounts }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []ServiceAccountTemplateSource { return t.ServiceAccounts }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []ServiceAccountTemplateSource { return t.ServiceAccounts })
		},
	},

	"roles": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []PlaceholderSource { return t.Roles }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []PlaceholderSource { return t.Roles }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []PlaceholderSource { return t.Roles })
		},
	},

	"rolebindings": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []PlaceholderSource { return t.RoleBindings }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []PlaceholderSource { return t.RoleBindings }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []PlaceholderSource { return t.RoleBindings })
		},
	},

	// ───────────────────────────────────────────────
	// Autoscaling, Disruption, Scheduling (TODOs now supported)
	// ───────────────────────────────────────────────

	"horizontalpodautoscalers": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []HPATemplateSource { return t.HorizontalPodAutoscalers }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []HPATemplateSource { return t.HorizontalPodAutoscalers }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []HPATemplateSource { return t.HorizontalPodAutoscalers })
		},
	},

	"poddisruptionbudgets": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []PDBTemplateSource { return t.PodDisruptionBudgets }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []PDBTemplateSource { return t.PodDisruptionBudgets }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []PDBTemplateSource { return t.PodDisruptionBudgets })
		},
	},

	"podtemplates": {
		Get: func(crd CRDEntry) bool {
			rc := crd.OperatorBox
			return usesTemplates(rc.OnCreate, func(t *HookTemplates) []PlaceholderSource { return t.PodTemplates }) ||
				usesTemplates(rc.OnReconcile, func(t *HookTemplates) []PlaceholderSource { return t.PodTemplates }) ||
				usesTemplates(rc.OnDelete, func(t *HookTemplates) []PlaceholderSource { return t.PodTemplates })
		},
	},
}
