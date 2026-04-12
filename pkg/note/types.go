package note

import (
	"fmt"
	"text/template"
)

func typeNotes() template.FuncMap {
	return template.FuncMap{
		"toInt":    noteToInt,
		"toFloat":  noteToFloat,
		"toBool":   noteToBool,
		"toString": noteToString,

		// typeOf v — returns the type name as a string
		// "string", "number", "bool", "map", "slice", "null", "unknown"
		"typeOf": TypeOf,

		// len v — returns element count or string length
		// Overrides the built-in template len with one that handles maps
		"len": OrkLen,
	}
}

// noteToInt converts any value to int64.
//
//	{{ toInt "3" }}    →  3
//	{{ toInt 3.7 }}    →  3   (truncates)
//	{{ toInt true }}   →  1
func noteToInt(v interface{}) (int64, error) {
	if b, ok := v.(bool); ok {
		if b {
			return 1, nil
		}
		return 0, nil
	}
	f, err := anyToFloat(v)
	if err != nil {
		return 0, fmt.Errorf("toInt: %w", err)
	}
	return int64(f), nil
}

// noteToFloat converts any value to float64.
func noteToFloat(v interface{}) (float64, error) {
	return anyToFloat(v)
}

// noteToBool converts a value to bool.
// Truthy: true, 1, "true", "yes", "on", "1", "True", "TRUE", "YES".
// Falsy: false, 0, "", "false", "no", "off", "0", "False", "FALSE".
//
//	{{ toBool "yes" }}   →  true
//	{{ toBool 1 }}       →  true
//	{{ toBool "" }}      →  false
func noteToBool(v interface{}) (bool, error) {
	switch val := v.(type) {
	case bool:
		return val, nil
	case int, int32, int64, float32, float64:
		f, _ := anyToFloat(v)
		return f != 0, nil
	case string:
		switch val {
		case "true", "True", "TRUE", "1", "yes", "YES", "Yes", "on", "ON":
			return true, nil
		case "false", "False", "FALSE", "0", "no", "NO", "No", "off", "OFF", "":
			return false, nil
		}
		return false, fmt.Errorf("toBool: unrecognised value %q", val)
	}
	return false, fmt.Errorf("toBool: cannot convert %T", v)
}

// noteToString converts any value to its string representation.
//
//	{{ toString 42 }}       →  "42"
//	{{ toString true }}     →  "true"
//	{{ toString 3.14 }}     →  "3.14"
func noteToString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// typeOf — returns the runtime type name of any value.
// orkLen — returns the length of a string, slice, or map.
//
// typeOf is used in:
//   - Template expressions: {{ typeOf .spec.schedule }} → "string" or "map"
//   - when: conditions:    operator: typeOf, value: map
//
// The condition operator path uses NavigateRawPath → note.TypeOf directly.
// The template expression path uses the FuncMap entry registered in note.Map().
// Both must return the same strings for the Katalog to be predictable.
//
// Type strings (match Python/JavaScript convention for familiarity):
//
//	"string"  → Go string
//	"number"  → Go float64, int64, int (JSON numbers come back as float64)
//	"bool"    → Go bool
//	"map"     → Go map[string]interface{} (YAML objects, structured fields)
//	"slice"   → Go []interface{} (YAML arrays, list fields)
//	"null"    → nil
//	"unknown" → any other type
//
// YAML type behaviour:
//
//	spec:
//	  schedule: "*/5 * * * *"          → typeOf returns "string"
//	  schedule:                          → typeOf returns "map"
//	    minute: "*/5"
//	  regions: [us-east-1, eu-west-1]   → typeOf returns "slice"
//	  replicas: 3                        → typeOf returns "number"
//	  enabled: true                      → typeOf returns "bool"
//
// TypeOf returns the type name of any interface{} value.
// Exported so pkg/types/when.go can call it from EvaluateOneCond
// without the template engine being involved.
func TypeOf(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case string:
		return "string"
	case float64, float32:
		return "number"
	case int, int32, int64, uint, uint32, uint64:
		return "number"
	case bool:
		return "bool"
	case map[string]interface{}:
		return "map"
	case []interface{}:
		return "slice"
	default:
		return "unknown"
	}
}

// OrkLen returns the length of a string, slice, or map.
// Named orkLen to avoid shadowing Go's built-in len.
// Registered in note.Map() as "len" since Go templates do not
// have a built-in len that handles all three types uniformly.
//
// Template: {{ len .spec.regions }}          → 3 (slice with 3 elements)
// Template: {{ len .spec.schedule }}         → 5 (map with 5 fields)
// Template: {{ len .metadata.name }}         → 12 (string length)
func OrkLen(v interface{}) int {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case string:
		return len(t)
	case []interface{}:
		return len(t)
	case map[string]interface{}:
		return len(t)
	default:
		return 0
	}
}
