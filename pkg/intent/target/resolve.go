package target

import "github.com/orkspace/orkestra/pkg/labels"

// ResolveTargetFromAnnotations extracts the effective target from a CR's annotations.
// Resolution order:
//  1. serve-alias annotation (most specific)
//  2. serve-target annotation (primary target)
//  3. Empty string (no target found)
func ResolveTargetFromAnnotations(annotations map[string]string) string {
	if annotations == nil {
		return ""
	}

	// 1. Check alias first (most specific)
	if alias, ok := annotations[labels.AnnotationServeAlias]; ok && alias != "" {
		return alias
	}

	// 2. Fall back to target
	if target, ok := annotations[labels.AnnotationServeTarget]; ok && target != "" {
		return target
	}

	return ""
}

