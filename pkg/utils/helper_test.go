package utils

import "testing"

func TestNestedMap(t *testing.T) {
	obj := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": map[string]interface{}{"found": true},
			},
		},
	}
	got, ok := NestedMap(obj, "a", "b", "c")
	if !ok {
		t.Fatal("nestedMap: not found")
	}
	if got["found"] != true {
		t.Errorf("nestedMap result = %v", got)
	}
	_, ok = NestedMap(obj, "a", "missing")
	if ok {
		t.Error("nestedMap should return false for missing key")
	}
}
