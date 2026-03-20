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
	switch strings.ToLower(k.ork.Environment) {
	case DevShort, Development:
		k.ork.Environment = Development
	case StagingShort, Staging:
		k.ork.Environment = Staging
	case Live, ProdShort, Production:
		k.ork.Environment = Production
	default:
		k.ork.Environment = Development
	}
}
