package konfig

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

// -----------------------------------------------------------------------------
func Validate() *validator.Validate {
	validate := validator.New(validator.WithRequiredStructEnabled())
	return validate
}

// Normalize environment
func (k *Konfig) normalizeEnvironment() {
	// Normalize ork environment
	switch strings.ToLower(k.ork.environment) {
	case DevShort, Development:
		k.ork.environment = Development
	case StagingShort, Staging:
		k.ork.environment = Staging
	case Live, ProdShort, Production:
		k.ork.environment = Production
	default:
		k.ork.environment = Development
	}
}
