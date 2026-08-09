package api

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
	Target      string                               `json:"target"`
	Title       string                               `json:"title,omitempty"`
	Description string                               `json:"description,omitempty"`
	Aliases     []string                             `json:"aliases,omitempty"`
	Fields      map[string]orktypes.ServeFieldConfig `json:"fields"`
	Required    []string                             `json:"required,omitempty"`
}

// CatalogEntry is one entry in the catalog returned by GET /api/v1/schema.
type CatalogEntry struct {
	Target      string   `json:"target"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Category    string   `json:"category,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
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
			if !hasAnySchemaPermission(r, kat.ServeEnabledCRDs) {
				writeJSONError(w, http.StatusForbidden, "permission denied", "token lacks required permission")
				return
			}
			handleSchemaCatalog(w, r, kat.ServeEnabledCRDs)
			return
		}

		// ─── Per-target mode ──────────────────────────────────────────────────
		resolution := kat.LookupByTargetOrAlias(target)
		if resolution == nil || !resolution.CRD.ServeEnabled() {
			writeJSONError(w, http.StatusNotFound, "target not found",
				fmt.Sprintf("target %q not found", target),
			)
			return
		}
		crd := resolution.CRD
		alias := resolution.Alias

		// Check get permission on this specific target (alias-aware)
		if !checkServePermission(w, r, crd, orktypes.ServeClassSchema, orktypes.ServeOpGet, "", alias) {
			return
		}

		title := crd.Serve.Title
		if title == "" {
			title = crd.Kind()
		}

		writeJSON(w, http.StatusOK, SchemaResponse{
			Target:      crd.ServeTarget(),
			Title:       title,
			Description: crd.Serve.Description,
			Aliases:     crd.AliasNames(),
			Fields:      crd.AllServeFields(),
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
		title := crd.Serve.Title
		if title == "" {
			title = crd.Kind()
		}

		entries = append(entries, CatalogEntry{
			Target:      crd.ServeTarget(),
			Title:       title,
			Description: crd.Serve.Description,
			Category:    crd.Serve.Category,
			Aliases:     crd.AliasNames(),
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

// getRequiredFields returns the names of all required fields in the serve config.
func getRequiredFields(crd *orktypes.CRDEntry) []string {
	var required []string
	for name, field := range crd.AllServeFields() {
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
		allowed, _ := crd.TokenAllowedFor("", tokenName, orktypes.ServeOpList, "", orktypes.ServeClassSchema)
		if allowed {
			return true
		}
	}
	return false
}
