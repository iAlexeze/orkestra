package note

import "text/template"

// nodeNotes registers helpers for inspecting Node status fields.
// These notes navigate a Node object directly — useful for operators that
// manage or reference nodes.
//
// Usage:
//
//	tmpl.Funcs(note.nodeNotes())
//
// Template examples:
//
//	{{ nodeReady .children.node }}
//	{{ nodeAllocatableCPU .children.node }}
//	{{ nodeAllocatableMemory .children.node }}
//	{{ nodeCondition .children.node "MemoryPressure" }}
//	{{ nodeTaints .children.node }}
//
// No enrichment required — all notes navigate the Node object directly.
func nodeNotes() template.FuncMap {
	return template.FuncMap{
		"nodeReady":             noteNodeReady,
		"nodeAllocatableCPU":    noteNodeAllocatableCPU,
		"nodeAllocatableMemory": noteNodeAllocatableMemory,
		"nodeCondition":         noteNodeCondition,
		"nodeTaints":            noteNodeTaints,
	}
}

// ── Node notes ────────────────────────────────────────────────────────────────

// noteNodeReady returns true when the node's Ready condition status is "True".
//
//	{{ nodeReady .children.node }}
func noteNodeReady(obj interface{}) bool {
	return noteNodeCondition(obj, "Ready") == "True"
}

// noteNodeAllocatableCPU returns status.allocatable.cpu.
// Returns "" when not set.
//
//	{{ nodeAllocatableCPU .children.node }}  → "3920m"
func noteNodeAllocatableCPU(obj interface{}) string {
	return nodeAllocatable(obj, "cpu")
}

// noteNodeAllocatableMemory returns status.allocatable.memory.
// Returns "" when not set.
//
//	{{ nodeAllocatableMemory .children.node }}  → "15032020Ki"
func noteNodeAllocatableMemory(obj interface{}) string {
	return nodeAllocatable(obj, "memory")
}

// noteNodeCondition returns the status string of a named node condition.
// Returns "" when the condition is absent. Common condition types:
// "Ready", "MemoryPressure", "DiskPressure", "PIDPressure", "NetworkUnavailable".
//
//	{{ nodeCondition .children.node "Ready" }}           → "True"
//	{{ nodeCondition .children.node "MemoryPressure" }}  → "False"
func noteNodeCondition(obj interface{}, condType string) string {
	status := noteStatus(obj)
	conditions, _ := status["conditions"].([]interface{})
	for _, c := range conditions {
		cm, _ := c.(map[string]interface{})
		if cm == nil {
			continue
		}
		if t, _ := cm["type"].(string); t == condType {
			s, _ := cm["status"].(string)
			return s
		}
	}
	return ""
}

// noteNodeTaints returns a comma-separated list of taint keys on the node.
// Returns "" when no taints are present.
//
//	{{ nodeTaints .children.node }}  → "node.kubernetes.io/not-ready"
func noteNodeTaints(obj interface{}) string {
	m, ok := obj.(map[string]interface{})
	if !ok {
		return ""
	}
	spec, _ := m["spec"].(map[string]interface{})
	if spec == nil {
		return ""
	}
	taints, _ := spec["taints"].([]interface{})
	var keys []string
	for _, t := range taints {
		tm, _ := t.(map[string]interface{})
		if tm == nil {
			continue
		}
		if k, _ := tm["key"].(string); k != "" {
			keys = append(keys, k)
		}
	}
	result := ""
	for i, k := range keys {
		if i > 0 {
			result += ", "
		}
		result += k
	}
	return result
}

func nodeAllocatable(obj interface{}, resource string) string {
	status := noteStatus(obj)
	allocatable, _ := status["allocatable"].(map[string]interface{})
	if allocatable == nil {
		return ""
	}
	v, _ := allocatable[resource].(string)
	return v
}
