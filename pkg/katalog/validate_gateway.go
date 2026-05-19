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
	if k.HasNotification() {
		reasons = append(reasons, "  • notification")
	}

	return fmt.Errorf(
		"orkestra: gateway endpoint required but not configured\n\n"+
			"The following features are enabled and require a companion gateway:\n\n"+
			"%s\n\n"+
			"The gateway process owns webhook serving (TLS, /validate, /mutate,\n"+
			"/convert, /deletion-protection) and notification dispatch.\n"+
			"Without it these features will silently do nothing.\n\n"+
			"Fix — set gatewayEndpoint in your Katalog security block:\n\n"+
			"  security:\n"+
			"    gatewayEndpoint: \"http://orkestra-gateway.<namespace>.svc:8080\"\n\n"+
			"Or set the environment variable on the runtime deployment:\n\n"+
			"  ORK_GATEWAY_ENDPOINT=http://orkestra-gateway.<namespace>.svc:8080\n\n"+
			"To generate gateway manifests run:\n\n"+
			"  ork generate bundle --for gateway",
		strings.Join(reasons, "\n"),
	)
}
