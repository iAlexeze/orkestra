package note

import "text/template"

// jobNotes registers helpers for inspecting Job status fields.
//
// Usage:
//
//	tmpl.Funcs(note.jobNotes())
//
// Template examples:
//
//	{{ jobSucceeded .children.job }}
//	{{ jobFailed .children.job }}
//	{{ jobActive .children.job }}
//
// These helpers allow gating dependent resources on Job lifecycle
// conditions—useful for cleanup tasks, chained workflows, or ensuring
// that follow‑up resources only apply after a Job has completed.
func jobNotes() template.FuncMap {
	return template.FuncMap{
		"jobSucceeded": noteJobSucceeded,
		"jobFailed":    noteJobFailed,
		"jobActive":    noteJobActive,
	}
}

// ── Job / batch notes ─────────────────────────────────────────────────────────

// noteJobSucceeded returns true when at least one pod has succeeded.
//
//	{{ jobSucceeded .children.job }}
func noteJobSucceeded(obj interface{}) bool {
	status := noteStatus(obj)
	return toInt64(status["succeeded"]) > 0
}

// noteJobFailed returns true when at least one pod has failed.
// Use to gate dependent resources on clean job completion:
//
//	when:
//	  - field: "{{ jobFailed .children.job }}"
//	    equals: "false"
//	  - field: "{{ jobSucceeded .children.job }}"
//	    equals: "true"
func noteJobFailed(obj interface{}) bool {
	status := noteStatus(obj)
	return toInt64(status["failed"]) > 0
}

// noteJobActive returns true when at least one pod is currently running.
//
//	{{ jobActive .children.job }}
func noteJobActive(obj interface{}) bool {
	status := noteStatus(obj)
	return toInt64(status["active"]) > 0
}
