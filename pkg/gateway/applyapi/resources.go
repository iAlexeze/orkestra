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

		if !crd.IDPEnabled() {
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
	if !checkIDPPermission(w, r, crd, orktypes.IDPClassResources, orktypes.IDPOpGet, ns) {
		http.Error(w, "permission denied", http.StatusForbidden)
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
	if !checkIDPPermission(w, r, crd, orktypes.IDPClassResources, orktypes.IDPOpList, ns) {
		http.Error(w, "permission denied", http.StatusForbidden)
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

	if !checkIDPPermission(w, r, crd, orktypes.IDPClassResources, orktypes.IDPOpDelete, ns) {
		http.Error(w, "permission denied", http.StatusForbidden)
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
	class orktypes.IDPEndpointClass,
	op, ns string,
) bool {
	if crd == nil || crd.IDP == nil || !crd.IDP.HasTokenRestrictions() {
		// No restrictions declared — proceed.
		return true
	}

	tokenName := TokenNameFromContext(r.Context())
	allowed, reason := crd.IDP.TokenAllowed(tokenName, op, ns, class)
	if !allowed {
		http.Error(
			w,
			reason.Message(tokenName, op, crd.APITypes.Kind, ns),
			http.StatusForbidden,
		)
		return false
	}
	return true
}
