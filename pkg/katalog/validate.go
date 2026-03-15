package katalog

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/ialexeze/orkestra/pkg/logger"
	ork_runtime "github.com/ialexeze/orkestra/pkg/runtime"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// -----------------------------------------------------------------------------
// Entry point
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// Validation: Pretty error reporting
// -----------------------------------------------------------------------------

func (k *Katalog) handleValidationErrors(err error) {
	logger.Info().Msg("Validation error:")

	if errs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range errs {
			// Extract index from namespace: Katalog.CRDs[3].Workers
			var index int
			fmt.Sscanf(e.StructNamespace(), "Katalog.CRDs[%d]", &index)

			crdName := "(unknown)"
			if index >= 0 && index < len(k.enabledCRDs) {
				crdName = k.enabledCRDs[index].Name
			}

			fmt.Printf("CRD '%s': field '%s' failed on '%s'\n",
				crdName, e.Field(), e.Tag())
		}
	} else {
		fmt.Println(err)
	}
}

// -----------------------------------------------------------------------------
// Validation: GVK uniqueness
// -----------------------------------------------------------------------------

func (k *Katalog) validateGVKUniqueness() error {
	seen := make(map[string]string) // key -> name

	for _, crd := range k.enabledCRDs {
		key := fmt.Sprintf("%s/%s/%s", crd.APITypes.Group, crd.APITypes.Version, crd.APITypes.Kind)

		if existing, ok := seen[key]; ok {
			return fmt.Errorf(
				"duplicate GVK detected: %s/%s, Kind=%s (CRDs: %s and %s)",
				crd.APITypes.Group, crd.APITypes.Version, crd.APITypes.Kind, existing, crd.Name,
			)
		}

		seen[key] = crd.Name
	}

	return nil
}

// -----------------------------------------------------------------------------
// Validation: dependsOn existence + cycle detection
// -----------------------------------------------------------------------------

func (k *Katalog) validateDependsOn() error {
	// Build lookup map
	exists := make(map[string]bool)
	for _, crd := range k.enabledCRDs {
		exists[crd.Name] = true
	}

	// Validate references
	for _, crd := range k.enabledCRDs {
		for _, dep := range crd.DependsOn {
			if dep == crd.Name {
				return fmt.Errorf("CRD '%s' cannot depend on itself", crd.Name)
			}
			if !exists[dep] {
				return fmt.Errorf("CRD '%s' depends on unknown or disabled CRD '%s'", crd.Name, dep)
			}
		}
	}

	// Detect cycles
	return k.detectDependencyCycles()
}

// -----------------------------------------------------------------------------
// Cycle detection (DFS)
// -----------------------------------------------------------------------------

func (k *Katalog) detectDependencyCycles() error {
	graph := make(map[string][]string)
	for _, crd := range k.enabledCRDs {
		graph[crd.Name] = crd.DependsOn
	}

	visited := make(map[string]bool)
	stack := make(map[string]bool)

	var dfs func(node string) error
	dfs = func(node string) error {
		if stack[node] {
			return fmt.Errorf("dependency cycle detected involving '%s'", node)
		}
		if visited[node] {
			return nil
		}

		visited[node] = true
		stack[node] = true

		for _, dep := range graph[node] {
			if err := dfs(dep); err != nil {
				return err
			}
		}

		stack[node] = false
		return nil
	}

	for name := range graph {
		if err := dfs(name); err != nil {
			return err
		}
	}

	return nil
}

// ---------------------------------------------------------------------------------
//
// Set GroupVersionKind
func (k *Katalog) setGroupVersionKind() error {
	for i := range k.enabledCRDs {
		crd := &k.enabledCRDs[i]

		crd.GroupVersionKind = schema.GroupVersionKind{
			Group:   crd.APITypes.Group,
			Version: crd.APITypes.Version,
			Kind:    crd.APITypes.Kind,
		}
		crd.GroupVersionResource = schema.GroupVersionResource{
			Group:    crd.APITypes.Group,
			Version:  crd.APITypes.Version,
			Resource: crd.APITypes.Plural,
		}

		crd.GroupVersion = &schema.GroupVersion{
			Group:   crd.APITypes.Group,
			Version: crd.APITypes.Version,
		}

		if crd.GroupVersionKind.Empty() {
			return fmt.Errorf("GroupVersionKind is empty. Enter a valid Group, Version and Kind for the CRD")
		}
	}
	return nil
}

// ---------------------------------------------------------------------------------
//
// Set SetDefaults
func (k *Katalog) setDefaults() error {
	for i := range k.enabledCRDs {
		crd := &k.enabledCRDs[i]

		// Handle namespaced and cluster-scoped crds
		if !crd.Namespaced && crd.Namespace != "" {
			logger.Warn().Msgf("%s is clusterscoped. Namespace %s will be ignored", crd.APITypes.Kind, crd.Namespace)
			crd.Namespace = ""
		}

		// Handle API path
		if crd.APITypes.APIPath == "" {
			logger.Info().Msgf("API path for Kind=%s is empty. Setting to '/apis'", crd.APITypes.Kind)
			crd.APITypes.APIPath = "/apis"
		}

		// Handle plural name
		crd.Name = strings.ToLower(crd.Name)

		if crd.APITypes.Plural == "" {
			logger.Info().Msgf("Plural name for %s is empty. Setting to '%ss'", crd.APITypes.Kind, crd.Name)
			crd.APITypes.Plural = fmt.Sprintf("%ss", strings.ToLower(crd.Name))
		}

		// Handle description
		if crd.Description == "" {
			crd.Description = fmt.Sprintf("%s CRD, GVK: %s", crd.APITypes.Kind, crd.GroupVersionKind.String())
		}

		// Handle finalizers
		if len(crd.ReconcilerConfig.Finalizers) == 0 {
			crd.ReconcilerConfig.Finalizers = k.Spec.Finalizers
		}
	}
	return nil
}

// ---------------------------------------------------------------------------------
//
// Add RuntimeObjects
func (k *Katalog) addRuntimeObjects() error {
	for i := range k.enabledCRDs {
		crd := &k.enabledCRDs[i]
		gvk := crd.GroupVersionKind

		if crd.IsUnstructured() {
			// Set unstructured factories so GetRuntimeObjects works consistently
			g := crd.APITypes.Group
			v := crd.APITypes.Version
			ki := crd.APITypes.Kind

			crd.ObjectYamlMode = func() runtime.Object {
				u := &unstructured.Unstructured{}
				u.SetGroupVersionKind(schema.GroupVersionKind{
					Group: g, Version: v, Kind: ki,
				})
				return u
			}
			crd.ListObjectYamlMode = func() runtime.Object {
				ul := &unstructured.UnstructuredList{}
				ul.SetGroupVersionKind(schema.GroupVersionKind{
					Group: g, Version: v, Kind: ki + "List",
				})
				return ul
			}
			continue
		}

		// Typed mode — look up from registry
		objFn, ok := ork_runtime.ObjectRegistry[gvk]
		if !ok {
			return fmt.Errorf("addRuntimeObjects: no object constructor registered for %s", gvk)
		}
		listFn, ok := ork_runtime.ListRegistry[gvk]
		if !ok {
			return fmt.Errorf("addRuntimeObjects: no list constructor registered for %s", gvk)
		}

		crd.ObjectYamlMode = objFn
		crd.ListObjectYamlMode = listFn
	}
	return nil
}

// ---------------------------------------------------------------------------------
// Add reconcilers
func (k *Katalog) addReconcilers() error {
	for i := range k.enabledCRDs {
		crd := &k.enabledCRDs[i]

		// Default → skip registry lookup
		if crd.ReconcilerConfig.Default {
			continue
		}

		constructorFn, ok := ork_runtime.ReconcilerRegistry[crd.GroupVersionKind]
		if !ok {
			return fmt.Errorf(
				"CRD %q: no constructor registered — "+
					"check reconciler.constructor in Katalog and re-run ork generate registry",
				crd.Name,
			)
		}

		crd.ReconcilerConfig.Constructor = constructorFn // ← sets the Go function field
	}
	return nil
}

// ---------------------------------------------------------------------------------
// Add hooks
func (k *Katalog) addHooks() error {
	for i := range k.enabledCRDs {
		crd := &k.enabledCRDs[i]
		if !crd.ReconcilerConfig.Default {
			continue
		}
		if hookFn, ok := ork_runtime.HookRegistry[crd.GroupVersionKind]; ok {
			crd.ReconcilerConfig.HookFactory = hookFn
		}
		// not found — fine, GenericReconciler runs without hooks
	}
	return nil
}
