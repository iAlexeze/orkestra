// pkg/gateway/resources/handler.go
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
package applyapi

import (
	"fmt"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/utils"
)

// KindMapper resolves a kind name to a GroupVersionResource.
// Provided by the caller (gateway server) so this package stays pure HTTP.
type KindMapper func(kind string) (schema.GroupVersionResource, error)

// Handler returns the http.HandlerFunc for /api/v1/resources/... routes.
// URL pattern: /api/v1/resources/{kind}/{namespace}[/{name}]
// The auth middleware must wrap this handler before registration.
func resourcesHandler(kube kubeclient.KubeClient, mapper KindMapper) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kind, ns, name, err := parsePath(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		gvr, err := mapper(kind)
		if err != nil {
			http.Error(w, fmt.Sprintf("unknown kind %q: %v", kind, err), http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			if name == "" {
				listResources(w, r, kube, gvr, ns)
			} else {
				getResource(w, r, kube, gvr, ns, name)
			}
		case http.MethodDelete:
			if name == "" {
				http.Error(w, "name required for DELETE", http.StatusBadRequest)
				return
			}
			deleteResource(w, r, kube, gvr, ns, name)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func getResource(w http.ResponseWriter, r *http.Request, kube kubeclient.KubeClient, gvr schema.GroupVersionResource, ns, name string) {
	obj, err := kube.DynamicClient().Resource(gvr).Namespace(ns).Get(r.Context(), name, metav1.GetOptions{})
	if err != nil {
		writeKubeError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, obj.Object)
}

func listResources(w http.ResponseWriter, r *http.Request, kube kubeclient.KubeClient, gvr schema.GroupVersionResource, ns string) {
	list, err := kube.DynamicClient().Resource(gvr).Namespace(ns).List(r.Context(), metav1.ListOptions{})
	if err != nil {
		writeKubeError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, list.Object)
}

func deleteResource(w http.ResponseWriter, r *http.Request, kube kubeclient.KubeClient, gvr schema.GroupVersionResource, ns, name string) {
	err := kube.DynamicClient().Resource(gvr).Namespace(ns).Delete(r.Context(), name, metav1.DeleteOptions{})
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
