package katalog

import (
	"fmt"
	"strings"
)

// validateGateway fails fast when gateway-dependent features are configured but
// no gatewayEndpoint is set. It collects every active feature that needs a
// gateway so the error tells the user everything at once.
func (k *Katalog) validateGateway() error {
	if !k.NeedsGateway() {
		return nil
	}
	if k.IsStandaloneGateway() {
		return nil
	}
	if k.GatewayEndpoint() != "" {
		return nil
	}

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
