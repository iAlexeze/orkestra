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
	"errors"
	"fmt"
	"io"
	"net/http"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
)

// ApplyResponse is returned for every POST /api/v1/apply request.
type ApplyResponse struct {
	// Accepted is true when the CR was applied (or would be applied, for dry runs) without error.
	Accepted bool `json:"accepted"`
	// DryRun is true when the request was a dry-run preview (?dryRun=true).
	DryRun bool `json:"dryRun,omitempty"`
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
	// Violations is a structured list of field-level errors from admission or
	// validation. Populated when Accepted=false and kubernetes returns Status details.
	Violations []ApplyViolation `json:"violations,omitempty"`
}

// ApplyViolation is one field-level error returned from Kubernetes on a failed apply.
type ApplyViolation struct {
	// Field is the dot-notation path to the offending field (e.g. "spec.environment").
	Field string `json:"field,omitempty"`
	// Message is the human-readable error from Kubernetes or the admission webhook.
	Message string `json:"message"`
	// Severity is "error" (blocks apply) or "warning" (informational).
	Severity string `json:"severity,omitempty"`
}

// Handler returns the http.HandlerFunc for POST /api/v1/apply.
// The auth middleware must wrap this handler before registration.
func Handler(kube kubeclient.KubeClient, lookup func(kind string) *orktypes.CRDEntry) http.HandlerFunc {
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

		dryRun := r.URL.Query().Get("dryRun") == "true"
		overwrite := r.URL.Query().Get("overwrite") == "true"

		// CRD-level forceConflict is a katalog declaration; ?overwrite=true is a
		// per-request override. Either one sets Force=true.
		if !overwrite {
			if crd := lookup(obj.GetKind()); crd != nil && crd.IDP != nil {
				overwrite = crd.IDP.ForceConflict
			}
		}

		patchOpts := metav1.PatchOptions{
			FieldManager: konfig.FieldManagerGateway,
			Force:        boolPtr(overwrite),
		}
		if dryRun {
			patchOpts.DryRun = []string{metav1.DryRunAll}
		}

		ns := obj.GetNamespace()
		result, err := kube.DynamicClient().
			Resource(gvr).
			Namespace(ns).
			Patch(r.Context(), obj.GetName(), k8stypes.ApplyPatchType, body, patchOpts)
		if err != nil {
			logger.FromContext(r.Context()).Warn().
				Str("kind", obj.GetKind()).
				Str("name", obj.GetName()).
				Bool("dryRun", dryRun).
				Err(err).
				Msg("apply API: SSA rejected")
			utils.WriteJSON(w, http.StatusUnprocessableEntity, ApplyResponse{
				DryRun:     dryRun,
				Message:    err.Error(),
				Violations: extractViolations(err),
			})
			return
		}

		utils.WriteJSON(w, http.StatusOK, ApplyResponse{
			Accepted:        true,
			DryRun:          dryRun,
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

// extractViolations pulls field-level causes out of a Kubernetes Status error.
func extractViolations(err error) []ApplyViolation {
	var statusErr *k8serrors.StatusError
	if !errors.As(err, &statusErr) || statusErr.ErrStatus.Details == nil {
		return nil
	}
	causes := statusErr.ErrStatus.Details.Causes
	if len(causes) == 0 {
		return nil
	}
	vs := make([]ApplyViolation, 0, len(causes))
	for _, c := range causes {
		v := ApplyViolation{
			Field:   c.Field,
			Message: c.Message,
		}
		if c.Type == metav1.CauseTypeFieldValueInvalid ||
			c.Type == metav1.CauseTypeFieldValueRequired ||
			c.Type == metav1.CauseTypeFieldValueDuplicate ||
			c.Type == metav1.CauseTypeFieldValueNotSupported {
			v.Severity = "error"
		} else {
			v.Severity = "warning"
		}
		vs = append(vs, v)
	}
	return vs
}
