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
func (c *Konfig) normalizeEnvironment() {
	// Normalize app environment
	switch strings.ToLower(c.app.Environment) {
	case "dev", "development":
		c.app.Environment = "development"
	case "uat", "staging":
		c.app.Environment = "staging"
	case "live", "prod", "production":
		c.app.Environment = "production"
	default:
		c.app.Environment = "development"
	}
}

// Validate CRD registry Konfiguration
func (c *Konfig) validateCRDKonfig() error {
	mode := strings.ToLower(c.katalog.Mode)
	if mode == "" {
		c.katalog.Mode = "go"
	}

	if mode != "go" && mode != "yaml" {
		return fmt.Errorf("invalid CRD registry mode: %s. Must be 'go' or 'yaml'", mode)
	}

	// Be sure yaml has file path
	if mode == "yaml" {
		if c.katalog.Path == "" {
			return fmt.Errorf("CRD registry path must be specified for yaml mode. Use 'KATALOG_PATH' env variable")
		}
	}

	c.katalog.Mode = mode
	return nil
}
