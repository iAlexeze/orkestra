// pkg/reconciler/children.go
package reconciler

import (
	"context"
	"fmt"
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
	kube *kubeclient.Kubeclient,
	obj domain.Object,
	resolver *orktmpl.Resolver,
	operatorBox orktypes.OperatorBoxConfig,
) map[string]interface{} {
	children := map[string]interface{}{}

	// Collect all template sources across onCreate and onReconcile.
	// We only read resources that are declared — not all resources in the namespace.
	templates := mergeTemplates(operatorBox)

	// ── Deployments ───────────────────────────────────────────────────────
	if len(templates.Deployments) > 0 {
		dNames := deploymentNames(resolver, templates.Deployments)
		m := readResourceGroup(ctx, kube, obj, resolver, deploymentGVR, dNames)
		children["deployments"] = m
		children["deployment"] = firstValue(m) // singular shorthand
	}

	// ── Services ──────────────────────────────────────────────────────────
	if len(templates.Services) > 0 {
		svcNames := serviceNames(resolver, templates.Services)
		m := readResourceGroup(ctx, kube, obj, resolver, serviceGVR, svcNames)
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
		children["secrets"] = m
		children["secret"] = firstValue(m)
	}

	// ── ConfigMaps ────────────────────────────────────────────────────────
	if len(templates.ConfigMaps) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, configMapGVR,
			configMapNames(resolver, templates.ConfigMaps))
		children["configmaps"] = m
		children["configmap"] = firstValue(m)
	}

	// ── Jobs ──────────────────────────────────────────────────────────────
	if len(templates.Jobs) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, jobGVR,
			jobNames(resolver, templates.Jobs))
		children["jobs"] = m
		children["job"] = firstValue(m)
	}

	// ── CronJobs ──────────────────────────────────────────────────────────
	if len(templates.CronJobs) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, cronJobGVR,
			cronJobNames(resolver, templates.CronJobs))
		children["cronjobs"] = m
		children["cronjob"] = firstValue(m)
	}

	// ── Pods ──────────────────────────────────────────────────────────────
	if len(templates.Pods) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, podGVR,
			podNames(resolver, templates.Pods))
		children["pods"] = m
		children["pod"] = firstValue(m)
	}

	// ── ServiceAccounts ───────────────────────────────────────────────────
	if len(templates.ServiceAccounts) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, serviceAccountGVR,
			serviceAccountNames(resolver, templates.ServiceAccounts))
		children["serviceaccounts"] = m
		children["serviceaccount"] = firstValue(m)
	}

	// ── Namespaces ───────────────────────────────────────────────────
	if len(templates.Namespaces) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, namespaceGVR,
			namespaceNames(resolver, templates.Namespaces))
		children["namespaces"] = m
		children["namespace"] = firstValue(m)
	}

	return children
}

// readEndpointSlicesForServices lists the EndpointSlice for each declared Service
// using the kubernetes.io/service-name label. The result is keyed by service name
// so templates can reference {{ .children.endpointslice }} for single-service katalogs.
func readEndpointSlicesForServices(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
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
	kube *kubeclient.Kubeclient,
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

		// Future when added
		t.StorageClasses = append(t.StorageClasses, operatorBox.OnCreate.StorageClasses...)
		t.ClusterRoles = append(t.ClusterRoles, operatorBox.OnCreate.ClusterRoles...)
		t.ClusterRoleBindings = append(t.ClusterRoleBindings, operatorBox.OnCreate.ClusterRoleBindings...)
		t.Roles = append(t.Roles, operatorBox.OnCreate.Roles...)
		t.RoleBindings = append(t.RoleBindings, operatorBox.OnCreate.RoleBindings...)
		t.LimitRanges = append(t.LimitRanges, operatorBox.OnCreate.LimitRanges...)
		t.ResourceQuotas = append(t.ResourceQuotas, operatorBox.OnCreate.ResourceQuotas...)
		t.PriorityClasses = append(t.PriorityClasses, operatorBox.OnCreate.PriorityClasses...)
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

		// Future when added
		t.StorageClasses = append(t.StorageClasses, operatorBox.OnReconcile.StorageClasses...)
		t.ClusterRoles = append(t.ClusterRoles, operatorBox.OnReconcile.ClusterRoles...)
		t.ClusterRoleBindings = append(t.ClusterRoleBindings, operatorBox.OnReconcile.ClusterRoleBindings...)
		t.Roles = append(t.Roles, operatorBox.OnReconcile.Roles...)
		t.RoleBindings = append(t.RoleBindings, operatorBox.OnReconcile.RoleBindings...)
		t.LimitRanges = append(t.LimitRanges, operatorBox.OnReconcile.LimitRanges...)
		t.ResourceQuotas = append(t.ResourceQuotas, operatorBox.OnReconcile.ResourceQuotas...)
		t.PriorityClasses = append(t.PriorityClasses, operatorBox.OnReconcile.PriorityClasses...)
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
