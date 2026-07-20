// pkg/gateway/schema/handler.go
//
// GET /api/v1/schema/{kind}
//
// Returns the OpenAPI spec schema for a CRD, with idp.fields hints merged in.
// Only served for CRDs where idp.enabled: true.
//
// The Control Center uses this endpoint to render the [+ Create] form.
// External callers (Terraform providers, custom portals) use it to discover
// the shape of a CR before POSTing to /api/v1/apply.
package schema

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

// SchemaResponse is returned by GET /api/v1/schema/{kind}.
type SchemaResponse struct {
	Kind       string                             `json:"kind"`
	APIVersion string                             `json:"apiVersion"`
	Properties map[string]interface{}             `json:"properties"`
	IDPFields  map[string]orktypes.IDPFieldConfig `json:"idpFields,omitempty"`
}

var crdGVR = schema.GroupVersionResource{
	Group:    "apiextensions.k8s.io",
	Version:  "v1",
	Resource: "customresourcedefinitions",
}

// Handler returns the http.HandlerFunc for GET /api/v1/schema/{kind}.
// The auth middleware must wrap this handler before registration.
func Handler(kube kubeclient.KubeClient, lookup CRDLookup) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		kind := strings.TrimPrefix(r.URL.Path, "/api/v1/schema/")
		kind = strings.Trim(kind, "/")
		if kind == "" {
			http.Error(w, "kind required", http.StatusBadRequest)
			return
		}

		entry := lookup(kind)
		if entry == nil || entry.IDP == nil || !entry.IDP.Enabled {
			http.Error(w, fmt.Sprintf("schema not available for kind %q", kind), http.StatusNotFound)
			return
		}

		props, err := fetchSpecProperties(r.Context(), kube, entry)
		if err != nil {
			http.Error(w, fmt.Sprintf("fetching CRD schema: %v", err), http.StatusInternalServerError)
			return
		}

		utils.WriteJSON(w, http.StatusOK, SchemaResponse{
			Kind:       entry.APITypes.Kind,
			APIVersion: entry.APITypes.Group + "/" + entry.APITypes.Version,
			Properties: props,
			IDPFields:  entry.IDP.Fields,
		})
	}
}

// fetchSpecProperties reads the CRD from Kubernetes and returns the spec
// properties from the storage version's openAPIV3Schema.spec.properties block.
func fetchSpecProperties(ctx context.Context, kube kubeclient.KubeClient, entry *orktypes.CRDEntry) (map[string]interface{}, error) {
	crdName := entry.APITypes.Plural + "." + entry.APITypes.Group
	obj, err := kube.DynamicClient().Resource(crdGVR).Get(ctx, crdName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get CRD %q: %w", crdName, err)
	}

	// Navigate: spec.versions[storage=true].schema.openAPIV3Schema.properties.spec.properties
	versions, ok := nestedSlice(obj.Object, "spec", "versions")
	if !ok {
		return nil, fmt.Errorf("CRD %q has no spec.versions", crdName)
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
		props, ok := nestedMap(ver, "schema", "openAPIV3Schema", "properties", "spec", "properties")
		if !ok {
			return nil, fmt.Errorf("CRD %q storage version has no spec properties schema", crdName)
		}
		return props, nil
	}
	return nil, fmt.Errorf("CRD %q has no storage version", crdName)
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
