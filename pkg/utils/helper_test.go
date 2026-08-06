package utils

import (
	"fmt"
	"reflect"
	"testing"
)

func TestToStringSet(t *testing.T) {
	tests := []struct {
		name string
		ops  []string
		want map[string]bool
	}{
		{
			name: "valid slice",
			ops:  []string{"read", "write", "execute"},
			want: map[string]bool{
				"read":    true,
				"write":   true,
				"execute": true,
			},
		},
		{
			name: "slice with duplicates",
			ops:  []string{"read", "read", "write", "write"},
			want: map[string]bool{
				"read":  true,
				"write": true,
			},
		},
		{
			name: "empty slice",
			ops:  []string{},
			want: map[string]bool{},
		},
		{
			name: "nil slice",
			ops:  nil,
			want: map[string]bool{},
		},
		{
			name: "single element",
			ops:  []string{"admin"},
			want: map[string]bool{
				"admin": true,
			},
		},
		{
			name: "with special characters",
			ops:  []string{"key.with.dots", "key-with-dashes", "key_with_underscores"},
			want: map[string]bool{
				"key.with.dots":        true,
				"key-with-dashes":      true,
				"key_with_underscores": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToStringSet(tt.ops)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToStringSet() = %v, want %v", got, tt.want)
			}
			// Verify all values are true
			for key, val := range got {
				if !val {
					t.Errorf("ToStringSet() key %q has value false, expected true", key)
				}
			}
		})
	}
}

func TestSetContains(t *testing.T) {
	tests := []struct {
		name string
		s    map[string]struct{}
		key  string
		want bool
	}{
		{
			name: "key exists",
			s: map[string]struct{}{
				"read":  {},
				"write": {},
			},
			key:  "read",
			want: true,
		},
		{
			name: "key does not exist",
			s: map[string]struct{}{
				"read":  {},
				"write": {},
			},
			key:  "execute",
			want: false,
		},
		{
			name: "nil map",
			s:    nil,
			key:  "read",
			want: false,
		},
		{
			name: "empty map",
			s:    map[string]struct{}{},
			key:  "read",
			want: false,
		},
		{
			name: "empty string key",
			s: map[string]struct{}{
				"": {},
			},
			key:  "",
			want: true,
		},
		{
			name: "key with special characters",
			s: map[string]struct{}{
				"key.with.dots": {},
				"key-with-dash": {},
			},
			key:  "key.with.dots",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SetContains(tt.s, tt.key)
			if got != tt.want {
				t.Errorf("SetContains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapContains(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want bool
	}{
		{
			name: "key exists with string value",
			m: map[string]interface{}{
				"name": "myapp",
				"port": 8080,
			},
			key:  "name",
			want: true,
		},
		{
			name: "key exists with int value",
			m: map[string]interface{}{
				"name": "myapp",
				"port": 8080,
			},
			key:  "port",
			want: true,
		},
		{
			name: "key exists with nil value",
			m: map[string]interface{}{
				"name": "myapp",
				"nil":  nil,
			},
			key:  "nil",
			want: true,
		},
		{
			name: "key does not exist",
			m: map[string]interface{}{
				"name": "myapp",
				"port": 8080,
			},
			key:  "missing",
			want: false,
		},
		{
			name: "nil map",
			m:    nil,
			key:  "name",
			want: false,
		},
		{
			name: "empty map",
			m:    map[string]interface{}{},
			key:  "name",
			want: false,
		},
		{
			name: "key with special characters",
			m: map[string]interface{}{
				"key.with.dots": "value1",
				"key-with-dash": "value2",
			},
			key:  "key.with.dots",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapContains(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("MapContains() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Additional tests for generic MapContains with different value types
func TestMapContainsGeneric(t *testing.T) {
	// Test with int values
	t.Run("int values", func(t *testing.T) {
		m := map[string]int{
			"one": 1,
			"two": 2,
		}
		if !MapContains(m, "one") {
			t.Errorf("MapContains() with int map should return true for existing key")
		}
		if MapContains(m, "three") {
			t.Errorf("MapContains() with int map should return false for missing key")
		}
	})

	// Test with bool values
	t.Run("bool values", func(t *testing.T) {
		m := map[string]bool{
			"enabled":  true,
			"disabled": false,
		}
		if !MapContains(m, "enabled") {
			t.Errorf("MapContains() with bool map should return true for existing key")
		}
		if MapContains(m, "missing") {
			t.Errorf("MapContains() with bool map should return false for missing key")
		}
	})

	// Test with struct values
	t.Run("struct values", func(t *testing.T) {
		type Person struct {
			Name string
			Age  int
		}
		m := map[string]Person{
			"alice": {Name: "Alice", Age: 30},
			"bob":   {Name: "Bob", Age: 25},
		}
		if !MapContains(m, "alice") {
			t.Errorf("MapContains() with struct map should return true for existing key")
		}
		if MapContains(m, "charlie") {
			t.Errorf("MapContains() with struct map should return false for missing key")
		}
	})

	// Test with slice values
	t.Run("slice values", func(t *testing.T) {
		m := map[string][]string{
			"fruits": {"apple", "banana"},
			"colors": {"red", "blue"},
		}
		if !MapContains(m, "fruits") {
			t.Errorf("MapContains() with slice map should return true for existing key")
		}
		if MapContains(m, "vegetables") {
			t.Errorf("MapContains() with slice map should return false for missing key")
		}
	})
}

// Benchmark tests
func BenchmarkToStringSet(b *testing.B) {
	ops := []string{"read", "write", "execute", "delete", "create", "update"}
	for i := 0; i < b.N; i++ {
		ToStringSet(ops)
	}
}

func BenchmarkSetContains(b *testing.B) {
	s := map[string]struct{}{
		"read":    {},
		"write":   {},
		"execute": {},
		"delete":  {},
		"create":  {},
		"update":  {},
	}
	for i := 0; i < b.N; i++ {
		SetContains(s, "read")
	}
}

func BenchmarkMapContains(b *testing.B) {
	m := map[string]interface{}{
		"read":    true,
		"write":   true,
		"execute": true,
		"delete":  true,
		"create":  true,
		"update":  true,
	}
	for i := 0; i < b.N; i++ {
		MapContains(m, "read")
	}
}

// Example usage tests
func ExampleToStringSet() {
	ops := []string{"read", "write", "execute"}
	set := ToStringSet(ops)
	if set["read"] {
		fmt.Println("read permission exists")
	}
	// Output: read permission exists
}

func ExampleSetContains() {
	permissions := map[string]struct{}{
		"read":  {},
		"write": {},
	}
	if SetContains(permissions, "read") {
		fmt.Println("has read permission")
	}
	// Output: has read permission
}

func ExampleMapContains() {
	config := map[string]interface{}{
		"port":  8080,
		"debug": true,
	}
	if MapContains(config, "port") {
		fmt.Println("port is configured")
	}
	// Output: port is configured
}
