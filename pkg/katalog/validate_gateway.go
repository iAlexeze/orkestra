package katalog

import (
	"fmt"
	"strings"
)

// validateGateway fails fast when gateway-dependent features are configured but
// no gatewayEndpoint is set. It collects every active feature that needs a
// gateway so the error tells the user everything at once.
// validateGateway fails fast when both standalone mode and an explicit endpoint are set.
// A simple "enabled: true" works
func (k *Katalog) validateGateway() error {
	if !k.NeedsGateway() {
		return nil
	}

	// If enabled, return early
	if k.IsGatewayEnabled() {
		return nil
	}

	// Conflict: only one of standalone or explicit endpoint can be used
	if k.IsStandaloneGateway() && k.GatewayEndpoint() != "" {
		return fmt.Errorf(
			"orkestra: cannot enable both gateway.standalone and gateway.endpoint\n\n"+
				"Conflicting configuration:\n"+
				"  • gateway.standalone = true\n"+
				"  • gateway.endpoint = %q\n\n"+
				"Fix - set only one:\n"+
				"  • Standalone mode: set gateway.standalone = true and unset gateway.endpoint\n"+
				"  • Runtime Companion mode: set gateway.endpoint and set gateway.standalone = false (or omit it)",
			k.GatewayEndpoint(),
		)
	}

	// Standalone gateway mode – no endpoint needed
	if k.IsStandaloneGateway() {
		return nil
	}

	// External gateway endpoint provided – ok
	if k.GatewayEndpoint() != "" {
		return nil
	}

	// If we reach here, gateway is needed but no standalone or endpoint was set.
	var reasons []string
	if k.IsDeletionProtectionEnabled() {
		reasons = append(reasons, "  • security.deletionProtection")
	}
	if k.IsAdmissionEnabled() {
		reasons = append(reasons, "  • security.webhooks.admission")
	}
	if k.IsConversionEnabled() {
		reasons = append(reasons, "  • security.conversion")
	}
	if k.IsNamespaceProtectionEnabled() {
		reasons = append(reasons, "  • security.namespaceProtection")
	}
	if k.HasNotification() && !k.IsNotificationStandalone() {
		reasons = append(reasons, "  • notification")
	}

	return fmt.Errorf(
		"orkestra: gateway endpoint required but not configured\n\n"+
			"Enabled features that need a gateway:\n%s\n\n"+
			"Fix - set gateway.endpoint, enable gateway.standalone, or set ORK_GATEWAY_ENDPOINT.",
		strings.Join(reasons, "\n"),
	)
}
