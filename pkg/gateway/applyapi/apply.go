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
package applyapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
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
	// Warnings is a list of advisory messages returned by the admission webhook
	// even when the request was accepted. Mirrors the kubectl warning experience.
	Warnings []string `json:"warnings,omitempty"`
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

// warningCapture collects Kubernetes API server warnings from a single request.
// It implements rest.WarningHandler and is safe for concurrent use.
type warningCapture struct {
	mu       sync.Mutex
	warnings []string
}

func (c *warningCapture) HandleWarningHeader(_ int, _ string, message string) {
	c.mu.Lock()
	c.warnings = append(c.warnings, message)
	c.mu.Unlock()
}

func (c *warningCapture) collect() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.warnings...)
}

// scopedDynamic returns a dynamic client that routes warnings to capture
// while reusing the existing HTTP transport from kube.
func scopedDynamic(kube kubeclient.KubeClient, capture *warningCapture) dynamic.Interface {
	cfg := kube.RestConfig()
	if cfg == nil {
		return kube.DynamicClient()
	}
	copy := *cfg
	copy.WarningHandler = capture
	d, err := dynamic.NewForConfig(&copy)
	if err != nil {
		return kube.DynamicClient()
	}
	return d
}

// Handler returns the http.HandlerFunc for POST /api/v1/apply.
// The auth middleware must wrap this handler before registration.
func applyHandler(kube kubeclient.KubeClient, lookup func(kind string) *orktypes.CRDEntry, notes orktypes.NoteRegistry) http.HandlerFunc {
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

		crd := lookup(obj.GetKind())

		// CRD-level forceConflict is a katalog declaration; ?overwrite=true is a
		// per-request override. Either one sets Force=true.
		if !overwrite && crd != nil && crd.IDP != nil {
			overwrite = crd.IDP.ForceConflict
		}

		patchOpts := metav1.PatchOptions{
			FieldManager: konfig.FieldManagerGateway,
			Force:        boolPtr(overwrite),
		}
		if dryRun {
			patchOpts.DryRun = []string{metav1.DryRunAll}
		}

		// idp.namespace, once declared, always decides — it overrides
		// whatever (if anything) the client sent, so namespace stops being
		// something any Apply API caller (Control Center, curl, CI) needs to
		// know or supply. Resolved against exactly what the client submitted
		// (labels/annotations/spec), same resolver+notes pattern the
		// admission webhook uses for validation.rules.
		if crd != nil && crd.HasIDPNamespace() {
			resolver := orktmpl.NewResolverFromMap(obj.Object).WithUserNotes(notes)
			if resolved, err := resolver.Resolve(crd.IDP.Namespace); err == nil && resolved != "" {
				obj.SetNamespace(resolved)
			} else {
				logger.FromContext(r.Context()).Warn().
					Str("kind", obj.GetKind()).
					Str("idp.namespace", crd.IDP.Namespace).
					Err(err).
					Msg("apply API: idp.namespace did not resolve to a value")
			}
		}

		ns := obj.GetNamespace()
		capture := &warningCapture{}
		result, err := scopedDynamic(kube, capture).
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
			violations := extractViolations(err)
			// err.Error() is the raw K8s-wrapped string — right for kubectl
			// (which just prints whatever the API server returns, unaffected
			// by anything here), but verbose here. Prefer the first violation's message
			// (same headline convention as ValidationResult.DenialMessage() — full detail
			// still lives in violations[] either way); fall back to err.Error()
			// only when there's nothing structured to summarize instead, e.g.
			// a plain 404 with no admission causes at all.
			message := err.Error()
			if len(violations) > 0 && violations[0].Message != "" {
				message = violations[0].Message
			}
			utils.WriteJSON(w, http.StatusUnprocessableEntity, ApplyResponse{
				DryRun:     dryRun,
				Message:    message,
				Warnings:   capture.collect(),
				Violations: violations,
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
			Warnings:        capture.collect(),
		})
	}
}

// mapperRefresher is satisfied by kubeclient.KubeClient's real implementation,
// deliberately not the interface itself — RefreshMapper is a real-cluster
// discovery-cache concern that simulate's fake client has no reason to
// implement, so resolveGVR type-asserts for it instead of widening the
// shared interface.
type mapperRefresher interface {
	RefreshMapper()
}

// resolveGVR maps apiVersion+kind to a GroupVersionResource via the REST
// mapper, retrying once after a cache refresh on a "no matches" error.
//
// The gateway's REST mapper is a DeferredDiscoveryRESTMapper built once at
// startup — it caches whatever CRDs existed at that point and is never told
// about ones applied afterward. The runtime avoids this because its
// reconciler explicitly calls RefreshMapper() from its own post-start hooks
// (pkg/runtime/kordinator/post_start_hooks.go) once it notices a CRD it was
// waiting on now exists. The gateway has no equivalent watch loop, so
// without this retry, an Apply API request for a CRD applied after the
// gateway pod started 404s forever — "no REST mapping" / "the server could
// not find the requested resource" — until the pod restarts and builds a
// fresh cache.
func resolveGVR(kube kubeclient.KubeClient, apiVersion, kind string) (schema.GroupVersionResource, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("invalid apiVersion %q: %w", apiVersion, err)
	}
	gk := schema.GroupKind{Group: gv.Group, Kind: kind}
	mapping, err := kube.Mapper().RESTMapping(gk, gv.Version)
	if err != nil && meta.IsNoMatchError(err) {
		if r, ok := kube.(mapperRefresher); ok {
			r.RefreshMapper()
			mapping, err = kube.Mapper().RESTMapping(gk, gv.Version)
		}
	}
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
