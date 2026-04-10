package note

import (
	"text/template"
)

// Kubernetes-aware helpers for templates.
// These functions safely navigate Kubernetes-style unstructured objects
// (map[string]interface{}) without panics, and provide convenient access
// to metadata, spec, status, ownerReferences, and conditions.
//
//
// Usage examples:
//   {{ meta .children.cronjob | json }}
//   {{ labels .children.cronjob }}
//   {{ annotations .children.cronjob }}
//   {{ spec .children.cronjob }}
//   {{ status .children.cronjob }}
//   {{ get .children.cronjob "status" "lastScheduleTime" }}
//   {{ ownerKind .children.cronjob }}
//   {{ ownerName .children.cronjob }}
//   {{ hasCondition .children.deployment "Available" }}
//   {{ boolTernary (boolDefault (get . "spec" "suspend") false) "Suspended" "Active" }}

func kubernetesNotes() template.FuncMap {
	return template.FuncMap{
		"meta":         noteMeta,
		"labels":       noteLabels,
		"annotations":  noteAnnotations,
		"spec":         noteSpec,
		"status":       noteStatus,
		"get":          noteGet,
		"ownerKind":    noteOwnerKind,
		"ownerName":    noteOwnerName,
		"hasCondition": noteHasCondition,
	}
}

//
// ────────────────────────────────────────────────────────────────
//   1. METADATA HELPERS
// ────────────────────────────────────────────────────────────────
//

// noteMeta returns the metadata map of a Kubernetes object.
// Safe for nil or missing metadata.
//
//	{{ meta .children.cronjob | json }}
func noteMeta(obj interface{}) map[string]interface{} {
	if m, ok := obj.(map[string]interface{}); ok {
		if meta, ok := m["metadata"].(map[string]interface{}); ok {
			return meta
		}
	}
	return map[string]interface{}{}
}

// noteLabels returns the labels map from metadata.
// Returns an empty map if labels are missing.
//
//	{{ labels .children.cronjob }}
func noteLabels(obj interface{}) map[string]interface{} {
	meta := noteMeta(obj)
	if lbls, ok := meta["labels"].(map[string]interface{}); ok {
		return lbls
	}
	return map[string]interface{}{}
}

// noteAnnotations returns the annotations map from metadata.
// Returns an empty map if annotations are missing.
//
//	{{ annotations .children.cronjob }}
func noteAnnotations(obj interface{}) map[string]interface{} {
	meta := noteMeta(obj)
	if ann, ok := meta["annotations"].(map[string]interface{}); ok {
		return ann
	}
	return map[string]interface{}{}
}

//
// ────────────────────────────────────────────────────────────────
//   2. SPEC / STATUS HELPERS
// ────────────────────────────────────────────────────────────────
//

// noteSpec returns the spec map of a Kubernetes object.
// Safe for nil or missing spec.
//
//	{{ spec .children.cronjob }}
func noteSpec(obj interface{}) map[string]interface{} {
	if m, ok := obj.(map[string]interface{}); ok {
		if spec, ok := m["spec"].(map[string]interface{}); ok {
			return spec
		}
	}
	return map[string]interface{}{}
}

// noteStatus returns the status map of a Kubernetes object.
// Safe for nil or missing status.
//
//	{{ status .children.cronjob }}
func noteStatus(obj interface{}) map[string]interface{} {
	if m, ok := obj.(map[string]interface{}); ok {
		if status, ok := m["status"].(map[string]interface{}); ok {
			return status
		}
	}
	return map[string]interface{}{}
}

//
// ────────────────────────────────────────────────────────────────
//   3. SAFE NESTED FIELD LOOKUP
// ────────────────────────────────────────────────────────────────
//

// noteGet retrieves a nested field from a Kubernetes object using a path.
// Returns nil if any segment is missing.
//
//	{{ get .children.cronjob "status" "lastScheduleTime" }}
func noteGet(obj interface{}, path ...string) interface{} {
	current := obj
	for _, p := range path {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[p]
		if current == nil {
			return nil
		}
	}
	return current
}

//
// ────────────────────────────────────────────────────────────────
//   4. OWNERREFERENCE HELPERS
// ────────────────────────────────────────────────────────────────
//

// noteOwnerKind returns the kind of the first ownerReference.
// Useful for debugging controller relationships.
//
//	{{ ownerKind .children.cronjob }}
func noteOwnerKind(obj interface{}) string {
	meta := noteMeta(obj)
	owners, ok := meta["ownerReferences"].([]interface{})
	if !ok || len(owners) == 0 {
		return ""
	}
	if first, ok := owners[0].(map[string]interface{}); ok {
		if kind, ok := first["kind"].(string); ok {
			return kind
		}
	}
	return ""
}

// noteOwnerName returns the name of the first ownerReference.
//
//	{{ ownerName .children.cronjob }}
func noteOwnerName(obj interface{}) string {
	meta := noteMeta(obj)
	owners, ok := meta["ownerReferences"].([]interface{})
	if !ok || len(owners) == 0 {
		return ""
	}
	if first, ok := owners[0].(map[string]interface{}); ok {
		if name, ok := first["name"].(string); ok {
			return name
		}
	}
	return ""
}

//
// ────────────────────────────────────────────────────────────────
//   5. CONDITION HELPERS
// ────────────────────────────────────────────────────────────────
//

// noteHasCondition checks if a Kubernetes status.conditions array
// contains a condition of a given type with status "True".
//
//	{{ hasCondition .children.deployment "Available" }}
func noteHasCondition(obj interface{}, condType string) bool {
	status := noteStatus(obj)
	conds, ok := status["conditions"].([]interface{})
	if !ok {
		return false
	}
	for _, c := range conds {
		if cm, ok := c.(map[string]interface{}); ok {
			if t, ok := cm["type"].(string); ok && t == condType {
				if s, ok := cm["status"].(string); ok && s == "True" {
					return true
				}
			}
		}
	}
	return false
}
