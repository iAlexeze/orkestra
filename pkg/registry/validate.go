package registry

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/ialexeze/orkestra/initialize"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/reconciler"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// -----------------------------------------------------------------------------
// Entry point
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// Validation: Pretty error reporting
// -----------------------------------------------------------------------------

func (r *CRDRegistry) handleValidationErrors(err error) {
	logger.Info().Msg("Validation error:")

	if errs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range errs {
			// Extract index from namespace: CRDRegistry.CRDs[3].Workers
			var index int
			fmt.Sscanf(e.StructNamespace(), "CRDRegistry.CRDs[%d]", &index)

			crdName := "(unknown)"
			if index >= 0 && index < len(r.CRDs) {
				crdName = r.CRDs[index].Name
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

func (r *CRDRegistry) validateGVKUniqueness() error {
	seen := make(map[string]string) // key -> name

	for _, crd := range r.CRDs {
		key := fmt.Sprintf("%s/%s/%s", crd.Group, crd.Version, crd.Kind)

		if existing, ok := seen[key]; ok {
			return fmt.Errorf(
				"duplicate GVK detected: %s/%s, Kind=%s (CRDs: %s and %s)",
				crd.Group, crd.Version, crd.Kind, existing, crd.Name,
			)
		}

		seen[key] = crd.Name
	}

	return nil
}

// -----------------------------------------------------------------------------
// Validation: dependsOn existence + cycle detection
// -----------------------------------------------------------------------------

func (r *CRDRegistry) validateDependsOn() error {
	// Build lookup map
	exists := make(map[string]bool)
	for _, crd := range r.CRDs {
		exists[crd.Name] = true
	}

	// Validate references
	for _, crd := range r.CRDs {
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
	return r.detectDependencyCycles()
}

// -----------------------------------------------------------------------------
// Cycle detection (DFS)
// -----------------------------------------------------------------------------

func (r *CRDRegistry) detectDependencyCycles() error {
	graph := make(map[string][]string)
	for _, crd := range r.CRDs {
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
func (r *CRDRegistry) SetGroupVersionKind() error {
	for i := range r.CRDs {
		crd := &r.CRDs[i]

		crd.GroupVersionKind = schema.GroupVersionKind{
			Group:   crd.Group,
			Version: crd.Version,
			Kind:    crd.Kind,
		}

		crd.GroupVersion = &schema.GroupVersion{
			Group:   crd.Group,
			Version: crd.Version,
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
func (r *CRDRegistry) SetDefaults() error {
	for i := range r.CRDs {
		crd := &r.CRDs[i]

		// Handle namespaced and cluster-scoped crds
		if !crd.Namespaced && crd.Namespace != "" {
			logger.Warn().Msgf("%s is clusterscoped. Namespace %s will be ignored", crd.Kind, crd.Namespace)
			crd.Namespace = ""
		}

		// Handle API path
		if crd.APIPath == "" {
			logger.Warn().Msgf("API path for Kind=%s is empty. Setting to '/apis'", crd.Kind)
			crd.APIPath = "/apis"
		}

		// Handle plural name
		crd.Name = strings.ToLower(crd.Name)

		if crd.Plural == "" {
			logger.Warn().Msgf("Plural name for %s is empty. Setting to '%ss'", crd.Kind, crd.Name)
			crd.Plural = fmt.Sprintf("%ss", strings.ToLower(crd.Name))
		}

		// Handle description
		if crd.Description == "" {
			crd.Description = fmt.Sprintf("%s CRD, GVK: %s", crd.Kind, crd.GroupVersionKind.String())
		}
	}
	return nil
}

// ---------------------------------------------------------------------------------
//
// Add RuntimeObjects
func (r *CRDRegistry) addRuntimeObjects() error {
	for i := range r.CRDs {
		crd := &r.CRDs[i]
		gvk := crd.GroupVersionKind

		objFn, ok := initialize.ObjectRegistry[gvk]
		if !ok {
			return fmt.Errorf("no object constructor registered for %s", gvk)
		}

		listFn, ok := initialize.ListRegistry[gvk]
		if !ok {
			return fmt.Errorf("no list constructor registered for %s", gvk)
		}

		crd.ObjectYamlMode = objFn
		crd.ListObjectYamlMode = listFn
	}

	return nil
}

// ---------------------------------------------------------------------------------
// Add reconcilers
func (r *CRDRegistry) addReconcilers() error {
	recs := reconciler.RegisterReconcilers()

	for i := range r.CRDs {
		crd := &r.CRDs[i]

		fn, ok := recs[crd.Name]
		if !ok {
			return fmt.Errorf("CRD '%s' has no registered reconciler", crd.Name)
		}

		crd.Reconciler = fn
	}

	return nil
}
