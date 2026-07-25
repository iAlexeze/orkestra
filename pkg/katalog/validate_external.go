package katalog

import (
	"fmt"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// validateExternalCalls checks all external: lists on a CRD entry.
//
// Enforces:
//  1. Each call must have a non-empty name.
//  2. No duplicate names within the same list.
func (k *Katalog) validateExternalCalls() error {
	for crdName, entry := range k.enabledCRDs {
		if ht := entry.OperatorBox.OnReconcile; ht != nil {
			if err := checkExternalNames(crdName, "onReconcile.external", ht.External); err != nil {
				return err
			}
		}
		if ht := entry.OperatorBox.OnCreate; ht != nil {
			if err := checkExternalNames(crdName, "onCreate.external", ht.External); err != nil {
				return err
			}
		}
		if r := entry.OperatorBox.Reconciler; r != nil && r.Hooks != nil {
			if err := checkExternalNames(crdName, "hooks.external", r.Hooks.External); err != nil {
				return err
			}
		}
		if entry.Validation != nil {
			if err := checkExternalNames(crdName, "validation.external", entry.Validation.External); err != nil {
				return err
			}
		}
		if entry.Mutation != nil {
			if err := checkExternalNames(crdName, "mutation.external", entry.Mutation.External); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkExternalNames(crdName, location string, calls []orktypes.ExternalCallSpec) error {
	seen := make(map[string]bool, len(calls))
	for i, call := range calls {
		if call.Name == "" {
			return fmt.Errorf("CRD %q: %s[%d]: name must not be empty", crdName, location, i)
		}
		if seen[call.Name] {
			return fmt.Errorf("CRD %q: %s: duplicate call name %q — names must be unique", crdName, location, call.Name)
		}
		seen[call.Name] = true
	}
	return nil
}
