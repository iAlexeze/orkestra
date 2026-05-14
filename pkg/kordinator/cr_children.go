// pkg/kordinator/cr_children.go
//
// Child resource fetching and readiness for the CR detail endpoint.
//
// Fixes vs original:
//
// Fix 1 — Multiple children per kind.
//
//	map[string]ChildSummary only held one child per kind (last-writer-wins).
//	Two Deployments (payments-service-with-secret, with-configmap) collapsed
//	into one. Now returns map[string]interface{} where the value is either a
//	single ChildSummary (when one child) or []ChildSummary (when multiple).
//	Backward-compatible with the UI: single children still serialize as objects.
//
// Fix 2 — Parent CR readiness uses builtInRegistry directly.
//
//	crd.IsStatuslessType() had a GVK string format mismatch —
//	k8s uses "/v1, Kind=ConfigMap", registry used "v1/ConfigMap".
//	extractParentReady now calls katalog.BuiltInMeta(parentKind) by plain
//	Kind string — no format dependency.
//
// Fix 3 — Annotation-based readiness for statusless CRDs.
//
//	ConfigMap has no /status subresource so Orkestra writes phase to annotations.
//	extractParentReady reads orkestra.io/phase annotation when the CR is statusless.
//
// Fix 4 — Dynamic GVR filtering + TTL cache.
//
//	readChildrenForEndpoint previously queried all 22 known child GVR types on
//	every HTTP request. Now it only queries the types declared in the Katalog's
//	onCreate/onReconcile templates (using the OperatorBoxConfig). Results are
//	cached for childrenCacheTTL per CR so repeated UI polls reuse in-memory data.
package kordinator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// childrenCacheTTL controls how long endpoint children results are served from
// memory before re-fetching from the API server. Matched to a typical resync
// period — data is at most this stale, which is acceptable for observability.
const childrenCacheTTL = 30 * time.Second

type cachedChildren struct {
	data map[string]interface{}
	at   time.Time
}

// endpointChildrenCache stores endpoint-ready children per CR, keyed by "namespace/name".
// Eliminates repeated concurrent LIST calls when the UI polls the detail endpoint.
var endpointChildrenCache sync.Map

// childGVREntry carries the GVR and readiness metadata for one child kind.
type childGVREntry struct {
	GVR        schema.GroupVersionResource
	Key        string // lowercase kind — matches .children.<key>
	Statusless bool   // from builtInRegistry — not inherited from parent CRD
}

// knownChildGVRs is built once at startup from builtInRegistry.
// Statusless is per-child-kind, independent of the parent CRD's statusless flag.
var knownChildGVRs = func() []childGVREntry {
	defs := katalog.ChildGVRs()

	out := make([]childGVREntry, len(defs))
	for i, d := range defs {
		m := katalog.BuiltInMeta(d.Key)
		out[i] = childGVREntry{
			GVR:        d.GVR,
			Key:        d.Key,
			Statusless: m.Statusless || m.SkipStatusSubresource,
		}
	}
	return out
}()

// ─────────────────────────────────────────────────────────────────────────────
// Children fetch
// ─────────────────────────────────────────────────────────────────────────────

// readChildrenForEndpoint fetches all child resources for a CR.
// Returns map[string]interface{} where:
//   - single child  → ChildSummary         (UI renders as object)
//   - multiple      → []ChildSummary        (UI renders as array)
//   - none          → key absent from map
//
// Only queries the resource types declared in rc (onCreate/onReconcile templates).
// Results are cached for childrenCacheTTL to avoid repeated concurrent LISTs.
func readChildrenForEndpoint(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	owner map[string]interface{},
	rc orktypes.OperatorBoxConfig,
) map[string]interface{} {
	if kube == nil {
		return map[string]interface{}{}
	}

	ns := metaField(owner, "namespace")
	name := metaField(owner, "name")
	cacheKey := ns + "/" + name

	// Serve from cache if still fresh.
	if cached, ok := endpointChildrenCache.Load(cacheKey); ok {
		c := cached.(cachedChildren)
		if time.Since(c.at) < childrenCacheTTL {
			return c.data
		}
	}

	fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	labelSelector := fmt.Sprintf("orkestra-owner=%s", name)

	// Only query types declared in the Katalog — avoids firing 22 concurrent
	// LISTs when the CR uses 2-3 resource types. Falls back to all types when
	// no config is provided (e.g. Katalog with Go hooks instead of templates).
	gvrs := relevantGVRsFromConfig(rc)

	type result struct {
		key   string
		items []ChildSummary
	}

	ch := make(chan result, len(gvrs))

	for _, entry := range gvrs {
		entry := entry
		go func() {
			ch <- result{key: entry.Key, items: fetchChildKind(fetchCtx, kube, ns, labelSelector, entry)}
		}()
	}

	children := make(map[string]interface{}, len(gvrs))
	for range gvrs {
		select {
		case r := <-ch:
			switch len(r.items) {
			case 0:
				// nothing — omit key
			case 1:
				children[r.key] = r.items[0] // flat object — backward compatible
			default:
				children[r.key] = r.items // array when multiple of same kind
			}
		case <-fetchCtx.Done():
			endpointChildrenCache.Store(cacheKey, cachedChildren{data: children, at: time.Now()})
			return children
		}
	}

	// Custom resources have dynamic GVRs — fetch separately from standard types.
	customSrcs := mergeCustomResourceSrcs(rc)
	if len(customSrcs) > 0 {
		customs := fetchCustomChildren(fetchCtx, kube, ns, labelSelector, customSrcs)
		switch len(customs) {
		case 1:
			children["custom"] = customs[0]
		default:
			if len(customs) > 1 {
				children["custom"] = customs
			}
		}
	}

	endpointChildrenCache.Store(cacheKey, cachedChildren{data: children, at: time.Now()})
	return children
}

// InvalidateChildrenCache removes the cached children for a CR.
// Called by the reconciler after a successful reconcile so the next endpoint
// request reflects the latest state immediately.
func InvalidateChildrenCache(namespace, name string) {
	endpointChildrenCache.Delete(namespace + "/" + name)
}

// ─────────────────────────────────────────────────────────────────────────────
// GVR filtering
// ─────────────────────────────────────────────────────────────────────────────

// relevantGVRsFromConfig returns only the GVR entries for resource types
// declared in the OperatorBoxConfig's onCreate/onReconcile templates.
// When no config is provided (empty templates), all known GVRs are returned
// so Katalogs using Go hooks still show their children.
func relevantGVRsFromConfig(rc orktypes.OperatorBoxConfig) []childGVREntry {
	needed := relevantKeys(rc)
	if len(needed) == 0 {
		return knownChildGVRs
	}
	result := make([]childGVREntry, 0, len(needed))
	for _, e := range knownChildGVRs {
		if needed[e.Key] {
			result = append(result, e)
		}
	}
	return result
}

// relevantKeys extracts the set of child resource type keys declared in
// the OperatorBoxConfig's onCreate/onReconcile templates.
func relevantKeys(rc orktypes.OperatorBoxConfig) map[string]bool {
	needed := map[string]bool{}
	addKeys := func(t *orktypes.HookTemplates) {
		if t == nil {
			return
		}
		if len(t.Deployments) > 0 {
			needed["deployment"] = true
		}
		if len(t.ReplicaSets) > 0 {
			needed["replicaset"] = true
		}
		if len(t.StatefulSets) > 0 {
			needed["statefulset"] = true
		}
		if len(t.Services) > 0 {
			needed["service"] = true
		}
		if len(t.Secrets) > 0 {
			needed["secret"] = true
		}
		if len(t.ConfigMaps) > 0 {
			needed["configmap"] = true
		}
		if len(t.Jobs) > 0 {
			needed["job"] = true
		}
		if len(t.CronJobs) > 0 {
			needed["cronjob"] = true
		}
		if len(t.Pods) > 0 {
			needed["pod"] = true
		}
		if len(t.ServiceAccounts) > 0 {
			needed["serviceaccount"] = true
		}
		if len(t.Namespaces) > 0 {
			needed["namespace"] = true
		}
		if len(t.Ingresses) > 0 {
			needed["ingress"] = true
		}
		if len(t.PersistentVolumes) > 0 {
			needed["persistentvolume"] = true
		}
		if len(t.PersistentVolumeClaims) > 0 {
			needed["persistentvolumeclaim"] = true
		}
		if len(t.Roles) > 0 {
			needed["role"] = true
		}
		if len(t.RoleBindings) > 0 {
			needed["rolebinding"] = true
		}
	}
	addKeys(rc.OnCreate)
	addKeys(rc.OnReconcile)
	return needed
}

// mergeCustomResourceSrcs collects all custom resource template entries from
// both onCreate and onReconcile, deduplicating by APIVersion+Kind so each
// unique GVR is only queried once.
func mergeCustomResourceSrcs(rc orktypes.OperatorBoxConfig) []orktypes.CustomResourceTemplateSource {
	type key struct{ apiVersion, kind string }
	seen := map[key]bool{}
	var out []orktypes.CustomResourceTemplateSource
	add := func(t *orktypes.HookTemplates) {
		if t == nil {
			return
		}
		for _, src := range t.CustomResource {
			if src.APIVersion == "" || src.Kind == "" {
				continue
			}
			k := key{src.APIVersion, src.Kind}
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, src)
		}
	}
	add(rc.OnCreate)
	add(rc.OnReconcile)
	return out
}

// fetchCustomChildren lists all custom resources declared in srcs that are
// owned by the CR (matched by orkestra-owner label). Each unique APIVersion/Kind
// is resolved to a GVR via the REST mapper and listed once.
func fetchCustomChildren(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	ns, labelSelector string,
	srcs []orktypes.CustomResourceTemplateSource,
) []ChildSummary {
	var result []ChildSummary
	for i := range srcs {
		src := &srcs[i]
		gvr, err := src.ResolveGVR(kube.Mapper())
		if err != nil {
			continue
		}
		resource := kube.DynamicClient().Resource(gvr)
		opts := metav1.ListOptions{
			LabelSelector:   labelSelector,
			ResourceVersion: "0",
		}
		var list *unstructured.UnstructuredList
		if src.IsNamespaced() && ns != "" {
			list, err = resource.Namespace(ns).List(ctx, opts)
		} else {
			list, err = resource.List(ctx, opts)
		}
		if err != nil || len(list.Items) == 0 {
			continue
		}
		for _, obj := range list.Items {
			status, _ := obj.Object["status"].(map[string]interface{})
			if status == nil {
				status = map[string]interface{}{}
			}
			result = append(result, ChildSummary{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
				Kind:      src.Kind,
				Status:    status,
				Ready:     isChildReady(obj.Object, false),
			})
		}
	}
	return result
}

// fetchChildKind lists all resources of one kind owned by the CR.
func fetchChildKind(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	ns, labelSelector string,
	entry childGVREntry,
) []ChildSummary {
	resource := kube.DynamicClient().Resource(entry.GVR)

	opts := metav1.ListOptions{
		LabelSelector:   labelSelector,
		Limit:           listOptionsLimit,
		ResourceVersion: "0",
	}

	var list *unstructured.UnstructuredList
	var err error

	if ns != "" {
		list, err = resource.Namespace(ns).List(ctx, opts)
	} else {
		list, err = resource.List(ctx, opts)
	}

	if err != nil || len(list.Items) == 0 {
		return nil
	}

	result := make([]ChildSummary, 0, len(list.Items))
	for _, obj := range list.Items {
		status, _ := obj.Object["status"].(map[string]interface{})
		if status == nil {
			status = map[string]interface{}{}
		}
		kind := obj.GetKind()
		if kind == "" {
			kind = strings.ToUpper(entry.Key[:1]) + entry.Key[1:]
		}
		result = append(result, ChildSummary{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
			Kind:      kind,
			Status:    status,
			Ready:     isChildReady(obj.Object, entry.Statusless),
		})
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Parent CR readiness — extractParentReady
// ─────────────────────────────────────────────────────────────────────────────

// extractParentReady determines readiness for the parent CR.
// Uses katalog.BuiltInMeta(parentKind) directly to avoid the GVK string
// format mismatch in crd.IsStatuslessType().
//
// Resolution order:
//  1. Statusless (ConfigMap, Secret, etc.):
//     a. orkestra.io/phase annotation → use it (Orkestra writes here for statusless CRDs)
//     b. No annotation → ready on existence
//  2. Non-statusless: check status.conditions[type=Ready]
func deprecatedExtractParentReady(objMap map[string]interface{}, parentKind string) (ready bool, reason, message string) {
	m := katalog.BuiltInMeta(parentKind)
	statusless := m.Statusless || m.SkipStatusSubresource

	if statusless {
		meta, _ := objMap["metadata"].(map[string]interface{})
		annotations, _ := meta["annotations"].(map[string]interface{})
		if phase, ok := annotations["orkestra.io/phase"].(string); ok {
			switch phase {
			case "Ready", "Succeeded":
				return true, phase, ""
			case "Failed", "Degraded", "Error":
				return false, phase, ""
			default:
				return true, phase, "" // Running, Pending, unknown → optimistic
			}
		}
		// No phase annotation yet — first reconcile not complete or CR just created.
		// Return true: the CR exists, which is the correct "ready" semantic for
		// statusless types. The phase annotation will appear after the first reconcile.
		return true, "Exists", ""
	}

	// Standard path: look for Ready condition in status
	status, _ := objMap["status"].(map[string]interface{})
	conditionsRaw, _ := status["conditions"].([]interface{})
	var conditions []interface{} = conditionsRaw
	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if cond["type"] == "Ready" {
			ready = cond["status"] == "True"
			reason = fmt.Sprint(cond["reason"])
			message = fmt.Sprint(cond["message"])
			if message == "<nil>" {
				message = ""
			}
			return ready, reason, message
		}
	}

	// No Ready condition — first reconcile not yet complete
	return false, "Pending", ""
}

// ─────────────────────────────────────────────────────────────────────────────
// Child readiness
// ─────────────────────────────────────────────────────────────────────────────

// isChildReady determines readiness for one child resource.
// statusless is from builtInRegistry (per-kind, not from parent CRD).
func isChildReady(obj map[string]interface{}, statusless bool) bool {
	if statusless {
		return true // ConfigMap, Secret, ServiceAccount — ready on existence
	}

	status, _ := obj["status"].(map[string]interface{})
	if status == nil {
		return false
	}

	kind := strings.ToLower(fmt.Sprint(obj["kind"]))

	switch kind {
	case "job":
		s, _ := status["succeeded"].(int64)
		return s > 0

	case "deployment", "statefulset", "replicaset":
		rr, _ := status["readyReplicas"].(int64)
		if rr == 0 {
			return false
		}
		if spec, _ := obj["spec"].(map[string]interface{}); spec != nil {
			if desired, _ := spec["replicas"].(int64); desired > 0 {
				return rr >= desired
			}
		}
		return rr > 0

	case "daemonset":
		desired, _ := status["desiredNumberScheduled"].(int64)
		ready, _ := status["numberReady"].(int64)
		return desired > 0 && ready >= desired

	case "cronjob":
		_, has := status["lastScheduleTime"]
		return has

	case "service":
		return true // Services are always considered available

	case "pod":
		phase, _ := status["phase"].(string)
		return phase == "Running" || phase == "Succeeded"

	default:
		// Check standard conditions
		conditions, _ := status["conditions"].([]interface{})
		for _, c := range conditions {
			if cond, ok := c.(map[string]interface{}); ok {
				if cond["type"] == "Ready" {
					return cond["status"] == "True"
				}
			}
		}
		return true // optimistic for unknown types
	}
}
