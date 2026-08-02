package katalog

import (
	"fmt"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// validateIDPTokenRestrictions confirms:
//
//  1. Every token name in idp.allowedTokens exists in gateway.applyAPI.auth.tokens.
//     A typo here would silently deny a token that should have access.
//
//  2. Every permission string is one of the valid IDPOperation constants.
//
//  3. Every idp permitted namespace must be allowed at crd level.
//
//  4. (Warning) A token entry with an empty permissions list grants no access.
//     Likely a misconfiguration; the token can authenticate but can do nothing.
//
//  5. (Warning) Namespace restrictions on a cluster-scoped CRD are silently
//     ignored at runtime. Surface this so the author can remove the dead config.
func (k *Katalog) validateIDPTokenRestrictions() error {
	// Build a set of token names that exist at the gateway level so we can
	// cross-reference them cheaply.
	gatewayTokens := k.gatewayTokenNames()
	knownTokens := make(map[string]struct{}, len(gatewayTokens))
	for _, name := range gatewayTokens {
		knownTokens[name] = struct{}{}
	}

	for crdName, crd := range k.enabledCRDs {
		if crd.IDP == nil || !crd.IDP.HasTokenRestrictions() {
			continue
		}

		gatewayTokensStr := strings.Join(gatewayTokens, ", ")
		for tokenName, perms := range crd.IDP.AllowedTokens {
			// 1. Token must exist at the gateway level.
			if _, ok := knownTokens[tokenName]; !ok {
				return fmt.Errorf(
					"crd %q: idp.allowedTokens[%q] — token %q is not declared in gateway.applyAPI.auth.tokens\n"+
						"  Add the token there or remove it from allowedTokens.\n"+
						"  Available tokens: %s",
					crdName, tokenName, tokenName, yellow(gatewayTokensStr),
				)
			}

			// 2. Every permission string must be a valid operation and unique
			ops := strings.Join(orktypes.ValidIDPOperations(), ", ")
			// 2. Every permission string must be a valid operation and unique
			seenOps := make(map[string]bool)
			var duplicates []string

			for _, p := range perms.Permissions {
				if !orktypes.IsValidIDPOperation(p) {
					return fmt.Errorf(
						"crd %q: idp.allowedTokens[%q].permissions: %q is not a valid operation\n"+
							"  Valid values: %s",
						crdName, tokenName, p, yellow(ops),
					)
				}
				if seenOps[p] {
					duplicates = append(duplicates, p)
				}
				seenOps[p] = true
			}

			if len(duplicates) > 0 {
				return fmt.Errorf(
					"crd %q: idp.allowedTokens[%q].permissions: duplicate operations found: %s",
					crdName, tokenName, red(strings.Join(duplicates, ", ")),
				)
			}

			// 3. Every namespace must be allowed at crd level
			for _, ns := range perms.Namespaces {
				if !crd.IsNamespaceAuthorized(ns) {
					if crd.HasAllowedNamespaces() {
						return fmt.Errorf(
							"namespace %q is not allowed for CRD %q.\n"+
								"Allowed namespaces: %s\n"+
								"Unauthorized namespace: %s",
							ns, crd.Name, strings.Join(crd.AllowedNamespaces, ","), red(ns),
						)
					}

					if crd.HasRestrictedNamespaces() {
						return fmt.Errorf(
							"namespace %q is restricted for CRD %q.\n"+
								"Restricted namespaces: %s\n"+
								"Unauthorized namespace: %s",
							ns, crd.Name, strings.Join(crd.RestrictedNamespaces, ","), red(ns),
						)
					}
				}
			}

			// 4. (Warning) Empty permissions list.
			if perms.Empty() {
				fmt.Printf("DEBUG: Empty permissions for %s/%s\n", crdName, tokenName)
				crd.Warnings.AddWarning(fmt.Sprintf(
					"crd %q: idp.allowedTokens[%q] has no permissions declared — "+
						"the token can authenticate but cannot perform any operation on this CRD",
					crdName, tokenName,
				))
			}

			// 5. (Warning) Namespace restrictions on a cluster-scoped CRD.
			if !crd.IsNamespaced() && !perms.Empty() {
				crd.Warnings.AddWarning(fmt.Sprintf(
					"crd %q: idp.allowedTokens[%q].namespaces is set but %q is cluster-scoped — "+
						"namespace restrictions are ignored for cluster-scoped resources",
					crdName, tokenName, crdName,
				))
			}
		}
	}

	return nil
}

// validateGatewayTokens ensures no duplicate token names in the gateway config
func (k *Katalog) validateGatewayTokens() error {
	if !k.IsGatewayEnabled() || !k.Gateway.HasApplyAPI() || k.Gateway.ApplyAPI.Auth.IsEmpty() {
		return nil
	}

	seenTokens := make(map[string]bool)
	var duplicates []string

	for _, t := range k.Gateway.ApplyAPI.Auth.Tokens {
		if seenTokens[t.Name] {
			duplicates = append(duplicates, t.Name)
		}
		seenTokens[t.Name] = true
	}

	if len(duplicates) > 0 {
		return fmt.Errorf(
			"gateway.applyAPI.auth.tokens: duplicate token names found: %s\n"+
				"Each token name must be unique",
			red(strings.Join(duplicates, ", ")),
		)
	}

	return nil
}

// gatewayTokenNames returns the names of all tokens declared in
// gateway.applyAPI.auth.tokens. Returns nil when the gateway is not enabled.
func (k *Katalog) gatewayTokenNames() []string {
	if !k.IsGatewayEnabled() {
		return nil
	}

	// Gateway is enabled, so ApplyAPI should exist
	if !k.Gateway.HasApplyAPI() || k.Gateway.ApplyAPI.Auth.IsEmpty() {
		return nil
	}

	names := make([]string, 0, len(k.Gateway.ApplyAPI.Auth.Tokens))
	for _, t := range k.Gateway.ApplyAPI.Auth.Tokens {
		names = append(names, t.Name)
	}
	return names
}
