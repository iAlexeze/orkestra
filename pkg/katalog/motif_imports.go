// pkg/katalog/motif_imports.go
//
// Expands imports: blocks at the CRD level (spec.crds[].imports).
// Each import loads the referenced Motif, binds its inputs from with:,
// and merges the expanded resources, status, and admission rules into the CRD.
package katalog

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/motif"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// expandMotifImports resolves all imports entries across enabled CRDs.
// For each import, it expands the motif and merges the result into the CRD entry.
// After expansion, the imports list is cleared (they have been inlined).
//
// Note: Imports are defined at the CRD level (spec.crds[].imports),
// not inside operatorBox. This allows motifs to contribute to multiple
// aspects of the CRD (resources, status, admission rules) without being
// tied to the operatorbox configuration.
func (k *Katalog) expandMotifImports() error {
	for name, entry := range k.enabledCRDs {
		if len(entry.Imports) == 0 {
			continue
		}

		for i, imp := range entry.Imports {
			expanded, err := k.loadAndExpandImport(&imp)
			if err != nil {
				return fmt.Errorf("CRD %q: operatorBox.imports[%d]: %w", name, i, err)
			}
			if err := k.mergeExpandedMotif(&entry, expanded); err != nil {
				return fmt.Errorf("CRD %q: operatorBox.imports[%d]: merging motif %q: %w",
					name, i, imp.Motif, err)
			}
		}

		// Clear imports after successful expansion
		entry.Imports = nil
		k.enabledCRDs[name] = entry
	}
	return nil
}

// loadAndExpandImport loads the motif from its source (file, OCI, Git) and expands
// it using the provided bindings. Returns the expanded motif or an error.
func (k *Katalog) loadAndExpandImport(imp *orktypes.MotifImport) (*motif.ExpandedMotif, error) {
	m, err := motif.LoadImport(imp)
	if err != nil {
		return nil, fmt.Errorf("loading motif %q: %w", imp.Motif, err)
	}
	expanded, err := motif.Expand(m, imp.With)
	if err != nil {
		return nil, fmt.Errorf("expanding motif %q: %w", imp.Motif, err)
	}
	return expanded, nil
}

// mergeExpandedMotif merges the resources, status, and admission rules from an
// expanded motif into the target CRD entry. It respects the existing fields
// (appending rules, preserving order, and merging condition flags sensibly).
func (k *Katalog) mergeExpandedMotif(entry *orktypes.CRDEntry, expanded *motif.ExpandedMotif) error {
	// Merge resources into operatorBox.OnReconcile
	if expanded.HasResources() {
		if !entry.HasOnReconcile() {
			entry.OperatorBox.OnReconcile = &orktypes.HookTemplates{}
		}
		motif.MergeHookTemplates(entry.OperatorBox.OnReconcile, expanded.Resources)
	}

	// Merge status fields
	if expanded.HasStatus() {
		if !entry.HasStatusFields() {
			entry.OperatorBox.Status = &orktypes.StatusConfig{}
		}
		entry.OperatorBox.Status.Fields = append(entry.OperatorBox.Status.Fields, expanded.Status.Fields...)
		if entry.OperatorBox.Status.Conditions == nil && expanded.Status.Conditions != nil {
			entry.OperatorBox.Status.Conditions = expanded.Status.Conditions
		}
	}

	// Merge admission (validation + mutation) rules – these are at CRD level, not operatorBox
	if expanded.HasAdmission() {
		// Merge validation rules
		if expanded.Admission.HasValidationRules() {
			if entry.Validation == nil {
				entry.Validation = &orktypes.ValidationConfig{}
			}
			entry.Validation.Rules = append(entry.Validation.Rules, expanded.Admission.Validation.Rules...)
		}
		// Merge mutation rules
		if expanded.Admission.HasMutationRules() {
			if entry.Mutation == nil {
				entry.Mutation = &orktypes.MutationConfig{}
			}
			entry.Mutation.Rules = append(entry.Mutation.Rules, expanded.Admission.Mutation.Rules...)
		}
	}

	return nil
}
