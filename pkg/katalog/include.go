package katalog

import (
	"fmt"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func populateAllServeFieldsFromInclude(entry *orktypes.CRDEntry, katalogDir string) error {
	if err := orktypes.ExpandServeInclude(entry.Serve, katalogDir); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	if err := orktypes.ExpandServeTargetShorthand(entry.Serve); err != nil {
		return fmt.Errorf("serve.target: %w", err)
	}
	if err := orktypes.ExpandServeTargetIncludes(entry.Serve, katalogDir); err != nil {
		return fmt.Errorf("serve.target: %w", err)
	}
	return nil
}

func populateStatusFieldsFromInclude(entry *orktypes.CRDEntry, katalogDir string) error {
	if err := orktypes.ExpandStatusInclude(entry.OperatorBox.Status, katalogDir); err != nil {
		return fmt.Errorf("status: %w", err)
	}
	return nil
}

func populateValidationRulesFromInclude(entry *orktypes.CRDEntry, katalogDir string) error {
	if err := orktypes.ExpandValidationInclude(entry.Validation, katalogDir); err != nil {
		return fmt.Errorf("validation: %w", err)
	}
	return nil
}

func populateMutationRulesFromInclude(entry *orktypes.CRDEntry, katalogDir string) error {
	if err := orktypes.ExpandMutationInclude(entry.Mutation, katalogDir); err != nil {
		return fmt.Errorf("mutation: %w", err)
	}
	return nil
}

func populateConversionPathsFromInclude(entry *orktypes.CRDEntry, katalogDir string) error {
	if err := orktypes.ExpandConversionInclude(entry.Conversion, katalogDir); err != nil {
		return fmt.Errorf("conversion: %w", err)
	}
	return nil
}

func populateExternalCallsFromInclude(entry *orktypes.CRDEntry, katalogDir string) error {
	var err error
	if entry.OperatorBox.OnReconcile != nil {
		entry.OperatorBox.OnReconcile.External, err = orktypes.ExpandExternalCalls(entry.OperatorBox.OnReconcile.External, katalogDir)
		if err != nil {
			return fmt.Errorf("onReconcile.external: %w", err)
		}
	}
	if entry.OperatorBox.OnCreate != nil {
		entry.OperatorBox.OnCreate.External, err = orktypes.ExpandExternalCalls(entry.OperatorBox.OnCreate.External, katalogDir)
		if err != nil {
			return fmt.Errorf("onCreate.external: %w", err)
		}
	}
	if r := entry.OperatorBox.Reconciler; r != nil && r.Hooks != nil {
		r.Hooks.External, err = orktypes.ExpandExternalCalls(r.Hooks.External, katalogDir)
		if err != nil {
			return fmt.Errorf("hooks.external: %w", err)
		}
	}
	if entry.Validation != nil {
		entry.Validation.External, err = orktypes.ExpandExternalCalls(entry.Validation.External, katalogDir)
		if err != nil {
			return fmt.Errorf("validation.external: %w", err)
		}
	}
	if entry.Mutation != nil {
		entry.Mutation.External, err = orktypes.ExpandExternalCalls(entry.Mutation.External, katalogDir)
		if err != nil {
			return fmt.Errorf("mutation.external: %w", err)
		}
	}
	return nil
}
