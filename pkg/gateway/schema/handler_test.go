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

func noopLister() []*orktypes.CRDEntry { return nil }

func TestHandler_MethodNotAllowed(t *testing.T) {
	h := Handler(nil, func(kind string) *orktypes.CRDEntry { return nil }, noopLister)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schema/Platform", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestHandler_IDPNotEnabled(t *testing.T) {
	lookup := func(kind string) *orktypes.CRDEntry {
		return &orktypes.CRDEntry{IDP: &orktypes.IDPConfig{Enabled: false}}
	}
	h := Handler(nil, lookup, noopLister)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema/Platform", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestHandler_UnknownKind(t *testing.T) {
	h := Handler(nil, func(kind string) *orktypes.CRDEntry { return nil }, noopLister)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema/Unknown", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestHandler_Catalog_Empty(t *testing.T) {
	h := Handler(nil, func(kind string) *orktypes.CRDEntry { return nil }, noopLister)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var cat CatalogResponse
	if err := json.NewDecoder(rr.Body).Decode(&cat); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if cat.Schemas == nil {
		t.Error("Schemas should not be nil")
	}
	if len(cat.Schemas) != 0 {
		t.Errorf("Schemas len = %d, want 0", len(cat.Schemas))
	}
}

func TestHandler_Catalog_WithEntries(t *testing.T) {
	appEntry := &orktypes.CRDEntry{
		Name: "apprequests",
		APITypes: orktypes.APITypes{
			Group:   "platform.orkestra.io",
			Version: "v1alpha1",
			Kind:    "AppRequest",
			Plural:  "apprequests",
		},
		IDP: &orktypes.IDPConfig{
			Enabled:     true,
			Category:    "Compute",
			Description: "Self-service app deployment",
		},
	}
	dbEntry := &orktypes.CRDEntry{
		Name: "databases",
		APITypes: orktypes.APITypes{
			Group:   "platform.orkestra.io",
			Version: "v1alpha1",
			Kind:    "Database",
			Plural:  "databases",
		},
		Description: "Managed database",
		IDP: &orktypes.IDPConfig{
			Enabled:  true,
			Category: "Data",
			// no IDP.Description — falls back to CRDEntry.Description
		},
	}
	lister := func() []*orktypes.CRDEntry { return []*orktypes.CRDEntry{appEntry, dbEntry} }
	h := Handler(nil, func(kind string) *orktypes.CRDEntry { return nil }, lister)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var cat CatalogResponse
	if err := json.NewDecoder(rr.Body).Decode(&cat); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if len(cat.Schemas) != 2 {
		t.Fatalf("Schemas len = %d, want 2", len(cat.Schemas))
	}

	byKind := make(map[string]CatalogEntry)
	for _, s := range cat.Schemas {
		byKind[s.Kind] = s
	}

	app := byKind["AppRequest"]
	if app.Category != "Compute" {
		t.Errorf("AppRequest.Category = %q", app.Category)
	}
	if app.Description != "Self-service app deployment" {
		t.Errorf("AppRequest.Description = %q", app.Description)
	}

	db := byKind["Database"]
	if db.Category != "Data" {
		t.Errorf("Database.Category = %q", db.Category)
	}
	if db.Description != "Managed database" {
		t.Errorf("Database.Description = %q (want fallback from CRDEntry)", db.Description)
	}
}

func TestSchemaResponse_JSON(t *testing.T) {
	resp := SchemaResponse{
		Kind:         "Platform",
		APIVersion:   "platform.orkestra.io/v1alpha1",
		Required:     []string{"team"},
		IgnoreFields: []string{"internalId"},
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
	if len(got.Required) != 1 || got.Required[0] != "team" {
		t.Errorf("Required = %v", got.Required)
	}
	if len(got.IgnoreFields) != 1 || got.IgnoreFields[0] != "internalId" {
		t.Errorf("IgnoreFields = %v", got.IgnoreFields)
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
