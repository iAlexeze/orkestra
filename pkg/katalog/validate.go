package katalog

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// -----------------------------------------------------------------------------
// Validation: Pretty error reporting
// -----------------------------------------------------------------------------

func (k *Katalog) handleValidationErrors(err error) {
	logger.Info().Msg("Validation error:")

	if errs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range errs {
			fmt.Printf("CRD field '%s' failed on '%s'\n", e.Field(), e.Tag())
		}
	} else {
		fmt.Println(err)
	}
}

// -----------------------------------------------------------------------------
// Validate uniqueness
// -----------------------------------------------------------------------------
func (k *Katalog) validateUniqueness() error {
	// Name uniqueness is guaranteed by map keys — only check GVK uniqueness.
	return k.validateGVKUniqueness()
}

// -----------------------------------------------------------------------------
// Validation: GVK uniqueness
// -----------------------------------------------------------------------------

func (k *Katalog) validateGVKUniqueness() error {
	seen := make(map[string]string) // key -> name

	for name, crd := range k.enabledCRDs {
		key := fmt.Sprintf("%s/%s/%s", crd.APITypes.Group, crd.APITypes.Version, crd.APITypes.Kind)

		if existing, ok := seen[key]; ok {
			return fmt.Errorf(
				"duplicate GVK detected: %s/%s, Kind=%s (CRDs: %s and %s)",
				crd.APITypes.Group, crd.APITypes.Version, crd.APITypes.Kind, existing, name,
			)
		}

		seen[key] = name
	}

	return nil
}

func (k *Katalog) validateReconcilerMode() error {
	for name, crd := range k.enabledCRDs {
		mode := crd.Mode

		switch mode {
		case "":
			logger.Debug().
				Str("crd", name).
				Msg("reconciler mode not set — defaulting to 'dynamic'")
			crd.Mode = orktypes.CRDModeDynamic

		case orktypes.CRDModeDynamic, orktypes.CRDModeTyped:
			// Valid — nothing to do

		default:
			return fmt.Errorf(
				"CRD %q: reconciler mode %q is not supported — use %q or %q",
				name, mode,
				orktypes.CRDModeDynamic,
				orktypes.CRDModeTyped,
			)
		}

		k.enabledCRDs[name] = crd
	}
	return nil
}

// -----------------------------------------------------------------------------
// Validation: dependsOn existence + cycle detection
// -----------------------------------------------------------------------------

func (k *Katalog) validateDependsOn() error {
	// Build lookup set from enabled CRD names (map keys)
	exists := make(map[string]bool, len(k.enabledCRDs))
	for name := range k.enabledCRDs {
		exists[name] = true
	}

	// Validate references
	for name, crd := range k.enabledCRDs {
		for dep := range crd.DependsOn {
			if dep == name {
				return fmt.Errorf("CRD '%s' cannot depend on itself", name)
			}
			if !exists[dep] {
				return fmt.Errorf("CRD '%s' depends on unknown or disabled CRD '%s'", name, dep)
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
	graph := make(map[string][]string, len(k.enabledCRDs))
	for name, crd := range k.enabledCRDs {
		graph[name] = crd.DependsOn.Names()
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
	for name, crd := range k.enabledCRDs {
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
			return fmt.Errorf("CRD '%s': missing required fields: apiTypes.group, apiTypes.version, apiTypes.kind", name)
		}

		if crd.GroupVersion.Empty() {
			return fmt.Errorf("CRD '%s': missing required fields: apiTypes.group, apiTypes.version", name)
		}

		if crd.APITypes.Kind == "" {
			return fmt.Errorf("CRD '%s': missing required field: apiTypes.kind", name)
		}

		if crd.GroupVersionResource.Empty() {
			return fmt.Errorf("CRD '%s': missing required fields: apiTypes.plural", name)
		}

		k.enabledCRDs[name] = crd
	}
	return nil
}

// ---------------------------------------------------------------------------------
//
// Set SetDefaults
func (k *Katalog) setDefaults(kfg *konfig.Konfig) error {
	for name, crd := range k.enabledCRDs {
		// Add katalog Name
		if k.metadata.Name != "" {
			crd.KatalogName = k.metadata.Name
		} else {
			crd.KatalogName = kfg.Cluster().Name
		}

		// Name is already set from map key — normalise it
		crd.Name = strings.ReplaceAll(crd.Name, " ", "")
		crd.Name = strings.ToLower(crd.Name)

		if crd.Name == "" {
			return fmt.Errorf("CRD with key '%s': empty name after normalisation", name)
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
		if len(crd.OperatorBox.Finalizers) == 0 {
			crd.OperatorBox.Finalizers = k.Spec.Finalizers
		}

		// Handle Resync
		if crd.Resync == 0 {
			crd.Resync = kfg.Cluster().DefaultResync
		}

		// Handle Workers
		if crd.Workers == 0 {
			crd.Workers = kfg.Cluster().DefaultWorkers
		}

		// Handle Notifications
		if k.IsEmailNotificationEnabled() || k.IsSlackNotificationEnabled() {
			crd.NotificationEnabled = true
		}

		k.enabledCRDs[name] = crd
	}
	return nil
}

// ---------------------------------------------------------------------------------
//
// Add RuntimeObjects
func (k *Katalog) addRuntimeObjects() error {
	for name, crd := range k.enabledCRDs {
		gvk := crd.GroupVersionKind

		if crd.IsDynamic() {
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
			k.enabledCRDs[name] = crd
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
		k.enabledCRDs[name] = crd
	}
	return nil
}

// ---------------------------------------------------------------------------------
// Add reconcilers
func (k *Katalog) addReconcilers() error {
	for name, crd := range k.enabledCRDs {
		rc := crd.OperatorBox

		// Add providers block
		if len(rc.ProviderBlocks) > 0 {
			blocks, err := orktypes.ParseProviderBlocks(rc.RawProviders)
			if err != nil {
				return err
			}
			rc.ProviderBlocks = blocks
		}

		if !crd.IsDynamic() {
			if crd.DefaultReconcile() {
				continue
			}

			constructorFn, ok := orktypes.ReconcilerRegistry[crd.GroupVersionKind]
			if !ok {
				return fmt.Errorf(
					"CRD %q: no constructor registered — "+
						"check reconciler.constructor in Katalog and re-run ork generate runtime",
					name,
				)
			}

			rc.Constructor = constructorFn
		}

		crd.OperatorBox = rc
		k.enabledCRDs[name] = crd
	}
	return nil
}

// ---------------------------------------------------------------------------------
// Add hooks
func (k *Katalog) addHooks() error {
	for name, crd := range k.enabledCRDs {
		if !crd.DefaultReconcile() {
			continue
		}
		if hookFn, ok := orktypes.HookRegistry[crd.GroupVersionKind]; ok {
			crd.OperatorBox.HookFactory = hookFn
			k.enabledCRDs[name] = crd
		}
		// not found — fine, GenericReconciler runs without hooks
	}
	return nil
}

// validateStatus sets IgnoreStatusPatch and IgnoreObservedGeneration on each
// enabled CRD entry based on the built-in resource registry.
//
// Called once during Katalog loading — flags are set once, checked cheaply
// in the hot reconcile path.
func (k *Katalog) validateStatus() {
	for name, crd := range k.enabledCRDs {
		// Look up by Kind directly — avoids GVK string format mismatch.
		// BuiltInMeta returns zero value for unknown kinds (safe).
		meta := BuiltInMeta(crd.APITypes.Kind)

		if meta.SkipStatusSubresource {
			// ConfigMap, Secret, ServiceAccount, Role, ClusterRole, etc.
			// These have no /status subresource — PATCH would return 404.
			crd.IgnoreStatusPatch = true
		}

		if meta.SkipObservedGeneration {
			// Namespace, Node, Service, Pod, PVC, etc.
			// These have status but no observedGeneration field.
			crd.IgnoreObservedGeneration = true
		}

		k.enabledCRDs[name] = crd
	}
}

// validateAutoscalerMetrics ensures only supported metrics.* fields are used.
// This is a fail-fast mechanism to avoid runtime errors.
func (k *Katalog) validateAutoscalerMetrics() error {
	for name, crd := range k.enabledCRDs {
		if !crd.AutoscaleEnabled() {
			continue
		}

		conds := crd.OperatorBox.Autoscale.Conditions

		// Validate anyOf
		for _, c := range conds.AnyOf {
			if strings.HasPrefix(c.Field, "metrics.") {
				if err := crd.ValidateMetricField(c.Field); err != nil {
					k.handleValidationErrors(err)
					return err
				}
			}
		}

		// Validate when
		for _, c := range conds.When {
			if strings.HasPrefix(c.Field, "metrics.") {
				if err := crd.ValidateMetricField(c.Field); err != nil {
					k.handleValidationErrors(err)
					return err
				}
			}
		}

		k.enabledCRDs[name] = crd
	}
	return nil
}
