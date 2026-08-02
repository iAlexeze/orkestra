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
	"strings"
	"sync"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

	"github.com/orkspace/orkestra/pkg/katalog"
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
	// PollURL is the GET /api/v1/resources/{kind}/{namespace}/{name} path for
	// this CR — callers can jq it straight out of the response instead of
	// hand-assembling it from Kind/Namespace/Name.
	PollURL string `json:"pollUrl,omitempty"`
	// Message carries the rejection reason on Accepted=false.
	Message string `json:"message,omitempty"`
	// Warnings is a list of advisory messages returned by the admission webhook
	// even when the request was accepted. Mirrors the kubectl warning experience.
	Warnings []string `json:"warnings,omitempty"`
	// Violations is a structured list of field-level errors from admission or
	// validation. Populated when Accepted=false and kubernetes returns Status details.
	Violations []ApplyViolation `json:"violations,omitempty"`

	// Payload carries the platform team's curated view of the submitted CR,
	// evaluated from idp.config.response at apply time.
	//
	// At apply time .status is not yet available — payload fields that
	// reference status resolve to "". The caller should poll
	// GET /api/v1/resources/{kind}/{ns}/{name} for live status values.
	//
	// Omitted when the CRD has no idp.config.response declared.
	Payload map[string]interface{} `json:"payload,omitempty"`
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

// applyHandler returns the http.HandlerFunc for POST /api/v1/apply.
// The auth middleware must wrap this handler before registration.
func applyHandler(
	kube kubeclient.KubeClient,
	kat *katalog.Katalog,
	notes orktypes.NoteRegistry,
) http.HandlerFunc {
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

		// Decode into a raw map first to detect target vs full CR mode.
		var raw map[string]interface{}
		if err := json.Unmarshal(body, &raw); err != nil {
			utils.WriteJSON(w, http.StatusBadRequest, ApplyResponse{
				Message: fmt.Sprintf("invalid JSON: %v", err),
			})
			return
		}

		dryRun := r.URL.Query().Get("dryRun") == "true"
		overwrite := r.URL.Query().Get("overwrite") == "true"

		var (
			obj *unstructured.Unstructured
			crd *orktypes.CRDEntry
			gvr schema.GroupVersionResource
		)

		// ── Format detection ──────────────────────────────────────────────────
		if isTargetRequest(raw) {
			// ── Target mode ───────────────────────────────────────────────────
			// Caller submitted flat fields alongside a target identifier.
			// The gateway builds the full CR from the IDP field declaration.
			target, _ := raw["target"].(string)
			if strings.TrimSpace(target) == "" {
				utils.WriteJSON(w, http.StatusBadRequest, ApplyResponse{
					Message: `"target" must be a non-empty string`,
				})
				return
			}

			crd = kat.LookupByTarget(target)
			if crd == nil {
				utils.WriteJSON(w, http.StatusBadRequest, ApplyResponse{
					Message: fmt.Sprintf(
						"unknown target %q — available: %s",
						target,
						strings.Join(kat.AvailableTargets(), ", "),
					),
				})
				return
			}

			built, err := BuildCRFromTarget(raw, crd, notes)
			if err != nil {
				utils.WriteJSON(w, http.StatusBadRequest, ApplyResponse{
					Message: err.Error(),
				})
				return
			}
			obj = built
			gvr = crd.GVR()

		} else {
			// ── Full CR mode ──────────────────────────────────────────────────
			// Caller provided a complete Kubernetes CR. Unmarshal directly.
			var full unstructured.Unstructured
			if err := json.Unmarshal(body, &full); err != nil {
				utils.WriteJSON(w, http.StatusBadRequest, ApplyResponse{
					Message: fmt.Sprintf("invalid CR JSON: %v", err),
				})
				return
			}

			if full.GetKind() == "" || full.GetAPIVersion() == "" {
				utils.WriteJSON(w, http.StatusBadRequest, ApplyResponse{
					Message: `request must include "apiVersion" and "kind" in full CR mode`,
				})
				return
			}

			crd = kat.LookupByKind(full.GetKind())
			if crd == nil {
				utils.WriteJSON(w, http.StatusBadRequest, ApplyResponse{
					Message: fmt.Sprintf("unknown kind %q", full.GetKind()),
				})
				return
			}

			if crd == nil || !crd.IDPEnabled() {
				http.Error(w, fmt.Sprintf("idp not enabled for %q", full.GetKind()), http.StatusBadRequest)
				return
			}

			// Resolve idp.name and idp.namespace when declared on the CRD.
			// In full CR mode the submitted spec is the resolver data source.
			obj = &full
			if err := resolveIDPMeta(obj, crd, notes); err != nil {
				utils.WriteJSON(w, http.StatusBadRequest, ApplyResponse{
					Message: err.Error(),
				})
				return
			}

			// Resolve GVR from the CRD entry.
			gvr = crd.GVR()
		}

		// ── Token permission check ────────────────────────────────────────────────
		// When the CRD declares idp.allowedTokens, the authenticated token must
		// have permission to perform the operation it is attempting.
		//
		// We determine whether this is a create or update by probing the API server
		// for the resource before applying. SSA would succeed either way, but the
		// permission model distinguishes them so platform teams can allow CI to
		// create staging CRs without allowing it to overwrite production ones.
		tokenName := TokenNameFromContext(r.Context())
		if crd != nil && crd.IDP != nil && crd.IDP.HasTokenRestrictions() {
			// Determine create vs update before the SSA call.
			op := orktypes.IDPOpCreate
			_, probeErr := kube.DynamicClient().
				Resource(gvr).
				Namespace(obj.GetNamespace()).
				Get(r.Context(), obj.GetName(), metav1.GetOptions{})
			if probeErr == nil {
				op = orktypes.IDPOpUpdate
			}

			allowed, reason := crd.IDP.TokenAllowed(tokenName, op, obj.GetNamespace())
			if !allowed {
				msg := reason.Message(tokenName, op, obj.GetKind(), obj.GetNamespace())
				utils.WriteJSON(w, http.StatusForbidden, ApplyResponse{
					Message: msg,
					Violations: []ApplyViolation{{
						Field:    "metadata",
						Message:  msg,
						Severity: "error",
					}},
				})
				return
			}
		}

		// CRD-level forceConflict is a katalog declaration; ?overwrite=true is a
		// per-request override. Either one sets Force=true.
		if !overwrite && crd != nil && crd.IDP != nil {
			overwrite = crd.IDP.ForceConflict
		}

		// idp.name, once declared, always decides — it overrides whatever
		// (if anything) the client sent. Resolved against exactly what the
		// client submitted (labels/annotations/spec), same resolver+notes
		// pattern the admission webhook uses for validation.rules. When it
		// is not declared, a name is required from the caller — reject here
		// with a structured violation instead of letting the SSA patch fail
		// with a raw "metadata.name is required" from the API server.
		// Note: In target mode, BuildCRFromTarget already handles this.
		// In full CR mode, resolveIDPMeta handles it above.
		if obj.GetName() == "" {
			utils.WriteJSON(w, http.StatusUnprocessableEntity, ApplyResponse{
				DryRun:  dryRun,
				Message: "name is required",
				Violations: []ApplyViolation{{
					Field:    "metadata.name",
					Message:  "name is required",
					Severity: "error",
				}},
			})
			return
		}

		// idp.namespace works the same way as idp.name above, so namespace
		// stops being something any Apply API caller (Control Center, curl,
		// CI) needs to know or supply.
		if obj.GetNamespace() == "" {
			utils.WriteJSON(w, http.StatusUnprocessableEntity, ApplyResponse{
				DryRun:  dryRun,
				Message: "namespace is required",
				Violations: []ApplyViolation{{
					Field:    "metadata.namespace",
					Message:  "namespace is required",
					Severity: "error",
				}},
			})
			return
		}

		patchOpts := metav1.PatchOptions{
			FieldManager: konfig.FieldManagerGateway,
			Force:        boolPtr(overwrite),
		}
		if dryRun {
			patchOpts.DryRun = []string{metav1.DryRunAll}
		}

		// Evaluate idp.config.response.payload against the submitted CR.
		// At apply time the stored result is not yet available — we evaluate
		// against the body we just applied. The caller can poll GET for live values.
		payload := EvaluatePayload(obj.Object, crd, notes)

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
				Payload:    payload,
			})
			return
		}

		utils.WriteJSON(w, http.StatusOK, ApplyResponse{
			Accepted:   true,
			DryRun:     dryRun,
			Name:       result.GetName(),
			Namespace:  result.GetNamespace(),
			Kind:       result.GetKind(),
			APIVersion: result.GetAPIVersion(),
			PollURL:    resourcePath(result.GetKind(), result.GetNamespace(), result.GetName()),
			Warnings:   capture.collect(),
			Payload:    payload,
		})
	}
}

// resolveIDPMeta resolves idp.name and idp.namespace on a full CR in-place.
// Called in full CR mode when the platform team has set these expressions on
// the CRD — the submitted spec fields are the resolver data source.
func resolveIDPMeta(
	obj *unstructured.Unstructured,
	crd *orktypes.CRDEntry,
	notes orktypes.NoteRegistry,
) error {
	if crd.IDP == nil {
		return nil
	}

	// Build resolver from the submitted spec so expressions like
	// `{{ .repository }}` resolve against spec fields.
	data := map[string]interface{}{}
	if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
		for k, v := range spec {
			data[k] = v
		}
	}
	resolver := orktmpl.NewResolverFromMap(data).WithUserNotes(notes)

	if crd.HasIDPName() {
		name, err := resolver.Resolve(crd.IDP.Name)
		if err != nil || strings.TrimSpace(name) == "" {
			return fmt.Errorf(
				"idp.name %q could not be resolved: %w",
				crd.IDP.Name, err,
			)
		}
		obj.SetName(strings.TrimSpace(name))
	}

	if crd.HasIDPNamespace() {
		ns, err := resolver.Resolve(crd.IDP.Namespace)
		if err != nil || strings.TrimSpace(ns) == "" {
			return fmt.Errorf(
				"idp.namespace %q could not be resolved: %w",
				crd.IDP.Namespace, err,
			)
		}
		obj.SetNamespace(strings.TrimSpace(ns))
	}

	return nil
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
