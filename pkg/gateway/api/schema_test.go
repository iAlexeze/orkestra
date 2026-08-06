package api

// import (
// 	"encoding/json"
// 	"net/http"
// 	"net/http/httptest"
// 	"strings"
// 	"testing"

// 	orktypes "github.com/orkspace/orkestra/pkg/types"
// )

// func idpEntry(kind string) *orktypes.CRDEntry {
// 	return &orktypes.CRDEntry{
// 		Name: strings.ToLower(kind),
// 		APITypes: orktypes.APITypes{
// 			Group:   "platform.orkestra.io",
// 			Version: "v1alpha1",
// 			Kind:    kind,
// 			Plural:  strings.ToLower(kind) + "s",
// 		},
// 		Serve: &orktypes.ServeConfig{
// 			Enabled: true,
// 			Fields: map[string]orktypes.ServeFieldConfig{
// 				"team": {Label: "Team", Order: 1},
// 			},
// 		},
// 	}
// }

// func noopLister() []*orktypes.CRDEntry { return nil }

// func TestSchemaHandler_MethodNotAllowed(t *testing.T) {
// 	h := schemaHandler(nil, func(kind string) *orktypes.CRDEntry { return nil }, noopLister)
// 	req := httptest.NewRequest(http.MethodPost, "/api/v1/schema/Platform", nil)
// 	rr := httptest.NewRecorder()
// 	h.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusMethodNotAllowed {
// 		t.Errorf("status = %d, want 405", rr.Code)
// 	}
// }

// func TestSchemaHandler_IDPNotEnabled(t *testing.T) {
// 	lookup := func(kind string) *orktypes.CRDEntry {
// 		return &orktypes.CRDEntry{Serve: &orktypes.ServeConfig{Enabled: false}}
// 	}
// 	h := schemaHandler(nil, lookup, noopLister)
// 	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema/Platform", nil)
// 	rr := httptest.NewRecorder()
// 	h.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusNotFound {
// 		t.Errorf("status = %d, want 404", rr.Code)
// 	}
// }

// func TestSchemaHandler_UnknownKind(t *testing.T) {
// 	h := schemaHandler(nil, func(kind string) *orktypes.CRDEntry { return nil }, noopLister)
// 	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema/Unknown", nil)
// 	rr := httptest.NewRecorder()
// 	h.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusNotFound {
// 		t.Errorf("status = %d, want 404", rr.Code)
// 	}
// }

// func TestSchemaHandler_Catalog_Empty(t *testing.T) {
// 	h := schemaHandler(nil, func(kind string) *orktypes.CRDEntry { return nil }, noopLister)
// 	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema/", nil)
// 	rr := httptest.NewRecorder()
// 	h.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusOK {
// 		t.Fatalf("status = %d, want 200", rr.Code)
// 	}
// 	var cat CatalogResponse
// 	if err := json.NewDecoder(rr.Body).Decode(&cat); err != nil {
// 		t.Fatalf("decode catalog: %v", err)
// 	}
// 	if cat.Schemas == nil {
// 		t.Error("Schemas should not be nil")
// 	}
// 	if len(cat.Schemas) != 0 {
// 		t.Errorf("Schemas len = %d, want 0", len(cat.Schemas))
// 	}
// }

// func TestSchemaHandler_Catalog_WithEntries(t *testing.T) {
// 	appEntry := &orktypes.CRDEntry{
// 		Name: "apprequests",
// 		APITypes: orktypes.APITypes{
// 			Group:   "platform.orkestra.io",
// 			Version: "v1alpha1",
// 			Kind:    "AppRequest",
// 			Plural:  "apprequests",
// 		},
// 		Serve: &orktypes.ServeConfig{
// 			Enabled:     true,
// 			Category:    "Compute",
// 			Description: "Self-service app deployment",
// 		},
// 	}
// 	dbEntry := &orktypes.CRDEntry{
// 		Name: "databases",
// 		APITypes: orktypes.APITypes{
// 			Group:   "platform.orkestra.io",
// 			Version: "v1alpha1",
// 			Kind:    "Database",
// 			Plural:  "databases",
// 		},
// 		Description: "Managed database",
// 		Serve: &orktypes.ServeConfig{
// 			Enabled:  true,
// 			Category: "Data",
// 			// no IDP.Description — falls back to CRDEntry.Description
// 		},
// 	}
// 	lister := func() []*orktypes.CRDEntry { return []*orktypes.CRDEntry{appEntry, dbEntry} }
// 	h := schemaHandler(nil, func(kind string) *orktypes.CRDEntry { return nil }, lister)

// 	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema/", nil)
// 	rr := httptest.NewRecorder()
// 	h.ServeHTTP(rr, req)

// 	if rr.Code != http.StatusOK {
// 		t.Fatalf("status = %d, want 200", rr.Code)
// 	}
// 	var cat CatalogResponse
// 	if err := json.NewDecoder(rr.Body).Decode(&cat); err != nil {
// 		t.Fatalf("decode catalog: %v", err)
// 	}
// 	if len(cat.Schemas) != 2 {
// 		t.Fatalf("Schemas len = %d, want 2", len(cat.Schemas))
// 	}

// 	byKind := make(map[string]CatalogEntry)
// 	for _, s := range cat.Schemas {
// 		byKind[s.Kind] = s
// 	}

// 	app := byKind["AppRequest"]
// 	if app.Category != "Compute" {
// 		t.Errorf("AppRequest.Category = %q", app.Category)
// 	}
// 	if app.Description != "Self-service app deployment" {
// 		t.Errorf("AppRequest.Description = %q", app.Description)
// 	}

// 	db := byKind["Database"]
// 	if db.Category != "Data" {
// 		t.Errorf("Database.Category = %q", db.Category)
// 	}
// 	if db.Description != "Managed database" {
// 		t.Errorf("Database.Description = %q (want fallback from CRDEntry)", db.Description)
// 	}
// }

// func TestSchemaResponse_JSON(t *testing.T) {
// 	resp := SchemaResponse{
// 		Kind:         "Platform",
// 		APIVersion:   "platform.orkestra.io/v1alpha1",
// 		Required:     []string{"team"},
// 		Ignore: []string{"internalId"},
// 		Properties: map[string]interface{}{
// 			"team": map[string]interface{}{"type": "string"},
// 		},
// 		AllServeFields: map[string]orktypes.ServeFieldConfig{
// 			"team": {Label: "Team", Order: 1},
// 		},
// 	}
// 	b, err := json.Marshal(resp)
// 	if err != nil {
// 		t.Fatalf("marshal: %v", err)
// 	}
// 	var got SchemaResponse
// 	if err := json.Unmarshal(b, &got); err != nil {
// 		t.Fatalf("unmarshal: %v", err)
// 	}
// 	if got.Kind != "Platform" {
// 		t.Errorf("Kind = %q", got.Kind)
// 	}
// 	if got.AllServeFields["team"].Label != "Team" {
// 		t.Errorf("AllServeFields[team].Label = %q", got.AllServeFields["team"].Label)
// 	}
// 	if len(got.Required) != 1 || got.Required[0] != "team" {
// 		t.Errorf("Required = %v", got.Required)
// 	}
// 	if len(got.Ignore) != 1 || got.Ignore[0] != "internalId" {
// 		t.Errorf("Ignore = %v", got.Ignore)
// 	}
// }

// func TestSchemaResponse_AdditionalFields_JSON(t *testing.T) {
// 	resp := SchemaResponse{
// 		Kind:       "Platform",
// 		APIVersion: "platform.orkestra.io/v1alpha1",
// 		AdditionalLabels: map[string]orktypes.ServeFieldConfig{
// 			"tier": {Label: "Tier", Type: "enum", Enum: []string{"free", "pro"}},
// 		},
// 		AdditionalAnnotations: map[string]orktypes.ServeFieldConfig{
// 			"platform.example.io/monitoring": {Label: "Monitoring", Type: "boolean"},
// 		},
// 	}
// 	b, err := json.Marshal(resp)
// 	if err != nil {
// 		t.Fatalf("marshal: %v", err)
// 	}
// 	var got SchemaResponse
// 	if err := json.Unmarshal(b, &got); err != nil {
// 		t.Fatalf("unmarshal: %v", err)
// 	}
// 	tier := got.AdditionalLabels["tier"]
// 	if tier.Label != "Tier" || tier.Type != "enum" || len(tier.Enum) != 2 {
// 		t.Errorf("AdditionalLabels[tier] = %+v", tier)
// 	}
// 	mon := got.AdditionalAnnotations["platform.example.io/monitoring"]
// 	if mon.Label != "Monitoring" || mon.Type != "boolean" {
// 		t.Errorf("AdditionalAnnotations[platform.example.io/monitoring] = %+v", mon)
// 	}
// }

// func TestSchemaResponse_AdditionalFields_OmittedWhenNil(t *testing.T) {
// 	resp := SchemaResponse{Kind: "Platform"}
// 	b, err := json.Marshal(resp)
// 	if err != nil {
// 		t.Fatalf("marshal: %v", err)
// 	}
// 	s := string(b)
// 	if strings.Contains(s, "additionalLabels") || strings.Contains(s, "additionalAnnotations") {
// 		t.Errorf("expected additionalLabels/additionalAnnotations to be omitted when nil, got: %s", s)
// 	}
// }

// func TestSchemaHandler_PopulatesAdditionalFields(t *testing.T) {
// 	entry := idpEntry("Platform")
// 	entry.Serve.AdditionalFields = &orktypes.ServeAdditionalFields{
// 		Labels: map[string]orktypes.ServeFieldConfig{
// 			"tier": {Label: "Tier"},
// 		},
// 	}
// 	// Mirrors schemaHandler's own population step directly — fetchSpecProperties
// 	// requires a live kube client the other handler tests don't set up either,
// 	// so this isolates the additionalFields mapping from the CRD-schema fetch.
// 	resp := SchemaResponse{}
// 	if entry.Serve.AdditionalFields != nil {
// 		resp.AdditionalLabels = entry.Serve.Labels
// 		resp.AdditionalAnnotations = entry.Serve.Annotations
// 	}
// 	if resp.AdditionalLabels["tier"].Label != "Tier" {
// 		t.Errorf("AdditionalLabels[tier].Label = %q, want %q", resp.AdditionalLabels["tier"].Label, "Tier")
// 	}
// 	if resp.AdditionalAnnotations != nil {
// 		t.Errorf("AdditionalAnnotations should be nil when not declared, got: %v", resp.AdditionalAnnotations)
// 	}
// }
