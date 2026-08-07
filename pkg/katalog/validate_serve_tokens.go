package katalog

import (
	"fmt"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// validateServeTokenRestrictions confirms:
//
//  1. Every token name in serve.tokens exists in gateway.api.auth.tokens.
//  2. Every permission string is one of the valid ServeOperation constants.
//  3. No permission list (global/schema/resources) repeats the same operation.
//  4. Schema permissions only support get/list — create/update/delete are ignored
//     when schema inherits from global.
//  5. When global is non-empty, class lists must be subsets of it.
//  6. (Warning) All three permission lists are empty — the token grants nothing.
//  7. (Warning) Namespace restrictions on a cluster-scoped CRD are ignored.
//  8. Every serve permitted namespace must be allowed at crd level.
//  9. (Warning) A token entry with an empty permissions list grants no access.
func (k *Katalog) validateServeTokenRestrictions() error {
	gatewayTokens := k.GatewayTokenNames()
	knownTokens := make(map[string]struct{}, len(gatewayTokens))
	for _, name := range gatewayTokens {
		knownTokens[name] = struct{}{}
	}

	allowedNamespaces := make(map[string]map[string]bool)
	restrictedNamespaces := make(map[string]map[string]bool)

	// Keyed by the enabledCRDs map key.
	for crdName, crd := range k.enabledCRDs {
		allowedNamespaces[crdName] = make(map[string]bool)
		for _, ns := range crd.AllowedNamespaces {
			allowedNamespaces[crdName][ns] = true
		}
		restrictedNamespaces[crdName] = make(map[string]bool)
		for _, ns := range crd.RestrictedNamespaces {
			restrictedNamespaces[crdName][ns] = true
		}
	}

	validSchemaOps := map[string]bool{
		orktypes.ServeOpGet:  true,
		orktypes.ServeOpList: true,
	}

	for crdName, crd := range k.enabledCRDs {
		if crd.Serve == nil || !crd.Serve.HasTokenRestrictions() {
			continue
		}

		for tokenName, perms := range crd.Serve.TokensMap() {
			gatewayTokensStr := strings.Join(gatewayTokens, ", ")

			// 1. Token must exist at the gateway level.
			if _, ok := knownTokens[tokenName]; !ok {
				return fmt.Errorf(
					"crd %q: serve.tokens[%q] — token %q is not declared in gateway.api.auth.tokens\n"+
						"  Add the token there or remove it from tokens.\n"+
						"  Available tokens: %s",
					crdName, tokenName, tokenName, yellow(gatewayTokensStr),
				)
			}

			// 2. Validate every operation string in all three lists.
			permissionLists := []struct {
				name string
				ops  []string
			}{
				{"global", perms.Permissions.Global},
				{"schema", perms.Permissions.Schema},
				{"resources", perms.Permissions.Resources},
			}

			for _, list := range permissionLists {
				for _, op := range list.ops {
					if !orktypes.IsValidServeOperation(op) {
						return fmt.Errorf(
							"crd %q: serve.tokens[%q].permissions.%s: %q is not a valid operation\n"+
								"  Valid values: get, list, create, update, delete, *",
							crdName, tokenName, list.name, op,
						)
					}
				}
			}

			// 3. No permission list repeats the same operation.
			for _, list := range permissionLists {
				seen := make(map[string]bool, len(list.ops))
				for _, op := range list.ops {
					if seen[op] {
						return fmt.Errorf(
							"crd %q: serve.tokens[%q].permissions.%s: %q is listed more than once",
							crdName, tokenName, list.name, op,
						)
					}
					seen[op] = true
				}
			}

			// 4. Schema permissions validation.
			if len(perms.Permissions.Schema) > 0 {
				// Explicit schema list — validate each operation.
				for _, op := range perms.Permissions.Schema {
					if !validSchemaOps[op] {
						return fmt.Errorf(
							"crd %q: serve.tokens[%q].permissions.schema: %q is not allowed for schema endpoints\n"+
								"  Schema endpoints only support: get, list",
							crdName, tokenName, op,
						)
					}
				}
			} else if len(perms.Permissions.Global) > 0 {
				// Schema inherits from global — check for create/update/delete warnings.
				for _, op := range perms.Permissions.Global {
					if op == orktypes.ServeOpAll {
						continue // Wildcard is fine
					}
					if !validSchemaOps[op] {
						crd.Warnings.AddWarning(fmt.Sprintf(
							"crd %q: serve.tokens[%q].permissions.global contains %q which is not valid for schema endpoints — "+
								"it will be ignored for schema and only apply to resources",
							crdName, tokenName, op,
						))
						k.enabledCRDs[crdName] = crd
					}
				}
			}

			// 5. When global is non-empty, class lists must be subsets.
			if perms.HasGlobalPermissions() && !perms.HasGlobalWildcard() {
				globalSet := toStringSet(perms.Permissions.Global)

				for _, classEntry := range []struct {
					name string
					ops  []string
				}{
					{"schema", perms.Permissions.Schema},
					{"resources", perms.Permissions.Resources},
				} {
					for _, op := range classEntry.ops {
						if op == orktypes.ServeOpAll {
							return fmt.Errorf(
								"crd %q: serve.tokens[%q].permissions.%s: %q "+
									"is broader than global permissions %v\n"+
									"  Set global: [\"*\"] to allow wildcard, or remove \"*\" from %s.",
								crdName, tokenName, classEntry.name, op,
								perms.Permissions.Global, classEntry.name,
							)
						}
						if !globalSet[op] {
							return fmt.Errorf(
								"crd %q: serve.tokens[%q].permissions.%s: %q "+
									"is not in global permissions %v\n"+
									"  Class permissions must be a subset of global, "+
									"or set global to empty for fine-grained mode.",
								crdName, tokenName, classEntry.name, op,
								perms.Permissions.Global,
							)
						}
					}
				}
			}

			// 6. (Warning) All permissions are empty — token grants nothing.
			if perms.Permissions.IsEmpty() {
				crd.Warnings.AddWarning(fmt.Sprintf(
					"crd %q: serve.tokens[%q] has no permissions declared — "+
						"the token can authenticate but cannot perform any operation on this CRD",
					crdName, tokenName,
				))
				k.enabledCRDs[crdName] = crd
			}

			// 7. (Warning) Namespace restrictions on a cluster-scoped CRD.
			if !crd.IsNamespaced() && perms.IsNamespaceRestricted() {
				crd.Warnings.AddWarning(fmt.Sprintf(
					"crd %q: serve.tokens[%q].namespaces is set but %q is cluster-scoped — "+
						"namespace restrictions are ignored for cluster-scoped resources",
					crdName, tokenName, crdName,
				))
				k.enabledCRDs[crdName] = crd
			}

			// 8. Every namespace must be allowed at CRD level.
			for _, ns := range perms.Namespaces {
				if crd.HasAllowedNamespaces() {
					if !allowedNamespaces[crdName][ns] {
						allowedList := strings.Join(crd.AllowedNamespaces, ", ")
						return fmt.Errorf(
							"crd %q: serve.tokens[%q].namespaces: %q is not in allowedNamespaces\n"+
								"  Allowed namespaces: %s",
							crdName, tokenName, ns, allowedList,
						)
					}
				}

				if crd.HasRestrictedNamespaces() {
					if restrictedNamespaces[crdName][ns] {
						restrictedList := strings.Join(crd.RestrictedNamespaces, ", ")
						return fmt.Errorf(
							"crd %q: serve.tokens[%q].namespaces: %q is in restrictedNamespaces\n"+
								"  Restricted namespaces: %s",
							crdName, tokenName, ns, restrictedList,
						)
					}
				}
			}
		}
	}

	return nil
}

// validateGatewayTokens checks gateway.api.auth.tokens for:
//  1. Duplicate names.
//  2. Each entry must have exactly one of token or secretRef set.
//  3. token values must be ${ENV_VAR} references — literals are rejected at
//     gateway startup, so we catch them here to give faster feedback.
//  4. secretRef entries must supply both name and key.
func (k *Katalog) validateGatewayTokens() error {
	if !k.IsGatewayEnabled() || !k.Gateway.HasAPI() || k.Gateway.API.Auth.IsEmpty() {
		return nil
	}

	seenTokens := make(map[string]bool)
	var duplicates []string
	var errs []string

	for _, t := range k.Gateway.API.Auth.Tokens {
		if seenTokens[t.Name] {
			duplicates = append(duplicates, t.Name)
		}
		seenTokens[t.Name] = true

		hasToken := strings.TrimSpace(t.Token) != ""
		hasRef := t.SecretRef != nil

		if !hasToken && !hasRef {
			errs = append(errs, fmt.Sprintf(
				"  • gateway.api.auth.tokens[%q]: must set either token or secretRef", t.Name))
			continue
		}
		if hasToken && hasRef {
			errs = append(errs, fmt.Sprintf(
				"  • gateway.api.auth.tokens[%q]: token and secretRef are mutually exclusive", t.Name))
			continue
		}

		if hasToken {
			v := strings.TrimSpace(t.Token)
			if !strings.HasPrefix(v, "${") || !strings.HasSuffix(v, "}") {
				errs = append(errs, fmt.Sprintf(
					"  • gateway.api.auth.tokens[%q]: token must be an ${ENV_VAR} reference, got literal — "+
						"set the value via extraEnv in Helm and reference it as ${MY_VAR}",
					t.Name))
			}
		}

		if hasRef {
			if strings.TrimSpace(t.SecretRef.Name) == "" {
				errs = append(errs, fmt.Sprintf(
					"  • gateway.api.auth.tokens[%q]: secretRef.name is required", t.Name))
			}
			if strings.TrimSpace(t.SecretRef.Key) == "" {
				errs = append(errs, fmt.Sprintf(
					"  • gateway.api.auth.tokens[%q]: secretRef.key is required", t.Name))
			}
		}
	}

	if len(duplicates) > 0 {
		errs = append([]string{fmt.Sprintf(
			"  • gateway.api.auth.tokens: duplicate names: %s — each token name must be unique",
			red(strings.Join(duplicates, ", ")))}, errs...)
	}

	if len(errs) > 0 {
		return fmt.Errorf("gateway.api.auth.tokens validation failed:\n%s", strings.Join(errs, "\n"))
	}

	return nil
}

// GatewayTokenNames returns the names of all tokens declared in
// gateway.api.auth.tokens. Returns nil when the gateway is not enabled.
func (k *Katalog) GatewayTokenNames() []string {
	if !k.IsGatewayEnabled() {
		return nil
	}

	// Gateway is enabled, so API should exist
	if !k.Gateway.HasAPI() || k.Gateway.API.Auth.IsEmpty() {
		return nil
	}

	names := make([]string, 0, len(k.Gateway.API.Auth.Tokens))
	for _, t := range k.Gateway.API.Auth.Tokens {
		names = append(names, t.Name)
	}
	return names
}
