package registry

import (
	"fmt"
	"reflect"

	"github.com/ialexeze/multi-crd-controller/pkg/config/initialize"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// -----------------------------------------------------------------------------
// Entry point
// -----------------------------------------------------------------------------

// NewCRDRegistry returns a list of CRD data
func NewCRDRegistry(mode, path string) *CRDRegistry {
	registry := &CRDRegistry{}
	var entries []initialize.CRDEntry

	switch mode {
	case GoMode:
		entries = registry.buildCRDRegistryFromGo()
	case YamlMode:
		// Register runtime objects
		initialize.RegisterRuntimeObjects()
		// Build CRDs
		var err error
		entries, err = registry.buildCRDRegistryFromYaml(path)
		if err != nil {
			panic(err)
		}
	default:
		panic("must specify either Go or YAML CRD registry mode")
	}

	reg, err := registry.validateConfig(entries)
	if err != nil {
		panic(err)
	}

	return reg.updateResourceMapAndReturn()
}

// NewSchemeRegistry returns a new scheme
func NewSchemeRegistry(r *CRDRegistry) (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	// 1. Register built-in Kubernetes types
	metav1.AddToGroupVersion(scheme, metav1.SchemeGroupVersion)

	// 2. Register core Kubernetes types
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}

	// 3. Register CRDs
	var err error
	if r.Mode.Yaml {
		if scheme, err = initialize.RegisterScheme(scheme); err != nil {
			return nil, err
		}
	} else if r.Mode.Go {
		if scheme, err = r.registerGoScheme(scheme); err != nil {
			return nil, err
		}
	}

	return scheme, nil
}

// Helpers
// Update resource map
func (r *CRDRegistry) updateResourceMapAndReturn() *CRDRegistry {
	// Map the type of the object
	for _, c := range r.CRDs {
		if r.Mode.Yaml {
			resourceTypeMap[reflect.TypeOf(c.ObjectYamlMode)] = c.GroupVersionKind.String()
		} else if r.Mode.Go {
			resourceTypeMap[reflect.TypeOf(c.ObjectGoMode)] = c.GroupVersionKind.String()
		} else {
			panic("must specify either Go or YAML CRD registry mode")
		}
	}

	return r
}

func (r *CRDRegistry) registerGoScheme(scheme *runtime.Scheme) (*runtime.Scheme, error) {
	for _, c := range r.CRDs {
		if err := c.Scheme(scheme); err != nil {
			return nil, fmt.Errorf("failed to register %s: %w", c.GroupVersionKind, err)
		}
	}
	return scheme, nil
}
