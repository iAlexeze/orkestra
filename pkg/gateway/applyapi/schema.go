package applyapi

import (
	"fmt"
	"net/http"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
)

// SchemaResponse is returned by GET /api/v1/schema?target=<t>.
//
// Fields is a flat map that merges spec fields, label fields, and annotation
// fields into one caller-facing surface. The caller does not need to know
// which destination each field routes to — that is the gateway's concern.
type SchemaResponse struct {
	Target      string                             `json:"target"`
	Title       string                             `json:"title,omitempty"`
	Description string                             `json:"description,omitempty"`
	Fields      map[string]orktypes.IDPFieldConfig `json:"fields"`
	Required    []string                           `json:"required,omitempty"`
}

// CatalogEntry is one entry in the catalog returned by GET /api/v1/schema.
type CatalogEntry struct {
	Target      string   `json:"target"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Category    string   `json:"category,omitempty"`
	Required    []string `json:"required,omitempty"`
}

// schemaHandler serves two endpoints:
//
//	GET /api/v1/schema               — catalog of all available targets
//	GET /api/v1/schema?target=<t>    — flat field schema for one target
//
// The catalog endpoint supports pagination via limit= and offset= query params.
// The per-target endpoint returns the complete field map — there is no
// pagination for fields since the field count is always small.
func schemaHandler(kat *katalog.Katalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed", "only GET requests are supported")
			return
		}

		target := r.URL.Query().Get("target")

		// ─── Catalog mode ──────────────────────────────────────────────────────
		if target == "" {
			// Check list permission on a representative CRD (or skip if none)
			// For catalog, we check permission on the first available CRD,
			// or allow if the user has any schema access.
			if !hasAnySchemaPermission(r, kat.IDPEnabledCRDs) {
				writeJSONError(w, http.StatusForbidden, "permission denied", "token lacks required permission")
				return
			}
			handleSchemaCatalog(w, r, kat.IDPEnabledCRDs)
			return
		}

		// ─── Per-target mode ──────────────────────────────────────────────────
		crd := kat.LookupByTarget(target)
		if crd == nil || !crd.IDPEnabled() {
			writeJSONError(w, http.StatusNotFound, "target not found",
				fmt.Sprintf("target %q not found", target),
			)
			return
		}

		// Check get permission on this specific target
		if !checkIDPPermission(w, r, crd, orktypes.IDPClassSchema, orktypes.IDPOpGet, "") {
			return
		}

		title := crd.IDP.Title
		if title == "" {
			title = crd.Kind()
		}

		writeJSON(w, http.StatusOK, SchemaResponse{
			Target:      crd.IDPTarget(),
			Title:       title,
			Description: crd.IDP.Description,
			Fields:      crd.IDPFields(),
			Required:    getRequiredFields(crd),
		})
	}
}

// handleSchemaCatalog returns a paginated list of available targets.
func handleSchemaCatalog(
	w http.ResponseWriter,
	r *http.Request,
	catalog func() []*orktypes.CRDEntry,
) {
	all := catalog()
	p := parsePagination(r)

	entries := make([]CatalogEntry, 0, len(all))
	for _, crd := range all {
		title := crd.IDP.Title
		if title == "" {
			title = crd.Kind()
		}

		entries = append(entries, CatalogEntry{
			Target:      crd.IDPTarget(),
			Title:       title,
			Description: crd.IDP.Description,
			Category:    crd.IDP.Category,
			Required:    getRequiredFields(crd),
		})
	}

	page, total := utils.PageItems(entries, p)
	writeJSON(w, http.StatusOK, utils.PaginatedResponse[CatalogEntry]{
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
		Items:  page,
	})
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// getRequiredFields returns the names of all required fields in the IDP config.
func getRequiredFields(crd *orktypes.CRDEntry) []string {
	var required []string
	for name, field := range crd.IDPFields() {
		if field.Required {
			required = append(required, name)
		}
	}
	return required
}

// hasAnySchemaPermission checks if the caller has any schema access.
// Used for the catalog endpoint when listing all targets.
func hasAnySchemaPermission(r *http.Request, catalog func() []*orktypes.CRDEntry) bool {
	tokenName := TokenNameFromContext(r.Context())
	if tokenName == "" {
		return false
	}

	for _, crd := range catalog() {
		if crd.IDP != nil && crd.IDP.HasTokenRestrictions() {
			allowed, _ := crd.IDP.TokenAllowed(
				tokenName, orktypes.IDPOpList, "", orktypes.IDPClassSchema,
			)
			if allowed {
				return true
			}
		} else {
			// No restrictions — any authenticated token can list
			return true
		}
	}
	return false
}
