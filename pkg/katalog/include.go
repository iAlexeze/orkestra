package katalog

import (
	"fmt"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func populateIDPFieldsFromInclude(entry *orktypes.CRDEntry, katalogDir string) error {
	if err := orktypes.ExpandIDPInclude(entry.IDP, katalogDir); err != nil {
		return fmt.Errorf("idp: %w", err)
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
