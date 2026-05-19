package note

import (
	"strings"
	"text/template"
)

// podNotes registers helpers for inspecting enriched pod lists embedded by
// children.go under the reserved "_pods" key.
//
// Usage:
//
//	tmpl.Funcs(note.podNotes())
//
// Template examples:
//
//	"{{ podNames .children.deployment }}"        → "web-abc, web-def"
//	"{{ podIPs .children.deployment }}"          → "10.0.0.1, 10.0.0.2"
//	"{{ podCount .children.deployment }}"        → 2
//	"{{ readyPodCount .children.deployment }}"   → 2
//	"{{ hasCrashingPod .children.deployment }}"  → false
//
// These notes require pod enrichment to be enabled for the CRD:
//
//	enrich: [pods]   # or enrichAll: true
//
// Without enrichment the "_pods" key is absent and all notes return their
// zero values ("", 0, false) — the same safe fallback as any missing field.
func podNotes() template.FuncMap {
	return template.FuncMap{
		"podNames":       notePodNames,
		"podIPs":         notePodIPs,
		"podCount":       notePodCount,
		"readyPodCount":  noteReadyPodCount,
		"hasCrashingPod": noteHasCrashingPod,
	}
}

// notePodNames returns a comma-separated list of pod names owned by the
// enriched resource. Returns "" when no pods are present.
//
//	{{ podNames .children.deployment }}  → "web-abc, web-def"
func notePodNames(obj interface{}) string {
	return joinPodField(obj, "name")
}

// notePodIPs returns a comma-separated list of pod IP addresses.
// Returns "" when no pods are present or IPs have not yet been assigned.
//
//	{{ podIPs .children.deployment }}  → "10.0.0.1, 10.0.0.2"
func notePodIPs(obj interface{}) string {
	return joinPodField(obj, "ip")
}

// notePodCount returns the total number of pods owned by the enriched resource.
//
//	{{ podCount .children.deployment }}  → 3
func notePodCount(obj interface{}) int {
	return len(getPods(obj))
}

// noteReadyPodCount returns the number of pods whose Ready condition is True.
//
//	{{ readyPodCount .children.deployment }}  → 2
func noteReadyPodCount(obj interface{}) int {
	pods := getPods(obj)
	count := 0
	for _, p := range pods {
		pod, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		if ready, _ := pod["ready"].(bool); ready {
			count++
		}
	}
	return count
}

// noteHasCrashingPod returns true when any pod has restarted more than twice.
// A restart count above 2 is a strong signal of a crash loop.
//
//	{{ hasCrashingPod .children.deployment }}  → false
func noteHasCrashingPod(obj interface{}) bool {
	for _, p := range getPods(obj) {
		pod, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		switch v := pod["restartCount"].(type) {
		case int64:
			if v > 2 {
				return true
			}
		case float64:
			if v > 2 {
				return true
			}
		case int:
			if v > 2 {
				return true
			}
		}
	}
	return false
}

// ── internal helpers ──────────────────────────────────────────────────────

// getPods navigates the "_pods" key of an enriched resource map.
// Returns nil when the key is absent or enrichment was not enabled.
func getPods(obj interface{}) []interface{} {
	m, ok := obj.(map[string]interface{})
	if !ok {
		return nil
	}
	pods, _ := m["_pods"].([]interface{})
	return pods
}

// joinPodField collects field from every pod in _pods and joins with ", ".
func joinPodField(obj interface{}, field string) string {
	pods := getPods(obj)
	if len(pods) == 0 {
		return ""
	}
	parts := make([]string, 0, len(pods))
	for _, p := range pods {
		pod, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		if v, _ := pod[field].(string); v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, ", ")
}
