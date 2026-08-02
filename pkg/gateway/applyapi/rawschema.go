package applyapi

import (
	"net/http"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RawSchemaResponse is returned by GET /api/v1/raw-schema?kind=<k>.
//
// It exposes the raw Kubernetes OpenAPI v3 schema for a CRD alongside the IDP
// metadata destinations (labels, annotations). The caller receives the full
// structural picture and decides how to map their fields — no Orkestra
// abstraction is applied.
//
// Intended for:
//   - Callers who want direct access to the CRD spec without IDP field config.
//   - Developers building tooling that understands Kubernetes CR structure.
//   - Advanced use cases where idp.fields is not declared or is deliberately
//     omitted in favour of letting the caller construct the full CR.
//
// Companion: POST /api/v1/apply in full CR mode accepts a complete CR whose
// structure matches what this endpoint describes.
type RawSchemaResponse struct {
	// Kind is the resolved CRD kind string.
	Kind string `json:"kind"`

	// APIVersion is the CRD's group/version.
	APIVersion string `json:"apiVersion"`

	// Spec contains the OpenAPI v3 schema for the CRD's spec field.
	// Properties describes each spec field; Required lists the mandatory ones.
	Spec RawSchemaSection `json:"spec"`

	// Labels describes fields the IDP expects in metadata.labels, when declared
	// in idp.additionalFields.labels. Empty when not configured.
	Labels map[string]orktypes.IDPFieldConfig `json:"labels,omitempty"`

	// Annotations describes fields the IDP expects in metadata.annotations,
	// when declared in idp.additionalFields.annotations. Empty when not configured.
	Annotations map[string]orktypes.IDPFieldConfig `json:"annotations,omitempty"`
}

// RawSchemaSection is the spec portion of the raw schema.
type RawSchemaSection struct {
	// Properties is the raw OpenAPI v3 properties map for the spec.
	// Values are the schema objects as returned by the Kubernetes API.
	Properties map[string]interface{} `json:"properties,omitempty"`

	// Required lists the spec field names that are required.
	Required []string `json:"required,omitempty"`
}

// rawSchemaHandler serves GET /api/v1/raw-schema?kind=<k>.
//
// It fetches the CRD definition from the Kubernetes API server and returns the
// spec schema alongside any IDP-declared label and annotation fields. The
// caller sees all four CR destinations (spec, labels, annotations, and implicit
// metadata like name/namespace) and is responsible for mapping their data.
//
// Unlike GET /api/v1/schema?target=<t>, this endpoint:
//   - Requires ?kind= (the Kubernetes kind string, not the IDP target).
//   - Returns raw OpenAPI properties, not the curated IDPFieldConfig list.
//   - Works even when idp.fields is not declared on the CRD entry.
//   - Is intended for callers who will use full CR mode in POST /api/v1/apply.
func rawSchemaHandler(
	kube kubeclient.KubeClient,
	kat *katalog.Katalog,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		kind := r.URL.Query().Get("kind")
		if kind == "" {
			http.Error(w, `"kind" query parameter is required`, http.StatusBadRequest)
			return
		}
		apiVersion := r.URL.Query().Get("apiVersion")

		var crd *orktypes.CRDEntry
		if apiVersion != "" && kind != "" {
			crd = kat.LookupByAPIVersionAndKind(apiVersion, kind)
		} else {
			crd = kat.LookupByKind(kind)
		}

		// Resolve the CRD entry.
		if crd == nil {
			http.Error(w, "kind not found", http.StatusNotFound)
			return
		}

		// Permission check — schema class, get operation.
		tokenName := TokenNameFromContext(r.Context())
		if crd.IDP != nil && crd.IDP.HasTokenRestrictions() {
			allowed, reason := crd.IDP.TokenAllowed(
				tokenName, orktypes.IDPOpGet, "", orktypes.IDPClassSchema,
			)
			if !allowed {
				http.Error(
					w,
					reason.Message(tokenName, orktypes.IDPOpGet, kind, ""),
					http.StatusForbidden,
				)
				return
			}
		}

		// Fetch the CRD definition from the Kubernetes API.
		crdGVR := schema.GroupVersionResource{
			Group:    "apiextensions.k8s.io",
			Version:  "v1",
			Resource: "customresourcedefinitions",
		}

		// CRD name is <plural>.<group>.
		crdName := crd.APITypes.Plural + "." + crd.APITypes.Group
		obj, err := kube.DynamicClient().
			Resource(crdGVR).
			Get(r.Context(), crdName, metav1.GetOptions{})
		if err != nil {
			writeKubeError(w, err)
			return
		}

		// Navigate to the OpenAPI schema for the storage version.
		// The storage version is the one with storage: true in the CRD spec.
		props, required := extractStorageSpecSchema(obj.Object)

		resp := RawSchemaResponse{
			Kind:       kind,
			APIVersion: crd.APIVersion(),
			Spec: RawSchemaSection{
				Properties: props,
				Required:   required,
			},
		}

		// Attach IDP label and annotation fields when declared.
		// These are not in the Kubernetes schema — they are Orkestra extensions.
		if crd.IDP != nil {
			if labels := crd.AdditionalLabelFields(); len(labels) > 0 {
				resp.Labels = labels
			}
			if annotations := crd.AdditionalAnnotationFields(); len(annotations) > 0 {
				resp.Annotations = annotations
			}
		}

		utils.WriteJSON(w, http.StatusOK, resp)
	}
}

// extractStorageSpecSchema navigates the CRD unstructured object to find the
// OpenAPI v3 spec schema for the storage version (storage: true).
//
// Returns nil, nil when the storage version cannot be found — callers render
// an empty properties map rather than an error.
func extractStorageSpecSchema(
	crdObj map[string]interface{},
) (properties map[string]interface{}, required []string) {
	versions, ok := utils.NestedSlice(crdObj, "spec", "versions")
	if !ok {
		return nil, nil
	}

	for _, v := range versions {
		ver, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this is the storage version.
		storage, _ := ver["storage"].(bool)
		if !storage {
			continue
		}

		// Navigate to the spec schema.
		specSchema, ok := utils.NestedMap(ver, "schema", "openAPIV3Schema", "properties", "spec")
		if !ok {
			return nil, nil
		}

		props, _ := specSchema["properties"].(map[string]interface{})
		if props == nil {
			return nil, nil
		}

		var req []string
		if reqRaw, ok := specSchema["required"].([]interface{}); ok {
			for _, r := range reqRaw {
				if s, ok := r.(string); ok {
					req = append(req, s)
				}
			}
		}
		return props, req
	}

	return nil, nil
}
