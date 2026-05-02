package reconciler

import (
	"github.com/orkspace/orkestra/domain"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// RunValidation exposes runValidation for integration tests.
func RunValidation(obj domain.Object, cfg *orktypes.ValidationConfig, crdName string) *ValidationResult {
	return runValidation(obj, cfg, crdName)
}
