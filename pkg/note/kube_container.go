package note

import (
	"fmt"
	"strings"
	"text/template"
)

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
		"containerImage":      noteContainerImage,
		"containerEnv":        noteContainerEnv,
		"containerPort":       noteContainerPort,
		"containerPortByName": noteContainerPortByName,
		"containerPorts":      noteContainerPorts,
		"containerCount":      noteContainerCount,
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

// ── Container notes ───────────────────────────────────────

// noteContainerPort returns the container port indexed.
//
//	{{ containerPort .children.deployment 0 8080 }}
func noteContainerPort(obj interface{}, index int, port int) bool {
	containers := getContainers(obj)
	if index < 0 || index >= len(containers) {
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
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		if int(toInt64(pm["containerPort"])) == port {
			return true
		}
	}
	return false
}

// noteContainerPortByName returns the port number for a named container port.
// Returns 0 when not found.
//
//	{{ containerPortByName .children.deployment 0 "http" }}  → 8080
func noteContainerPortByName(obj interface{}, index int, name string) int {
	containers := getContainers(obj)
	if index < 0 || index >= len(containers) {
		return 0
	}
	cm, ok := containers[index].(map[string]interface{})
	if !ok {
		return 0
	}
	ports, ok := cm["ports"].([]interface{})
	if !ok {
		return 0
	}
	for _, p := range ports {
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		if n, _ := pm["name"].(string); n == name {
			return int(toInt64(pm["containerPort"]))
		}
	}
	return 0
}

// noteContainerPorts returns a comma-separated list of exposed port numbers.
//
//	{{ containerPorts .children.deployment 0 }}  → "8080,9090"
func noteContainerPorts(obj interface{}, index int) string {
	containers := getContainers(obj)
	if index < 0 || index >= len(containers) {
		return ""
	}
	cm, ok := containers[index].(map[string]interface{})
	if !ok {
		return ""
	}
	ports, ok := cm["ports"].([]interface{})
	if !ok {
		return ""
	}
	var result []string
	for _, p := range ports {
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		result = append(result, fmt.Sprintf("%d", int(toInt64(pm["containerPort"]))))
	}
	return strings.Join(result, ",")
}

// noteContainerCount returns the number of containers in the pod spec.
//
//	{{ containerCount .children.deployment }}  → 2
func noteContainerCount(obj interface{}) int {
	return len(getContainers(obj))
}

func getContainers(obj interface{}) []interface{} {
	spec := noteGet(obj, "spec", "template", "spec")
	m, ok := spec.(map[string]interface{})
	if !ok {
		return nil
	}
	containers, _ := m["containers"].([]interface{})
	return containers
}
