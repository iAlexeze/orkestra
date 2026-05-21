package reconciler

import "github.com/orkspace/orkestra/pkg/children"

// GVR aliases used by reconciler-internal files (run_delete_ordered.go).
// All authoritative GVR definitions live in pkg/children.
var (
	deploymentGVR     = children.DeploymentGVR
	statefulSetGVR    = children.StatefulSetGVR
	serviceGVR        = children.ServiceGVR
	secretGVR         = children.SecretGVR
	configMapGVR      = children.ConfigMapGVR
	serviceAccountGVR = children.ServiceAccountGVR
	jobGVR            = children.JobGVR
	cronJobGVR        = children.CronJobGVR
	ingressGVR        = children.IngressGVR
	pvcGVR            = children.PersistentVolumeClaimGVR
	pvGVR             = children.PersistentVolumeGVR
	hpaGVR            = children.HorizontalPodAutoscalerGVR
	pdbGVR            = children.PodDisruptionBudgetGVR
	namespaceGVR      = children.NamespaceGVR
)
