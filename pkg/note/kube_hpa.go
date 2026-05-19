package note

import "text/template"

// hpaNotes registers helpers for inspecting HorizontalPodAutoscaler status fields.
//
// Usage:
//
//	tmpl.Funcs(note.hpaNotes())
//
// Template examples:
//
//	{{ hpaCurrentReplicas .children.hpa }}
//	{{ hpaDesiredReplicas .children.hpa }}
//	{{ hpaMinReplicas .children.hpa }}
//	{{ hpaMaxReplicas .children.hpa }}
//	{{ hpaScaling .children.hpa }}
//	{{ hpaAtMax .children.hpa }}
//
// No enrichment required — all notes navigate the HPA object directly.
func hpaNotes() template.FuncMap {
	return template.FuncMap{
		"hpaCurrentReplicas": noteHPACurrentReplicas,
		"hpaDesiredReplicas": noteHPADesiredReplicas,
		"hpaMinReplicas":     noteHPAMinReplicas,
		"hpaMaxReplicas":     noteHPAMaxReplicas,
		"hpaScaling":         noteHPAScaling,
		"hpaAtMax":           noteHPAAtMax,
	}
}

// ── HPA notes ─────────────────────────────────────────────────────────────────

// noteHPACurrentReplicas returns status.currentReplicas.
//
//	{{ hpaCurrentReplicas .children.hpa }}  → 3
func noteHPACurrentReplicas(obj interface{}) int64 {
	status := noteStatus(obj)
	return toInt64(status["currentReplicas"])
}

// noteHPADesiredReplicas returns status.desiredReplicas.
//
//	{{ hpaDesiredReplicas .children.hpa }}  → 5
func noteHPADesiredReplicas(obj interface{}) int64 {
	status := noteStatus(obj)
	return toInt64(status["desiredReplicas"])
}

// noteHPAMinReplicas returns spec.minReplicas.
// Returns 1 when not set (Kubernetes default).
//
//	{{ hpaMinReplicas .children.hpa }}  → 2
func noteHPAMinReplicas(obj interface{}) int64 {
	m, ok := obj.(map[string]interface{})
	if !ok {
		return 1
	}
	spec, _ := m["spec"].(map[string]interface{})
	if spec == nil {
		return 1
	}
	v := toInt64(spec["minReplicas"])
	if v == 0 {
		return 1
	}
	return v
}

// noteHPAMaxReplicas returns spec.maxReplicas.
//
//	{{ hpaMaxReplicas .children.hpa }}  → 10
func noteHPAMaxReplicas(obj interface{}) int64 {
	m, ok := obj.(map[string]interface{})
	if !ok {
		return 0
	}
	spec, _ := m["spec"].(map[string]interface{})
	if spec == nil {
		return 0
	}
	return toInt64(spec["maxReplicas"])
}

// noteHPAScaling returns true when the HPA is actively scaling
// (currentReplicas != desiredReplicas).
//
//	{{ hpaScaling .children.hpa }}
func noteHPAScaling(obj interface{}) bool {
	return noteHPACurrentReplicas(obj) != noteHPADesiredReplicas(obj)
}

// noteHPAAtMax returns true when currentReplicas has reached maxReplicas.
// Useful for surfacing capacity pressure in status.
//
//	{{ hpaAtMax .children.hpa }}
func noteHPAAtMax(obj interface{}) bool {
	max := noteHPAMaxReplicas(obj)
	if max == 0 {
		return false
	}
	return noteHPACurrentReplicas(obj) >= max
}
