// pkg/kubeclient/args.go
package kubeclient

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Args is the typed view of the args: map declared in katalog.yaml
// under hooks.args or constructor.args.
//
// Values come in as raw YAML scalars (string, int, bool, map, slice).
// Accessors convert on the fly and return zero values when the key
// is absent or the wrong type — hooks should treat absent args as "use default".
type Args map[string]interface{}

// String returns the value for key as a string.
// Returns "" if the key is absent or is not a string.
func (a Args) String(key string) string {
	v, _ := a[key].(string)
	return v
}

// Bool returns the value for key as a bool.
// Returns false if absent or not a bool.
func (a Args) Bool(key string) bool {
	v, _ := a[key].(bool)
	return v
}

// Int returns the value for key as an int.
// YAML unmarshals integers as int, int64, or float64 depending on the decoder;
// all three are tried. Template-evaluated args arrive as strings — those are
// parsed with strconv so {{ default "3" .spec.replicas }} works naturally.
func (a Args) Int(key string) int {
	switch v := a[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 0
}

// Sub returns the nested map at key as an Args value.
// Returns an empty Args if the key is absent or is not a map.
func (a Args) Sub(key string) Args {
	switch v := a[key].(type) {
	case map[string]interface{}:
		return Args(v)
	case Args:
		return v
	}
	return Args{}
}

// Slice returns the slice at key as []interface{}.
// Returns nil if the key is absent or is not a slice.
func (a Args) Slice(key string) []interface{} {
	v, _ := a[key].([]interface{})
	return v
}

// ResolveArgsMap walks rawArgs and evaluates any string values that contain
// Go template expressions using eval. Non-string and non-template values pass through unchanged.
func ResolveArgsMap(rawArgs map[string]interface{}, eval func(string) (string, bool)) Args {
	out := make(Args, len(rawArgs))
	for k, v := range rawArgs {
		out[k] = resolveArgValue(v, eval)
	}
	return out
}

func resolveArgValue(v interface{}, eval func(string) (string, bool)) interface{} {
	switch val := v.(type) {
	case string:
		if !strings.Contains(val, "{{") {
			return val
		}
		if resolved, ok := eval(val); ok {
			return resolved
		}
		return val
	case map[string]interface{}:
		sub := make(map[string]interface{}, len(val))
		for k, sv := range val {
			sub[k] = resolveArgValue(sv, eval)
		}
		return sub
	default:
		return v
	}
}

// BindArgs JSON-round-trips the args map into dst (must be a pointer to a struct).
// Use this when you want a strongly-typed args struct in your hook or constructor.
//
//	type MyArgs struct {
//	    Region string `json:"region"`
//	}
//	var cfg MyArgs
//	if err := kube.Args().BindArgs(&cfg); err != nil { ... }
func (a Args) BindArgs(dst interface{}) error {
	b, err := json.Marshal(map[string]interface{}(a))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}
