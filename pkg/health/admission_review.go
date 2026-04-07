// health/admission_review.go
package health

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ── Kubernetes AdmissionReview types ─────────────────────────────────────────
//
// These mirror the Kubernetes admissionregistration.k8s.io/v1 types.
// We declare them here to avoid importing the full k8s.io/api module
// into the health package — the health package is lightweight by design.
//
// The Kubernetes API server sends an AdmissionReview to /validate and /mutate.
// Orkestra reads the Request, evaluates the object against the Katalog's rules,
// and writes the Response. The API server reads the Response and either stores
// the object (allowed) or rejects it (denied) based on the response.
//
// Mutation webhooks (/mutate) return a JSON patch in Response.Patch.
// Validation webhooks (/validate) return allowed: true/false in Response.Allowed.
// Both may return warnings — strings shown to the user via kubectl.

// AdmissionReview is the top-level wrapper sent by the API server and
// expected in the response. The same struct is used for both request and
// response — the Request field is populated by the API server, the Response
// field is populated by Orkestra.
type AdmissionReview struct {
	// TypeMeta mirrors metav1.TypeMeta — we inline it to avoid the import.
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	// Request — populated by the API server. Contains the object being admitted.
	// Nil in the response direction.
	Request *AdmissionRequest `json:"request,omitempty"`

	// Response — populated by Orkestra. Must reference the same UID as Request.
	// Nil in the request direction.
	Response *AdmissionResponse `json:"response,omitempty"`
}

// AdmissionRequest contains the object the API server is asking about.
// Orkestra reads this to determine which CRD's rules to apply and to
// evaluate those rules against the object.
type AdmissionRequest struct {
	// UID — must be echoed verbatim in the Response.UID.
	// The API server uses this to correlate requests and responses.
	UID string `json:"uid"`

	// Kind — the GVK of the object being admitted.
	Kind metav1.GroupVersionKind `json:"kind"`

	// Resource — the GVR of the object being admitted.
	Resource metav1.GroupVersionResource `json:"resource"`

	// Name — the name of the object. May be empty for CREATE operations
	// when the name is server-generated.
	Name string `json:"name,omitempty"`

	// Namespace — the namespace of the object. Empty for cluster-scoped resources.
	Namespace string `json:"namespace,omitempty"`

	// Operation — the operation being performed: CREATE, UPDATE, DELETE, or CONNECT.
	Operation string `json:"operation"`

	// Object — the full object being admitted, as raw JSON.
	// Present for CREATE and UPDATE operations.
	// For UPDATE, this is the new (incoming) version of the object.
	Object json.RawMessage `json:"object,omitempty"`

	// OldObject — the existing object, as raw JSON.
	// Present for UPDATE and DELETE operations.
	OldObject json.RawMessage `json:"oldObject,omitempty"`

	// DryRun — true when the operation is a dry run (kubectl apply --dry-run=server).
	// Orkestra should evaluate rules but return allowed: true without side effects.
	DryRun *bool `json:"dryRun,omitempty"`
}

// AdmissionResponse is what Orkestra writes in reply to an AdmissionRequest.
type AdmissionResponse struct {
	// UID — must exactly match AdmissionRequest.UID.
	UID string `json:"uid"`

	// Allowed — true when the operation should proceed, false to reject.
	// For validation webhooks: false causes kubectl to show the Status message.
	// For mutation webhooks: should always be true (mutation doesn't reject).
	Allowed bool `json:"allowed"`

	// Status — populated when Allowed is false. Shown to the user as the
	// rejection reason. Code should be 400 for validation failures.
	Status *AdmissionStatus `json:"status,omitempty"`

	// Patch — JSON patch (RFC 6902) to apply to the object.
	// Only meaningful for mutation webhooks.
	// Must be base64-encoded. PatchType must be "JSONPatch" when set.
	Patch []byte `json:"patch,omitempty"`

	// PatchType — the type of patch. Always "JSONPatch" when Patch is set.
	PatchType *string `json:"patchType,omitempty"`

	// Warnings — strings shown to the user via kubectl as Warning: header lines.
	// Used by Orkestra for action: warn validation rules — the object is allowed
	// but the warnings surface in the user's terminal.
	// Supported by Kubernetes API server 1.19+.
	Warnings []string `json:"warnings,omitempty"`
}

// AdmissionStatus is the rejection reason returned when Allowed is false.
type AdmissionStatus struct {
	// Message — the human-readable rejection message shown by kubectl.
	Message string `json:"message"`

	// Code — HTTP status code. Use 400 for validation failures.
	Code int32 `json:"code"`
}

// jsonPatchType is the constant string for the JSON patch type.
const jsonPatchType = "JSONPatch"

// ptrString returns a pointer to a string. Used for PatchType.
func ptrString(s string) *string { return &s }
