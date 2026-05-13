// pkg/reconciler/children_gvr.go
package reconciler

import kat "github.com/orkspace/orkestra/pkg/katalog"

// ── Child resource GVRs (local variables sourced from katalog) ─────────────
// These remain local to the reconciler package, but now pull directly from
// katalog’s authoritative exported GVRs. No duplication, no string lookups.

var (
	deploymentGVR         = kat.DeploymentGVR
	serviceGVR            = kat.ServiceGVR
	secretGVR             = kat.SecretGVR
	configMapGVR          = kat.ConfigMapGVR
	jobGVR                = kat.JobGVR
	cronJobGVR            = kat.CronJobGVR
	podGVR                = kat.PodGVR
	serviceAccountGVR     = kat.ServiceAccountGVR
	statefulSetGVR        = kat.StatefulSetGVR
	ingressGVR            = kat.IngressGVR
	pvcGVR                = kat.PersistentVolumeClaimGVR
	pvGVR                 = kat.PersistentVolumeGVR
	hpaGVR                = kat.HorizontalPodAutoscalerGVR
	pdbGVR                = kat.PodDisruptionBudgetGVR
	namespaceGVR          = kat.NamespaceGVR
	daemonSetGVR          = kat.DaemonSetGVR
	replicaSetGVR         = kat.ReplicaSetGVR
	networkPolicyGVR      = kat.NetworkPolicyGVR
	roleGVR               = kat.RoleGVR
	roleBindingGVR        = kat.RoleBindingGVR
	clusterRoleGVR        = kat.ClusterRoleGVR
	clusterRoleBindingGVR = kat.ClusterRoleBindingGVR
	nodeGVR               = kat.NodeGVR
	endpointSliceGVR      = kat.EndpointSliceGVR
)
