// pkg/merger/merger.go
package merger

import (
	"fmt"

	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// Merger loads one or more Katalog files, resolves their sources
// (files, URLs, Helm charts), merges all CRD entries, and exposes
// the result through Enabled(), All(), and Get().
//
// Entry point: one or more file paths from the CLI or konstructOrkestra.
// Everything else — source resolution, Helm rendering, deduplication —
// is internal to the merger.
//
// Merge rules:
//   - Sources within a Katalog are loaded before spec.crds
//   - Inline spec.crds win on name conflict (local overrides remote)
//   - Duplicate names across independent Katalog files are errors
//   - disabled CRDs are preserved and filtered by Enabled()
type Merger struct {
	// entryPoints are the initial file paths/URLs passed from the CLI (ork run or or generate)
	entryPoints []string

	// result holds the merged CRD entries after Merge() completes
	result []orktypes.CRDEntry

	// merged tracks whether Merge() has been called
	merged bool

	// metadata gets the metadata from the document processed
	// used by cli and health endpoints
	metadata orktypes.KatalogMeta
}

// New creates a Merger with the given entry point file paths or URLs.
// Accepts one or more paths — the same as passing --katalog multiple times or comma separated.
func New(paths ...string) *Merger {
	return &Merger{entryPoints: paths}
}

// Add appends additional entry point paths after construction.
// Returns the Merger for chaining.
func (m *Merger) Add(paths ...string) *Merger {
	m.entryPoints = append(m.entryPoints, paths...)
	return m
}

// Merge loads all entry points and their declared sources,
// resolves Helm charts, and produces a single deduplicated CRD list.
// Safe to call multiple times — re-merges on each call.
func (m *Merger) Merge() error {
	// seen lives here — top level only
	seen := map[string]string{}
	var merged []orktypes.CRDEntry

	for _, path := range m.entryPoints {
		// loadKatalogFile manages its OWN internal dedup
		// it does NOT receive seen — only returns the final CRD list
		crds, err := m.loadKatalogFile(path)
		if err != nil {
			return fmt.Errorf("merger: loading %q: %w", path, err)
		}

		// Check duplicates HERE at the top level across entry points
		for _, crd := range crds {
			if err := checkDuplicate(seen, crd.Name, path); err != nil {
				return err
			}
			seen[crd.Name] = path
			merged = append(merged, crd)
		}
	}

	m.result = merged
	m.merged = true

	logger.Info().
		Int("entryPoints", len(m.entryPoints)).
		Int("total", len(m.result)).
		Int("enabled", len(m.Enabled())).
		Msg("merger: complete")
	return nil
}

// ── Query methods ─────────────────────────────────────────────────────────────

// Enabled returns only CRD entries where enabled: true.
func (m *Merger) Enabled() []orktypes.CRDEntry {
	m.mustBeMerged()
	var out []orktypes.CRDEntry
	for _, crd := range m.result {
		if crd.IsEnabled() {
			out = append(out, crd)
		}
	}
	return out
}

// All returns all CRD entries including disabled ones.
func (m *Merger) All() []orktypes.CRDEntry {
	m.mustBeMerged()
	return m.result
}

// Get returns a CRD entry by name. Returns (entry, true) if found.
func (m *Merger) Get(name string) (orktypes.CRDEntry, bool) {
	m.mustBeMerged()
	for _, crd := range m.result {
		if crd.Name == name {
			return crd, true
		}
	}
	return orktypes.CRDEntry{}, false
}

// Count returns total CRD count across all sources.
func (m *Merger) Count() int {
	m.mustBeMerged()
	return len(m.result)
}

// EnabledCount returns the number of enabled CRDs.
func (m *Merger) EnabledCount() int {
	return len(m.Enabled())
}

// ToSpec returns the merged result as a KatalogSpec.
// Used by NewKatalog and generate.Registry to consume the merged result.
func (m *Merger) ToSpec() orktypes.KatalogSpec {
	m.mustBeMerged()
	return orktypes.KatalogSpec{CRDs: m.result}
}

// ToMeta returns the merged result as a KatalogMeta.
func (m *Merger) ToMeta() orktypes.KatalogMeta {
	m.mustBeMerged()
	return m.metadata
}
