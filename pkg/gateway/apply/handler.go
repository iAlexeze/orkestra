// pkg/gateway/apply/handler.go
//
// POST /api/v1/apply
//
// Pipeline: auth (handled by middleware) → decode body → SSA → translate response.
//
// The Apply API does not duplicate admission logic. Webhooks enforce admission
// rules when enabled; the reconciler enforces them otherwise. This handler's
// only job is to accept a CR body, apply it via server-side apply, and return
// a structured result.
package apply

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/utils"
)

// ApplyResponse is returned for every POST /api/v1/apply request.
type ApplyResponse struct {
	// Accepted is true when the CR was applied without error.
	Accepted bool `json:"accepted"`
	// Name is the CR name as stored in Kubernetes.
	Name string `json:"name,omitempty"`
	// Namespace is the CR namespace.
	Namespace string `json:"namespace,omitempty"`
	// Kind is the CR kind.
	Kind string `json:"kind,omitempty"`
	// APIVersion is the CR apiVersion.
	APIVersion string `json:"apiVersion,omitempty"`
	// ResourceVersion is the Kubernetes resourceVersion after apply.
	ResourceVersion string `json:"resourceVersion,omitempty"`
	// Message carries the rejection reason on Accepted=false.
	Message string `json:"message,omitempty"`
}

// Handler returns the http.HandlerFunc for POST /api/v1/apply.
// The auth middleware must wrap this handler before registration.
func Handler(kube kubeclient.KubeClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB cap
		if err != nil {
			utils.WriteJSON(w, http.StatusBadRequest, ApplyResponse{
				Message: "failed to read request body",
			})
			return
		}

		// Decode into unstructured so we can resolve the GVR.
		var obj unstructured.Unstructured
		if err := json.Unmarshal(body, &obj); err != nil {
			utils.WriteJSON(w, http.StatusBadRequest, ApplyResponse{
				Message: fmt.Sprintf("invalid JSON: %v", err),
			})
			return
		}

		gvr, err := resolveGVR(kube, obj.GetAPIVersion(), obj.GetKind())
		if err != nil {
			utils.WriteJSON(w, http.StatusBadRequest, ApplyResponse{
				Message: fmt.Sprintf("unknown resource: %v", err),
			})
			return
		}

		ns := obj.GetNamespace()
		result, err := kube.DynamicClient().
			Resource(gvr).
			Namespace(ns).
			Patch(r.Context(), obj.GetName(), k8stypes.ApplyPatchType, body, metav1.PatchOptions{
				FieldManager: konfig.FieldManagerGateway,
				Force:        boolPtr(true),
			})
		if err != nil {
			logger.FromContext(r.Context()).Warn().
				Str("kind", obj.GetKind()).
				Str("name", obj.GetName()).
				Err(err).
				Msg("apply API: SSA rejected")
			status := http.StatusUnprocessableEntity
			utils.WriteJSON(w, status, ApplyResponse{
				Message: err.Error(),
			})
			return
		}

		utils.WriteJSON(w, http.StatusOK, ApplyResponse{
			Accepted:        true,
			Name:            result.GetName(),
			Namespace:       result.GetNamespace(),
			Kind:            result.GetKind(),
			APIVersion:      result.GetAPIVersion(),
			ResourceVersion: result.GetResourceVersion(),
		})
	}
}

// resolveGVR maps apiVersion+kind to a GroupVersionResource via the REST mapper.
func resolveGVR(kube kubeclient.KubeClient, apiVersion, kind string) (schema.GroupVersionResource, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("invalid apiVersion %q: %w", apiVersion, err)
	}
	mapping, err := kube.Mapper().RESTMapping(schema.GroupKind{Group: gv.Group, Kind: kind}, gv.Version)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("no REST mapping for %s/%s: %w", apiVersion, kind, err)
	}
	return mapping.Resource, nil
}

func boolPtr(b bool) *bool { return &b }
