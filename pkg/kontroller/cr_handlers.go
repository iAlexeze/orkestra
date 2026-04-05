// pkg/kontroller/cr_handlers.go
//
// CR-level endpoints — list, detail, and events per CRD instance.
//
// These endpoints extend the existing /katalog/{crd} surface with:
//
//   GET /katalog/{crd}/cr
//       List all CR instances for a CRD — the kubectl get equivalent.
//       Reads from the informer cache — no API server calls.
//       Returns name, namespace, phase, ready, age, generation.
//
//   GET /katalog/{crd}/cr/{name}           (cluster-scoped CRDs)
//   GET /katalog/{crd}/cr/{namespace}/{name} (namespaced CRDs)
//       Full CR detail: status fields, conditions, children.
//       Children are read on demand from the API server.
//
//   GET /katalog/{crd}/cr/{name}/events
//   GET /katalog/{crd}/cr/{namespace}/{name}/events
//       Recent Kubernetes events for this CR — both emitted by Orkestra
//       (Reconciled, ReconcileError) and by Kubernetes itself.
//
// Registration — add to wherever BuildKatalogHandler is registered:
//
//   crHandler := kontroller.NewCRHandler(kube, reg, rcMap)
//   mux.Handle("/katalog/", crHandler.Route(existingKatalogMux))
//
// Or call BuildCRListHandler / BuildCRDetailHandler / BuildCREventsHandler
// directly alongside the existing per-CRD handler registrations.

package kontroller

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ialexeze/orkestra/pkg/kubeclient"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// ─────────────────────────────────────────────────────────────────────────────
// GVRs for all resource types Orkestra manages as children.
// Defined here (not imported from reconciler) to avoid import cycles.
// These mirror the GVR vars in pkg/reconciler/run_children.go.
// ─────────────────────────────────────────────────────────────────────────────
var (
	crDeploymentGVR     = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	crServiceGVR        = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	crSecretGVR         = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	crConfigMapGVR      = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	crJobGVR            = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
	crCronJobGVR        = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}
	crServiceAccountGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}
)

// knownChildGVRs is the ordered list of resource types Orkestra may create
// as children. For each, the lowercase kind key used in the children map.
var knownChildGVRs = []struct {
	GVR schema.GroupVersionResource
	Key string // lowercase kind — matches .children.<key> in templates
}{
	{crDeploymentGVR, "deployment"},
	{crServiceGVR, "service"},
	{crConfigMapGVR, "configmap"},
	{crSecretGVR, "secret"},
	{crJobGVR, "job"},
	{crCronJobGVR, "cronjob"},
	{crServiceAccountGVR, "serviceaccount"},
}

// ─────────────────────────────────────────────────────────────────────────────
// Response types
// ─────────────────────────────────────────────────────────────────────────────

// CRSummary is one row in the CR list — equivalent to one line of kubectl get.
type CRSummary struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	Phase       string `json:"phase,omitempty"` // from status.phase if declared
	Ready       bool   `json:"ready"`           // from status.conditions[type=Ready]
	ReadyReason string `json:"readyReason,omitempty"`
	Age         string `json:"age"`
	Generation  int64  `json:"generation"`
}

// CRListResponse is returned by GET /katalog/{crd}/cr.
type CRListResponse struct {
	CRD   string      `json:"crd"`
	GVK   string      `json:"gvk"`
	Total int         `json:"total"`
	Items []CRSummary `json:"items"`
}

// ChildSummary is a condensed view of one child resource.
// Only the fields that matter for observability are included — not the full object.
type ChildSummary struct {
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace,omitempty"`
	Kind      string                 `json:"kind"`
	Status    map[string]interface{} `json:"status,omitempty"`
	Ready     bool                   `json:"ready"`
}

// CRDetailResponse is returned by GET /katalog/{crd}/cr/{namespace}/{name}.
type CRDetailResponse struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace,omitempty"`
	Generation        int64             `json:"generation"`
	CreationTimestamp string            `json:"creationTimestamp"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`

	// Ready reflects the Orkestra Ready condition.
	// true = operator reconciled successfully.
	// The CR may still be in a non-terminal phase (e.g. Running/build).
	Ready        bool   `json:"ready"`
	ReadyReason  string `json:"readyReason,omitempty"`
	ReadyMessage string `json:"readyMessage,omitempty"`

	// Status is the full status subresource as returned by the API server.
	// Includes phase, conditions, and any declared status fields.
	Status map[string]interface{} `json:"status,omitempty"`

	// Children holds the child resources created by Orkestra for this CR.
	// Keyed by lowercase kind (e.g. "deployment", "job", "cronjob").
	// Populated on demand from the API server — may be empty on first reconcile.
	Children map[string]ChildSummary `json:"children,omitempty"`

	// EventsEndpoint is the URL to fetch recent events for this CR.
	EventsEndpoint string `json:"eventsEndpoint"`
}

// CREvent is one Kubernetes event involving this CR or its children.
type CREvent struct {
	Type      string `json:"type"` // Normal or Warning
	Reason    string `json:"reason"`
	Message   string `json:"message"`
	Source    string `json:"source"` // component that emitted the event
	Object    string `json:"object"` // involved object kind/name
	Count     int32  `json:"count"`
	FirstSeen string `json:"firstSeen"`
	LastSeen  string `json:"lastSeen"`
	Age       string `json:"age"`
}

// CREventsResponse is returned by GET /katalog/{crd}/cr/{namespace}/{name}/events.
type CREventsResponse struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace,omitempty"`
	Total     int       `json:"total"`
	Events    []CREvent `json:"events"`
}

// ─────────────────────────────────────────────────────────────────────────────
// CR List Handler
// GET /katalog/{crd}/cr
//
// Lists all CR instances from the informer cache.
// No API server calls — purely in-memory.
// Sort order: namespace/name ascending for deterministic output.
// ─────────────────────────────────────────────────────────────────────────────
func BuildCRListHandler(
	crd orktypes.CRDEntry,
	inf cache.SharedIndexInformer,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if inf == nil {
			utils.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "informer not ready — CRD may still be starting",
			})
			return
		}

		objs := inf.GetIndexer().List()
		items := make([]CRSummary, 0, len(objs))

		for _, raw := range objs {
			u, ok := raw.(*unstructured.Unstructured)
			if !ok {
				continue
			}
			items = append(items, summariseCR(u))
		}

		sort.Slice(items, func(i, j int) bool {
			ki := items[i].Namespace + "/" + items[i].Name
			kj := items[j].Namespace + "/" + items[j].Name
			return ki < kj
		})

		utils.WriteJSON(w, http.StatusOK, CRListResponse{
			CRD:   crd.Name,
			GVK:   crd.GVK().String(),
			Total: len(items),
			Items: items,
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CR Detail Handler
// GET /katalog/{crd}/cr/{name}              (cluster-scoped)
// GET /katalog/{crd}/cr/{namespace}/{name}  (namespaced)
//
// Returns the full CR status and its child resources.
// CR is read from the informer cache — no API server call.
// Children are read from the API server on demand.
//
// rcMap maps GVK → ReconcilerConfig so we know which resource types
// to look for as children (from onCreate/onReconcile templates).
// ─────────────────────────────────────────────────────────────────────────────
func BuildCRDetailHandler(
	crd orktypes.CRDEntry,
	inf cache.SharedIndexInformer,
	kube *kubeclient.Kubeclient,
	rc orktypes.ReconcilerConfig,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if inf == nil {
			utils.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "informer not ready",
			})
			return
		}

		// Parse name and optional namespace from the URL suffix.
		// The mux strips /katalog/{crd}/cr/ before this handler is called,
		// leaving either "name" or "namespace/name".
		suffix := strings.TrimPrefix(r.URL.Path, "/")
		parts := strings.SplitN(suffix, "/", 2)

		var namespace, name string
		switch len(parts) {
		case 1:
			name = parts[0] // cluster-scoped
		case 2:
			namespace = parts[0]
			name = parts[1]
		default:
			http.NotFound(w, r)
			return
		}

		// Look up in informer cache — no API call
		key := name
		if namespace != "" {
			key = namespace + "/" + name
		}
		raw, exists, err := inf.GetIndexer().GetByKey(key)
		if err != nil || !exists {
			utils.WriteJSON(w, http.StatusNotFound, map[string]string{
				"error": fmt.Sprintf("CR %q not found", key),
			})
			return
		}

		u, ok := raw.(*unstructured.Unstructured)
		if !ok {
			utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "unexpected object type in cache",
			})
			return
		}

		// Read child resources from the API server.
		// This is the same set of children ReadChildren uses in the reconciler.
		children := readChildrenForEndpoint(r.Context(), kube, u, rc)

		// Build the events endpoint URL for this CR
		eventsPath := buildEventsPath(crd, namespace, name)

		utils.WriteJSON(w, http.StatusOK, buildCRDetail(u, children, eventsPath))
	}
}

// BuildCRDetailAndEventsHandler handles all sub-paths under /katalog/{crd}/cr/
// Dispatches to detail or events based on whether the path ends with /events.
// Register at "/katalog/{crd}/cr/" (with trailing slash).
func BuildCRDetailAndEventsHandler(
	crd orktypes.CRDEntry,
	inf cache.SharedIndexInformer,
	kube *kubeclient.Kubeclient,
) http.HandlerFunc {
	detail := BuildCRDetailHandler(crd, inf, kube, orktypes.ReconcilerConfig{})
	events := BuildCREventsHandler(crd, kube)

	return func(w http.ResponseWriter, r *http.Request) {
		prefix := "/katalog/" + strings.ToLower(crd.Name) + "/cr/"
		suffix := strings.TrimPrefix(r.URL.Path, prefix)

		if strings.HasSuffix(suffix, "/events") {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/" + strings.TrimSuffix(suffix, "/events")
			events.ServeHTTP(w, r2)
			return
		}

		r2 := r.Clone(r.Context())
		r2.URL.Path = "/" + suffix
		detail.ServeHTTP(w, r2)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CR Events Handler
// GET /katalog/{crd}/cr/{name}/events              (cluster-scoped)
// GET /katalog/{crd}/cr/{namespace}/{name}/events  (namespaced)
//
// Returns recent Kubernetes events involving this CR.
// Events are listed from the API server using field selectors on
// involvedObject.name and involvedObject.namespace.
//
// Includes events emitted by Orkestra (reason: Reconciled, ReconcileError,
// ValidationWarning, Deleting) and by Kubernetes itself.
//
// Events are sorted newest-first and capped at 100.
// ─────────────────────────────────────────────────────────────────────────────
func BuildCREventsHandler(
	crd orktypes.CRDEntry,
	kube *kubeclient.Kubeclient,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse namespace and name from URL suffix (after /events stripped by mux)
		suffix := strings.TrimSuffix(
			strings.TrimPrefix(r.URL.Path, "/"),
			"/events",
		)
		parts := strings.SplitN(suffix, "/", 2)

		var namespace, name string
		switch len(parts) {
		case 1:
			name = parts[0]
		case 2:
			namespace = parts[0]
			name = parts[1]
		default:
			http.NotFound(w, r)
			return
		}

		events, err := fetchCREvents(r.Context(), kube, namespace, name, crd.APITypes.Kind)
		if err != nil {
			utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("listing events: %v", err),
			})
			return
		}

		utils.WriteJSON(w, http.StatusOK, CREventsResponse{
			Name:      name,
			Namespace: namespace,
			Total:     len(events),
			Events:    events,
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

// summariseCR extracts the summary fields from one CR object.
func summariseCR(u *unstructured.Unstructured) CRSummary {
	phase, _, _ := unstructured.NestedString(u.Object, "status", "phase")
	ready, readyReason := extractReadyCondition(u)
	age := formatAge(u.GetCreationTimestamp().Time)

	return CRSummary{
		Name:        u.GetName(),
		Namespace:   u.GetNamespace(),
		Phase:       phase,
		Ready:       ready,
		ReadyReason: readyReason,
		Age:         age,
		Generation:  u.GetGeneration(),
	}
}

// buildCRDetail assembles the full detail response for one CR.
func buildCRDetail(u *unstructured.Unstructured, children map[string]ChildSummary, eventsEndpoint string) CRDetailResponse {
	ready, readyReason, readyMsg := extractReadyConditionFull(u)

	status, _ := u.Object["status"].(map[string]interface{})

	// Filter Orkestra system annotations from the public response
	annotations := make(map[string]string)
	for k, v := range u.GetAnnotations() {
		if !strings.HasPrefix(k, "orkestra.konductor.io/") &&
			k != "kubectl.kubernetes.io/last-applied-configuration" {
			annotations[k] = v
		}
	}

	return CRDetailResponse{
		Name:              u.GetName(),
		Namespace:         u.GetNamespace(),
		Generation:        u.GetGeneration(),
		CreationTimestamp: u.GetCreationTimestamp().UTC().Format(time.RFC3339),
		Labels:            u.GetLabels(),
		Annotations:       annotations,
		Ready:             ready,
		ReadyReason:       readyReason,
		ReadyMessage:      readyMsg,
		Status:            status,
		Children:          children,
		EventsEndpoint:    eventsEndpoint,
	}
}

// readChildrenForEndpoint reads child resources from the API server.
//
// Rather than requiring the ReconcilerConfig (which would create an import
// cycle via mergeTemplates in pkg/reconciler), we query each known resource
// type by the orkestra-owner label. This is equivalent to ReadChildren in
// the reconciler but works from the kontroller package without any imports
// from pkg/reconciler.
//
// Returns a map keyed by lowercase kind — same convention as .children.* in templates.
func readChildrenForEndpoint(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	owner *unstructured.Unstructured,
	_ orktypes.ReconcilerConfig, // kept for signature compatibility, unused
) map[string]ChildSummary {
	result := map[string]ChildSummary{}
	if kube == nil {
		return result
	}

	ns := owner.GetNamespace()
	labelSelector := fmt.Sprintf("orkestra-owner=%s", owner.GetName())

	for _, entry := range knownChildGVRs {
		resource := kube.DynamicClient().Resource(entry.GVR)

		var list *unstructured.UnstructuredList
		var err error

		if ns != "" {
			list, err = resource.Namespace(ns).List(ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
				Limit:         1, // only need the first for the summary
			})
		} else {
			// Cluster-scoped owner — search all namespaces
			list, err = resource.List(ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
				Limit:         1,
			})
		}

		if err != nil || len(list.Items) == 0 {
			continue
		}

		obj := list.Items[0]
		status, _ := obj.Object["status"].(map[string]interface{})
		if status == nil {
			status = map[string]interface{}{}
		}

		// Use the object's own Kind field when available (populated by API server).
		// Fall back to the key title-cased (e.g. "job" → "Job").
		kind := obj.GetKind()
		if kind == "" {
			kind = strings.ToUpper(entry.Key[:1]) + entry.Key[1:]
		}

		result[entry.Key] = ChildSummary{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
			Kind:      kind,
			Status:    status,
			Ready:     isChildReady(obj.Object),
		}
	}

	return result
}

// fetchCREvents lists Kubernetes events for a specific CR by name.
// Uses field selectors — no label selector needed since events reference
// their involved object directly.
func fetchCREvents(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	namespace, name, kind string,
) ([]CREvent, error) {
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	// Field selector: events where involvedObject matches this CR
	selector := fields.AndSelectors(
		fields.OneTermEqualSelector("involvedObject.name", name),
		fields.OneTermEqualSelector("involvedObject.kind", kind),
	)
	if namespace != metav1.NamespaceAll {
		selector = fields.AndSelectors(
			selector,
			fields.OneTermEqualSelector("involvedObject.namespace", namespace),
		)
	}

	eventList, err := kube.Clientset().CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: selector.String(),
		Limit:         100, // cap at 100 — events are noisy
	})
	if err != nil {
		return nil, err
	}

	events := make([]CREvent, 0, len(eventList.Items))
	for _, ev := range eventList.Items {
		events = append(events, summariseEvent(ev))
	}

	// Sort newest-first — most recent event is what operators care about
	sort.Slice(events, func(i, j int) bool {
		return events[i].LastSeen > events[j].LastSeen
	})

	return events, nil
}

// summariseEvent converts a Kubernetes Event to the API response shape.
func summariseEvent(ev corev1.Event) CREvent {
	firstSeen := ev.FirstTimestamp.UTC().Format(time.RFC3339)
	lastSeen := ev.LastTimestamp.UTC().Format(time.RFC3339)

	// Use EventTime for newer events that use the eventTime field instead
	if !ev.EventTime.IsZero() {
		lastSeen = ev.EventTime.UTC().Format(time.RFC3339)
	}

	return CREvent{
		Type:      ev.Type,
		Reason:    ev.Reason,
		Message:   ev.Message,
		Source:    ev.Source.Component,
		Object:    fmt.Sprintf("%s/%s", ev.InvolvedObject.Kind, ev.InvolvedObject.Name),
		Count:     ev.Count,
		FirstSeen: firstSeen,
		LastSeen:  lastSeen,
		Age:       formatAge(ev.LastTimestamp.Time),
	}
}

// extractReadyCondition reads the Ready condition from status.conditions.
// Returns (ready bool, reason string).
func extractReadyCondition(u *unstructured.Unstructured) (bool, string) {
	ready, reason, _ := extractReadyConditionFull(u)
	return ready, reason
}

// extractReadyConditionFull reads Ready condition with message.
func extractReadyConditionFull(u *unstructured.Unstructured) (bool, string, string) {
	conditions, _, _ := unstructured.NestedSlice(u.Object, "status", "conditions")
	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if cond["type"] == "Ready" {
			ready := cond["status"] == "True"
			reason := fmt.Sprint(cond["reason"])
			message := fmt.Sprint(cond["message"])
			if message == "<nil>" {
				message = ""
			}
			return ready, reason, message
		}
	}
	return false, "", ""
}

// isChildReady determines readiness from a child resource's status.
// For Jobs: ready when succeeded > 0.
// For Deployments: ready when readyReplicas == replicas.
// For everything else: ready when no failed conditions exist.
func isChildReady(obj map[string]interface{}) bool {
	status, _ := obj["status"].(map[string]interface{})
	if status == nil {
		return false
	}

	// Job: succeeded count > 0
	if succeeded, ok := status["succeeded"]; ok {
		s, _ := succeeded.(int64)
		return s > 0
	}

	// Deployment: readyReplicas matches replicas
	if readyReplicas, ok := status["readyReplicas"]; ok {
		spec, _ := obj["spec"].(map[string]interface{})
		if spec != nil {
			desired, _ := spec["replicas"].(int64)
			ready, _ := readyReplicas.(int64)
			return desired > 0 && ready == desired
		}
	}

	// CronJob: has a lastScheduleTime
	if _, ok := status["lastScheduleTime"]; ok {
		return true
	}

	return true // default optimistic for other resource types
}

// buildEventsPath constructs the events endpoint URL for a CR.
func buildEventsPath(crd orktypes.CRDEntry, namespace, name string) string {
	base := "/katalog/" + strings.ToLower(crd.Name) + "/cr"
	if namespace != "" {
		return fmt.Sprintf("%s/%s/%s/events", base, namespace, name)
	}
	return fmt.Sprintf("%s/%s/events", base, name)
}

// formatAge returns a human-readable age string from a time.Time.
func formatAge(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t).Round(time.Second)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
