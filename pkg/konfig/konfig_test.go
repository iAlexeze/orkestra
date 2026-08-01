package konfig

import (
	"testing"
	"time"
)

func TestGetStrEnv(t *testing.T) {
	t.Run("unset falls back to default", func(t *testing.T) {
		if got := GetStrEnv("KONFIG_TEST_UNSET_STR", "fallback"); got != "fallback" {
			t.Errorf("got %q, want fallback", got)
		}
	})
	t.Run("set overrides default", func(t *testing.T) {
		t.Setenv("KONFIG_TEST_STR", "value")
		if got := GetStrEnv("KONFIG_TEST_STR", "fallback"); got != "value" {
			t.Errorf("got %q, want value", got)
		}
	})
	t.Run("set to empty string overrides default (LookupEnv sees it as present)", func(t *testing.T) {
		t.Setenv("KONFIG_TEST_STR_EMPTY", "")
		if got := GetStrEnv("KONFIG_TEST_STR_EMPTY", "fallback"); got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})
}

func TestGetStrSliceEnv(t *testing.T) {
	t.Run("unset falls back to default", func(t *testing.T) {
		def := []string{"a", "b"}
		got := GetStrSliceEnv("KONFIG_TEST_UNSET_SLICE", def)
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Errorf("got %v, want default %v", got, def)
		}
	})
	t.Run("set wraps the raw value as a single-element slice", func(t *testing.T) {
		t.Setenv("KONFIG_TEST_SLICE", "a,b,c")
		got := GetStrSliceEnv("KONFIG_TEST_SLICE", nil)
		if len(got) != 1 || got[0] != "a,b,c" {
			t.Errorf("got %v, want a single element %q — no comma-splitting here", got, "a,b,c")
		}
	})
}

func TestGetBoolEnv(t *testing.T) {
	t.Run("unset falls back to default", func(t *testing.T) {
		if got := GetBoolEnv("KONFIG_TEST_UNSET_BOOL", true); !got {
			t.Error("got false, want default true")
		}
	})
	t.Run("set true", func(t *testing.T) {
		t.Setenv("KONFIG_TEST_BOOL", "true")
		if got := GetBoolEnv("KONFIG_TEST_BOOL", false); !got {
			t.Error("got false, want true")
		}
	})
	t.Run("set false", func(t *testing.T) {
		t.Setenv("KONFIG_TEST_BOOL_FALSE", "false")
		if got := GetBoolEnv("KONFIG_TEST_BOOL_FALSE", true); got {
			t.Error("got true, want false")
		}
	})
	t.Run("unparseable value falls back to default", func(t *testing.T) {
		t.Setenv("KONFIG_TEST_BOOL_GARBAGE", "not-a-bool")
		if got := GetBoolEnv("KONFIG_TEST_BOOL_GARBAGE", true); !got {
			t.Error("got false, want default true")
		}
	})
}

func TestGetIntEnv(t *testing.T) {
	t.Run("unset falls back to default", func(t *testing.T) {
		if got := GetIntEnv("KONFIG_TEST_UNSET_INT", 42); got != 42 {
			t.Errorf("got %d, want 42", got)
		}
	})
	t.Run("set overrides default", func(t *testing.T) {
		t.Setenv("KONFIG_TEST_INT", "7")
		if got := GetIntEnv("KONFIG_TEST_INT", 42); got != 7 {
			t.Errorf("got %d, want 7", got)
		}
	})
	t.Run("unparseable value falls back to default", func(t *testing.T) {
		t.Setenv("KONFIG_TEST_INT_GARBAGE", "not-a-number")
		if got := GetIntEnv("KONFIG_TEST_INT_GARBAGE", 42); got != 42 {
			t.Errorf("got %d, want default 42", got)
		}
	})
}

func TestGetDurEnvSeconds(t *testing.T) {
	t.Run("unset falls back to default seconds", func(t *testing.T) {
		if got := GetDurEnvSeconds("KONFIG_TEST_UNSET_DUR", 15); got != 15*time.Second {
			t.Errorf("got %v, want 15s", got)
		}
	})
	t.Run("set overrides default, interpreted as seconds", func(t *testing.T) {
		t.Setenv("KONFIG_TEST_DUR", "30")
		if got := GetDurEnvSeconds("KONFIG_TEST_DUR", 15); got != 30*time.Second {
			t.Errorf("got %v, want 30s", got)
		}
	})
	t.Run("unparseable value falls back to default", func(t *testing.T) {
		t.Setenv("KONFIG_TEST_DUR_GARBAGE", "not-a-number")
		if got := GetDurEnvSeconds("KONFIG_TEST_DUR_GARBAGE", 15); got != 15*time.Second {
			t.Errorf("got %v, want default 15s", got)
		}
	})
}
