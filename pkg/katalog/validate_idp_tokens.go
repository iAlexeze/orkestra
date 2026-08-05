package katalog

import (
	"fmt"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// validateIDPTokenRestrictions confirms:
//
//  1. Every token name in idp.allowedTokens exists in gateway.applyAPI.auth.tokens.
//  2. Every permission string is one of the valid IDPOperation constants.
//  3. No permission list (global/schema/resources) repeats the same operation.
//  4. Schema permissions only support get/list — create/update/delete are ignored
//     when schema inherits from global.
//  5. When global is non-empty, class lists must be subsets of it.
//  6. (Warning) All three permission lists are empty — the token grants nothing.
//  7. (Warning) Namespace restrictions on a cluster-scoped CRD are ignored.
//  8. Every idp permitted namespace must be allowed at crd level.
//  9. (Warning) A token entry with an empty permissions list grants no access.
func (k *Katalog) validateIDPTokenRestrictions() error {
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
		orktypes.IDPOpGet:  true,
		orktypes.IDPOpList: true,
	}

	for crdName, crd := range k.enabledCRDs {
		if crd.IDP == nil || !crd.IDP.HasTokenRestrictions() {
			continue
		}

		for tokenName, perms := range crd.IDP.AllowedTokensMap() {
			gatewayTokensStr := strings.Join(gatewayTokens, ", ")

			// 1. Token must exist at the gateway level.
			if _, ok := knownTokens[tokenName]; !ok {
				return fmt.Errorf(
					"crd %q: idp.allowedTokens[%q] — token %q is not declared in gateway.applyAPI.auth.tokens\n"+
						"  Add the token there or remove it from allowedTokens.\n"+
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
					if !orktypes.IsValidIDPOperation(op) {
						return fmt.Errorf(
							"crd %q: idp.allowedTokens[%q].permissions.%s: %q is not a valid operation\n"+
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
							"crd %q: idp.allowedTokens[%q].permissions.%s: %q is listed more than once",
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
							"crd %q: idp.allowedTokens[%q].permissions.schema: %q is not allowed for schema endpoints\n"+
								"  Schema endpoints only support: get, list",
							crdName, tokenName, op,
						)
					}
				}
			} else if len(perms.Permissions.Global) > 0 {
				// Schema inherits from global — check for create/update/delete warnings.
				for _, op := range perms.Permissions.Global {
					if op == orktypes.IDPOpAll {
						continue // Wildcard is fine
					}
					if !validSchemaOps[op] {
						crd.Warnings.AddWarning(fmt.Sprintf(
							"crd %q: idp.allowedTokens[%q].permissions.global contains %q which is not valid for schema endpoints — "+
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
						if op == orktypes.IDPOpAll {
							return fmt.Errorf(
								"crd %q: idp.allowedTokens[%q].permissions.%s: %q "+
									"is broader than global permissions %v\n"+
									"  Set global: [\"*\"] to allow wildcard, or remove \"*\" from %s.",
								crdName, tokenName, classEntry.name, op,
								perms.Permissions.Global, classEntry.name,
							)
						}
						if !globalSet[op] {
							return fmt.Errorf(
								"crd %q: idp.allowedTokens[%q].permissions.%s: %q "+
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
					"crd %q: idp.allowedTokens[%q] has no permissions declared — "+
						"the token can authenticate but cannot perform any operation on this CRD",
					crdName, tokenName,
				))
				k.enabledCRDs[crdName] = crd
			}

			// 7. (Warning) Namespace restrictions on a cluster-scoped CRD.
			if !crd.IsNamespaced() && perms.IsNamespaceRestricted() {
				crd.Warnings.AddWarning(fmt.Sprintf(
					"crd %q: idp.allowedTokens[%q].namespaces is set but %q is cluster-scoped — "+
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
							"crd %q: idp.allowedTokens[%q].namespaces: %q is not in allowedNamespaces\n"+
								"  Allowed namespaces: %s",
							crdName, tokenName, ns, allowedList,
						)
					}
				}

				if crd.HasRestrictedNamespaces() {
					if restrictedNamespaces[crdName][ns] {
						restrictedList := strings.Join(crd.RestrictedNamespaces, ", ")
						return fmt.Errorf(
							"crd %q: idp.allowedTokens[%q].namespaces: %q is in restrictedNamespaces\n"+
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

// GatewayTokenNames returns the names of all tokens declared in
// gateway.applyAPI.auth.tokens. Returns nil when the gateway is not enabled.
func (k *Katalog) GatewayTokenNames() []string {
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
