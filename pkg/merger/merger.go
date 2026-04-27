// pkg/merger/merger.go
//
// Package merger resolves and merges Katalog and Komposer YAML files into a
// single unified CRD map that the Katalog runtime consumes. It is the
// ingestion layer between raw YAML on disk (or remote sources) and the
// operator's live configuration.
//
// Entry point: New(paths...).Merge() — call once; query with Enabled, All,
// ToSpec, ToSecurity, ToNotification, and ToProviders.
//
// See README.md for merge rules, source-loading order, and top-level field
// accumulation semantics.
package merger

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
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

	// result holds the merged CRD entries after Merge() completes — keyed by CRD name
	result map[string]orktypes.CRDEntry

	// security holds the security configuration of the final katalog
	security orktypes.KatalogSecurity

	// notification holds the merged notification configuration of the final katalog
	notification *orktypes.KatalogNotification

	// providers holds the top-level provider requirements of the final katalog
	providers []orktypes.KatalogProviderRequirement

	// merged tracks whether Merge() has been called
	merged bool

	// metadata gets the metadata from the document processed
	// used by cli and health endpoints
	apiMetadata apiMetadata

	registryURL string // set from ORK_REGISTRY via SetRegistryURL
}

type apiMetadata struct {
	APIVersion string               `json:"apiVersion" yaml:"apiVersion"`
	Kind       string               `json:"kind" yaml:"kind"`
	Metadata   orktypes.KatalogMeta `json:"metadata" yaml:"metadata"`
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
// resolves Helm charts, and produces a single deduplicated CRD map.
// Safe to call multiple times — re-merges on each call.
func (m *Merger) Merge() error {
	// seen lives here — top level only (tracks which file each name came from)
	seen := map[string]string{}
	merged := make(map[string]orktypes.CRDEntry)

	for _, path := range m.entryPoints {
		// loadKatalogFile manages its OWN internal dedup
		// it does NOT receive seen — only returns the final CRD map for this file
		crds, err := m.loadKatalogFile(path)
		if err != nil {
			return fmt.Errorf("merger: loading %q: %w", path, err)
		}

		// Check duplicates HERE at the top level across entry points
		for name, crd := range crds {
			if err := checkDuplicate(seen, name, path); err != nil {
				return err
			}
			seen[name] = path
			merged[name] = crd
		}
	}

	m.result = merged
	m.merged = true

	logger.Debug().
		Int("entryPoints", len(m.entryPoints)).
		Int("total", len(m.result)).
		Int("enabled", len(m.Enabled())).
		Msg("merger: complete")
	return nil
}

// mergeCRDEntry merges an override entry on top of a base entry.
// The override wins on any field it explicitly declares.
// Fields that are zero/nil/empty in the override are inherited from the base.
//
// This is how Komposer spec.crds works: you declare only what you want
// to change. Everything else — reconciler templates, validation rules,
// mutation rules, status config — is inherited from the source Katalog.
func mergeCRDEntry(base, override orktypes.CRDEntry) orktypes.CRDEntry {
	result := base // start from the full base

	// ── Identity ─────────────────────────────────────────────────────────
	// Name always comes from the override (it's the merge key)
	// APITypes: override wins if declared, otherwise keep base
	if override.APITypes.Kind != "" {
		result.APITypes = override.APITypes
	}

	// ── Enabled ───────────────────────────────────────────────────────────
	// Only override if explicitly set to false — zero value (true) means
	// "not declared", so we keep the base value
	if !override.IsEnabled() {
		result.Enabled = override.Enabled
	}

	// ── Runtime tuning ────────────────────────────────────────────────────
	// Override only when non-zero — zero means "not declared in override"
	if override.Workers > 0 {
		result.Workers = override.Workers
	}
	if override.Resync != 0 {
		result.Resync = override.Resync
	}
	if override.Queue.MaxQueueDepth > 0 {
		result.Queue.MaxQueueDepth = override.Queue.MaxQueueDepth
	}
	if override.Queue.DegradeThreshold > 0 {
		result.Queue.DegradeThreshold = override.Queue.DegradeThreshold
	}

	// ── Namespace ─────────────────────────────────────────────────────────
	if override.Namespace != "" {
		result.Namespace = override.Namespace
	}

	// ── Critical ──────────────────────────────────────────────────────────
	// if override.IsCritical() {
	// 	result.Critical = override.Critical
	// }

	// ── Dependencies ──────────────────────────────────────────────────────
	// Override replaces entirely if declared — partial dependency override
	// makes no semantic sense
	if len(override.DependsOn) > 0 {
		result.DependsOn = override.DependsOn
	}

	// ── Restricted namespaces — additive ──────────────────────────────────
	// Restrictions are additive: override adds to base, never removes
	if len(override.RestrictedNamespaces) > 0 {
		seen := map[string]struct{}{}
		for _, ns := range result.RestrictedNamespaces {
			seen[ns] = struct{}{}
		}
		for _, ns := range override.RestrictedNamespaces {
			if _, ok := seen[ns]; !ok {
				result.RestrictedNamespaces = append(result.RestrictedNamespaces, ns)
			}
		}
	}

	// ── Allowed namespaces — additive ─────────────────────────────────────
	// Allowances are additive: override adds to base, never removes
	if len(override.AllowedNamespaces) > 0 {
		seen := map[string]struct{}{}
		for _, ns := range result.AllowedNamespaces {
			seen[ns] = struct{}{}
		}
		for _, ns := range override.AllowedNamespaces {
			if _, ok := seen[ns]; !ok {
				result.AllowedNamespaces = append(result.AllowedNamespaces, ns)
			}
		}
	}

	// ── Finalizers — additive ─────────────────────────────────────────────
	if len(override.OperatorBox.Finalizers) > 0 {
		seen := map[string]struct{}{}
		for _, f := range result.OperatorBox.Finalizers {
			seen[f] = struct{}{}
		}
		for _, f := range override.OperatorBox.Finalizers {
			if _, ok := seen[f]; !ok {
				result.OperatorBox.Finalizers = append(
					result.OperatorBox.Finalizers, f)
			}
		}
	}

	// ── Reconciler config — override only declared blocks ─────────────────
	// If the override declares onCreate, it replaces the base onCreate.
	// If it doesn't declare it, the base onCreate is preserved.
	// Same for onReconcile, onDelete, hooks, constructor, status.
	rc := &result.OperatorBox

	if override.OperatorBox.OnCreate != nil {
		rc.OnCreate = override.OperatorBox.OnCreate
	}
	if override.OperatorBox.OnReconcile != nil {
		rc.OnReconcile = override.OperatorBox.OnReconcile
	}
	if override.OperatorBox.OnDelete != nil {
		rc.OnDelete = override.OperatorBox.OnDelete
	}
	if override.OperatorBox.HookFactory != nil {
		rc.HookFactory = override.OperatorBox.HookFactory
	}
	if override.OperatorBox.Constructor != nil {
		rc.Constructor = override.OperatorBox.Constructor
	}
	if override.OperatorBox.Status != nil {
		rc.Status = override.OperatorBox.Status
	}

	// ── Validation and mutation — override replaces if declared ───────────
	// Platform teams may want to add stricter rules in production via the
	// Komposer. Replacing rather than merging is the safe behaviour —
	// merging rules from two sources could produce unexpected combinations.
	if override.Validation != nil {
		result.Validation = override.Validation
	}
	if override.Mutation != nil {
		result.Mutation = override.Mutation
	}

	// ── Conversion — override replaces if declared ────────────────────────
	if override.Conversion != nil {
		result.Conversion = override.Conversion
	}

	// ── Endpoints ─────────────────────────────────────────────────────────
	if override.IsEnabledAllEndpoints() {
		result.Endpoints = override.Endpoints
	}

	return result
}

// ── Query methods ─────────────────────────────────────────────────────────────

// Enabled returns only CRD entries where enabled: true.
func (m *Merger) Enabled() map[string]orktypes.CRDEntry {
	m.mustBeMerged()
	out := make(map[string]orktypes.CRDEntry)
	for name, crd := range m.result {
		if crd.IsEnabled() {
			out[name] = crd
		}
	}
	return out
}

// All returns all CRD entries including disabled ones.
func (m *Merger) All() map[string]orktypes.CRDEntry {
	m.mustBeMerged()
	return m.result
}

// Get returns a CRD entry by name. Returns (entry, true) if found.
func (m *Merger) Get(name string) (orktypes.CRDEntry, bool) {
	m.mustBeMerged()
	crd, ok := m.result[name]
	return crd, ok
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

// ToSecurity returns the security config of the merged result as a KatalogSecurity
// Used by NewKatalog consume the merged result.
func (m *Merger) ToSecurity() orktypes.KatalogSecurity {
	m.mustBeMerged()
	return m.security
}

// ToProviders returns the top-level provider requirements of the merged result.
// Used by KomposeRuntimeKatalog to populate Katalog.Providers.
func (m *Merger) ToProviders() []orktypes.KatalogProviderRequirement {
	m.mustBeMerged()
	return m.providers
}

// ToNotification returns the merged notification configuration of the merged result.
// When a Komposer references multiple source Katalogs, teams from all sources are
// merged — source teams are inherited and the Komposer's own teams win on conflict.
// Used by KomposeRuntimeKatalog to populate Katalog.Notification.
func (m *Merger) ToNotification() *orktypes.KatalogNotification {
	m.mustBeMerged()
	return m.notification
}

// APIMetadata returns the merged result as a KatalogMeta with apiversion and kind.
func (m *Merger) APIMetadata() apiMetadata {
	m.mustBeMerged()
	return m.apiMetadata
}

func (m *Merger) SetRegistryURL(url string) {
	m.registryURL = url
}

func (m *Merger) GetRegistryURL() string {
	m.mustBeMerged()
	return m.registryURL
}

// ToUI returns a UI-friendly representation of the merged Katalog.
// This method extracts only the fields needed for display in the Control Center:
//   - API version and kind (always "Katalog" at runtime)
//   - Metadata (name, description, version, author, license)
//   - All merged CRD definitions
//
// Internal fields (Scheme, GroupVersionKind, etc.) are excluded because they
// have `yaml:"-" json:"-"` tags and won't be serialized to JSON.
//
// This method is used by the /katalog/raw endpoint to provide a clean,
// readable view of the Katalog that created the current operator.
func (m *Merger) ToUI() *orktypes.KatalogForUI {
	m.mustBeMerged()

	return &orktypes.KatalogForUI{
		APIVersion: m.apiMetadata.APIVersion,
		Kind:       konfig.KatalogKind(),
		Metadata:   m.apiMetadata.Metadata,
		Spec: orktypes.KatalogSpecForUI{
			CRDs: m.result,
		},
	}
}
