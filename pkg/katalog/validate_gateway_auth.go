package katalog

import (
	"fmt"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// validateGatewayTokens is the dispatcher for gateway.api.auth.tokens validation.
func (k *Katalog) validateGatewayTokens() error {
	if !k.IsGatewayEnabled() || !k.Gateway.HasAPI() || k.Gateway.API.Auth.Empty() {
		return nil
	}
	if err := k.validateGatewayTokensCommon(); err != nil {
		return err
	}
	if err := k.validateGatewayStaticTokens(); err != nil {
		return err
	}
	return k.validateGatewayOIDCTokens()
}

// gatewayAPIAuthTokens returns the token entries from gateway.api.auth.tokens.
// Callers must have already confirmed the gateway and API are enabled.
func (k *Katalog) gatewayAPIAuthTokens() []orktypes.APIToken {
	return k.Gateway.API.Auth.Tokens
}

// GatewayTokenNames returns the names of all tokens declared in
// gateway.api.auth.tokens. Returns nil when the gateway is not enabled.
func (k *Katalog) GatewayTokenNames() []string {
	if !k.IsGatewayEnabled() {
		return nil
	}

	// Gateway is enabled, so API should exist
	if !k.Gateway.HasAPI() || k.Gateway.API.Auth.Empty() {
		return nil
	}

	names := make([]string, 0, len(k.gatewayAPIAuthTokens()))
	for _, t := range k.gatewayAPIAuthTokens() {
		names = append(names, t.Name)
	}
	return names
}

// validateGatewayTokensCommon checks for duplicate names and that each entry
// declares exactly one credential source.
func (k *Katalog) validateGatewayTokensCommon() error {
	seen := make(map[string]bool)
	var duplicates []string
	var errs []string

	for _, t := range k.gatewayAPIAuthTokens() {
		if seen[t.Name] {
			duplicates = append(duplicates, t.Name)
		}
		seen[t.Name] = true

		sourceCount := 0
		if strings.TrimSpace(t.Token) != "" {
			sourceCount++
		}
		if t.SecretRef != nil {
			sourceCount++
		}
		if t.IsOIDC() {
			sourceCount++
		}

		switch sourceCount {
		case 0:
			errs = append(errs, fmt.Sprintf(
				"  • gateway.api.auth.tokens[%q]: must set exactly one of: token, secretRef, githubOIDC, gitlabOIDC, vaultOIDC, oidc",
				t.Name))
		case 1:
			// ok
		default:
			errs = append(errs, fmt.Sprintf(
				"  • gateway.api.auth.tokens[%q]: only one of token, secretRef, githubOIDC, gitlabOIDC, vaultOIDC, oidc may be set",
				t.Name))
		}
	}

	if len(duplicates) > 0 {
		errs = append([]string{fmt.Sprintf(
			"  • gateway.api.auth.tokens: duplicate names: %s — each token name must be unique",
			red(strings.Join(duplicates, ", ")))}, errs...)
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s gateway.api.auth.tokens validation failed:\n%s", failureMark(), strings.Join(errs, "\n"))
	}
	return nil
}
