package katalog

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
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
// Validate uniqueness
// -----------------------------------------------------------------------------
func (k *Katalog) validateUniqueness() error {
	if err := k.validateGVKUniqueness(); err != nil {
		return err
	}
	if err := k.validateNameUniqueness(); err != nil {
		return err
	}
	// if err := k.validatePluralUniqueness(); err != nil {
	// 	return err
	// }
	return nil
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
// Validation: Name uniqueness
// -----------------------------------------------------------------------------

func (k *Katalog) validateNameUniqueness() error {
	seen := make(map[string]bool)

	for _, crd := range k.enabledCRDs {
		if seen[crd.Name] {
			return fmt.Errorf("duplicate CRD name detected: %s", crd.Name)
		}
		seen[crd.Name] = true
	}

	return nil
}

// -----------------------------------------------------------------------------
// Validation: Name uniqueness
// -----------------------------------------------------------------------------

// func (k *Katalog) validatePluralUniqueness() error {
// 	seen := make(map[string]string)

// 	for _, crd := range k.enabledCRDs {
// 		if existing, ok := seen[crd.APITypes.Plural]; ok {
// 			return fmt.Errorf("duplicate plural detected: %s (CRDs: %s and %s)",
// 				crd.APITypes.Plural, existing, crd.Name)
// 		}
// 		seen[crd.APITypes.Plural] = crd.Name
// 	}

// 	return nil
// }

func (k *Katalog) validateReconcilerMode() error {
	for i := range k.enabledCRDs {
		crd := &k.enabledCRDs[i]
		mode := crd.Mode

		switch mode {
		case "":
			// No mode declared — default to dynamic
			logger.Debug().
				Str("crd", crd.Name).
				Msg("reconciler mode not set — defaulting to 'dynamic'")

			crd.Mode = orktypes.CRDModeDynamic

		case orktypes.CRDModeDynamic, orktypes.CRDModeTyped:
			// Valid — nothing to do

		default:
			return fmt.Errorf(
				"CRD %q: reconciler mode %q is not supported — use %q or %q",
				crd.Name, mode,
				orktypes.CRDModeDynamic,
				orktypes.CRDModeTyped,
			)
		}
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
			return fmt.Errorf("CRD '%s': missing required fields: apiTypes.group, apiTypes.version, apiTypes.kind", crd.Name)
		}

		if crd.GroupVersion.Empty() {
			return fmt.Errorf("CRD '%s': missing required fields: apiTypes.group, apiTypes.version", crd.Name)
		}

		if crd.APITypes.Kind == "" {
			return fmt.Errorf("CRD '%s': missing required field: apiTypes.kind", crd.Name)
		}

		if crd.GroupVersionResource.Empty() {
			return fmt.Errorf("CRD '%s': missing required fields: apiTypes.plural", crd.Name)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------------
//
// Set SetDefaults
func (k *Katalog) setDefaults(kfg *konfig.Konfig) error {
	for i := range k.enabledCRDs {
		crd := &k.enabledCRDs[i]

		// Add labels
		crd.Labels = append(crd.Labels, orktypes.ResourceLabel{
			Key:   konfig.LabelManaged,
			Value: konfig.LabelManagedValue,
		})

		// Remove all whitespaces
		crd.Name = strings.ReplaceAll(crd.Name, " ", "")
		crd.Name = strings.ToLower(crd.Name)

		if crd.Name == "" {
			return fmt.Errorf("CRD '%s': missing required field: name", crd.Name)
		}

		// Handle namespaced and cluster-scoped crds
		if !crd.IsNamespaced() && crd.Namespace != "" {
			logger.Warn().Msgf("%s is clusterscoped. Namespace %s will be ignored", crd.APITypes.Kind, crd.Namespace)
			crd.Namespace = ""
		}

		// Handle API path
		if crd.APITypes.APIPath == "" {
			logger.Debug().Msgf("API path for Kind=%s is empty. Setting to '/apis'", crd.APITypes.Kind)
			crd.APITypes.APIPath = "/apis"
		}

		// Handle plural name
		if crd.APITypes.Plural == "" {
			logger.Debug().Msgf("Plural name for %s is empty. Setting to '%ss'", crd.APITypes.Kind, crd.Name)
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

		// Handle Resync
		if crd.Resync == 0 {
			crd.Resync = kfg.Cluster().DefaultResync
		}

		// Handle Workers
		if crd.Workers == 0 {
			crd.Workers = kfg.Cluster().DefaultWorkers
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

		if crd.IsDynamic() {
			// Set dynamic factories so GetRuntimeObjects works consistently
			g := crd.APITypes.Group
			v := crd.APITypes.Version
			ki := crd.APITypes.Kind

			crd.DynamicModeObject = func() runtime.Object {
				u := &unstructured.Unstructured{}
				u.SetGroupVersionKind(schema.GroupVersionKind{
					Group: g, Version: v, Kind: ki,
				})
				return u
			}
			crd.ListDynamicModeObject = func() runtime.Object {
				ul := &unstructured.UnstructuredList{}
				ul.SetGroupVersionKind(schema.GroupVersionKind{
					Group: g, Version: v, Kind: ki + "List",
				})
				return ul
			}
			continue
		}

		// Typed mode — look up from registry
		objFn, ok := orktypes.ObjectRegistry[gvk]
		if !ok {
			return fmt.Errorf("addRuntimeObjects: no object constructor registered for %s", gvk)
		}
		listFn, ok := orktypes.ListRegistry[gvk]
		if !ok {
			return fmt.Errorf("addRuntimeObjects: no list constructor registered for %s", gvk)
		}

		crd.DynamicModeObject = objFn
		crd.ListDynamicModeObject = listFn
	}
	return nil
}

// ---------------------------------------------------------------------------------
// Add reconcilers
func (k *Katalog) addReconcilers() error {
	for i := range k.enabledCRDs {
		crd := &k.enabledCRDs[i]

		if !crd.IsDynamic() {

			// Default → skip registry lookup
			if crd.DefaultReconcile() {
				continue
			}

			constructorFn, ok := orktypes.ReconcilerRegistry[crd.GroupVersionKind]
			if !ok {
				return fmt.Errorf(
					"CRD %q: no constructor registered — "+
						"check reconciler.constructor in Katalog and re-run ork generate runtime",
					crd.Name,
				)
			}

			crd.ReconcilerConfig.Constructor = constructorFn // ← sets the Go function field
		}
	}
	return nil
}

// ---------------------------------------------------------------------------------
// Add hooks
func (k *Katalog) addHooks() error {
	for i := range k.enabledCRDs {
		crd := &k.enabledCRDs[i]
		if !crd.DefaultReconcile() {
			continue
		}
		if hookFn, ok := orktypes.HookRegistry[crd.GroupVersionKind]; ok {
			crd.ReconcilerConfig.HookFactory = hookFn
		}
		// not found — fine, GenericReconciler runs without hooks
	}
	return nil
}
