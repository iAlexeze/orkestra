// Package children reads and enriches child Kubernetes resources that are
// declared in a Katalog's operatorBox. It is the bridge between the template
// resolver (which holds the owner CR's fields) and the Kubernetes API (which
// holds the live state of every child resource).
//
// The single entry point is [ReadChildren]. It:
//  1. Merges onCreate and onReconcile template declarations.
//  2. For each declared resource type, reads the live objects from Kubernetes.
//  3. Applies enrichment layers (pods, endpoints, warnings, PV).
//  4. Returns a structured map injected into the template resolver under "children".
//
// # File layout
//
//   - children.go   — this file; package doc and ReadChildren entry point
//   - read.go       — readResourceGroup, firstValue, mergeTemplates
//   - names.go      — name resolution (resolvedChildName, *Names helpers)
//   - foreach.go    — forEach expansion for all built-in resource types
//   - foreach_customresources.go — forEach expansion for custom resources
//   - enrich_pods.go         — _pods enrichment and pod summary building
//   - enrich_endpoints.go    — _endpoints enrichment
//   - enrich_warnings.go     — _warnings enrichment (workload + pod events)
//   - enrich_pvc.go          — _pv enrichment for PersistentVolumeClaims
//   - enrich_pv.go           — _pvc enrichment for PersistentVolumes
//   - enrich_replicasets.go  — _owner for ReplicaSets; _replicaSets for Deployments
//   - enrich_cronjobs.go     — _activeJobs, _lastJob, _lastSuccessfulJob for CronJobs
//   - enrich_statefulsets.go — _pvcs for StatefulSets
//   - enrich_storageclass.go — _storageClass for PersistentVolumeClaims
//   - enrich_service_pods.go — _backingPods for Services
//   - enrich_ingress.go      — _loadBalancerIPs, _tlsSecrets for Ingresses
//   - enrich_node.go         — _node for Pods
//   - enrich_hpa.go          — _currentMetrics, _scaleTarget for HPAs
//
// See docs/ for a progressive walkthrough of each layer.
package children

import (
	"context"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// ReadChildren reads all child resources declared in the Katalog's onCreate
// templates and returns a structured map for use in status field expressions.
//
// The returned map is injected into the template resolver under the "children"
// key. Status field expressions can then reference child resource state:
//
//	# Singular — the first/only resource of this type
//	{{ .children.deployment.status.readyReplicas }}
//	{{ .children.service.status.loadBalancer.ingress }}
//
//	# Plural — all resources of this type, by name
//	{{ (index .children.deployments "my-site-api").status.readyReplicas }}
//
// ReadChildren is called after runTemplateReconcile — child resources exist
// at this point. Missing status fields resolve to "" (missingkey=zero),
// which is correct eventual consistency behaviour for newly created resources.
//
// API cost: one GET per child resource type × count.
// For the common case (one Deployment + one Service), this is two GETs.
//
// This function never returns an error that should fail the reconcile.
// Read failures are logged and the child is omitted from the map.
// Status patching proceeds with whatever children were successfully read.
func ReadChildren(
	ctx context.Context,
	kube kubeclient.KubeClient,
	obj domain.Object,
	resolver *orktmpl.Resolver,
	crd orktypes.CRDEntry,
) map[string]interface{} {
	children := map[string]interface{}{}

	// Apply conditional enrichment gates — filter crd.Enrich to only the targets
	// whose when:/anyOf: conditions pass for the current CR state.
	// Unconditional targets (no conditions) always pass.
	// enrichmentEnabled() called deep inside each enrich_*.go sees the filtered list.
	if !crd.EnrichAll {
		crd.Enrich = crd.ActiveEnrichTargets(resolver.Data(), resolver.TemplateEvaluator())
	}

	// Collect all template sources across onCreate and onReconcile.
	// We only read resources that are declared — not all resources in the namespace.
	templates := mergeTemplates(crd.OperatorBox)

	// ── Deployments ───────────────────────────────────────────────────────
	if len(templates.Deployments) > 0 {
		dNames := deploymentNames(resolver, templates.Deployments)
		m := readResourceGroup(ctx, kube, obj, resolver, DeploymentGVR, dNames)
		// Deployments do not directly own pods — their ReplicaSets do.
		// Filtering by ownerKind=ReplicaSet excludes Job pods that share the
		// same orkestra-owner label but have a different immediate controller.
		enrichGroupWithPods(ctx, kube, m, crd, "ReplicaSet")
		enrichGroupWithReplicaSets(ctx, kube, m, crd)
		enrichGroupWithWarnings(ctx, kube, m, crd, "Deployment")
		children["deployments"] = m
		children["deployment"] = firstValue(m)

		// When replicasets enrichment is active but no RSes are declared in onCreate,
		// synthesize .children.replicaset from the active RS embedded in _replicaSets.
		// This makes replicaSetOwnerName and replicaSetOwnerKind work on .children.replicaset.
		if crd.ShouldEnrich("replicasets") && len(templates.ReplicaSets) == 0 {
			if rsGroup := activeReplicaSetGroup(m); len(rsGroup) > 0 {
				children["replicasets"] = rsGroup
				children["replicaset"] = firstValue(rsGroup)
			}
		}
	}

	// ── StatefulSets ───────────────────────────────────────────────────────
	if len(templates.StatefulSets) > 0 {
		dNames := statefulSetNames(resolver, templates.StatefulSets)
		m := readResourceGroup(ctx, kube, obj, resolver, StatefulSetGVR, dNames)
		enrichGroupWithPods(ctx, kube, m, crd, "StatefulSet")
		enrichGroupWithStatefulSetPVCs(ctx, kube, m, crd)
		enrichGroupWithWarnings(ctx, kube, m, crd, "StatefulSet")
		children["statefulsets"] = m
		children["statefulset"] = firstValue(m)
	}

	// ── ReplicaSets ───────────────────────────────────────────────────────
	if len(templates.ReplicaSets) > 0 {
		dNames := replicaSetNames(resolver, templates.ReplicaSets)
		m := readResourceGroup(ctx, kube, obj, resolver, ReplicaSetGVR, dNames)
		enrichGroupWithPods(ctx, kube, m, crd, "ReplicaSet")
		enrichGroupWithOwner(ctx, kube, m, crd)
		enrichGroupWithWarnings(ctx, kube, m, crd, "ReplicaSet")
		children["replicasets"] = m
		children["replicaset"] = firstValue(m)
	}

	// ── CustomResources ───────────────────────────────────────────────────────
	// Each entry has its own APIVersion/Kind, so GVR is resolved per entry via
	// RESTMapper rather than using a single shared GVR.
	if len(templates.CustomResource) > 0 {
		m := readCustomResourceGroup(ctx, kube, obj, resolver, templates.CustomResource)
		enrichGroupWithWarnings(ctx, kube, m, crd, "")
		children["customs"] = m
		children["custom"] = firstValue(m)
	}

	// ── Services ──────────────────────────────────────────────────────────
	if len(templates.Services) > 0 {
		svcNames := serviceNames(resolver, templates.Services)
		m := readResourceGroup(ctx, kube, obj, resolver, ServiceGVR, svcNames)
		enrichGroupWithEndpoints(ctx, kube, m, crd)
		enrichGroupWithBackingPods(ctx, kube, m, crd)
		enrichGroupWithWarnings(ctx, kube, m, crd, "Service")
		children["services"] = m
		children["service"] = firstValue(m)

		// Auto-fetch the EndpointSlice for each declared Service.
		esMap := readEndpointSlicesForServices(ctx, kube, obj, svcNames)
		if len(esMap) > 0 {
			children["endpointslices"] = esMap
			children["endpointslice"] = firstValue(esMap)
		}
	}

	// ── Secrets ───────────────────────────────────────────────────────────
	if len(templates.Secrets) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, SecretGVR,
			secretNames(resolver, templates.Secrets))
		enrichGroupWithWarnings(ctx, kube, m, crd, "Secret")
		children["secrets"] = m
		children["secret"] = firstValue(m)
	}

	// ── ConfigMaps ────────────────────────────────────────────────────────
	if len(templates.ConfigMaps) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, ConfigMapGVR,
			configMapNames(resolver, templates.ConfigMaps))
		enrichGroupWithWarnings(ctx, kube, m, crd, "ConfigMap")
		children["configmaps"] = m
		children["configmap"] = firstValue(m)
	}

	// ── Jobs ──────────────────────────────────────────────────────────────
	if len(templates.Jobs) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, JobGVR,
			jobNames(resolver, templates.Jobs))
		enrichGroupWithPods(ctx, kube, m, crd, "Job")
		enrichGroupWithWarnings(ctx, kube, m, crd, "Job")
		children["jobs"] = m
		children["job"] = firstValue(m)
	}

	// ── CronJobs ──────────────────────────────────────────────────────────
	if len(templates.CronJobs) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, CronJobGVR,
			cronJobNames(resolver, templates.CronJobs))
		enrichGroupWithCronJobChildren(ctx, kube, m, crd)
		enrichGroupWithWarnings(ctx, kube, m, crd, "CronJob")
		children["cronjobs"] = m
		children["cronjob"] = firstValue(m)
	}

	// ── Pods ──────────────────────────────────────────────────────────────
	if len(templates.Pods) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, PodGVR,
			podNames(resolver, templates.Pods))
		enrichGroupWithNode(ctx, kube, m, crd)
		enrichGroupWithWarnings(ctx, kube, m, crd, "Pod")
		children["pods"] = m
		children["pod"] = firstValue(m)
	}

	// ── ServiceAccounts ───────────────────────────────────────────────────
	if len(templates.ServiceAccounts) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, ServiceAccountGVR,
			serviceAccountNames(resolver, templates.ServiceAccounts))
		enrichGroupWithWarnings(ctx, kube, m, crd, "ServiceAccount")
		children["serviceaccounts"] = m
		children["serviceaccount"] = firstValue(m)
	}

	// ── Namespaces ───────────────────────────────────────────────────
	if len(templates.Namespaces) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, NamespaceGVR,
			namespaceNames(resolver, templates.Namespaces))
		enrichGroupWithWarnings(ctx, kube, m, crd, "Namespace")
		children["namespaces"] = m
		children["namespace"] = firstValue(m)
	}

	// ── Ingresses ────────────────────────────────────────────────────────
	if len(templates.Ingresses) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, IngressGVR,
			ingressNames(resolver, templates.Ingresses))
		enrichGroupWithIngressData(ctx, kube, m, crd)
		enrichGroupWithWarnings(ctx, kube, m, crd, "Ingress")
		children["ingresses"] = m
		children["ingress"] = firstValue(m)
	}

	// ── HorizontalPodAutoscalers ─────────────────────────────────────────
	if len(templates.HorizontalPodAutoscalers) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, HorizontalPodAutoscalerGVR,
			hpaNames(resolver, templates.HorizontalPodAutoscalers))
		enrichGroupWithHPAData(ctx, kube, m, crd)
		enrichGroupWithWarnings(ctx, kube, m, crd, "HorizontalPodAutoscaler")
		children["hpas"] = m
		children["hpa"] = firstValue(m)
	}

	// ── PersistentVolumeClaims ────────────────────────────────────────────
	if len(templates.PersistentVolumeClaims) > 0 {
		pvcNms := pvcNames(resolver, templates.PersistentVolumeClaims)
		m := readResourceGroup(ctx, kube, obj, resolver, PersistentVolumeClaimGVR, pvcNms)
		enrichGroupWithPV(ctx, kube, m, crd)
		enrichGroupWithStorageClass(ctx, kube, m, crd)
		enrichGroupWithWarnings(ctx, kube, m, crd, "PersistentVolumeClaim")
		children["persistentvolumeclaims"] = m
		children["pvc"] = firstValue(m)
	}

	// ── PersistentVolumes ─────────────────────────────────────────────────
	// PVs are cluster-scoped — readResourceGroup skips namespace when empty.
	if len(templates.PersistentVolumes) > 0 {
		pvNms := pvNames(resolver, templates.PersistentVolumes)
		m := readResourceGroup(ctx, kube, obj, resolver, PersistentVolumeGVR, pvNms)
		enrichGroupWithPVC(ctx, kube, m, crd)
		enrichGroupWithWarnings(ctx, kube, m, crd, "PersistentVolume")
		children["persistentvolumes"] = m
		children["pv"] = firstValue(m)
	}

	return children
}
