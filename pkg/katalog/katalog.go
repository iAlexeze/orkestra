package katalog

import (
	"fmt"
	"reflect"

	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	ork_runtime "github.com/orkspace/orkestra/pkg/typeregistry"
	orktypes "github.com/orkspace/orkestra/pkg/types"

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
func NewKatalog(kfg *konfig.Konfig, m *merger.Merger) *Katalog {
	katalog := &Katalog{}
	katalog.konfig = kfg

	paths := katalog.konfig.Katalog().Paths()

	// Register runtime objects
	ork_runtime.RegisterRuntimeObjects()

	// Build CRDs
	entries, err := katalog.KomposeRuntimeKatalog(kfg, m, paths...)
	if err != nil {
		exit(err)
	}

	if len(entries) == 0 && !katalog.IsStandaloneGateway() {
		exit(fmt.Errorf("validation error: katalog empty"))
	}

	// Guard: if ObjectRegistry is empty, user forgot to run ork generate
	for _, crd := range entries {
		if len(orktypes.ObjectRegistry) == 0 && !crd.IsDynamic() {
			exit(fmt.Errorf(
				"ObjectRegistry is empty — run 'ork generate registry --file <my-katalog.yaml>' first",
			))
		}
	}

	kat, err := katalog.ValidateConfig(kfg)
	if err != nil {
		exit(err)
	}

	if err := kat.CheckDeprecationPolicy(); err != nil {
		exit(err)
	}

	kat, err = kat.updateResourceMapAndReturn()
	if err != nil {
		exit(err)
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
	// Register dynamic scheme
	if scheme, err = k.registerDynamicScheme(scheme); err != nil {
		return nil, err
	}
	// Register typed scheme
	if scheme, err = ork_runtime.RegisterTypedScheme(scheme); err != nil {
		return nil, err
	}

	// 4. Register external GVKs from custom: blocks so the fake dynamic client's
	// object tracker can create/get them during simulate (in-memory cluster).
	// Without this, the tracker has no schema entry and Create calls fail silently.
	if scheme, err = k.registerCustomResourceScheme(scheme); err != nil {
		return nil, err
	}

	return scheme, nil
}

// Helpers
// Update resource map
func (k *Katalog) updateResourceMapAndReturn() (*Katalog, error) {
	// Map the type of the object
	for _, c := range k.enabledCRDs {
		if len(k.enabledCRDs) == 0 {
			return nil, fmt.Errorf("no enabled CRDs found")
		}

		// Map the type of the object
		logger.Debug().Msgf("updating resource map for %s", c.GroupVersionKind.String())
		resourceTypeMap[reflect.TypeOf(c.DynamicModeObject)] = c.GroupVersionKind.String()
	}

	return k, nil
}

// registerCustomResourceScheme registers external GVKs declared in custom: blocks
// (onCreate and onReconcile) into the scheme so the fake dynamic client's object
// tracker can store and retrieve them during simulate. Deduplicates by GVK string.
func (k *Katalog) registerCustomResourceScheme(scheme *runtime.Scheme) (*runtime.Scheme, error) {
	seen := make(map[string]bool)
	for _, crd := range k.enabledCRDs {
		if crd.HasOnCreate() && crd.OperatorBox.OnCreate.CustomResource != nil {
			for i := range crd.OperatorBox.OnCreate.CustomResource {
				cr := &crd.OperatorBox.OnCreate.CustomResource[i]
				key := cr.APIVersion + "/" + cr.Kind
				if seen[key] || cr.APIVersion == "" || cr.Kind == "" {
					continue
				}
				seen[key] = true
				gvk, err := cr.BuildGVK()
				if err != nil {
					continue
				}
				scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
				scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind + "List"}, &unstructured.UnstructuredList{})
			}
		}

		if crd.HasOnReconcile() && crd.OperatorBox.OnReconcile.CustomResource != nil {
			for i := range crd.OperatorBox.OnReconcile.CustomResource {
				cr := &crd.OperatorBox.OnReconcile.CustomResource[i]
				key := cr.APIVersion + "/" + cr.Kind
				if seen[key] || cr.APIVersion == "" || cr.Kind == "" {
					continue
				}
				seen[key] = true
				gvk, err := cr.BuildGVK()
				if err != nil {
					continue
				}
				scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
				scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind + "List"}, &unstructured.UnstructuredList{})
			}
		}

	}
	return scheme, nil
}

// Register dynamic CRDs — tells the watch stream to decode
// these GVKs as *unstructured.Unstructured instead of failing
func (k *Katalog) registerDynamicScheme(scheme *runtime.Scheme) (*runtime.Scheme, error) {
	for _, crd := range k.enabledCRDs {
		if crd.IsDynamic() && crd.APITypes.Location == "" && !crd.IsBuiltInType() {
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
