// pkg/gateway/applyapi/resources.go
//
// GET  /api/v1/resources/{kind}/{namespace}/{name}  — read one CR
// GET  /api/v1/resources/{kind}/{namespace}         — list CRs in a namespace
// DELETE /api/v1/resources/{kind}/{namespace}/{name} — delete a CR
//
// These endpoints are for external callers (CI, Terraform, Slack bots) that
// need raw Kubernetes CR state without kubeconfig. The Control Center uses the
// runtime's /katalog/{crd}/cr/... endpoints (richer, informer-cached); these
// endpoints call the Kubernetes API directly via the dynamic client.
//
// Deletion protection is enforced by the admission webhook when enabled.
// This handler does not duplicate that check — it issues the DELETE and lets
// Kubernetes reject it if the webhook blocks.
//
// {kind} is the lowercased kind string or, when the CRD has idp.target set,
// the target value. The handler resolves the CRD via the kind lookup.
package applyapi

import (
	"fmt"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
)

// resourcesHandler returns the http.HandlerFunc for /api/v1/resources/... routes.
// URL pattern: /api/v1/resources/{kind}/{namespace}[/{name}]
// The auth middleware must wrap this handler before registration.
//
// Two lookup modes are supported:
//   - Kind lookup: resolves the raw kind string to a CRD entry
//   - Target lookup: resolves the target identifier to a CRD entry (when idp.target is set)
func resourcesHandler(
	kube kubeclient.KubeClient,
	kat *katalog.Katalog,
	notes orktypes.NoteRegistry,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kind, ns, name, err := parsePath(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Resolve CRD entry — accept both lowercased kind and target identifier.
		crd := kat.LookupByKind(kind)
		if crd == nil {
			http.Error(w, fmt.Sprintf("unknown kind or target %q", kind), http.StatusNotFound)
			return
		}

		if crd == nil || !crd.IDPEnabled() {
			http.Error(w, fmt.Sprintf("idp not enabled for %q", kind), http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			if name == "" {
				listResources(w, r, kube, ns, crd, notes)
			} else {
				getResource(w, r, kube, ns, name, crd, notes)
			}
		case http.MethodDelete:
			if name == "" {
				http.Error(w, "name required for DELETE", http.StatusBadRequest)
				return
			}
			deleteResource(w, r, kube, ns, name, crd)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// getResource handles GET /api/v1/resources/{kind}/{ns}/{name}.
// Returns the full stored CR with the idp.config.response payload evaluated
// against the live CR — status fields written by the runtime are available.
func getResource(
	w http.ResponseWriter,
	r *http.Request,
	kube kubeclient.KubeClient,
	ns, name string,
	crd *orktypes.CRDEntry,
	notes orktypes.NoteRegistry,
) {
	// When the CRD declares idp.allowedTokens, the authenticated token must
	// have permission to perform the operation it is attempting.
	if !checkIDPPermission(w, r, crd, orktypes.IDPOpGet, ns) {
		return
	}

	gvr := crd.GVR()
	obj, err := kube.DynamicClient().Resource(gvr).Namespace(ns).
		Get(r.Context(), name, metav1.GetOptions{})
	if err != nil {
		writeKubeError(w, err)
		return
	}

	response := obj.Object

	field := r.URL.Query().Get("field")
	if field != "" {
		value, ok := orktypes.ResolveScalarField(response, field)
		if !ok {
			http.Error(w, "invalid field path", http.StatusBadRequest)
			return
		}
		// Return just the value
		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"name":  name,
			"field": field,
			"value": value,
		})
		return
	}

	// Evaluate idp.config.response against the full stored CR.
	// At GET time .status is available because the runtime has written it —
	// payload fields referencing status are now populated.
	//
	// EvaluatePayload returns nil when no config is declared, in which case
	// we skip injecting the "payload" key so the response stays identical to
	// today for CRDs without idp.config.response.
	if payload := EvaluatePayload(obj.Object, crd, notes); payload != nil {
		response["payload"] = payload
	}

	utils.WriteJSON(w, http.StatusOK, response)
}

// listResources handles GET /api/v1/resources/{kind}/{ns}.
// Returns a paginated list of CRs. Supports Kubernetes continuation tokens
// via the ?continue= query param for server-side pagination.
func listResources(
	w http.ResponseWriter,
	r *http.Request,
	kube kubeclient.KubeClient,
	ns string,
	crd *orktypes.CRDEntry,
	notes orktypes.NoteRegistry,
) {
	if !checkIDPPermission(w, r, crd, orktypes.IDPOpList, ns) {
		return
	}

	p := utils.ParsePagination(r)

	listOpts := metav1.ListOptions{
		Limit:    int64(p.Limit),
		Continue: p.Continue,
	}

	gvr := crd.GVR()
	list, err := kube.DynamicClient().
		Resource(gvr).
		Namespace(ns).
		List(r.Context(), listOpts)
	if err != nil {
		writeKubeError(w, err)
		return
	}

	field := r.URL.Query().Get("field")

	// Evaluate payload for each item when configured.
	items := list.Items
	type itemWithPayload struct {
		Object  map[string]interface{} `json:"object"`
		Payload map[string]interface{} `json:"payload,omitempty"`
	}
	result := make([]itemWithPayload, 0, len(items))
	for _, item := range items {
		entry := itemWithPayload{Object: item.Object}
		if payload := EvaluatePayload(item.Object, crd, notes); payload != nil {
			entry.Payload = payload
		}
		result = append(result, entry)
		if field != "" {
			value, ok := orktypes.ResolveScalarField(item.Object, field)
			if !ok {
				http.Error(w, "invalid field path", http.StatusBadRequest)
				return
			}
			// Return just the value
			utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"name":  entry.Object["metadata"].(map[string]interface{})["name"],
				"field": field,
				"value": value,
			})
			return
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.PaginatedResponse[itemWithPayload]{
		Total:    len(result), // client-side total for this page
		Limit:    p.Limit,
		Offset:   p.Offset,
		Continue: list.GetContinue(), // Kubernetes server-side continue token
		Items:    result,
	})
}

// deleteResource handles DELETE /api/v1/resources/{kind}/{ns}/{name}.
// Respects deletion protection — the Kubernetes webhook enforces it; the
// gateway surfaces the structured rejection rather than a raw API server error.
func deleteResource(
	w http.ResponseWriter,
	r *http.Request,
	kube kubeclient.KubeClient,
	ns, name string,
	crd *orktypes.CRDEntry,
) {
	if name == "" {
		http.Error(w, "name is required for DELETE", http.StatusBadRequest)
		return
	}

	if !checkIDPPermission(w, r, crd, orktypes.IDPOpDelete, ns) {
		return
	}

	gvr := crd.GVR()
	err := kube.DynamicClient().Resource(gvr).Namespace(ns).
		Delete(r.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		writeKubeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// resourcePath builds the canonical /api/v1/resources/{kind}/{namespace}/{name}
// path for a CR. namespace is "" for cluster-scoped kinds — parsePath treats
// an empty middle segment as "no namespace", so the result is a literal
// doubled slash (e.g. /api/v1/resources/AppRequest//payments-api), which is
// the documented shape for cluster-scoped lookups, not a bug.
func resourcePath(kind, namespace, name string) string {
	return fmt.Sprintf("/api/v1/resources/%s/%s/%s", kind, namespace, name)
}

// parsePath extracts kind, namespace, and optional name from
// /api/v1/resources/{kind}/{namespace}[/{name}]
func parsePath(path string) (kind, ns, name string, err error) {
	// Strip prefix — the handler is registered at /api/v1/resources/
	path = strings.TrimPrefix(path, "/api/v1/resources/")
	path = strings.TrimPrefix(path, "/") // normalise double-slash

	parts := strings.SplitN(path, "/", 3)
	switch len(parts) {
	case 2:
		return parts[0], parts[1], "", nil
	case 3:
		return parts[0], parts[1], parts[2], nil
	default:
		return "", "", "", fmt.Errorf("path must be /{kind}/{namespace}[/{name}], got %q", path)
	}
}

// writeKubeError maps common Kubernetes API errors to HTTP status codes.
func writeKubeError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found") || strings.Contains(msg, "NotFound"):
		http.Error(w, msg, http.StatusNotFound)
	case strings.Contains(msg, "forbidden") || strings.Contains(msg, "Forbidden"):
		http.Error(w, msg, http.StatusForbidden)
	default:
		http.Error(w, msg, http.StatusInternalServerError)
	}
}

// checkIDPPermission is a single function used by all resource handlers
// (get, list, delete). Keeps the permission logic in one place so changes to
// the model propagate automatically.
//
// Returns true when the request should proceed. Writes the 403 response and
// returns false when it should not — callers must return immediately.
func checkIDPPermission(
	w http.ResponseWriter,
	r *http.Request,
	crd *orktypes.CRDEntry,
	op string,
	namespace string,
) bool {
	if crd == nil || crd.IDP == nil || !crd.IDP.HasTokenRestrictions() {
		// No restrictions declared — proceed.
		return true
	}

	tokenName := TokenNameFromContext(r.Context())
	allowed, reason := crd.IDP.TokenAllowed(tokenName, op, namespace)
	if !allowed {
		http.Error(
			w,
			reason.Message(tokenName, op, crd.APITypes.Kind, namespace),
			http.StatusForbidden,
		)
		return false
	}
	return true
}
