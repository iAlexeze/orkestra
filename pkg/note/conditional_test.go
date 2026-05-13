// pkg/note/conditional_test.go
package note

import (
	"reflect"
	"testing"
)

func TestNoteTernary(t *testing.T) {
	tests := []struct {
		name     string
		cond     interface{}
		trueVal  interface{}
		falseVal interface{}
		want     interface{}
	}{
		{"truthy string", "hello", "yes", "no", "yes"},
		{"empty string", "", "yes", "no", "no"},
		{"truthy int", 1, "yes", "no", "yes"},
		{"zero int", 0, "yes", "no", "no"},
		{"truthy bool true", true, "yes", "no", "yes"},
		{"truthy bool false", false, "yes", "no", "no"},
		{"non-empty slice", []interface{}{1}, "yes", "no", "yes"},
		{"empty slice", []interface{}{}, "yes", "no", "no"},
		{"nil", nil, "yes", "no", "no"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noteTernary(tt.cond, tt.trueVal, tt.falseVal)
			if got != tt.want {
				t.Errorf("noteTernary(%v, %v, %v) = %v, want %v", tt.cond, tt.trueVal, tt.falseVal, got, tt.want)
			}
		})
	}
}

func TestNoteCoalesce(t *testing.T) {
	tests := []struct {
		name string
		vals []interface{}
		want interface{}
	}{
		{"first non-empty", []interface{}{"", nil, "value", "other"}, "value"},
		{"all empty", []interface{}{"", nil, 0, false}, nil},
		{"single value", []interface{}{"hello"}, "hello"},
		{"empty list", []interface{}{}, nil},
		{"numbers", []interface{}{0, 0, 5, 10}, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noteCoalesce(tt.vals...)
			if got != tt.want {
				t.Errorf("noteCoalesce(%v) = %v, want %v", tt.vals, got, tt.want)
			}
		})
	}
}

func TestNoteDefault(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		def  interface{}
		want interface{}
	}{
		{"non-empty string", "foo", "default", "foo"},
		{"empty string", "", "default", "default"},
		{"nil", nil, "default", "default"},
		{"zero int", 0, 100, 100},
		{"non-zero int", 42, 100, 42},
		{"false bool", false, true, true},
		{"true bool", true, false, true},
		{"empty slice", []interface{}{}, "default", "default"},
		{"non-empty slice", []interface{}{1}, "default", []interface{}{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noteDefault(tt.val, tt.def)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("noteDefault(%v, %v) = %v, want %v", tt.val, tt.def, got, tt.want)
			}
		})
	}
}

func TestNoteEmpty(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"<no value>", "<no value>", true},
		{"non-empty string", "hello", false},
		{"zero int", 0, true},
		{"non-zero int", 5, false},
		{"zero float", 0.0, true},
		{"non-zero float", 3.14, false},
		{"false bool", false, true},
		{"true bool", true, false},
		{"empty slice", []interface{}{}, true},
		{"non-empty slice", []interface{}{1}, false},
		{"empty map", map[string]interface{}{}, true},
		{"non-empty map", map[string]interface{}{"a": 1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noteEmpty(tt.v); got != tt.want {
				t.Errorf("noteEmpty(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestNoteNotEmpty(t *testing.T) {
	// Just inverse of empty
	if noteNotEmpty("") != false {
		t.Error("noteNotEmpty('') should be false")
	}
	if noteNotEmpty("hello") != true {
		t.Error("noteNotEmpty('hello') should be true")
	}
}

func TestNoteIsTruthy(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want bool
	}{
		{"nil", nil, false},
		{"empty string", "", false},
		{"non-empty string", "test", true},
		{"zero int", 0, false},
		{"non-zero int", 1, true},
		{"zero float", 0.0, false},
		{"non-zero float", 1.5, true},
		{"false bool", false, false},
		{"true bool", true, true},
		{"empty slice", []interface{}{}, false},
		{"non-empty slice", []interface{}{1}, true},
		{"empty map", map[string]interface{}{}, false},
		{"non-empty map", map[string]interface{}{"a": 1}, true},
		{"other type", struct{}{}, true}, // non-nil struct -> true
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noteIsTruthy(tt.v); got != tt.want {
				t.Errorf("noteIsTruthy(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestNoteBoolTernary(t *testing.T) {
	if got := noteBoolTernary(true, "yes", "no"); got != "yes" {
		t.Errorf("noteBoolTernary(true) = %v, want yes", got)
	}
	if got := noteBoolTernary(false, "yes", "no"); got != "no" {
		t.Errorf("noteBoolTernary(false) = %v, want no", got)
	}
}

func TestNoteBoolDefault(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		def  bool
		want bool
	}{
		{"bool true", true, false, true},
		{"bool false", false, true, false},
		{"non-bool string", "true", false, false},
		{"non-bool int", 1, true, true},
		{"nil", nil, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noteBoolDefault(tt.v, tt.def); got != tt.want {
				t.Errorf("noteBoolDefault(%v, %v) = %v, want %v", tt.v, tt.def, got, tt.want)
			}
		})
	}
}
