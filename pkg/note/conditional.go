package note

import (
	"fmt"
	"text/template"
)

func conditionalNotes() template.FuncMap {
	return template.FuncMap{
		"ternary":  noteTernary,
		"coalesce": noteCoalesce,
		"default":  noteDefault,
		"empty":    noteEmpty,
		"notEmpty": noteNotEmpty,
	}
}

// noteTernary returns trueVal when condition is truthy, falseVal otherwise.
// Replaces verbose {{ if }}...{{ else }}...{{ end }} in template expressions.
//
//	{{ ternary .spec.debug "debug" "info" }}
//	{{ ternary .spec.monitoring "enabled" "disabled" }}
func noteTernary(condition interface{}, trueVal, falseVal interface{}) interface{} {
	if noteIsTruthy(condition) {
		return trueVal
	}
	return falseVal
}

// noteCoalesce returns the first non-empty value from a variadic list.
//
//	{{ coalesce .spec.image .spec.defaultImage "nginx:latest" }}
func noteCoalesce(vals ...interface{}) interface{} {
	for _, v := range vals {
		if !noteEmpty(v) {
			return v
		}
	}
	return nil
}

// noteDefault returns val if non-empty, otherwise returns def.
//
//	{{ default .spec.replicas 2 }}     →  2 if spec.replicas is absent or zero
//	{{ default .spec.port "8080" }}    →  "8080" if spec.port is absent
func noteDefault(val, def interface{}) interface{} {
	if noteEmpty(val) {
		return def
	}
	return val
}

// noteEmpty reports whether a value is empty.
// Empty: nil, "", 0, false, empty slice, empty map, "<no value>".
//
//	{{ empty .spec.image }}   →  true when absent
func noteEmpty(v interface{}) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == "" || val == "<no value>"
	case bool:
		return !val
	case int:
		return val == 0
	case int32:
		return val == 0
	case int64:
		return val == 0
	case float32:
		return val == 0
	case float64:
		return val == 0
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	}
	return fmt.Sprintf("%v", v) == ""
}

// noteNotEmpty is the inverse of empty.
//
//	{{ notEmpty .spec.image }}   →  true when spec.image is set
func noteNotEmpty(v interface{}) bool {
	return !noteEmpty(v)
}

func noteIsTruthy(v interface{}) bool {
	return !noteEmpty(v)
}
