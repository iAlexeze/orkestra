package schema

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func idpEntry(kind string) *orktypes.CRDEntry {
	return &orktypes.CRDEntry{
		Name: strings.ToLower(kind),
		APITypes: orktypes.APITypes{
			Group:   "platform.orkestra.io",
			Version: "v1alpha1",
			Kind:    kind,
			Plural:  strings.ToLower(kind) + "s",
		},
		IDP: &orktypes.IDPConfig{
			Enabled: true,
			Fields: map[string]orktypes.IDPFieldConfig{
				"team": {Label: "Team", Order: 1},
			},
		},
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	h := Handler(nil, func(kind string) *orktypes.CRDEntry { return nil })
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schema/Platform", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestHandler_MissingKind(t *testing.T) {
	h := Handler(nil, func(kind string) *orktypes.CRDEntry { return nil })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestHandler_IDPNotEnabled(t *testing.T) {
	lookup := func(kind string) *orktypes.CRDEntry {
		return &orktypes.CRDEntry{IDP: &orktypes.IDPConfig{Enabled: false}}
	}
	h := Handler(nil, lookup)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema/Platform", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestHandler_UnknownKind(t *testing.T) {
	h := Handler(nil, func(kind string) *orktypes.CRDEntry { return nil })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema/Unknown", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestSchemaResponse_JSON(t *testing.T) {
	resp := SchemaResponse{
		Kind:       "Platform",
		APIVersion: "platform.orkestra.io/v1alpha1",
		Properties: map[string]interface{}{
			"team": map[string]interface{}{"type": "string"},
		},
		IDPFields: map[string]orktypes.IDPFieldConfig{
			"team": {Label: "Team", Order: 1},
		},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got SchemaResponse
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Kind != "Platform" {
		t.Errorf("Kind = %q", got.Kind)
	}
	if got.IDPFields["team"].Label != "Team" {
		t.Errorf("IDPFields[team].Label = %q", got.IDPFields["team"].Label)
	}
}

func TestNestedMap(t *testing.T) {
	obj := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": map[string]interface{}{"found": true},
			},
		},
	}
	got, ok := nestedMap(obj, "a", "b", "c")
	if !ok {
		t.Fatal("nestedMap: not found")
	}
	if got["found"] != true {
		t.Errorf("nestedMap result = %v", got)
	}
	_, ok = nestedMap(obj, "a", "missing")
	if ok {
		t.Error("nestedMap should return false for missing key")
	}
}
