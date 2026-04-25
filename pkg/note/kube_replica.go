package note

import "text/template"

// replicaNotes registers helpers for inspecting ReplicaSet and Deployment-style
// replica status fields.
//
// Usage:
//
//	tmpl.Funcs(note.replicaNotes())
//
// Template examples:
//
//	{{ replicasReady .children.deployment }}
//	{{ desiredReplicas .children.deployment }}
//	{{ readyReplicas .children.deployment }}
//	{{ availableReplicas .children.deployment }}
//	{{ updatedReplicas .children.deployment }}
//
// These helpers provide concise gates for rollout readiness, desired vs. actual
// replica counts, and updated pod availability—useful for orchestrating async
// workflows and ensuring dependent resources apply only after a rollout
// stabilizes.
func replicaNotes() template.FuncMap {
	return template.FuncMap{
		"replicasReady":     noteReplicasReady,
		"desiredReplicas":   noteDesiredReplicas,
		"readyReplicas":     noteReadyReplicas,
		"availableReplicas": noteAvailableReplicas,
		"updatedReplicas":   noteUpdatedReplicas,
	}
}

// ── Replica notes ─────────────────────────────────────────────────────────────

// noteReplicasReady returns true when readyReplicas matches the desired replica count.
// The single most common async gate condition — replaces the verbose:
//
//	  children.deployment.status.readyReplicas == spec.replicas
//
//		when:
//		  - field: "{{ replicasReady .children.deployment }}"
//		    equals: "true"
func noteReplicasReady(obj interface{}) bool {
	desired := noteDesiredReplicas(obj)
	ready := noteReadyReplicas(obj)
	return desired > 0 && ready >= desired
}

// noteDesiredReplicas returns spec.replicas. Returns 1 when not set (Kubernetes default).
//
//	{{ desiredReplicas .children.deployment }}  → 3
func noteDesiredReplicas(obj interface{}) int {
	spec := noteGet(obj, "spec")
	m, ok := spec.(map[string]interface{})
	if !ok {
		return 1
	}
	v := toInt64(m["replicas"])
	if v == 0 {
		return 1 // Kubernetes default when not specified
	}
	return int(v)
}

// noteReadyReplicas returns status.readyReplicas safely. Returns 0 when not set.
//
//	{{ readyReplicas .children.deployment }}  → 2
func noteReadyReplicas(obj interface{}) int {
	status := noteStatus(obj)
	return int(toInt64(status["readyReplicas"]))
}

// noteAvailableReplicas returns status.availableReplicas safely.
//
//	{{ availableReplicas .children.deployment }}  → 2
func noteAvailableReplicas(obj interface{}) int {
	status := noteStatus(obj)
	return int(toInt64(status["availableReplicas"]))
}

// noteUpdatedReplicas returns status.updatedReplicas — pods running the current spec.
// When updatedReplicas == desiredReplicas, the rollout is complete.
//
//	{{ updatedReplicas .children.deployment }}  → 3
func noteUpdatedReplicas(obj interface{}) int {
	status := noteStatus(obj)
	return int(toInt64(status["updatedReplicas"]))
}
