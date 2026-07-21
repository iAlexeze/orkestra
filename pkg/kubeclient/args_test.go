package kubeclient

import (
	"testing"
)

func TestArgs_String(t *testing.T) {
	a := Args{"key": "hello", "other": 42}
	if got := a.String("key"); got != "hello" {
		t.Errorf("String(key) = %q, want %q", got, "hello")
	}
	if got := a.String("missing"); got != "" {
		t.Errorf("String(missing) = %q, want empty", got)
	}
	if got := a.String("other"); got != "" {
		t.Errorf("String(other) = %q, want empty (wrong type)", got)
	}
}

func TestArgs_Bool(t *testing.T) {
	a := Args{"enabled": true, "disabled": false, "str": "true"}
	if !a.Bool("enabled") {
		t.Error("Bool(enabled) = false, want true")
	}
	if a.Bool("disabled") {
		t.Error("Bool(disabled) = true, want false")
	}
	if a.Bool("missing") {
		t.Error("Bool(missing) = true, want false")
	}
	// string "true" is not a bool — returns zero value
	if a.Bool("str") {
		t.Error("Bool(str) = true, want false (wrong type)")
	}
}

func TestArgs_Int(t *testing.T) {
	a := Args{
		"native":  3,
		"int64":   int64(7),
		"float64": float64(5),
		"str":     "9",
	}
	if got := a.Int("native"); got != 3 {
		t.Errorf("Int(native) = %d, want 3", got)
	}
	if got := a.Int("int64"); got != 7 {
		t.Errorf("Int(int64) = %d, want 7", got)
	}
	if got := a.Int("float64"); got != 5 {
		t.Errorf("Int(float64) = %d, want 5", got)
	}
	if got := a.Int("str"); got != 0 {
		t.Errorf("Int(str) = %d, want 0 (wrong type)", got)
	}
	if got := a.Int("missing"); got != 0 {
		t.Errorf("Int(missing) = %d, want 0", got)
	}
}

func TestArgs_Sub(t *testing.T) {
	a := Args{"nested": map[string]interface{}{"x": "y"}}
	sub := a.Sub("nested")
	if got := sub.String("x"); got != "y" {
		t.Errorf("Sub.String(x) = %q, want %q", got, "y")
	}
	empty := a.Sub("missing")
	if len(empty) != 0 {
		t.Errorf("Sub(missing) = %v, want empty", empty)
	}
}

func TestArgs_Slice(t *testing.T) {
	a := Args{"list": []interface{}{"a", "b"}}
	s := a.Slice("list")
	if len(s) != 2 {
		t.Errorf("Slice(list) len = %d, want 2", len(s))
	}
	if a.Slice("missing") != nil {
		t.Error("Slice(missing) should be nil")
	}
}

func TestArgs_BindArgs(t *testing.T) {
	type cfg struct {
		Region  string `json:"region"`
		Timeout int    `json:"timeout"`
	}
	a := Args{"region": "us-east-1", "timeout": float64(30)}
	var out cfg
	if err := a.BindArgs(&out); err != nil {
		t.Fatalf("BindArgs: %v", err)
	}
	if out.Region != "us-east-1" {
		t.Errorf("Region = %q, want us-east-1", out.Region)
	}
	if out.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", out.Timeout)
	}
}

func TestKubeclient_Args_Empty(t *testing.T) {
	k := &Kubeclient{}
	a := k.Args()
	if a == nil {
		t.Error("Args() should never return nil")
	}
	if len(a) != 0 {
		t.Errorf("Args() = %v, want empty", a)
	}
}

func TestKubeclient_WithArgs(t *testing.T) {
	k := &Kubeclient{name: "test"}
	args := Args{"region": "eu-west-1"}
	kWithArgs := k.WithArgs(args)

	// new instance has the args
	got := kWithArgs.Args()
	if got.String("region") != "eu-west-1" {
		t.Errorf("WithArgs: region = %q, want eu-west-1", got.String("region"))
	}

	// original is unchanged
	if len(k.Args()) != 0 {
		t.Error("WithArgs must not mutate the original Kubeclient")
	}
}
