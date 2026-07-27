// pkg/gateway/schema/handler.go
//
// GET /api/v1/schema/{kind}   — spec properties + idpFields for one CRD
// GET /api/v1/schema/         — catalog: list of all IDP-enabled CRDs
//
// Only served for CRDs where idp.enabled: true.
// The Control Center uses these endpoints to render the [+ Create] form
// and the service catalog picker.
package applyapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/orkspace/orkestra/pkg/kubeclient"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
)

// CRDLookup returns the CRDEntry for a given kind, or nil if not found or not IDP-enabled.
type CRDLookup func(kind string) *orktypes.CRDEntry

// CatalogLister returns all IDP-enabled CRDEntries.
type CatalogLister func() []*orktypes.CRDEntry

// SchemaResponse is returned by GET /api/v1/schema/{kind}.
type SchemaResponse struct {
	Kind         string                             `json:"kind"`
	APIVersion   string                             `json:"apiVersion"`
	Properties   map[string]interface{}             `json:"properties"`
	Required     []string                           `json:"required,omitempty"`
	IDPFields    map[string]orktypes.IDPFieldConfig `json:"idpFields,omitempty"`
	IgnoreFields []string                           `json:"ignoreFields,omitempty"`

	// AdditionalLabels/AdditionalAnnotations are idp.additionalFields entries —
	// form fields with no CRD schema counterpart, written to metadata on apply
	// instead of spec. No live-cluster read needed to populate these, unlike
	// Properties — labels/annotations have no OpenAPI schema to fetch.
	AdditionalLabels      map[string]orktypes.IDPFieldConfig `json:"additionalLabels,omitempty"`
	AdditionalAnnotations map[string]orktypes.IDPFieldConfig `json:"additionalAnnotations,omitempty"`
}

// CatalogEntry is one row in the service catalog.
type CatalogEntry struct {
	Kind        string `json:"kind"`
	APIVersion  string `json:"apiVersion"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
}

// CatalogResponse is returned by GET /api/v1/schema/ (no kind).
type CatalogResponse struct {
	Schemas []CatalogEntry `json:"schemas"`
}

var crdGVR = schema.GroupVersionResource{
	Group:    "apiextensions.k8s.io",
	Version:  "v1",
	Resource: "customresourcedefinitions",
}

// Handler returns the http.HandlerFunc for GET /api/v1/schema/ and /api/v1/schema/{kind}.
// The auth middleware must wrap this handler before registration.
func schemaHandler(kube kubeclient.KubeClient, lookup CRDLookup, list CatalogLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		kind := strings.TrimPrefix(r.URL.Path, "/api/v1/schema/")
		kind = strings.Trim(kind, "/")

		if kind == "" {
			handleCatalog(w, list)
			return
		}

		entry := lookup(kind)
		if entry == nil || !entry.IDPEnabled() {
			http.Error(w, fmt.Sprintf("schema not available for kind %q", kind), http.StatusNotFound)
			return
		}

		props, required, err := fetchSpecProperties(r.Context(), kube, entry)
		if err != nil {
			http.Error(w, fmt.Sprintf("fetching CRD schema: %v", err), http.StatusInternalServerError)
			return
		}

		utils.WriteJSON(w, http.StatusOK, SchemaResponse{
			Kind:                  entry.APITypes.Kind,
			APIVersion:            entry.APITypes.Group + "/" + entry.APITypes.Version,
			Properties:            props,
			Required:              required,
			IDPFields:             entry.IDP.Fields,
			IgnoreFields:          entry.IDP.IgnoreFields,
			AdditionalLabels:      entry.AdditionalLabelFields(),
			AdditionalAnnotations: entry.AdditionalAnnotationFields(),
		})
	}
}

// handleCatalog returns a list of all IDP-enabled CRDs — the service catalog.
func handleCatalog(w http.ResponseWriter, list CatalogLister) {
	entries := list()
	catalog := make([]CatalogEntry, 0, len(entries))
	for _, e := range entries {
		desc := e.Description
		if e.IDP.Description != "" {
			desc = e.IDP.Description
		}
		catalog = append(catalog, CatalogEntry{
			Kind:        e.APITypes.Kind,
			APIVersion:  e.APITypes.Group + "/" + e.APITypes.Version,
			Category:    e.IDP.Category,
			Description: desc,
		})
	}
	utils.WriteJSON(w, http.StatusOK, CatalogResponse{Schemas: catalog})
}

// fetchSpecProperties reads the CRD from Kubernetes and returns the spec
// properties and required fields from the storage version's openAPIV3Schema.
func fetchSpecProperties(ctx context.Context, kube kubeclient.KubeClient, entry *orktypes.CRDEntry) (map[string]interface{}, []string, error) {
	crdName := entry.APITypes.Plural + "." + entry.APITypes.Group
	obj, err := kube.DynamicClient().Resource(crdGVR).Get(ctx, crdName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("get CRD %q: %w", crdName, err)
	}

	versions, ok := nestedSlice(obj.Object, "spec", "versions")
	if !ok {
		return nil, nil, fmt.Errorf("CRD %q has no spec.versions", crdName)
	}

	for _, v := range versions {
		ver, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		storage, _ := ver["storage"].(bool)
		if !storage {
			continue
		}
		specSchema, ok := nestedMap(ver, "schema", "openAPIV3Schema", "properties", "spec")
		if !ok {
			return nil, nil, fmt.Errorf("CRD %q storage version has no spec schema", crdName)
		}
		props, _ := specSchema["properties"].(map[string]interface{})
		if props == nil {
			return nil, nil, fmt.Errorf("CRD %q storage version has no spec properties", crdName)
		}
		var required []string
		if req, ok := specSchema["required"].([]interface{}); ok {
			for _, r := range req {
				if s, ok := r.(string); ok {
					required = append(required, s)
				}
			}
		}
		return props, required, nil
	}
	return nil, nil, fmt.Errorf("CRD %q has no storage version", crdName)
}

func nestedSlice(obj map[string]interface{}, keys ...string) ([]interface{}, bool) {
	cur := obj
	for i, k := range keys {
		if i == len(keys)-1 {
			v, ok := cur[k].([]interface{})
			return v, ok
		}
		next, ok := cur[k].(map[string]interface{})
		if !ok {
			return nil, false
		}
		cur = next
	}
	return nil, false
}

func nestedMap(obj map[string]interface{}, keys ...string) (map[string]interface{}, bool) {
	cur := obj
	for _, k := range keys {
		next, ok := cur[k].(map[string]interface{})
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}
