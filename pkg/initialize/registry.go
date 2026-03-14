// initialize/registry.go
package initialize

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// RegisterRuntimeObjects
func RegisterRuntimeObjects() {}

// RegisterScheme registers all generated CRD types into the provided scheme.
// Called by NewSchemeRegistry() in YAML mode.
func RegisterScheme(scheme *runtime.Scheme) (*runtime.Scheme, error) {
	return scheme, nil
}
