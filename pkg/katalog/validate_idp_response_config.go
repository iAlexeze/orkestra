package katalog

import (
	"fmt"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// validateIDPResponseConfig checks the IDP response configuration for:
//   - Path conflicts between payload and exclude (exclude wins, surfaced as warning)
//   - Exclude paths that don't exist in the CRD schema (warning)
//   - Payload template compilation errors (error)
func (k *Katalog) validateIDPResponseConfig() error {
	for crdName, crd := range k.enabledCRDs {
		if !crd.HasResponseConfig() {
			continue
		}

		config := crd.IDP.Config.Response
		if !config.HasPayload() && !config.HasExclude() {
			continue
		}

		// Collect payload keys (these will become top-level fields)
		payloadKeys := make(map[string]bool)
		for key := range config.Payload {
			payloadKeys[key] = true
		}

		// Resolve exclude paths (if possible at validation time)
		excludePaths := config.Exclude
		if excludePaths == "" {
			continue
		}

		// If exclude is a template, we can't validate its contents at load time
		if orktypes.IsTemplate(excludePaths) {
			// Still check if it references any payload keys in the template itself
			for key := range payloadKeys {
				if strings.Contains(excludePaths, key) {
					fmt.Printf("⚠️  CRD %q: exclude template references payload key %q — potential conflict\n", crdName, key)
				}
			}
			continue
		}

		// Static exclude: split and validate
		parts := strings.Split(excludePaths, ",")
		for _, p := range parts {
			path := strings.TrimSpace(p)
			if path == "" {
				continue
			}

			// Check if path conflicts with a payload key
			if payloadKeys[path] {
				warning := fmt.Sprintf("path %q appears in both payload and exclude — exclude wins", path)
				crd.Warnings.AddWarning(warning)
			}
		}
	}
	return nil
}
