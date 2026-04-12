package katalog

// Status management
// Resources that have a status subresource but lack status.observedGeneration.
// Orkestra should NOT apply generation‑based readiness checks; treat as synced.
var skipObservedGenerationGVKs = []string{
	// Core v1
	"v1/Namespace",             // phase tracks lifecycle
	"v1/Node",                  // status reported by kubelet
	"v1/Service",               // loadBalancer populated by cloud provider
	"v1/Endpoints",             // managed by endpoints controller
	"v1/PersistentVolume",      // status reflects binding
	"v1/PersistentVolumeClaim", // phase tracks provisioning
	"v1/ResourceQuota",         // status holds calculated usage
	"v1/LimitRange",            // status injected by API server
	"v1/Pod",                   // no observedGeneration (alpha possible)

	// apiextensions.k8s.io
	"apiextensions.k8s.io/v1/CustomResourceDefinition", // conditions only

	// apiregistration.k8s.io
	"apiregistration.k8s.io/v1/APIService", // conditions, no observedGeneration

	// policy/v1
	"policy/v1/PodDisruptionBudget", // status fields calculated

	// discovery.k8s.io
	"discovery.k8s.io/v1/EndpointSlice", // managed by controller

	// storage.k8s.io
	"storage.k8s.io/v1/VolumeAttachment", // low‑level binding status
}

// Resources that DO NOT HAVE a /status subresource.
// Orkestra must NOT attempt any status patches on these types.
var skipStatusSubresourceGVKs = []string{
	// Core v1 – static data / config
	"v1/ConfigMap",
	"v1/Secret",
	"v1/ServiceAccount",
	"v1/Event",
	"v1/ComponentStatus",
	"v1/PodTemplate",

	// rbac.authorization.k8s.io – pure policy
	"rbac.authorization.k8s.io/v1/Role",
	"rbac.authorization.k8s.io/v1/RoleBinding",
	"rbac.authorization.k8s.io/v1/ClusterRole",
	"rbac.authorization.k8s.io/v1/ClusterRoleBinding",

	// admissionregistration.k8s.io – API server config
	"admissionregistration.k8s.io/v1/MutatingWebhookConfiguration",
	"admissionregistration.k8s.io/v1/ValidatingWebhookConfiguration",

	// networking.k8s.io
	"networking.k8s.io/v1/NetworkPolicy",

	// scheduling.k8s.io
	"scheduling.k8s.io/v1/PriorityClass",

	// events.k8s.io
	"events.k8s.io/v1/Event",

	// Jobs
	"batch/v1/Job",
	"batch/v1/CronJob",
}

// Export methods
// -----------------------------------------------------------------------------

// // SkipObservedGeneration returns a list of resources by GVK to be skipped during generation checks.
// func (k *Katalog) SkipObservedGeneration() []string {
// 	return skipObservedGenerationGVKs
// }

// // skipStatusSubresourceGVKs returns a list of resources by GVK to be skipped during status patching.
// func (k *Katalog) SkipStatusSubresourceGVKs() []string {
// 	return skipStatusSubresourceGVKs
// }
