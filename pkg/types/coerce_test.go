package types

import "testing"

func TestTryCoerceString(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want any
	}{
		{"integer", "42", float64(42)},
		{"negative integer", "-7", float64(-7)},
		{"float", "3.14", 3.14},
		{"true", "true", true},
		{"false", "false", false},
		{"plain string", "hello", "hello"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TryCoerceString(tt.in)
			if got != tt.want {
				t.Errorf("TryCoerceString(%q) = %#v (%T), want %#v (%T)", tt.in, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestTryCoerceString_JSONObject(t *testing.T) {
	got := TryCoerceString(`{"app": "payments-api"}`)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("got %#v (%T), want map[string]any", got, got)
	}
	if m["app"] != "payments-api" {
		t.Errorf(`m["app"] = %v, want "payments-api"`, m["app"])
	}
}

func TestTryCoerceString_JSONArray(t *testing.T) {
	got := TryCoerceString(`["a", "b", "c"]`)
	arr, ok := got.([]any)
	if !ok {
		t.Fatalf("got %#v (%T), want []any", got, got)
	}
	if len(arr) != 3 || arr[0] != "a" {
		t.Errorf("got %#v, want [a b c]", arr)
	}
}

func TestTryCoerceString_JSONWithWhitespace(t *testing.T) {
	got := TryCoerceString("  \n" + `{"app": "payments-api"}` + "\n  ")
	if _, ok := got.(map[string]any); !ok {
		t.Errorf("got %#v (%T), want map[string]any — leading/trailing whitespace should not block JSON detection", got, got)
	}
}

func TestTryCoerceString_MalformedJSONFallsBackToString(t *testing.T) {
	in := `{not valid json`
	got := TryCoerceString(in)
	if got != in {
		t.Errorf("got %#v, want the original string unchanged on parse failure", got)
	}
}

func TestTryCoerceString_BraceLikeButNotJSONFallsBackToString(t *testing.T) {
	// A string that starts with '{' but isn't JSON at all — must not panic,
	// must fall back to the original string.
	in := "{this is not json}"
	got := TryCoerceString(in)
	if got != in {
		t.Errorf("got %#v, want %q unchanged", got, in)
	}
}

func TestTryCoerceString_NumericPrecedesJSONCheck(t *testing.T) {
	// Sanity: numeric/bool parsing is attempted before JSON detection, so a
	// plain number never accidentally falls through to json.Unmarshal.
	got := TryCoerceString("123")
	if got != float64(123) {
		t.Errorf("got %#v, want float64(123)", got)
	}
}
