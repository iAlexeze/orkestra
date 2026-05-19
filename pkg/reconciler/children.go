// pkg/reconciler/children.go
package reconciler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ── Read children ────────────────────────────────────────────────────────
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
	operatorBox orktypes.OperatorBoxConfig,
	crd orktypes.CRDEntry,
) map[string]interface{} {
	children := map[string]interface{}{}

	// Collect all template sources across onCreate and onReconcile.
	// We only read resources that are declared — not all resources in the namespace.
	templates := mergeTemplates(operatorBox)

	// ── Deployments ───────────────────────────────────────────────────────
	if len(templates.Deployments) > 0 {
		dNames := deploymentNames(resolver, templates.Deployments)
		m := readResourceGroup(ctx, kube, obj, resolver, deploymentGVR, dNames)
		// Deployments do not directly own pods — their ReplicaSets do.
		// Filtering by ownerKind=ReplicaSet excludes Job pods that share the
		// same orkestra-owner label but have a different immediate controller.
		enrichGroupWithPods(ctx, kube, m, crd, "ReplicaSet")
		enrichGroupWithWarnings(ctx, kube, m, crd, "Deployment")
		children["deployments"] = m
		children["deployment"] = firstValue(m)
	}

	// ── StatefulSets ───────────────────────────────────────────────────────
	if len(templates.StatefulSets) > 0 {
		dNames := statefulSetNames(resolver, templates.StatefulSets)
		m := readResourceGroup(ctx, kube, obj, resolver, statefulSetGVR, dNames)
		enrichGroupWithPods(ctx, kube, m, crd, "StatefulSet")
		enrichGroupWithWarnings(ctx, kube, m, crd, "StatefulSet")
		children["statefulsets"] = m
		children["statefulset"] = firstValue(m)
	}

	// ── ReplicaSets ───────────────────────────────────────────────────────
	if len(templates.ReplicaSets) > 0 {
		dNames := replicaSetNames(resolver, templates.ReplicaSets)
		m := readResourceGroup(ctx, kube, obj, resolver, replicaSetGVR, dNames)
		enrichGroupWithPods(ctx, kube, m, crd, "ReplicaSet")
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
		m := readResourceGroup(ctx, kube, obj, resolver, serviceGVR, svcNames)
		// Embed _endpoints into each service map when endpoint enrichment is enabled.
		enrichGroupWithEndpoints(ctx, kube, m, crd)
		enrichGroupWithWarnings(ctx, kube, m, crd, "Service")
		children["services"] = m
		children["service"] = firstValue(m)

		// Auto-fetch the EndpointSlice for each declared Service.
		// EndpointSlices are created by Kubernetes (not Orkestra) and are labelled
		// with kubernetes.io/service-name=<service-name>, so we list by label.
		esMap := readEndpointSlicesForServices(ctx, kube, obj, svcNames)
		if len(esMap) > 0 {
			children["endpointslices"] = esMap
			children["endpointslice"] = firstValue(esMap)
		}
	}

	// ── Secrets ───────────────────────────────────────────────────────────
	if len(templates.Secrets) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, secretGVR,
			secretNames(resolver, templates.Secrets))
		enrichGroupWithWarnings(ctx, kube, m, crd, "Secret")
		children["secrets"] = m
		children["secret"] = firstValue(m)
	}

	// ── ConfigMaps ────────────────────────────────────────────────────────
	if len(templates.ConfigMaps) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, configMapGVR,
			configMapNames(resolver, templates.ConfigMaps))
		enrichGroupWithWarnings(ctx, kube, m, crd, "ConfigMap")
		children["configmaps"] = m
		children["configmap"] = firstValue(m)
	}

	// ── Jobs ──────────────────────────────────────────────────────────────
	if len(templates.Jobs) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, jobGVR,
			jobNames(resolver, templates.Jobs))
		// Job pods are owned directly by the Job controller.
		enrichGroupWithPods(ctx, kube, m, crd, "Job")
		enrichGroupWithWarnings(ctx, kube, m, crd, "Job")
		children["jobs"] = m
		children["job"] = firstValue(m)
	}

	// ── CronJobs ──────────────────────────────────────────────────────────
	if len(templates.CronJobs) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, cronJobGVR,
			cronJobNames(resolver, templates.CronJobs))
		enrichGroupWithWarnings(ctx, kube, m, crd, "CronJob")
		children["cronjobs"] = m
		children["cronjob"] = firstValue(m)
	}

	// ── Pods ──────────────────────────────────────────────────────────────
	if len(templates.Pods) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, podGVR,
			podNames(resolver, templates.Pods))
		enrichGroupWithWarnings(ctx, kube, m, crd, "Pod")
		children["pods"] = m
		children["pod"] = firstValue(m)
	}

	// ── ServiceAccounts ───────────────────────────────────────────────────
	if len(templates.ServiceAccounts) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, serviceAccountGVR,
			serviceAccountNames(resolver, templates.ServiceAccounts))
		enrichGroupWithWarnings(ctx, kube, m, crd, "ServiceAccount")
		children["serviceaccounts"] = m
		children["serviceaccount"] = firstValue(m)
	}

	// ── Namespaces ───────────────────────────────────────────────────
	if len(templates.Namespaces) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, namespaceGVR,
			namespaceNames(resolver, templates.Namespaces))
		enrichGroupWithWarnings(ctx, kube, m, crd, "Namespace")
		children["namespaces"] = m
		children["namespace"] = firstValue(m)
	}

	// ── Ingresses ────────────────────────────────────────────────────────
	if len(templates.Ingresses) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, ingressGVR,
			ingressNames(resolver, templates.Ingresses))
		enrichGroupWithWarnings(ctx, kube, m, crd, "Ingress")
		children["ingresses"] = m
		children["ingress"] = firstValue(m)
	}

	// ── HorizontalPodAutoscalers ─────────────────────────────────────────
	if len(templates.HorizontalPodAutoscalers) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, hpaGVR,
			hpaNames(resolver, templates.HorizontalPodAutoscalers))
		enrichGroupWithWarnings(ctx, kube, m, crd, "HorizontalPodAutoscaler")
		children["hpas"] = m
		children["hpa"] = firstValue(m)
	}

	// ── PersistentVolumeClaims ────────────────────────────────────────────
	if len(templates.PersistentVolumeClaims) > 0 {
		pvcNms := pvcNames(resolver, templates.PersistentVolumeClaims)
		m := readResourceGroup(ctx, kube, obj, resolver, pvcGVR, pvcNms)
		enrichGroupWithPV(ctx, kube, m, crd)
		enrichGroupWithWarnings(ctx, kube, m, crd, "PersistentVolumeClaim")
		children["persistentvolumeclaims"] = m
		children["pvc"] = firstValue(m)
	}

	// ── PersistentVolumes ─────────────────────────────────────────────────
	// PVs are cluster-scoped — readResourceGroup skips namespace when empty.
	if len(templates.PersistentVolumes) > 0 {
		pvNms := pvNames(resolver, templates.PersistentVolumes)
		m := readResourceGroup(ctx, kube, obj, resolver, pvGVR, pvNms)
		enrichGroupWithWarnings(ctx, kube, m, crd, "PersistentVolume")
		children["persistentvolumes"] = m
		children["pv"] = firstValue(m)
	}

	return children
}

// readCustomResourceGroup reads each custom resource entry using its own GVR resolved
// via the RESTMapper. Unlike built-in types, custom resources may have different
// APIVersions/Kinds within the same list, so GVR must be resolved per entry.
func readCustomResourceGroup(
	ctx context.Context,
	kube kubeclient.KubeClient,
	obj domain.Object,
	resolver *orktmpl.Resolver,
	srcs []orktypes.CustomResourceTemplateSource,
) map[string]interface{} {
	result := map[string]interface{}{}

	expanded := expandForEachCustomResources(resolver, srcs)
	for i := range expanded {
		src := &expanded[i]

		name, err := resolver.Resolve(src.Metadata.Name)
		if err != nil || name == "" {
			continue
		}

		gvr, err := src.ResolveGVR(kube.Mapper())
		if err != nil {
			logger.FromContext(ctx).Debug().
				Str("resource", name).
				Str("gvk", src.GVKString()).
				Err(err).
				Msg("children: failed to resolve GVR for custom resource — omitted from status context")
			continue
		}

		ns, _ := resolver.Resolve(src.Metadata.Namespace)
		if ns == "" {
			ns = obj.GetNamespace()
		}

		// hasStatus: false → skip this resource entirely (no status to propagate).
		// hasStatus: true or nil → read and include status.
		if src.HasStatus != nil && !*src.HasStatus {
			continue
		}

		var u *unstructured.Unstructured
		if src.IsNamespaced() && ns != "" {
			u, err = kube.DynamicClient().Resource(gvr).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
		} else {
			u, err = kube.DynamicClient().Resource(gvr).Get(ctx, name, metav1.GetOptions{})
		}

		if err != nil {
			if !strings.Contains(err.Error(), "not found") {
				logger.FromContext(ctx).Debug().
					Str("resource", fmt.Sprintf("%s/%s", ns, name)).
					Str("gvr", gvr.String()).
					Err(err).
					Msg("children: custom resource read failed — omitted from status context")
			}
			continue
		}

		o := u.Object
		if s, ok := o["status"]; !ok || s == nil {
			o["status"] = map[string]interface{}{}
		}
		result[name] = o
	}

	return result
}

// readEndpointSlicesForServices lists the EndpointSlice for each declared Service
// using the kubernetes.io/service-name label. The result is keyed by service name
// so templates can reference {{ .children.endpointslice }} for single-service katalogs.
func readEndpointSlicesForServices(
	ctx context.Context,
	kube kubeclient.KubeClient,
	obj domain.Object,
	svcNames []resolvedChildName,
) map[string]interface{} {
	result := map[string]interface{}{}
	for _, svc := range svcNames {
		ns := svc.namespace
		if ns == "" {
			ns = obj.GetNamespace()
		}
		list, err := kube.DynamicClient().
			Resource(endpointSliceGVR).
			Namespace(ns).
			List(ctx, metav1.ListOptions{
				LabelSelector:   fmt.Sprintf("kubernetes.io/service-name=%s", svc.name),
				Limit:           1,
				ResourceVersion: "0",
			})
		if err != nil || len(list.Items) == 0 {
			continue
		}
		esObj := list.Items[0].Object
		if s, ok := esObj["status"]; !ok || s == nil {
			esObj["status"] = map[string]interface{}{}
		}
		result[svc.name] = esObj
	}
	return result
}

// readResourceGroup reads one or more resources of the same type and returns
// a map[name → objectMap] for all that were found.
// Missing resources are omitted silently — they may not exist yet on the first reconcile.
func readResourceGroup(
	ctx context.Context,
	kube kubeclient.KubeClient,
	obj domain.Object,
	resolver *orktmpl.Resolver,
	gvr schema.GroupVersionResource,
	names []resolvedChildName,
) map[string]interface{} {
	result := map[string]interface{}{}

	for _, child := range names {
		ns := child.namespace
		if ns == "" {
			ns = obj.GetNamespace()
		}

		var err error
		var u *unstructured.Unstructured
		if ns != "" {
			// Namespaced — use resolved namespace (template or owner fallback)
			u, err = kube.DynamicClient().
				Resource(gvr).
				Namespace(ns).
				Get(ctx, child.name, metav1.GetOptions{})
		} else {
			// Cluster-scoped (e.g. Namespace, ClusterRole)
			u, err = kube.DynamicClient().
				Resource(gvr).
				Get(ctx, child.name, metav1.GetOptions{})
		}

		if err != nil {
			// Not found on first reconcile is expected and not an error.
			// Any other error is worth logging at debug level.
			if !strings.Contains(err.Error(), "not found") {
				logger.FromContext(ctx).Debug().
					Str("resource", fmt.Sprintf("%s/%s", ns, child.name)).
					Str("gvr", gvr.String()).
					Err(err).
					Msg("children: read failed — omitted from status context")
			}
			continue
		}

		// Ensure status is always a non-nil map.
		// A newly-created Deployment may have no status block yet — Kubernetes
		// hasn't populated readyReplicas. Without this guard, the template
		// "{{ .children.deployment.status.readyReplicas }}" causes a nil pointer
		// dereference because .status resolves to nil interface{}, not an empty map.
		// With this guard it resolves to "" via missingkey=zero — correct behaviour.
		obj := u.Object
		if s, ok := obj["status"]; !ok || s == nil {
			obj["status"] = map[string]interface{}{}
		}
		result[child.name] = obj
	}

	return result
}

// resolvedChildName holds a resolved (non-template) name and namespace
// for one child resource.
type resolvedChildName struct {
	name       string
	namespace  string
	namespaced bool
}

// firstValue returns the first value from a map[string]interface{}.
// Returns an empty object with an empty status map when the map is empty,
// so that templates using {{ .children.deployment.status.readyReplicas }}
// resolve to "" via missingkey=zero rather than nil-pointer-dereference.
// Used to provide the singular shorthand: .children.deployment (instead of
// requiring .children.deployments["my-site-deployment"] in the common case).
func firstValue(m map[string]interface{}) interface{} {
	for _, v := range m {
		return v
	}
	// Resource not yet created or name resolution failed.
	// Return a placeholder so template field access is safe.
	// _placeholder:true lets noteExists() distinguish this from a real resource.
	return map[string]interface{}{
		"_placeholder": true,
		"status":       map[string]interface{}{},
	}
}

// mergeTemplates merges onCreate and onReconcile templates into one set.
// We read back resources declared in either block — both produce child resources.
// Resources declared in both are deduplicated by name after resolution.
func mergeTemplates(operatorBox orktypes.OperatorBoxConfig) orktypes.HookTemplates {
	t := orktypes.HookTemplates{}
	if operatorBox.OnCreate != nil {
		t.Deployments = append(t.Deployments, operatorBox.OnCreate.Deployments...)
		t.ReplicaSets = append(t.ReplicaSets, operatorBox.OnCreate.ReplicaSets...)
		t.StatefulSets = append(t.StatefulSets, operatorBox.OnCreate.StatefulSets...)
		t.Services = append(t.Services, operatorBox.OnCreate.Services...)
		t.Secrets = append(t.Secrets, operatorBox.OnCreate.Secrets...)
		t.ConfigMaps = append(t.ConfigMaps, operatorBox.OnCreate.ConfigMaps...)
		t.Jobs = append(t.Jobs, operatorBox.OnCreate.Jobs...)
		t.CronJobs = append(t.CronJobs, operatorBox.OnCreate.CronJobs...)
		t.Pods = append(t.Pods, operatorBox.OnCreate.Pods...)
		t.ServiceAccounts = append(t.ServiceAccounts, operatorBox.OnCreate.ServiceAccounts...)
		t.Namespaces = append(t.Namespaces, operatorBox.OnCreate.Namespaces...)
		t.PersistentVolumes = append(t.PersistentVolumes, operatorBox.OnCreate.PersistentVolumes...)
		t.PersistentVolumeClaims = append(t.PersistentVolumeClaims, operatorBox.OnCreate.PersistentVolumeClaims...)
		t.Ingresses = append(t.Ingresses, operatorBox.OnCreate.Ingresses...)
		t.HorizontalPodAutoscalers = append(t.HorizontalPodAutoscalers, operatorBox.OnCreate.HorizontalPodAutoscalers...)

		// Future when added
		t.StorageClasses = append(t.StorageClasses, operatorBox.OnCreate.StorageClasses...)
		t.ClusterRoles = append(t.ClusterRoles, operatorBox.OnCreate.ClusterRoles...)
		t.ClusterRoleBindings = append(t.ClusterRoleBindings, operatorBox.OnCreate.ClusterRoleBindings...)
		t.Roles = append(t.Roles, operatorBox.OnCreate.Roles...)
		t.RoleBindings = append(t.RoleBindings, operatorBox.OnCreate.RoleBindings...)
		t.LimitRanges = append(t.LimitRanges, operatorBox.OnCreate.LimitRanges...)
		t.ResourceQuotas = append(t.ResourceQuotas, operatorBox.OnCreate.ResourceQuotas...)
		t.PriorityClasses = append(t.PriorityClasses, operatorBox.OnCreate.PriorityClasses...)
		t.CustomResource = append(t.CustomResource, operatorBox.OnCreate.CustomResource...)
	}
	if operatorBox.OnReconcile != nil {
		t.Deployments = append(t.Deployments, operatorBox.OnReconcile.Deployments...)
		t.ReplicaSets = append(t.ReplicaSets, operatorBox.OnReconcile.ReplicaSets...)
		t.StatefulSets = append(t.StatefulSets, operatorBox.OnReconcile.StatefulSets...)
		t.Services = append(t.Services, operatorBox.OnReconcile.Services...)
		t.Secrets = append(t.Secrets, operatorBox.OnReconcile.Secrets...)
		t.ConfigMaps = append(t.ConfigMaps, operatorBox.OnReconcile.ConfigMaps...)
		t.Jobs = append(t.Jobs, operatorBox.OnReconcile.Jobs...)
		t.CronJobs = append(t.CronJobs, operatorBox.OnReconcile.CronJobs...)
		t.Pods = append(t.Pods, operatorBox.OnReconcile.Pods...)
		t.ServiceAccounts = append(t.ServiceAccounts, operatorBox.OnReconcile.ServiceAccounts...)
		t.Namespaces = append(t.Namespaces, operatorBox.OnReconcile.Namespaces...)
		t.PersistentVolumes = append(t.PersistentVolumes, operatorBox.OnReconcile.PersistentVolumes...)
		t.PersistentVolumeClaims = append(t.PersistentVolumeClaims, operatorBox.OnReconcile.PersistentVolumeClaims...)
		t.Ingresses = append(t.Ingresses, operatorBox.OnReconcile.Ingresses...)
		t.HorizontalPodAutoscalers = append(t.HorizontalPodAutoscalers, operatorBox.OnReconcile.HorizontalPodAutoscalers...)

		// Future when added
		t.StorageClasses = append(t.StorageClasses, operatorBox.OnReconcile.StorageClasses...)
		t.ClusterRoles = append(t.ClusterRoles, operatorBox.OnReconcile.ClusterRoles...)
		t.ClusterRoleBindings = append(t.ClusterRoleBindings, operatorBox.OnReconcile.ClusterRoleBindings...)
		t.Roles = append(t.Roles, operatorBox.OnReconcile.Roles...)
		t.RoleBindings = append(t.RoleBindings, operatorBox.OnReconcile.RoleBindings...)
		t.LimitRanges = append(t.LimitRanges, operatorBox.OnReconcile.LimitRanges...)
		t.ResourceQuotas = append(t.ResourceQuotas, operatorBox.OnReconcile.ResourceQuotas...)
		t.PriorityClasses = append(t.PriorityClasses, operatorBox.OnReconcile.PriorityClasses...)
		t.CustomResource = append(t.CustomResource, operatorBox.OnReconcile.CustomResource...)
	}
	return t
}

// ── Name resolvers — one per resource type ────────────────────────────────
// Each reads the name and namespace template from the template source and
// resolves them. The default name pattern mirrors OrkestraRegistry defaults.

// resolveName resolves a name and namespace from raw template strings.
// Empty or unresolvable names are skipped — they indicate a resource whose
// name depends on a field the CR does not have.
func resolveName(resolver *orktmpl.Resolver, rawName, rawNamespace string) (resolvedChildName, bool) {
	name, err := resolver.Resolve(rawName)
	if err != nil || name == "" {
		return resolvedChildName{}, false
	}
	ns, _ := resolver.Resolve(rawNamespace) // namespace resolution failure is non-fatal
	return resolvedChildName{name: name, namespace: ns}, true
}

func deploymentNames(resolver *orktmpl.Resolver, srcs []orktypes.DeploymentTemplateSource) []resolvedChildName {
	expanded := expandForEachDeployments(resolver, srcs)
	names := make([]resolvedChildName, 0, len(expanded))
	for _, s := range expanded {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

func statefulSetNames(resolver *orktmpl.Resolver, srcs []orktypes.StatefulSetTemplateSource) []resolvedChildName {
	expanded := expandForEachStatefulSets(resolver, srcs)
	names := make([]resolvedChildName, 0, len(expanded))
	for _, s := range expanded {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

func customResourceNames(resolver *orktmpl.Resolver, srcs []orktypes.CustomResourceTemplateSource) []resolvedChildName {
	expanded := expandForEachCustomResources(resolver, srcs)
	names := make([]resolvedChildName, 0, len(expanded))
	for _, s := range expanded {
		if n, ok := resolveName(resolver, s.Metadata.Name, s.Metadata.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

func serviceNames(resolver *orktmpl.Resolver, srcs []orktypes.ServiceTemplateSource) []resolvedChildName {
	expanded := expandForEachServices(resolver, srcs)
	names := make([]resolvedChildName, 0, len(expanded))
	for _, s := range expanded {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

func secretNames(resolver *orktmpl.Resolver, srcs []orktypes.SecretTemplateSource) []resolvedChildName {
	expanded := expandForEachSecrets(resolver, srcs)
	names := make([]resolvedChildName, 0, len(expanded))
	for _, s := range expanded {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

func configMapNames(resolver *orktmpl.Resolver, srcs []orktypes.ConfigMapTemplateSource) []resolvedChildName {
	expanded := expandForEachConfigMaps(resolver, srcs)
	names := make([]resolvedChildName, 0, len(expanded))
	for _, s := range expanded {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

// func jobNames(resolver *orktmpl.Resolver, srcs []orktypes.JobTemplateSource) []resolvedChildName {
// 	names := make([]resolvedChildName, 0, len(srcs))
// 	for _, s := range srcs {
// 		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
// 			names = append(names, n)
// 		}
// 	}
// 	return names
// }

// jobNames collects all job names from the template list.
// Conditions are NOT evaluated here — we read all declared children
// so status can reference any of them regardless of phase.
// Conditions only gate creation in run_jobs.go.
func jobNames(resolver *orktmpl.Resolver, srcs []orktypes.JobTemplateSource) []resolvedChildName {
	expanded := expandForEachJobs(resolver, srcs)
	var names []resolvedChildName
	for _, src := range expanded {
		name, err := resolver.Resolve(src.Name)
		if err != nil || name == "" {
			continue
		}
		ns, _ := resolver.Resolve(src.Namespace)
		names = append(names, resolvedChildName{name: name, namespace: ns})
	}
	return names
}

func cronJobNames(resolver *orktmpl.Resolver, srcs []orktypes.CronJobTemplateSource) []resolvedChildName {
	expanded := expandForEachCronJobs(resolver, srcs)
	names := make([]resolvedChildName, 0, len(expanded))
	for _, s := range expanded {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

func podNames(resolver *orktmpl.Resolver, srcs []orktypes.PodTemplateSource) []resolvedChildName {
	names := make([]resolvedChildName, 0, len(srcs))
	for _, s := range srcs {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

func serviceAccountNames(resolver *orktmpl.Resolver, srcs []orktypes.ServiceAccountTemplateSource) []resolvedChildName {
	expanded := expandForEachServiceAccounts(resolver, srcs)
	names := make([]resolvedChildName, 0, len(expanded))
	for _, s := range expanded {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

func namespaceNames(resolver *orktmpl.Resolver, srcs []orktypes.NamespaceTemplateSource) []resolvedChildName {
	expanded := expandForEachNamespaces(resolver, srcs)
	names := make([]resolvedChildName, 0, len(expanded))
	for _, s := range expanded {
		if n, ok := resolveName(resolver, s.Name, ""); ok {
			names = append(names, n)
		}
	}
	return names
}

func ingressNames(resolver *orktmpl.Resolver, srcs []orktypes.IngressTemplateSource) []resolvedChildName {
	expanded := expandForEachIngresses(resolver, srcs)
	names := make([]resolvedChildName, 0, len(expanded))
	for _, s := range expanded {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

func hpaNames(resolver *orktmpl.Resolver, srcs []orktypes.HPATemplateSource) []resolvedChildName {
	expanded := expandForEachHPAs(resolver, srcs)
	names := make([]resolvedChildName, 0, len(expanded))
	for _, s := range expanded {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

func pvcNames(resolver *orktmpl.Resolver, srcs []orktypes.PVCTemplateSource) []resolvedChildName {
	expanded := expandForEachPVCs(resolver, srcs)
	names := make([]resolvedChildName, 0, len(expanded))
	for _, s := range expanded {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

func pvNames(resolver *orktmpl.Resolver, srcs []orktypes.PVTemplateSource) []resolvedChildName {
	expanded := expandForEachPVs(resolver, srcs)
	names := make([]resolvedChildName, 0, len(expanded))
	for _, s := range expanded {
		// PVs are cluster-scoped — namespace is always empty.
		if n, ok := resolveName(resolver, s.Name, ""); ok {
			names = append(names, n)
		}
	}
	return names
}

func replicaSetNames(resolver *orktmpl.Resolver, srcs []orktypes.ReplicaSetTemplateSource) []resolvedChildName {
	expanded := expandForEachReplicaSets(resolver, srcs)
	names := make([]resolvedChildName, 0, len(expanded))
	for _, s := range expanded {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

// ── Pod enrichment ────────────────────────────────────────────────────────
// enrichGroupWithPods embeds pod summaries under "_pods" for every resource
// in the group. A no-op when pods enrichment is not enabled on the CRD.
//
// ownerKind filters pods to only those whose immediate ownerReference matches
// the expected controller kind — e.g. "ReplicaSet" for Deployments, "StatefulSet"
// for StatefulSets. This prevents job pods from appearing in a Deployment's
// pod list when both share the same orkestra-owner label selector.
func enrichGroupWithPods(ctx context.Context, kube kubeclient.KubeClient, m map[string]interface{}, crd orktypes.CRDEntry, ownerKind string) {
	if !crd.ShouldEnrich("pods") {
		return
	}
	for _, v := range m {
		obj, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		meta, _ := obj["metadata"].(map[string]interface{})
		ns := ""
		if meta != nil {
			ns, _ = meta["namespace"].(string)
		}
		enrichWithPods(ctx, kube, ns, obj, ownerKind)
	}
}

// enrichWithPods lists pods matching the resource's spec.selector.matchLabels,
// filters to those owned by a controller of ownerKind, and embeds summaries
// as _pods in the resource map.
func enrichWithPods(ctx context.Context, kube kubeclient.KubeClient, ns string, obj map[string]interface{}, ownerKind string) {
	selector := podLabelSelector(obj)
	if selector == "" {
		return
	}
	list, err := kube.DynamicClient().
		Resource(podGVR).
		Namespace(ns).
		List(ctx, metav1.ListOptions{
			LabelSelector:   selector,
			ResourceVersion: "0",
		})
	if err != nil || list == nil {
		return
	}
	pods := make([]interface{}, 0, len(list.Items))
	for i := range list.Items {
		if !podOwnedBy(list.Items[i].Object, ownerKind) {
			continue
		}
		pods = append(pods, buildPodSummary(list.Items[i].Object))
	}
	obj["_pods"] = pods
}

// podOwnedBy returns true when any ownerReference on the pod has the given kind.
func podOwnedBy(obj map[string]interface{}, kind string) bool {
	meta, _ := obj["metadata"].(map[string]interface{})
	if meta == nil {
		return false
	}
	ownerRefs, _ := meta["ownerReferences"].([]interface{})
	for _, ref := range ownerRefs {
		r, _ := ref.(map[string]interface{})
		if r == nil {
			continue
		}
		if k, _ := r["kind"].(string); k == kind {
			return true
		}
	}
	return false
}

// podLabelSelector builds a comma-separated label selector from spec.selector.matchLabels.
// Returns "" when the field is absent or empty — no selector means no list.
func podLabelSelector(obj map[string]interface{}) string {
	spec, _ := obj["spec"].(map[string]interface{})
	if spec == nil {
		return ""
	}
	sel, _ := spec["selector"].(map[string]interface{})
	if sel == nil {
		return ""
	}
	matchLabels, _ := sel["matchLabels"].(map[string]interface{})
	if len(matchLabels) == 0 {
		return ""
	}
	parts := make([]string, 0, len(matchLabels))
	for k, v := range matchLabels {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, ",")
}

// buildPodSummary extracts the fields note functions navigate from _pods.
func buildPodSummary(obj map[string]interface{}) map[string]interface{} {
	meta, _ := obj["metadata"].(map[string]interface{})
	spec, _ := obj["spec"].(map[string]interface{})
	status, _ := obj["status"].(map[string]interface{})

	name := ""
	if meta != nil {
		name, _ = meta["name"].(string)
	}
	nodeName := ""
	if spec != nil {
		nodeName, _ = spec["nodeName"].(string)
	}
	podIP, phase := "", ""
	if status != nil {
		podIP, _ = status["podIP"].(string)
		phase, _ = status["phase"].(string)
	}
	return map[string]interface{}{
		"name":         name,
		"ip":           podIP,
		"phase":        phase,
		"ready":        isPodReady(status),
		"node":         nodeName,
		"restartCount": podTotalRestarts(status),
		"ordinal":      podOrdinal(name),
		"exitCode":     podExitCode(status),
		"containers":   buildContainerSummaries(status),
	}
}

// buildContainerSummaries extracts per-container state from pod status.containerStatuses.
// Each entry: {name, image, state, reason, ready, restartCount}
//
// state is one of: "Running", "Waiting", "Terminated", ""
// reason is state.waiting.reason or state.terminated.reason — e.g. "CrashLoopBackOff"
func buildContainerSummaries(status map[string]interface{}) []interface{} {
	if status == nil {
		return nil
	}
	containerStatuses, _ := status["containerStatuses"].([]interface{})
	if len(containerStatuses) == 0 {
		return nil
	}
	result := make([]interface{}, 0, len(containerStatuses))
	for _, cs := range containerStatuses {
		csMap, _ := cs.(map[string]interface{})
		if csMap == nil {
			continue
		}
		name, _ := csMap["name"].(string)
		image, _ := csMap["image"].(string)
		ready, _ := csMap["ready"].(bool)
		restartCount := int64(0)
		switch v := csMap["restartCount"].(type) {
		case int64:
			restartCount = v
		case float64:
			restartCount = int64(v)
		case int:
			restartCount = int64(v)
		}
		state, reason := containerStateAndReason(csMap)
		result = append(result, map[string]interface{}{
			"name":         name,
			"image":        image,
			"state":        state,
			"reason":       reason,
			"ready":        ready,
			"restartCount": restartCount,
		})
	}
	return result
}

// containerStateAndReason extracts the state name and reason from a containerStatus map.
func containerStateAndReason(csMap map[string]interface{}) (state, reason string) {
	stateMap, _ := csMap["state"].(map[string]interface{})
	if stateMap == nil {
		return "", ""
	}
	if _, ok := stateMap["running"]; ok {
		return "Running", ""
	}
	if w, ok := stateMap["waiting"].(map[string]interface{}); ok {
		r, _ := w["reason"].(string)
		return "Waiting", r
	}
	if t, ok := stateMap["terminated"].(map[string]interface{}); ok {
		r, _ := t["reason"].(string)
		return "Terminated", r
	}
	return "", ""
}

// podExitCode returns the exit code from the first terminated container.
// Returns -1 when no container has terminated.
func podExitCode(status map[string]interface{}) int64 {
	if status == nil {
		return -1
	}
	containerStatuses, _ := status["containerStatuses"].([]interface{})
	for _, cs := range containerStatuses {
		csMap, _ := cs.(map[string]interface{})
		if csMap == nil {
			continue
		}
		state, _ := csMap["state"].(map[string]interface{})
		if state == nil {
			continue
		}
		terminated, _ := state["terminated"].(map[string]interface{})
		if terminated == nil {
			continue
		}
		switch v := terminated["exitCode"].(type) {
		case int64:
			return v
		case float64:
			return int64(v)
		case int:
			return int64(v)
		}
	}
	return -1
}

// podOrdinal extracts the ordinal from a StatefulSet pod name (<name>-<ordinal>).
// Returns -1 for non-ordinal pods (Deployment, Job, etc.).
func podOrdinal(name string) int64 {
	parts := strings.Split(name, "-")
	if len(parts) < 2 {
		return -1
	}
	n, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return -1
	}
	return n
}

func isPodReady(status map[string]interface{}) bool {
	if status == nil {
		return false
	}
	conditions, _ := status["conditions"].([]interface{})
	for _, c := range conditions {
		cond, _ := c.(map[string]interface{})
		if cond == nil {
			continue
		}
		if t, _ := cond["type"].(string); t == "Ready" {
			s, _ := cond["status"].(string)
			return s == "True"
		}
	}
	return false
}

func podTotalRestarts(status map[string]interface{}) int64 {
	if status == nil {
		return 0
	}
	containerStatuses, _ := status["containerStatuses"].([]interface{})
	var total int64
	for _, cs := range containerStatuses {
		csMap, _ := cs.(map[string]interface{})
		if csMap == nil {
			continue
		}
		switch v := csMap["restartCount"].(type) {
		case int64:
			total += v
		case float64:
			total += int64(v)
		case int:
			total += int64(v)
		}
	}
	return total
}

// ── Endpoint enrichment ───────────────────────────────────────────────────
// enrichGroupWithEndpoints embeds _endpoints into each service map.
// Reads the EndpointSlice for each service and embeds IP:port pairs.
// A no-op when endpoint enrichment is not enabled on the CRD.
func enrichGroupWithEndpoints(ctx context.Context, kube kubeclient.KubeClient, m map[string]interface{}, crd orktypes.CRDEntry) {
	if !crd.ShouldEnrich("endpoints") {
		return
	}
	for _, v := range m {
		obj, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		meta, _ := obj["metadata"].(map[string]interface{})
		ns, svcName := "", ""
		if meta != nil {
			ns, _ = meta["namespace"].(string)
			svcName, _ = meta["name"].(string)
		}
		if svcName == "" {
			continue
		}
		enrichServiceWithEndpoints(ctx, kube, ns, svcName, obj)
	}
}

// enrichServiceWithEndpoints fetches the EndpointSlice for the named service
// and embeds a list of {ip, port, ready} maps under _endpoints.
func enrichServiceWithEndpoints(ctx context.Context, kube kubeclient.KubeClient, ns, svcName string, obj map[string]interface{}) {
	list, err := kube.DynamicClient().
		Resource(endpointSliceGVR).
		Namespace(ns).
		List(ctx, metav1.ListOptions{
			LabelSelector:   fmt.Sprintf("kubernetes.io/service-name=%s", svcName),
			Limit:           1,
			ResourceVersion: "0",
		})
	if err != nil || list == nil || len(list.Items) == 0 {
		return
	}
	endpoints := extractEndpointEntries(list.Items[0].Object)
	if len(endpoints) > 0 {
		obj["_endpoints"] = endpoints
	}
}

// extractEndpointEntries builds the flat _endpoints list from an EndpointSlice object.
func extractEndpointEntries(esObj map[string]interface{}) []interface{} {
	ports := extractSlicePorts(esObj)
	eps, _ := esObj["endpoints"].([]interface{})

	var result []interface{}
	for _, e := range eps {
		em, _ := e.(map[string]interface{})
		if em == nil {
			continue
		}
		cond, _ := em["conditions"].(map[string]interface{})
		ready, _ := cond["ready"].(bool)

		addrs, _ := em["addresses"].([]interface{})
		for _, addr := range addrs {
			ip, _ := addr.(string)
			if ip == "" {
				continue
			}
			for _, port := range ports {
				result = append(result, map[string]interface{}{
					"ip":    ip,
					"port":  port,
					"ready": ready,
				})
			}
		}
	}
	return result
}

// extractSlicePorts returns the port numbers from an EndpointSlice's ports array.
func extractSlicePorts(esObj map[string]interface{}) []int64 {
	portObjs, _ := esObj["ports"].([]interface{})
	ports := make([]int64, 0, len(portObjs))
	for _, p := range portObjs {
		pm, _ := p.(map[string]interface{})
		if pm == nil {
			continue
		}
		switch v := pm["port"].(type) {
		case int64:
			ports = append(ports, v)
		case float64:
			ports = append(ports, int64(v))
		case int:
			ports = append(ports, int64(v))
		}
	}
	return ports
}

// ── Warning event enrichment ──────────────────────────────────────────────────
// enrichGroupWithWarnings embeds warning events under "_warnings" for every
// resource in the group. A no-op when events enrichment is not enabled on the CRD.
//
// kind is the Kubernetes Kind used to scope the event field selector
// (involvedObject.kind). Pass "" to skip the kind filter — used for custom
// resources whose exact kind is not known statically.
func enrichGroupWithWarnings(ctx context.Context, kube kubeclient.KubeClient, m map[string]interface{}, crd orktypes.CRDEntry, kind string) {
	if !crd.ShouldEnrich("events") {
		return
	}
	for _, v := range m {
		obj, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		meta, _ := obj["metadata"].(map[string]interface{})
		ns, name := "", ""
		if meta != nil {
			ns, _ = meta["namespace"].(string)
			name, _ = meta["name"].(string)
		}
		if name == "" {
			continue
		}
		enrichWithWarnings(ctx, kube, ns, name, kind, obj)
	}
}

// enrichWithWarnings fetches Warning events scoped to the named resource and
// embeds a list of {reason, message, count, lastTimestamp} maps under _warnings.
//
// Field selector: involvedObject.name=<name>,type=Warning
// When kind is non-empty: involvedObject.kind=<kind> is appended.
// ResourceVersion "0" serves from the informer cache — no quorum read.
func enrichWithWarnings(ctx context.Context, kube kubeclient.KubeClient, ns, name, kind string, obj map[string]interface{}) {
	selector := fmt.Sprintf("involvedObject.name=%s,type=Warning", name)
	if kind != "" {
		selector += fmt.Sprintf(",involvedObject.kind=%s", kind)
	}
	if ns == "" {
		return
	}
	list, err := kube.DynamicClient().
		Resource(eventGVR).
		Namespace(ns).
		List(ctx, metav1.ListOptions{
			FieldSelector:   selector,
			ResourceVersion: "0",
		})
	if err != nil || list == nil || len(list.Items) == 0 {
		return
	}
	warnings := make([]interface{}, 0, len(list.Items))
	for i := range list.Items {
		warnings = append(warnings, buildWarningSummary(list.Items[i].Object))
	}
	obj["_warnings"] = warnings
}

// buildWarningSummary extracts the fields note functions navigate from _warnings.
func buildWarningSummary(obj map[string]interface{}) map[string]interface{} {
	reason, _ := obj["reason"].(string)
	message, _ := obj["message"].(string)
	lastTimestamp, _ := obj["lastTimestamp"].(string)
	count := int64(0)
	switch v := obj["count"].(type) {
	case int64:
		count = v
	case float64:
		count = int64(v)
	case int:
		count = int64(v)
	}
	return map[string]interface{}{
		"reason":        reason,
		"message":       message,
		"count":         count,
		"lastTimestamp": lastTimestamp,
	}
}

// ── PVC enrichment ────────────────────────────────────────────────────────────
// enrichGroupWithPV embeds the bound PV object under "_pv" for each PVC in
// the group. A no-op when pvc enrichment is not enabled on the CRD.
// The PV name comes from spec.volumeName on the PVC, set by Kubernetes once bound.
func enrichGroupWithPV(ctx context.Context, kube kubeclient.KubeClient, m map[string]interface{}, crd orktypes.CRDEntry) {
	if !crd.ShouldEnrich("pvc") {
		return
	}
	for _, v := range m {
		obj, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		spec, _ := obj["spec"].(map[string]interface{})
		if spec == nil {
			continue
		}
		volName, _ := spec["volumeName"].(string)
		if volName == "" {
			continue
		}
		u, err := kube.DynamicClient().
			Resource(pvGVR).
			Get(ctx, volName, metav1.GetOptions{})
		if err != nil {
			continue
		}
		pvObj := u.Object
		if s, ok := pvObj["status"]; !ok || s == nil {
			pvObj["status"] = map[string]interface{}{}
		}
		obj["_pv"] = pvObj
	}
}
