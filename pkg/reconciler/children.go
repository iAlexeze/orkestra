// pkg/reconciler/children.go
package reconciler

import (
	"context"
	"fmt"
	"strings"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ── Child resource GVRs ───────────────────────────────────────────────────
// These are the GVRs for every resource type the OrkestraRegistry creates.
// Used to read back child resources after reconcile completes.
// When you add a new resource, make it avaialble here to be read by orkestra

var (
	deploymentGVR     = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	serviceGVR        = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	secretGVR         = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	configMapGVR      = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	jobGVR            = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
	cronJobGVR        = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}
	podGVR            = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	serviceAccountGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}
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
	kube *kubeclient.Kubeclient,
	obj domain.Object,
	resolver *orktmpl.Resolver,
	rc orktypes.ReconcilerConfig,
) map[string]interface{} {
	children := map[string]interface{}{}

	// Collect all template sources across onCreate and onReconcile.
	// We only read resources that are declared — not all resources in the namespace.
	templates := mergeTemplates(rc)

	// ── Deployments ───────────────────────────────────────────────────────
	if len(templates.Deployments) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, deploymentGVR,
			deploymentNames(resolver, templates.Deployments))
		children["deployments"] = m
		children["deployment"] = firstValue(m) // singular shorthand
	}

	// ── Services ──────────────────────────────────────────────────────────
	if len(templates.Services) > 0 {
		m := readResourceGroup(ctx, kube, obj, resolver, serviceGVR,
			serviceNames(resolver, templates.Services))
		children["services"] = m
		children["service"] = firstValue(m)
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

	return children
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

		u, err := kube.DynamicClient().
			Resource(gvr).
			Namespace(ns).
			Get(ctx, child.name, metav1.GetOptions{})

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
	name      string
	namespace string
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
	return map[string]interface{}{
		"status": map[string]interface{}{},
	}
}

// mergeTemplates merges onCreate and onReconcile templates into one set.
// We read back resources declared in either block — both produce child resources.
// Resources declared in both are deduplicated by name after resolution.
func mergeTemplates(rc orktypes.ReconcilerConfig) orktypes.HookTemplates {
	t := orktypes.HookTemplates{}
	if rc.OnCreate != nil {
		t.Deployments = append(t.Deployments, rc.OnCreate.Deployments...)
		t.Services = append(t.Services, rc.OnCreate.Services...)
		t.Secrets = append(t.Secrets, rc.OnCreate.Secrets...)
		t.ConfigMaps = append(t.ConfigMaps, rc.OnCreate.ConfigMaps...)
		t.Jobs = append(t.Jobs, rc.OnCreate.Jobs...)
		t.CronJobs = append(t.CronJobs, rc.OnCreate.CronJobs...)
		t.Pods = append(t.Pods, rc.OnCreate.Pods...)
		t.ServiceAccounts = append(t.ServiceAccounts, rc.OnCreate.ServiceAccounts...)
	}
	if rc.OnReconcile != nil {
		t.Deployments = append(t.Deployments, rc.OnReconcile.Deployments...)
		t.Services = append(t.Services, rc.OnReconcile.Services...)
		t.Secrets = append(t.Secrets, rc.OnReconcile.Secrets...)
		t.ConfigMaps = append(t.ConfigMaps, rc.OnReconcile.ConfigMaps...)
		t.Jobs = append(t.Jobs, rc.OnReconcile.Jobs...)
		t.CronJobs = append(t.CronJobs, rc.OnReconcile.CronJobs...)
		t.Pods = append(t.Pods, rc.OnReconcile.Pods...)
		t.ServiceAccounts = append(t.ServiceAccounts, rc.OnReconcile.ServiceAccounts...)
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
	names := make([]resolvedChildName, 0, len(srcs))
	for _, s := range srcs {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

func serviceNames(resolver *orktmpl.Resolver, srcs []orktypes.ServiceTemplateSource) []resolvedChildName {
	names := make([]resolvedChildName, 0, len(srcs))
	for _, s := range srcs {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

func secretNames(resolver *orktmpl.Resolver, srcs []orktypes.SecretTemplateSource) []resolvedChildName {
	names := make([]resolvedChildName, 0, len(srcs))
	for _, s := range srcs {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}

func configMapNames(resolver *orktmpl.Resolver, srcs []orktypes.ConfigMapTemplateSource) []resolvedChildName {
	names := make([]resolvedChildName, 0, len(srcs))
	for _, s := range srcs {
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
	var names []resolvedChildName
	for _, src := range srcs {
		// ← no condition check here
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
	names := make([]resolvedChildName, 0, len(srcs))
	for _, s := range srcs {
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
	names := make([]resolvedChildName, 0, len(srcs))
	for _, s := range srcs {
		if n, ok := resolveName(resolver, s.Name, s.Namespace); ok {
			names = append(names, n)
		}
	}
	return names
}
