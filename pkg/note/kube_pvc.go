package note

import (
	"strings"
	"text/template"
)

// pvcNotes registers helpers for inspecting PersistentVolumeClaim fields and
// the bound PV embedded by the enrichment layer.
//
// Usage:
//
//	tmpl.Funcs(note.pvcNotes())
//
// Template examples:
//
//	{{ pvcBound .children.pvc }}
//	{{ pvcPhase .children.pvc }}
//	{{ pvcCapacity .children.pvc }}
//	{{ pvcStorageClass .children.pvc }}
//	{{ pvcAccessModes .children.pvc }}
//	{{ pvcProvisioner .children.pvc }}
//	{{ pvcVolumeMode .children.pvc }}
//
// pvcBound, pvcPhase, pvcCapacity, pvcStorageClass, pvcAccessModes require no
// enrichment — they navigate the PVC object directly.
//
// pvcProvisioner and pvcVolumeMode read the bound PV embedded under "_pv" and
// require enrich: [pvc] (or enrichAll: true) on the CRD.
func pvcNotes() template.FuncMap {
	return template.FuncMap{
		"pvcBound":        notePVCBound,
		"pvcPhase":        notePVCPhase,
		"pvcCapacity":     notePVCCapacity,
		"pvcStorageClass": notePVCStorageClass,
		"pvcAccessModes":  notePVCAccessModes,
		// Enriched PV notes — require enrich: [pvc] on the CRD.
		"pvcProvisioner": notePVCProvisioner,
		"pvcVolumeMode":  notePVCVolumeMode,
	}
}

// ── PVC notes (no enrichment required) ───────────────────────────────────────

// notePVCBound returns true when the PVC status.phase is "Bound".
//
//	{{ pvcBound .children.pvc }}
func notePVCBound(obj interface{}) bool {
	return notePVCPhase(obj) == "Bound"
}

// notePVCPhase returns the PVC status.phase ("Bound", "Pending", "Released", "Lost").
// Returns "" before the phase is set.
//
//	{{ pvcPhase .children.pvc }}  → "Bound"
func notePVCPhase(obj interface{}) string {
	status := noteStatus(obj)
	v, _ := status["phase"].(string)
	return v
}

// notePVCCapacity returns the requested storage capacity from spec.resources.requests.storage.
//
//	{{ pvcCapacity .children.pvc }}  → "10Gi"
func notePVCCapacity(obj interface{}) string {
	m, ok := obj.(map[string]interface{})
	if !ok {
		return ""
	}
	spec, _ := m["spec"].(map[string]interface{})
	if spec == nil {
		return ""
	}
	resources, _ := spec["resources"].(map[string]interface{})
	if resources == nil {
		return ""
	}
	requests, _ := resources["requests"].(map[string]interface{})
	if requests == nil {
		return ""
	}
	v, _ := requests["storage"].(string)
	return v
}

// notePVCStorageClass returns spec.storageClassName.
// Returns "" when not set (cluster default will be used).
//
//	{{ pvcStorageClass .children.pvc }}  → "standard"
func notePVCStorageClass(obj interface{}) string {
	m, ok := obj.(map[string]interface{})
	if !ok {
		return ""
	}
	spec, _ := m["spec"].(map[string]interface{})
	if spec == nil {
		return ""
	}
	v, _ := spec["storageClassName"].(string)
	return v
}

// notePVCAccessModes returns a comma-separated list of spec.accessModes.
//
//	{{ pvcAccessModes .children.pvc }}  → "ReadWriteOnce"
func notePVCAccessModes(obj interface{}) string {
	m, ok := obj.(map[string]interface{})
	if !ok {
		return ""
	}
	spec, _ := m["spec"].(map[string]interface{})
	if spec == nil {
		return ""
	}
	modes, _ := spec["accessModes"].([]interface{})
	parts := make([]string, 0, len(modes))
	for _, mode := range modes {
		if s, _ := mode.(string); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ", ")
}

// ── Enriched PV notes (require enrich: [pvc]) ─────────────────────────────────

// notePVCProvisioner returns the provisioner that created the bound PV.
// Read from the PV's metadata.annotations["pv.kubernetes.io/provisioned-by"].
// Returns "" when the PVC is unbound or enrichment is not enabled.
// Requires enrich: [pvc] on the CRD.
//
//	{{ pvcProvisioner .children.pvc }}  → "ebs.csi.aws.com"
func notePVCProvisioner(obj interface{}) string {
	pv := getBoundPV(obj)
	if pv == nil {
		return ""
	}
	meta, _ := pv["metadata"].(map[string]interface{})
	if meta == nil {
		return ""
	}
	annotations, _ := meta["annotations"].(map[string]interface{})
	if annotations == nil {
		return ""
	}
	v, _ := annotations["pv.kubernetes.io/provisioned-by"].(string)
	return v
}

// notePVCVolumeMode returns the volumeMode of the bound PV ("Filesystem" or "Block").
// Returns "" when the PVC is unbound or enrichment is not enabled.
// Requires enrich: [pvc] on the CRD.
//
//	{{ pvcVolumeMode .children.pvc }}  → "Filesystem"
func notePVCVolumeMode(obj interface{}) string {
	pv := getBoundPV(obj)
	if pv == nil {
		return ""
	}
	spec, _ := pv["spec"].(map[string]interface{})
	if spec == nil {
		return ""
	}
	v, _ := spec["volumeMode"].(string)
	return v
}

func getBoundPV(obj interface{}) map[string]interface{} {
	m, ok := obj.(map[string]interface{})
	if !ok {
		return nil
	}
	pv, _ := m["_pv"].(map[string]interface{})
	return pv
}
