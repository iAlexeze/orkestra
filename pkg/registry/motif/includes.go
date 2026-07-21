package motif

import (
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func expandIncludes(m *orktypes.Motif, dir string) error {
	if err := orktypes.ExpandNotesInclude(&m.Notes, dir); err != nil {
		return err
	}
	if err := orktypes.ExpandProfileInclude(&m.Profiles, dir); err != nil {
		return err
	}
	if err := orktypes.ExpandStatusInclude(m.Status, dir); err != nil {
		return err
	}
	if m.Admission != nil {
		if err := orktypes.ExpandValidationInclude(m.Admission.Validation, dir); err != nil {
			return err
		}
		if err := orktypes.ExpandMutationInclude(m.Admission.Mutation, dir); err != nil {
			return err
		}
	}
	return nil
}
