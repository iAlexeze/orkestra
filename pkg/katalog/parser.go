// pkg/katalog/parsek.go
package katalog

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/merger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// -----------------------------------------------------------------------------
//
//	YAML Builder
//
// -----------------------------------------------------------------------------

// BuildExpanded is the canonical pipeline for CLI commands that need a fully
// ready Katalog: merge → expand motifs → validate.
//
// Use this instead of calling KomposeRuntimeKatalog + ValidateConfig separately.
// For the rare case where validation must be skipped (e.g. ork template --no-validate),
// call KomposeRuntimeKatalog directly.
func BuildExpanded(kfg *konfig.Konfig, m *merger.Merger) (*Katalog, error) {
	var k Katalog
	if _, err := k.KomposeRuntimeKatalog(kfg, m); err != nil {
		return nil, err
	}
	if _, err := k.ValidateConfig(kfg); err != nil {
		return nil, err
	}
	return &k, nil
}

func (k *Katalog) KomposeRuntimeKatalog(kfg *konfig.Konfig, m *merger.Merger, paths ...string) (map[string]orktypes.CRDEntry, error) {
	k.Spec = m.ToSpec()
	k.Security = m.ToSecurity()
	k.Notification = m.ToNotification()
	k.Providers = m.ToProviders()
	k.projectInfo = m.ToProjectInfo()
	k.enabledCRDs = m.Enabled()           // Enabled CRDs for all operations
	k.metadata = m.APIMetadata().Metadata // Metadata for CLI and health endpoints
	k.APIVersion = m.APIMetadata().APIVersion
	k.Kind = m.APIMetadata().Kind
	k.konfig = kfg
	k.katalogDir = m.FirstEntryDir()

	for name, entry := range k.enabledCRDs {
		// Populate APITypes from crdFile before enrichment so isFullySpecified sees
		// the correct values. crdFile is the source of truth — overwrites any apiTypes.
		if entry.CRDFile != "" {
			if err := populateAPITypesFromCRDFile(&entry, k.katalogDir); err != nil {
				return nil, fmt.Errorf("CRD %q: %w", name, err)
			}
		}

		// Enrich enabled CRDs
		outcome, err := EnrichCRDEntry(&entry)
		if err != nil {
			return nil, err
		}
		entry.EnrichmentOutcome = outcome
		k.enabledCRDs[name] = entry
	}

	// Expand Motif imports declared in each operatorBox
	if err := k.expandMotifImports(); err != nil {
		return nil, err
	}

	// initialize conversion registry and admission registry
	k.conversionRegistry = NewInMemoryConversionRegistry()
	k.admissionRegistry = NewInMemoryAdmissionRegistry()

	// now safe to register rules
	for _, entry := range k.enabledCRDs {
		k.admissionRegistry.registerAdmissionRulesFromEntry(entry)
		k.conversionRegistry.registerConversionRulesFromSpec(entry)
	}

	// Apply defaults so CLI tools (simulate, validate, plan) get the same
	// field values as the runtime without needing to call ValidateConfig.
	if err := k.setDefaults(kfg); err != nil {
		return nil, err
	}

	return k.enabledCRDs, nil
}
