package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
)

// serveEntryKatalog builds a single-CRD katalog for kind "Platform" — its
// ServeTarget() defaults to the lowercased kind, "platform", which every
// test below uses directly. The outer map key is unrelated to target
// resolution (LookupByTargetOrAlias matches on ServeTarget(), not the key).
func serveEntryKatalog(serve *orktypes.ServeConfig) *katalog.Katalog {
	return katalog.NewFromEntryPointers(map[string]*orktypes.CRDEntry{
		"platform": {
			APITypes: orktypes.APITypes{
				Group:   "platform.orkestra.io",
				Version: "v1alpha1",
				Kind:    "Platform",
				Plural:  "platforms",
			},
			Serve: serve,
		},
	})
}

func TestSchemaHandler_MethodNotAllowed(t *testing.T) {
	h := schemaHandler(serveEntryKatalog(&orktypes.ServeConfig{Enabled: true}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schema", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestSchemaHandler_UnknownTarget(t *testing.T) {
	h := schemaHandler(serveEntryKatalog(&orktypes.ServeConfig{Enabled: true}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema?target=unknown", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestSchemaHandler_NotEnabled_UnreachableByTarget(t *testing.T) {
	// A disabled serve config means the CRD has no target at all —
	// LookupByTargetOrAlias can never resolve it, so this looks identical
	// to an unknown target from the caller's side.
	h := schemaHandler(serveEntryKatalog(&orktypes.ServeConfig{Enabled: false}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema?target=platform", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestSchemaHandler_PerTarget_ReturnsFields(t *testing.T) {
	h := schemaHandler(serveEntryKatalog(&orktypes.ServeConfig{
		Enabled:     true,
		Title:       "Platform Service",
		Description: "A platform-managed service",
		Fields: map[string]orktypes.ServeFieldConfig{
			"team":  {Label: "Team", Order: 1, Required: true},
			"image": {Label: "Image", Order: 2},
		},
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema?target=platform", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp SchemaResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Target != "platform" {
		t.Errorf("Target = %q, want %q", resp.Target, "platform")
	}
	if resp.Title != "Platform Service" {
		t.Errorf("Title = %q, want %q", resp.Title, "Platform Service")
	}
	if len(resp.Fields) != 2 {
		t.Errorf("Fields = %+v, want 2 entries", resp.Fields)
	}
	if len(resp.Required) != 1 || resp.Required[0] != "team" {
		t.Errorf("Required = %v, want [team]", resp.Required)
	}
}

func TestSchemaHandler_PerTarget_TitleFallsBackToKind(t *testing.T) {
	h := schemaHandler(serveEntryKatalog(&orktypes.ServeConfig{Enabled: true}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema?target=platform", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	var resp SchemaResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Title != "Platform" {
		t.Errorf("Title = %q, want the kind %q as fallback", resp.Title, "Platform")
	}
}

func TestSchemaHandler_Catalog_NoAuthorizedEntries_Forbidden(t *testing.T) {
	// hasAnySchemaPermission grants access by finding at least one CRD the
	// token can list — with zero CRDs, that loop can never grant, so even a
	// present, non-empty token name gets 403 rather than an empty catalog.
	h := schemaHandler(katalog.NewFromEntryPointers(map[string]*orktypes.CRDEntry{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema", nil)
	req = req.WithContext(contextWithTokenName(req.Context(), "test-token"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}

func TestSchemaHandler_Catalog_ListsServeEnabledEntries(t *testing.T) {
	h := schemaHandler(serveEntryKatalog(&orktypes.ServeConfig{
		Enabled:  true,
		Title:    "Platform Service",
		Category: "Infra",
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema", nil)
	req = req.WithContext(contextWithTokenName(req.Context(), "test-token"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var page utils.PaginatedResponse[CatalogEntry]
	if err := json.NewDecoder(rr.Body).Decode(&page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("Total = %d, want 1", page.Total)
	}
	if page.Items[0].Target != "platform" || page.Items[0].Category != "Infra" {
		t.Errorf("entry = %+v, want target=platform category=Infra", page.Items[0])
	}
}
