package note

import "text/template"

// containerNotes registers helpers for extracting container information.
//
// Usage:
//
//	tmpl.Funcs(note.containerNotes())
//
// Template examples:
//
//	{{ containerImage .children.deployment 0 }}
//	{{ containerEnv .children.deployment 0 "APP_ENV" }}
//	{{ containerPort .children.deployment 0 8080 }}
func containerNotes() template.FuncMap {
	return template.FuncMap{
		"containerImage": noteContainerImage,
		"containerEnv":   noteContainerEnv,
		"containerPort":  noteContainerPort,
	}
}

// noteContainerImage returns the image of containers[index].
//
//	{{ containerImage .children.deployment 0 }}
func noteContainerImage(obj interface{}, index int) string {
	spec := noteGet(obj, "spec", "template", "spec")
	m, ok := spec.(map[string]interface{})
	if !ok {
		return ""
	}
	containers, ok := m["containers"].([]interface{})
	if !ok || index < 0 || index >= len(containers) {
		return ""
	}
	cm, ok := containers[index].(map[string]interface{})
	if !ok {
		return ""
	}
	img, _ := cm["image"].(string)
	return img
}

// noteContainerEnv returns the value of an env var.
//
//	{{ containerEnv .children.deployment 0 "APP_ENV" }}
func noteContainerEnv(obj interface{}, index int, key string) string {
	spec := noteGet(obj, "spec", "template", "spec")
	m, ok := spec.(map[string]interface{})
	if !ok {
		return ""
	}
	containers, ok := m["containers"].([]interface{})
	if !ok || index < 0 || index >= len(containers) {
		return ""
	}
	cm, ok := containers[index].(map[string]interface{})
	if !ok {
		return ""
	}
	envList, ok := cm["env"].([]interface{})
	if !ok {
		return ""
	}
	for _, e := range envList {
		if em, ok := e.(map[string]interface{}); ok {
			if name, ok := em["name"].(string); ok && name == key {
				if val, ok := em["value"].(string); ok {
					return val
				}
			}
		}
	}
	return ""
}

// noteContainerPort returns true if container exposes the given port.
//
//	{{ containerPort .children.deployment 0 8080 }}
func noteContainerPort(obj interface{}, index int, port int) bool {
	spec := noteGet(obj, "spec", "template", "spec")
	m, ok := spec.(map[string]interface{})
	if !ok {
		return false
	}
	containers, ok := m["containers"].([]interface{})
	if !ok || index < 0 || index >= len(containers) {
		return false
	}
	cm, ok := containers[index].(map[string]interface{})
	if !ok {
		return false
	}
	ports, ok := cm["ports"].([]interface{})
	if !ok {
		return false
	}
	for _, p := range ports {
		if pm, ok := p.(map[string]interface{}); ok {
			if val, ok := pm["containerPort"].(int); ok && val == port {
				return true
			}
		}
	}
	return false
}
