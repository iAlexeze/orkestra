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

// idpValidOperations is the authoritative set used by ork validate.
var idpValidOperations = []string{
	IDPOpGet, IDPOpList, IDPOpCreate, IDPOpUpdate, IDPOpDelete, IDPOpAll,
}

// IDPTokenPermissions declares the operations a named token may perform on a
// CRD and, optionally, the namespaces it may access.
//
// Example — ci-pipeline may create and update in staging only:
//
//	allowedTokens:
//	  ci-pipeline:
//	    permissions: [create, update]
//	    namespaces:  [team-payments-staging]
//
// Example — control-center has full access everywhere:
//
//	allowedTokens:
//	  control-center:
//	    permissions: ["*"]
type IDPTokenPermissions struct {
	// Permissions is the list of allowed operations.
	// Valid values: get, list, create, update, delete, * (all).
	// An empty list is allowed by the parser but caught by ork validate as a
	// warning — the token is listed but grants no access.
	Permissions []string `yaml:"permissions,omitempty" json:"permissions,omitempty"`

	// Namespaces restricts the token to specific namespaces.
	// When empty (the default), all namespaces are permitted.
	// Ignored for cluster-scoped CRDs; ork validate emits a warning.
	Namespaces []string `yaml:"namespaces,omitempty" json:"namespaces,omitempty"`
}

// allows reports whether this permission set grants the given operation
// in the given namespace. The caller is responsible for checking that the
// token name is in the map before calling allows.
func (p IDPTokenPermissions) allows(op, namespace string) bool {
	// Namespace check — only when namespaces are declared.
	if len(p.Namespaces) > 0 && !slices.Contains(p.Namespaces, namespace) {
		return false
	}
	// Operation check.
	for _, perm := range p.Permissions {
		if perm == IDPOpAll || perm == op {
			return true
		}
	}
	return false
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

// Empty returns true if the permissions list has no entries.
func (p IDPTokenPermissions) Empty() bool {
	return len(p.Permissions) == 0
}

// HasWildcard returns true if the permissions include "*" (all operations).
func (p IDPTokenPermissions) HasWildcard() bool {
	return slices.Contains(p.Permissions, IDPOpAll)
}

// HasOperation returns true if the permission includes the given operation.
func (p IDPTokenPermissions) HasOperation(op string) bool {
	if p.HasWildcard() {
		return true
	}
	return slices.Contains(p.Permissions, op)
}
