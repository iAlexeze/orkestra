package types

import "slices"

// IDPOperation is a valid Apply API operation string.
type IDPOperation = string

const (
	// IDPOpGet lists or reads a single CR via GET /api/v1/resources/...
	IDPOpGet IDPOperation = "get"

	// IDPOpList lists all CRs via GET /api/v1/resources/{kind}/{ns}
	IDPOpList IDPOperation = "list"

	// IDPOpCreate creates a new CR via POST /api/v1/apply when the CR does
	// not already exist in the cluster.
	IDPOpCreate IDPOperation = "create"

	// IDPOpUpdate updates an existing CR via POST /api/v1/apply when the CR
	// already exists in the cluster.
	IDPOpUpdate IDPOperation = "update"

	// IDPOpDelete deletes a CR via DELETE /api/v1/resources/...
	IDPOpDelete IDPOperation = "delete"

	// IDPOpAll is a wildcard that grants every operation. When present in the
	// permissions list, all other entries are redundant.
	IDPOpAll IDPOperation = "*"
)

// idpValidOperations is the authoritative set used by ork validate and ork idp.
var idpValidOperations = []string{
	IDPOpGet, IDPOpList, IDPOpCreate, IDPOpUpdate, IDPOpDelete, IDPOpAll,
}

// idpValidClasses is the authoritative set used by ork idp
var idpValidClasses = []string{
	IDPClassSchema, IDPClassResources,
}

// IDPEndpointClass identifies which Apply API endpoint group a permission
// applies to.
type IDPEndpointClass = string

const (
	// IDPClassSchema covers GET /api/v1/schema and GET /api/v1/raw-schema.
	// The relevant operation is always "get".
	IDPClassSchema IDPEndpointClass = "schema"

	// IDPClassResources covers all /api/v1/resources/... endpoints:
	// get, list, create, update, delete.
	IDPClassResources IDPEndpointClass = "resources"
)

// IDPPermissionSet declares operations permitted across endpoint classes.
//
// Resolution priority per class:
//  1. Class-specific list (schema: or resources:) when non-empty.
//  2. Global list when class-specific is empty.
//  3. No access when both are empty.
//
// When global is non-empty, class-specific lists must be subsets of it.
// ork validate enforces this. When global is empty, each class list is
// fully independent — fine-grained mode.
//
// Example — one token, two different access levels:
//
//	permissions:
//	  schema:    [get]              # can browse the catalog
//	  resources: [create, update]   # can create and update CRs
//
// Example — global with a narrower schema:
//
//	permissions:
//	  global:    [get, list, create, update]
//	  schema:    [get]              # further restricted for schema
//	  # resources inherits global
//
// Example — full access (star):
//
//	permissions:
//	  global: ["*"]
type IDPPermissionSet struct {
	// Global applies to all endpoint classes that do not declare their own list.
	// When non-empty, schema and resources must be subsets of this list.
	Global []string `yaml:"global,omitempty" json:"global,omitempty"`

	// Schema controls access to the schema endpoints.
	// Falls back to global when empty.
	Schema []string `yaml:"schema,omitempty" json:"schema,omitempty"`

	// Resources controls access to the resource endpoints.
	// Falls back to global when empty.
	Resources []string `yaml:"resources,omitempty" json:"resources,omitempty"`
}

// IDPTokenPermissions declares the access a named token has on a CRD.
type IDPTokenPermissions struct {
	// Namespaces restricts the token to specific namespaces.
	// Empty means all namespaces. Ignored for cluster-scoped CRDs.
	Namespaces []string `yaml:"namespaces,omitempty" json:"namespaces,omitempty"`

	// Permissions declares the operations this token may perform, split by
	// endpoint class. See IDPPermissionSet for the resolution rules.
	Permissions IDPPermissionSet `yaml:"permissions,omitempty" json:"permissions,omitempty"`
}

// activePerms returns the permission list that applies to the given endpoint
// class for this token. Returns nil when no permissions apply (no access).
func (p IDPTokenPermissions) activePerms(class IDPEndpointClass) []string {
	switch class {
	case IDPClassSchema:
		if len(p.Permissions.Schema) > 0 {
			return p.Permissions.Schema
		}
	case IDPClassResources:
		if len(p.Permissions.Resources) > 0 {
			return p.Permissions.Resources
		}
	}
	// Fall back to global.
	return p.Permissions.Global
}

// IsEmpty reports whether no permissions at all are declared.
// A token with empty permissions can authenticate but cannot do anything.
func (p IDPPermissionSet) IsEmpty() bool {
	return len(p.Global) == 0 && len(p.Schema) == 0 && len(p.Resources) == 0
}

// IsGlobalEmpty reports whether no global permissions are declared.
func IsGlobalEmpty(p IDPPermissionSet) bool {
	return len(p.Global) == 0
}

// IsSchemaEmpty reports whether no schema permissions are declared.
func IsSchemaEmpty(p IDPPermissionSet) bool {
	return len(p.Schema) == 0
}

// IsResourcesEmpty reports whether no resource permissions are declared.
func IsResourcesEmpty(p IDPPermissionSet) bool {
	return len(p.Resources) == 0
}

// IsValidIDPOperation reports whether s is one of the declared IDPOperation
// constants. Defined here to avoid importing types into the validate file.
func IsValidIDPOperation(s string) bool {
	for _, op := range idpValidOperations {
		if s == op {
			return true
		}
	}
	return false
}

// ValidIDPOperations returns the list of valid IDP operations.
func ValidIDPOperations() []string {
	return idpValidOperations
}

// IsValidIDPEndpointClass reports whether is one of the declared IDPClass constants.
func IsValidIDPEndpointClass(s string) bool {
	for _, class := range idpValidClasses {
		if s == class {
			return true
		}
	}
	return false
}

// ValidIDPEndpointClasses returns the list of valid IDP endpoint classes.
func ValidIDPEndpointClasses() []string {
	return idpValidClasses
}

// HasNamespace returns true if the token is allowed in the given namespace.
// Empty namespaces list means all namespaces are allowed.
// For cluster-scoped CRDs, namespace checks are skipped.
func (p IDPTokenPermissions) HasNamespace(namespace string) bool {
	if len(p.Namespaces) == 0 {
		return true // All namespaces allowed
	}
	for _, ns := range p.Namespaces {
		if ns == namespace {
			return true
		}
	}
	return false
}

// AllowedTokensMap returns the map of allowed tokens.
// Convenience method to avoid dereferencing AllowedTokens.Tokens directly.
func (idp *IDPConfig) AllowedTokensMap() map[string]IDPTokenPermissions {
	if idp == nil {
		return nil
	}
	return idp.AllowedTokens.Tokens
}

// Namespaces returns the list of allowed namespaces.
// Empty slice means all namespaces are allowed.
func (p IDPTokenPermissions) TokenNamespaces() []string {
	return p.Namespaces
}

// IsNamespaceRestricted returns true if the token has namespace restrictions.
func (p IDPTokenPermissions) IsNamespaceRestricted() bool {
	return len(p.Namespaces) > 0
}

// ─── Scope Existence Methods ──────────────────────────────────────────────

// HasGlobalPermissions returns true if global permissions are declared.
func (p IDPTokenPermissions) HasGlobalPermissions() bool {
	return len(p.Permissions.Global) > 0
}

// HasSchemaPermissions returns true if schema permissions are declared.
func (p IDPTokenPermissions) HasSchemaPermissions() bool {
	return len(p.Permissions.Schema) > 0
}

// HasResourcesPermissions returns true if resource permissions are declared.
func (p IDPTokenPermissions) HasResourcesPermissions() bool {
	return len(p.Permissions.Resources) > 0
}

// HasAnyPermissions returns true if any permissions are declared.
func (p IDPTokenPermissions) HasAnyPermissions() bool {
	return p.HasGlobalPermissions() || p.HasSchemaPermissions() || p.HasResourcesPermissions()
}

// HasGlobalWildcard returns true if the permissions include "*" (all operations) at global level.
func (p IDPTokenPermissions) HasGlobalWildcard() bool {
	return slices.Contains(p.Permissions.Global, IDPOpAll)
}

// HasResourcesWildcard returns true if the permissions include "*" (all operations) at resource level.
func (p IDPTokenPermissions) HasResourcesWildcard() bool {
	return slices.Contains(p.Permissions.Resources, IDPOpAll)
}

// HasSchemaWildcard returns true if the permissions include "*" (all operations) at schema level.
func (p IDPTokenPermissions) HasSchemaWildcard() bool {
	return slices.Contains(p.Permissions.Schema, IDPOpAll)
}

// HasWildcard returns true if any permission list contains "*".
func (p IDPTokenPermissions) HasWildcard() bool {
	return p.HasGlobalWildcard() || p.HasSchemaWildcard() || p.HasResourcesWildcard()
}

// HasOperation returns true if the permission set includes the given operation
// for the specified endpoint class.
//
// If class-specific permissions are empty, falls back to global.
func (p IDPTokenPermissions) HasOperation(class IDPEndpointClass, op string) bool {
	perms := p.activePerms(class)
	if len(perms) == 0 {
		return false
	}
	for _, perm := range perms {
		if perm == IDPOpAll || perm == op {
			return true
		}
	}
	return false
}

// HasGlobalOperation checks if the operation is in the global list.
func (p IDPTokenPermissions) HasGlobalOperation(op string) bool {
	for _, perm := range p.Permissions.Global {
		if perm == IDPOpAll || perm == op {
			return true
		}
	}
	return false
}

// HasSchemaOperation checks if the operation is in the schema list.
func (p IDPTokenPermissions) HasSchemaOperation(op string) bool {
	for _, perm := range p.Permissions.Schema {
		if perm == IDPOpAll || perm == op {
			return true
		}
	}
	return false
}

// HasResourcesOperation checks if the operation is in the resources list.
func (p IDPTokenPermissions) HasResourcesOperation(op string) bool {
	for _, perm := range p.Permissions.Resources {
		if perm == IDPOpAll || perm == op {
			return true
		}
	}
	return false
}
