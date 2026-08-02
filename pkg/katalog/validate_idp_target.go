package katalog

import "fmt"

// validateIDPTarget is called from the main Validate() chain after the
// existing IDP checks. It catches:
//
//  1. Invalid target format — targets must be lowercase alphanumeric + hyphens.
//     The gateway uses the target as a URL path segment; anything else is unsafe.
//
//  2. Duplicate targets — two CRDs that resolve to the same target string would
//     make lookups ambiguous. Most commonly caused by two CRDs with the same
//     lowercased kind when idp.target is not set explicitly on one of them.
//
//     if err := k.validateIDPTarget(); err != nil { return err }
func (k *Katalog) validateIDPTarget() error {
	// Track seen targets → CRD name for duplicate detection.
	seen := make(map[string]string)

	for crdName, crd := range k.Spec.CRDs {
		if !crd.HasIDPTarget() {
			continue
		}

		target := crd.IDPTarget()

		// 1. Format — lowercase alphanumeric + hyphens only.
		if !isValidIDPTarget(target) {
			return fmt.Errorf(
				"crd %q: target %q is invalid — must be lowercase alphanumeric with "+
					"optional hyphens (a-z, 0-9, -)\n"+
					"  Set idp.target explicitly to a valid value.",
				crdName, target,
			)
		}

		// 2. Uniqueness — two CRDs cannot share a target.
		if first, clash := seen[target]; clash {
			return fmt.Errorf(
				"crd %q and %q both resolve to target %q\n"+
					"  Set idp.target explicitly on one or both to make them unique.",
				first, crdName, target,
			)
		}
		seen[target] = crdName
	}

	return nil
}

// isValidIDPTarget reports whether s is a valid target string.
// Valid: lowercase letters, digits, hyphens. Must be non-empty.
func isValidIDPTarget(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}
