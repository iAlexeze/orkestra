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
	"github.com/orkspace/orkestra/pkg/logger"
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
			writeJSONError(w, http.StatusBadRequest, "invalid path",
				fmt.Sprintf("could not parse path: %v", err),
			)
			return
		}

		// ─── Debug: Log available kinds ──────────────────────────────────────────────
		logger.FromContext(r.Context()).Debug().
			Str("kind", kind).
			Str("availableKinds", strings.Join(kat.ListKinds(), ", ")).
			Msg("resourcesHandler: lookup")

		// Resolve CRD entry — accept both lowercased kind and target identifier.
		crd := kat.LookupByKind(kind)
		if crd == nil {
			writeJSONError(w, http.StatusNotFound, "kind not found",
				fmt.Sprintf("unknown kind %q", kind),
			)
			return
		}

		if !crd.IDPEnabled() {
			writeJSONError(w, http.StatusBadRequest, "idp not enabled",
				fmt.Sprintf("IDP is not enabled for kind %q", kind),
			)
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
				writeJSONError(w, http.StatusBadRequest, "name required",
					"DELETE requires a resource name",
				)
				return
			}
			deleteResource(w, r, kube, ns, name, crd)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed",
				fmt.Sprintf("method %q is not supported for /api/v1/resources", r.Method),
			)
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

	// ── Step 1: Check for ?field= query param (lightweight polling) ──────
	field := r.URL.Query().Get("field")
	if field != "" {
		value, ok := resolveScalarField(response, field)
		if !ok {
			writeJSONError(w, http.StatusBadRequest, "invalid field path",
				fmt.Sprintf("field %q not found", field),
			)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"name":  name,
			"field": field,
			"value": value,
		})
		return
	}

	// ── Step 2: Apply exclusions ──────────────────────────────────────────
	// ApplyExclusions strips paths listed in idp.config.response.exclude.
	ApplyExclusions(response, crd, notes)

	// ── Step 3: Evaluate payload ──────────────────────────────────────────
	// EvaluatePayload returns ONLY the fields defined in idp.config.response.payload.
	payload := EvaluatePayload(response, crd, notes)

	// ── Step 4: Return response ──────────────────────────────────────────
	if payload != nil {
		// When default: false, return only the payload.
		if !crd.GetIDPResponseConfig().UseDefault() {
			writeJSON(w, http.StatusOK, payload)
			return
		}
		// When default: true, inject payload alongside the CR.
		response["payload"] = payload
	}

	writeJSON(w, http.StatusOK, response)
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
		return
	}

	p := parsePagination(r)

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
	usePayload := crd.HasIDPResponseConfig() && crd.GetIDPResponseConfig().HasPayload()

	// ── Build items ──────────────────────────────────────────────────────
	type responseItem struct {
		Object  map[string]interface{} `json:"object"`
		Payload map[string]interface{} `json:"payload,omitempty"`
	}

	items := make([]responseItem, 0, len(list.Items))
	for _, item := range list.Items {
		obj := item.Object

		// Step 1: Apply exclusions to each item
		ApplyExclusions(obj, crd, notes)

		// Step 2: Build payload for each item
		var payload map[string]interface{}
		if usePayload {
			payload = EvaluatePayload(obj, crd, notes)
		}

		entry := responseItem{Object: obj}
		// If default: false, use payload as the primary response
		if !crd.GetIDPResponseConfig().UseDefault() {
			entry.Object = nil
		}
		if payload != nil {
			entry.Payload = payload
		}

		items = append(items, entry)

		// ── ?field= query param ──────────────────────────────────────────
		if field != "" {
			value, ok := resolveScalarField(obj, field)
			if !ok {
				writeJSONError(w, http.StatusBadRequest, "invalid field path",
					fmt.Sprintf("field %q not found", field),
				)
				return
			}
			name, _ := obj["metadata"].(map[string]interface{})["name"].(string)
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"name":  name,
				"field": field,
				"value": value,
			})
			return
		}
	}

	// ── Return paginated response ───────────────────────────────────────
	writeJSON(w, http.StatusOK, utils.PaginatedResponse[responseItem]{
		Total:    len(items),
		Limit:    p.Limit,
		Offset:   p.Offset,
		Continue: list.GetContinue(),
		Items:    items,
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
		writeJSONError(w, http.StatusBadRequest, "name required",
			"DELETE requires a resource name",
		)
		return
	}

	if !checkIDPPermission(w, r, crd, orktypes.IDPClassResources, orktypes.IDPOpDelete, ns) {
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
		writeJSONError(w, http.StatusForbidden, "permission denied", reason.Message(tokenName, op, crd.APITypes.Kind, ns))
		return false
	}
	return true
}
