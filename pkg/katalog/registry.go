package katalog

import (
	"fmt"
	"reflect"

	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/logger"
	ork_runtime "github.com/ialexeze/orkestra/pkg/runtime"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// -----------------------------------------------------------------------------
// Entry point
// -----------------------------------------------------------------------------

// NewKatalog returns a list of CRD data
func NewKatalog(mode, path string) *Katalog {
	katalog := &Katalog{}
	var entries []orktypes.CRDEntry
	var err error

	switch mode {
	case konfig.TypedMode:
		entries, err = katalog.KomposeKatalogFromGo()
		if err != nil {
			utils.Exit(err)
		}
	case konfig.DynamicMode:
		// Register runtime objects
		ork_runtime.RegisterRuntimeObjects()

		// Guard: if ObjectRegistry is empty, user forgot to run ork generate
		for _, crd := range entries {
			if len(orktypes.ObjectRegistry) == 0 && !crd.IsDynamic() {
				utils.Exit(fmt.Errorf(
					"ObjectRegistry is empty — run 'ork generate registry --katalog %s' first",
					path,
				))
			}

		}
		// Build CRDs
		entries, err = katalog.KomposeKatalogFromYaml(path)
		if err != nil {
			utils.Exit(err)
		}
	default:
		utils.Exit(fmt.Errorf("must be 'dynamic' or 'typed' invalid katalog mode: %s", mode))
	}

	if len(entries) == 0 {
		utils.Exit(fmt.Errorf("validation error: katalog empty"))
	}

	// Pass to enabled
	katalog.enabledCRDs = entries

	kat, err := katalog.validateConfig()
	if err != nil {
		utils.Exit(err)
	}

	kat, err = kat.updateResourceMapAndReturn()
	if err != nil {
		utils.Exit(err)
	}

	return kat
}

// NewSchemeRegistry returns a new scheme
func NewSchemeRegistry(k *Katalog) (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	// 1. Register built-in Kubernetes types
	metav1.AddToGroupVersion(scheme, metav1.SchemeGroupVersion)

	// 2. Register core Kubernetes types
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}

	// 3. Register CRDs
	var err error
	if k.mode.Dynamic {
		if scheme, err = ork_runtime.RegisterScheme(scheme); err != nil {
			return nil, err
		}
		if scheme, err = k.registerDynamicScheme(scheme); err != nil {
			return nil, err
		}
	} else if k.mode.Typed {
		if scheme, err = k.registerGoScheme(scheme); err != nil {
			return nil, err
		}
	}

	return scheme, nil
}

// Helpers
// Update resource map
func (k *Katalog) updateResourceMapAndReturn() (*Katalog, error) {
	// Map the type of the object
	for _, c := range k.enabledCRDs {
		if k.enabledEmpty() {
			return nil, fmt.Errorf("no enabled CRDs found")
		}

		if k.mode.Dynamic {
			// Map the type of the object
			logger.Debug().Msgf("updating resource map for %s", c.GroupVersionKind.String())
			resourceTypeMap[reflect.TypeOf(c.DynamicModeObject)] = c.GroupVersionKind.String()
		} else if k.mode.Typed {
			resourceTypeMap[reflect.TypeOf(c.TypedModeObject)] = c.GroupVersionKind.String()
		}
	}

	return k, nil
}

func (k *Katalog) registerGoScheme(scheme *runtime.Scheme) (*runtime.Scheme, error) {
	for _, c := range k.enabledCRDs {
		if k.enabledEmpty() {
			return nil, fmt.Errorf("no enabled CRDs found")
		}

		if err := c.Scheme(scheme); err != nil {
			return nil, fmt.Errorf("failed to register %s: %w", c.GroupVersionKind, err)
		}
	}
	return scheme, nil
}

// Register dynamic CRDs — tells the watch stream to decode
// these GVKs as *unstructured.Unstructured instead of failing
func (k *Katalog) registerDynamicScheme(scheme *runtime.Scheme) (*runtime.Scheme, error) {
	for _, crd := range k.enabledCRDs {
		if crd.IsDynamic() {
			// Register Object
			scheme.AddKnownTypeWithName(
				schema.GroupVersionKind{
					Group:   crd.APITypes.Group,
					Version: crd.APITypes.Version,
					Kind:    crd.APITypes.Kind,
				},
				&unstructured.Unstructured{},
			)

			// Register List
			scheme.AddKnownTypeWithName(
				schema.GroupVersionKind{
					Group:   crd.APITypes.Group,
					Version: crd.APITypes.Version,
					Kind:    crd.APITypes.Kind + "List",
				},
				&unstructured.UnstructuredList{},
			)
		}
	}

	return scheme, nil
}
