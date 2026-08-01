package katalog

import (
	orkexternal "github.com/orkspace/orkestra/pkg/external"
)

func (k *Katalog) validateExternalCalls() error {
	for crdName, entry := range k.enabledCRDs {
		if ht := entry.OperatorBox.OnReconcile; ht != nil {
			if err := orkexternal.ValidateCalls(crdName, "onReconcile.external", ht.External); err != nil {
				return err
			}
		}
		if ht := entry.OperatorBox.OnCreate; ht != nil {
			if err := orkexternal.ValidateCalls(crdName, "onCreate.external", ht.External); err != nil {
				return err
			}
		}
		if r := entry.OperatorBox.Reconciler; r != nil && r.Hooks != nil {
			if err := orkexternal.ValidateCalls(crdName, "hooks.external", r.Hooks.External); err != nil {
				return err
			}
		}
		if entry.Validation != nil {
			if err := orkexternal.ValidateCalls(crdName, "validation.external", entry.Validation.External); err != nil {
				return err
			}
		}
		if entry.Mutation != nil {
			if err := orkexternal.ValidateCalls(crdName, "mutation.external", entry.Mutation.External); err != nil {
				return err
			}
		}
	}
	return nil
}
