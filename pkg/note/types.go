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
