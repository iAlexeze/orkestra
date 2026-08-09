// pkg/gateway/apply/handler.go
//
// POST /api/v1/apply
//
// Pipeline: auth (handled by middleware) → decode body → SSA → translate response.
//
// The Gateway API does not duplicate admission logic. Webhooks enforce admission
// rules when enabled; the reconciler enforces them otherwise. This handler's
// only job is to accept a CR body, apply it via server-side apply, and return
// a structured result.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
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
	// PollURL is the URL where the caller can GET the full state of this resource.
	// By default, it points to /api/v1/resources/{kind}/{namespace}/{name}.
	//
	// The platform team can override this via serve.config.response.poll:
	//   - poll.field:   appends ?field=<value> for lightweight polling
	//   - poll.url:     replaces the URL entirely with a custom template
	//
	// Callers should use this URL for subsequent GET requests instead of
	// assembling it themselves from Kind/Namespace/Name.
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
	// evaluated from serve.config.response at apply time.
	//
	// At apply time .status is not yet available — payload fields that
	// reference status resolve to "". The caller should poll
	// GET /api/v1/resources/{kind}/{ns}/{name} for live status values.
	//
	// Omitted when the CRD has no serve.config.response declared.
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
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed", "only POST requests are supported")
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB cap
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ApplyResponse{
				Message: "failed to read request body",
			})
			return
		}

		// Decode into a raw map first to detect target vs full CR mode.
		var raw map[string]interface{}
		if err := json.Unmarshal(body, &raw); err != nil {
			writeJSON(w, http.StatusBadRequest, ApplyResponse{
				Message: fmt.Sprintf("invalid JSON: %v", err),
			})
			return
		}

		dryRun := r.URL.Query().Get("dryRun") == "true"
		overwrite := r.URL.Query().Get("overwrite") == "true"
		override := r.URL.Query().Get("override") == "true"

		var (
			obj       *unstructured.Unstructured
			crd       *orktypes.CRDEntry
			alias     string // non-empty when caller used an alias
			gvr       schema.GroupVersionResource
			patchBody []byte
		)

		// ── Format detection ──────────────────────────────────────────────────
		if isTargetRequest(raw) {
			// ── Target mode ───────────────────────────────────────────────────
			// Caller submitted flat fields alongside a target identifier.
			// The gateway builds the full CR from the serve field declaration.
			target, _ := raw["target"].(string)
			if strings.TrimSpace(target) == "" {
				writeJSON(w, http.StatusBadRequest, ApplyResponse{
					Message: `"target" must be a non-empty string`,
				})
				return
			}

			resolution := kat.LookupByTargetOrAlias(target)
			if resolution == nil {
				writeJSON(w, http.StatusBadRequest, ApplyResponse{
					Message: fmt.Sprintf(
						"unknown target %q — available: %s",
						target,
						strings.Join(kat.AvailableTargets(), ", "),
					),
				})
				return
			}
			crd = resolution.CRD
			alias = resolution.Alias

			built, err := BuildCRFromTarget(raw, crd, notes)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, ApplyResponse{
					Message: err.Error(),
				})
				return
			}
			obj = built
			InjectProvenanceAnnotations(obj, crd.ServeTarget(), alias, OIDCSubFromContext(r.Context()))
			InjectIntentAnnotation(obj, raw)
			gvr = crd.GVR()

			// ─── Marshal built CR for the patch body ──────────────────────
			patchBody, err = json.Marshal(obj.Object)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, ApplyResponse{
					Message: "failed to marshal built CR",
				})
				return
			}

		} else {
			// ── Full CR mode ──────────────────────────────────────────────────
			// Built from the already-decoded raw map.
			full := unstructured.Unstructured{Object: raw}

			if full.GetKind() == "" || full.GetAPIVersion() == "" {
				writeJSON(w, http.StatusBadRequest, ApplyResponse{
					Message: `request must include either "target" (target mode) or "apiVersion" and "kind" (full CR mode)`,
				})
				return
			}

			crd = kat.LookupByKind(full.GetKind()).Entry()
			if crd == nil {
				writeJSON(w, http.StatusBadRequest, ApplyResponse{
					Message: fmt.Sprintf("unknown kind %q", full.GetKind()),
				})
				return
			}

			if !crd.ServeEnabled() {
				writeJSONError(w, http.StatusBadRequest, "serve not enabled",
					fmt.Sprintf("Serve is not enabled for kind %q", full.GetKind()),
				)
				return
			}

			obj = &full
			InjectProvenanceAnnotations(obj, crd.ServeTarget(), "", OIDCSubFromContext(r.Context()))
			gvr = crd.GVR()
			patchBody = body // raw body is already a valid CR

			// Resolve serve.name and serve.namespace when declared on the CRD.
			if err := resolveServeMeta(obj, crd, notes); err != nil {
				writeJSON(w, http.StatusBadRequest, ApplyResponse{
					Message: err.Error(),
				})
				return
			}
		}

		// ── Token permission check ────────────────────────────────────────────────
		// TokenAllowedFor resolves alias-specific tokens before falling back to
		// CRD-level tokens, then delegates to ServeConfig.TokenAllowed.
		// When no restrictions are declared at either level, every valid token passes.
		tokenName := TokenNameFromContext(r.Context())
		if crd != nil {
			// Determine create vs update before the SSA call.
			op := orktypes.ServeOpCreate
			existing, probeErr := kube.DynamicClient().
				Resource(gvr).
				Namespace(obj.GetNamespace()).
				Get(r.Context(), obj.GetName(), metav1.GetOptions{})
			if probeErr == nil {
				op = orktypes.ServeOpUpdate

				// ── Routing surface guard ─────────────────────────────────────
				// The CR's stamped surface (alias > target annotation) is immutable
				// without ?override=true.
				ann := existing.GetAnnotations()
				storedSurface := ann[labels.AnnotationServeAlias]
				if storedSurface == "" {
					storedSurface = ann[labels.AnnotationServeTarget]
				}
				incomingSurface := alias
				if incomingSurface == "" {
					incomingSurface = crd.ServeTarget()
				}
				if storedSurface != "" && storedSurface != incomingSurface {
					if !override {
						msg := fmt.Sprintf(
							"routing surface conflict: resource was created via %q, cannot update via %q without ?override=true",
							storedSurface, incomingSurface,
						)
						writeJSON(w, http.StatusConflict, ApplyResponse{
							Message: msg,
							Violations: []ApplyViolation{{
								Field:    "metadata.annotations",
								Message:  msg,
								Severity: "error",
							}},
						})
						return
					}
					logger.FromContext(r.Context()).Warn().
						Str("from", storedSurface).
						Str("to", incomingSurface).
						Str("name", obj.GetName()).
						Str("namespace", obj.GetNamespace()).
						Msg("routing surface override")
				}
			}

			allowed, reason := crd.TokenAllowedFor(alias, tokenName, op, obj.GetNamespace(), orktypes.ServeClassResources)
			if !allowed {
				msg := reason.Message(tokenName, op, obj.GetKind(), obj.GetNamespace())
				writeJSON(w, http.StatusForbidden, ApplyResponse{
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
		if !overwrite && crd != nil && crd.Serve != nil {
			overwrite = crd.Serve.ForceConflict
		}

		// serve.name, once declared, always decides — it overrides whatever
		// (if anything) the client sent. Resolved against exactly what the
		// client submitted (labels/annotations/spec), same resolver+notes
		// pattern the admission webhook uses for validation.rules. When it
		// is not declared, a name is required from the caller — reject here
		// with a structured violation instead of letting the SSA patch fail
		// with a raw "metadata.name is required" from the API server.
		// Note: In target mode, BuildCRFromTarget already handles this.
		// In full CR mode, resolveServeMeta handles it above.
		if obj.GetName() == "" {
			writeJSON(w, http.StatusUnprocessableEntity, ApplyResponse{
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

		// serve.namespace works the same way as serve.name above, so namespace
		// stops being something any Gateway API caller (Control Center, curl,
		// CI) needs to know or supply.
		if obj.GetNamespace() == "" {
			writeJSON(w, http.StatusUnprocessableEntity, ApplyResponse{
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

		// Evaluate serve.config.response.payload against the submitted CR.
		payload := EvaluatePayload(obj.Object, crd, alias, notes)

		ns := obj.GetNamespace()
		capture := &warningCapture{}
		result, err := scopedDynamic(kube, capture).
			Resource(gvr).
			Namespace(ns).
			Patch(r.Context(), obj.GetName(), k8stypes.ApplyPatchType, patchBody, patchOpts)
		if err != nil {
			// ─── Capture detailed error information ──────────────────────────────
			errorDetails := map[string]interface{}{
				"kind":       obj.GetKind(),
				"name":       obj.GetName(),
				"namespace":  obj.GetNamespace(),
				"gvr":        gvr.String(),
				"apiVersion": obj.GetAPIVersion(),
				"error":      err.Error(),
			}

			// Log the full error details
			logger.FromContext(r.Context()).Warn().
				Str("kind", obj.GetKind()).
				Str("name", obj.GetName()).
				Bool("dryRun", dryRun).
				Str("gvr", gvr.String()).
				Str("namespace", obj.GetNamespace()).
				Err(err).
				Interface("errorDetails", errorDetails).
				Msg("gateway API: SSA rejected")

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

			writeJSON(w, http.StatusUnprocessableEntity, ApplyResponse{
				DryRun:     dryRun,
				Message:    message,
				Warnings:   capture.collect(),
				Violations: violations,
				// Add the error details to the response for debugging
				Payload: map[string]interface{}{
					"debug": errorDetails,
				},
			})
			return
		}

		// Resolve configured polling url and field
		resolver := orktmpl.NewResolverFromMap(raw).WithUserNotes(notes)
		pollURL := resolvePollURL(
			result.GetKind(),
			result.GetNamespace(),
			result.GetName(),
			crd.GetServePollingConfig(),
			resolver,
		)

		writeJSON(w, http.StatusOK, ApplyResponse{
			Accepted:   true,
			DryRun:     dryRun,
			Name:       result.GetName(),
			Namespace:  result.GetNamespace(),
			Kind:       result.GetKind(),
			APIVersion: result.GetAPIVersion(),
			PollURL:    pollURL,
			Warnings:   capture.collect(),
			Payload:    payload,
		})
	}
}

// resolveServeMeta resolves serve.name and serve.namespace on a full CR in-place.
// Called in full CR mode when the platform team has set these expressions on
// the CRD — the submitted spec fields are the resolver data source.
func resolveServeMeta(
	obj *unstructured.Unstructured,
	crd *orktypes.CRDEntry,
	notes orktypes.NoteRegistry,
) error {
	if crd.Serve == nil {
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

	if crd.HasServeName() {
		name, err := resolver.Resolve(crd.Serve.Name)
		if err != nil || strings.TrimSpace(name) == "" {
			return fmt.Errorf(
				"serve.name %q could not be resolved: %w",
				crd.Serve.Name, err,
			)
		}
		obj.SetName(strings.TrimSpace(name))
	}

	if crd.HasServeNamespace() {
		ns, err := resolver.Resolve(crd.Serve.Namespace)
		if err != nil || strings.TrimSpace(ns) == "" {
			return fmt.Errorf(
				"serve.namespace %q could not be resolved: %w",
				crd.Serve.Namespace, err,
			)
		}
		obj.SetNamespace(strings.TrimSpace(ns))
	}

	return nil
}

func boolPtr(b bool) *bool { return &b }

// InjectProvenanceAnnotations stamps the three serve provenance annotations onto
// obj before SSA. Empty strings are skipped — callers pass "" for fields that
// don't apply to the current request path (e.g. alias is "" on primary targets).
func InjectProvenanceAnnotations(obj *unstructured.Unstructured, target, alias, source string) {
	ann := obj.GetAnnotations()
	if ann == nil {
		ann = make(map[string]string, 3)
	}
	if target != "" {
		ann[labels.AnnotationServeTarget] = target
	}
	if alias != "" {
		ann[labels.AnnotationServeAlias] = alias
	}
	if source != "" {
		ann[labels.AnnotationServeSource] = source
	}
	obj.SetAnnotations(ann)
}

// InjectIntentAnnotation stores the raw intent payload as a JSON-encoded
// annotation so the admission webhook can bind it as .request in validation
// rules, enabling intent-level gates before field translation.
func InjectIntentAnnotation(obj *unstructured.Unstructured, raw map[string]interface{}) {
	b, err := json.Marshal(raw)
	if err != nil {
		return
	}
	ann := obj.GetAnnotations()
	if ann == nil {
		ann = make(map[string]string, 1)
	}
	ann[labels.AnnotationServeIntent] = string(b)
	obj.SetAnnotations(ann)
}

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
