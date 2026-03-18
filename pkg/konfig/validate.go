package konfig

import (
	"fmt"
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
	// Normalize app environment
	switch strings.ToLower(k.app.Environment) {
	case "dev", "development":
		k.app.Environment = "development"
	case "uat", "staging":
		k.app.Environment = "staging"
	case "live", "prod", "production":
		k.app.Environment = "production"
	default:
		k.app.Environment = "development"
	}
}

// Validate CRD katalog Konfiguration
func (k *Konfig) validateKatalogKonfig() error {
	mode := strings.ToLower(k.katalog.Mode)
	if mode == "" {
		k.katalog.Mode = "dynamic"
	}

	if mode != "dynamic" && mode != "typed" {
		return fmt.Errorf("invalid katalog mode: %s. Must be 'typed' or 'dynamic'", mode)
	}

	// Be sure dynamic mode has file path
	if mode == "dynamic" {
		if k.katalog.Path == "" {
			return fmt.Errorf("katalog path must be specified for dynamic mode. Use 'KATALOG_PATH' env variable")
		}
	}

	k.katalog.Mode = mode
	return nil
}
