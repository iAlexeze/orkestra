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
	switch strings.ToLower(c.app.Environment) {
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
	mode := strings.ToLower(c.katalog.Mode)
	if mode == "" {
		k.katalog.Mode = "go"
	}

	if mode != "go" && mode != "yaml" {
		return fmt.Errorf("invalid katalog mode: %s. Must be 'go' or 'yaml'", mode)
	}

	// Be sure yaml has file path
	if mode == "yaml" {
		if k.katalog.Path == "" {
			return fmt.Errorf("katalog path must be specified for yaml mode. Use 'KATALOG_PATH' env variable")
		}
	}

	k.katalog.Mode = mode
	return nil
}
