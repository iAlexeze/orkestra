// pkg/types/ns_allowed.go
package types

// AllowedNamespaces declares namespaces where Orkestra *is permitted*
// to create child resources. If the list is empty, all namespaces are allowed.
//
// Applies before onCreate/onReconcile templates and before Go hooks run.
// The restriction is on child resources only — not on the CR itself.
//
// Supported at all three levels:
//   Komposer level — applies to all CRDs in the Komposer
//   Katalog level  — applies to all CRDs in the Katalog
//   CRD level      — applies to this specific CRD
//
// Rules from all three levels are merged — more specific levels add to,
// not replace, less specific levels. A namespace allowed at the Komposer
// level cannot be "disallowed" at the CRD level. This is intentional:
// platform-wide allowances are non-negotiable.
//
// Example:
//   allowedNamespaces:
//     - apps
//     - workloads
//     - team-*        # wildcard — any namespace starting with team-
//     - "*-sandbox"   # wildcard — any namespace ending with -sandbox

// AllowedNamespaces holds the set of allowed namespace patterns.
type AllowedNamespaces []string

// IsAllowed reports whether a given namespace matches any allowed pattern.
// If the list is empty, all namespaces are allowed.
// Supports exact matches and simple glob patterns (* prefix or suffix).
func (a AllowedNamespaces) IsAllowed(namespace string) bool {
	// Empty list → allow everything
	if len(a) == 0 {
		return true
	}

	for _, pattern := range a {
		if matchesPattern(namespace, pattern) {
			return true
		}
	}
	return false
}

// Merge combines two AllowedNamespaces sets, deduplicating entries.
// Since allowances are additive, a namespace allowed at any level
// is allowed at all levels — more specific levels cannot remove allowances.
func (a AllowedNamespaces) Merge(other AllowedNamespaces) AllowedNamespaces {
	seen := make(map[string]bool, len(a))
	merged := make(AllowedNamespaces, 0, len(a)+len(other))

	for _, ns := range a {
		if !seen[ns] {
			seen[ns] = true
			merged = append(merged, ns)
		}
	}
	for _, ns := range other {
		if !seen[ns] {
			seen[ns] = true
			merged = append(merged, ns)
		}
	}

	return merged
}
