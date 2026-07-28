package katalog

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/orkspace/orkestra/pkg/children"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/profiles"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// -----------------------------------------------------------------------------
// Validation: Pretty error reporting
// -----------------------------------------------------------------------------

func (k *Katalog) handleValidationErrors(err error) {
	logger.Error().Msg("Validation failed")

	// Case 1: struct field validation errors (validator.v10)
	if errs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range errs {
			fmt.Printf(
				"  • Field '%s' failed validation rule '%s'\n",
				e.Field(), e.Tag(),
			)
		}
		return
	}

	// Case 2: custom validation errors (like duration parsing)
	fmt.Printf("  • %s\n", err.Error())
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

		// Set require fields
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
			if crd.CRDFile != "" {
				return fmt.Errorf("CRD '%s': could not determine group/version/kind from crdFile %q", name, crd.CRDFile)
			}
			return fmt.Errorf("CRD '%s': missing required fields: apiTypes.group, apiTypes.version, apiTypes.kind (or declare crdFile: to read these from the CRD YAML)", name)
		}

		if crd.GroupVersion.Empty() {
			return fmt.Errorf("CRD '%s': missing required fields: apiTypes.group, apiTypes.version", name)
		}

		if crd.GroupVersionResource.Empty() {
			return fmt.Errorf("CRD '%s': missing required fields: apiTypes.plural", name)
		}

		// Handle description
		// Here, GroupVersionKind is not empty
		if crd.Description == "" {
			crd.Description = fmt.Sprintf("%s CRD, GVK: %s", crd.APITypes.Kind, crd.GroupVersionKind.String())
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
			crd.KatalogName = k.metadata.ClusterName + "-" + name
		}

		// Propagate katalog namespace — merger stamps it, but ensure the
		// default is applied for any path that bypasses the merger (e.g. Go-mode).
		if crd.KatalogNamespace == "" {
			if k.metadata.Namespace != "" {
				crd.KatalogNamespace = k.metadata.Namespace
			} else {
				crd.KatalogNamespace = "default"
			}
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

		// Handle finalizers
		if len(crd.OperatorBox.Finalizers) == 0 {
			crd.OperatorBox.Finalizers = k.Spec.Finalizers
		}

		// Ensure operatorBox.reconciler is always initialised so runtime callsites
		// can read from it directly without nil-checking.
		if crd.OperatorBox.Reconciler == nil {
			crd.OperatorBox.Reconciler = &orktypes.ReconcilerConfig{}
		}
		rec := crd.OperatorBox.Reconciler

		// Expand a named profile — inline fields win over profile values.
		if rec.Profile != "" {
			result, err := profiles.ApplyReconcilerProfile(rec.Profile, k.Profiles)
			if err != nil {
				return fmt.Errorf("CRD %q: %w", name, err)
			}
			if rec.Workers == 0 {
				rec.Workers = result.Workers
			}
			if rec.Resync.Duration == 0 {
				rec.Resync.Duration = result.Resync
			}
			if rec.Queue.MaxDepth == 0 {
				rec.Queue.MaxDepth = result.MaxDepth
			}
		}

		// Apply global defaults for any field still at zero.
		if rec.Workers == 0 {
			rec.Workers = kfg.Katalog().DefaultWorkers()
		}
		if rec.Resync.Duration == 0 {
			rec.Resync.Duration = kfg.Katalog().DefaultResync()
		}
		if rec.Queue.MaxDepth == 0 {
			rec.Queue.MaxDepth = kfg.Katalog().DefaultQueueDepth()
		}
		if rec.Queue.FailureThreshold == 0 {
			rec.Queue.FailureThreshold = kfg.Katalog().DefaultFailureThreshold()
		}
		crd.OperatorBox.Reconciler = rec

		// Handle Notifications
		if k.IsEmailNotificationEnabled() || k.IsSlackNotificationEnabled() {
			enabled := true
			crd.NotificationEnabled = &enabled
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
			err := fmt.Errorf("addRuntimeObjects: no object constructor registered for %s", gvk)
			if crd.RegistryRef != "" {
				return &TypedOperatorError{Ref: crd.RegistryRef, Err: err}
			}
			return err
		}
		listFn, ok := orktypes.ListRegistry[gvk]
		if !ok {
			err := fmt.Errorf("addRuntimeObjects: no list constructor registered for %s", gvk)
			if crd.RegistryRef != "" {
				return &TypedOperatorError{Ref: crd.RegistryRef, Err: err}
			}
			return err
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
						"check reconciler.constructor in Katalog and re-run ork generate registry",
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
		// children.BuiltInMeta returns zero value for unknown kinds (safe).
		meta := children.BuiltInMeta(crd.APITypes.Kind)

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
	for _, crd := range k.enabledCRDs {
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

	}
	return nil
}

// validateNamespaceProtection enforces the namespace‑rule invariants for all enabled CRDs.
//
// A CRD may define *either* allowedNamespaces (whitelist) *or* restrictedNamespaces (blacklist),
// but never both. Allowing both simultaneously creates an ambiguous and contradictory policy:
//   - allowedNamespaces = “only these namespaces are permitted”
//   - restrictedNamespaces = “these namespaces are forbidden”
//
// Combining them would require precedence rules and conflict resolution, which Orkestra
// intentionally avoids to keep namespace protection deterministic and easy to reason about.
//
// Valid states:
//  1. No namespace rules → all namespaces allowed
//  2. Only restrictedNamespaces → blacklist mode
//  3. Only allowedNamespaces → whitelist mode
//  4. Both defined → invalid (this function returns an error)
//
// This validator ensures each CRD selects exactly one model.
func (k *Katalog) validateNamespaceProtection() error {
	for name, crd := range k.enabledCRDs {
		if !crd.IsNamespaced() || !crd.HasNamespaceRules() {
			continue // valid
		}
		if crd.AllowedNamespacesOnly() || crd.RestrictedNamespacesOnly() {
			continue // valid
		}

		// Both restricted and allowed are defined → invalid
		if crd.HasAllowedNamespaces() && crd.HasRestrictedNamespaces() {
			return fmt.Errorf(
				"CRD %q cannot define both allowedNamespaces and restrictedNamespaces — choose one",
				name, // invalid
			)
		}
	}
	return nil
}

// validateTimeDuration validates all rotation-related duration fields
// across all enabled CRDs. It is fail-fast: the first invalid duration
// returns an error immediately.
//
// Supported units (extended by ParseTimeDuration):
//
//	d   = days (24h)
//	w   = weeks (7d)
//	mo  = months (30d)
//	y   = years (365d)
func (k *Katalog) validateTimeDuration() error {
	for name, crd := range k.enabledCRDs {
		if !crd.HasAnyHookTemplates() {
			continue
		}

		// Validate sleep durations across all resource types.
		// Skip template expressions — those are resolved at runtime.
		for _, e := range crd.CollectSleepEntries() {
			if orktypes.IsTemplate(e.Duration) {
				continue
			}
			if _, err := orktypes.ParseTimeDuration(e.Duration); err != nil {
				return durationError(name, e.ResourceName, "sleep", e.Duration, err)
			}
		}

		// Validate secret durations (rotateAfter, TLS.validFor)
		if !crd.HasAnySecrets() {
			continue
		}
		if crd.HasOnCreate() {
			for _, s := range crd.OperatorBox.OnCreate.Secrets {
				if s.RotateAfter != "" {
					if _, err := orktypes.ParseTimeDuration(s.RotateAfter); err != nil {
						return durationError(name, s.Name, "rotateAfter", s.RotateAfter, err)
					}
				}
				// Check per-secret TLS presence
				if s.TLS != nil && s.TLS.ValidFor != "" {
					if _, err := orktypes.ParseTimeDuration(s.TLS.ValidFor); err != nil {
						return durationError(name, s.Name, "validFor", s.TLS.ValidFor, err)
					}
				}
			}
		}

		if crd.HasOnReconcile() {
			for _, s := range crd.OperatorBox.OnReconcile.Secrets {
				if s.RotateAfter != "" {
					if _, err := orktypes.ParseTimeDuration(s.RotateAfter); err != nil {
						return durationError(name, s.Name, "rotateAfter", s.RotateAfter, err)
					}
				}
				// Check per-secret TLS presence
				if s.TLS != nil && s.TLS.ValidFor != "" {
					if _, err := orktypes.ParseTimeDuration(s.TLS.ValidFor); err != nil {
						return durationError(name, s.Name, "validFor", s.TLS.ValidFor, err)
					}
				}
			}
		}
	}
	return nil
}

// Helpers
func durationError(crdName, secretName, field, value string, err error) error {
	return fmt.Errorf(
		"invalid duration %q in CRD %q (secret %q, field %q): %v\n\n"+
			"Allowed units:\n"+
			"  d   = days (24h)\n"+
			"  w   = weeks (7d)\n"+
			"  mo  = months (30d)\n"+
			"  y   = years (365d)\n\n"+
			"Examples: 30d, 2w, 3mo, 1y",
		value, crdName, secretName, field, err,
	)
}

// validateHPAReference ensures that every HPA declaration has a valid ScaleTargetRef.
// Fail-fast: the first invalid reference returns an error immediately.
func (k *Katalog) validateHPAReference() error {
	for crdName, crd := range k.enabledCRDs {
		if !crd.HasAnyHookTemplates() || !crd.HasAnyHPA() {
			continue
		}

		// onCreate
		if crd.HasOnCreate() {
			for _, h := range crd.OperatorBox.OnCreate.HorizontalPodAutoscalers {
				if err := validateOneHPARef(crdName, h.Name, h.ScaleTargetRef); err != nil {
					return err
				}
			}
		}

		// onReconcile
		if crd.HasOnReconcile() {
			for _, h := range crd.OperatorBox.OnReconcile.HorizontalPodAutoscalers {
				if err := validateOneHPARef(crdName, h.Name, h.ScaleTargetRef); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func validateOneHPARef(crdName, hpaName string, ref orktypes.ScaleTargetRef) error {
	// Default apiVersion for the three core scalable workload kinds.
	appsV1Kinds := map[string]bool{"Deployment": true, "StatefulSet": true, "ReplicaSet": true}
	if ref.APIVersion == "" && appsV1Kinds[ref.Kind] {
		ref.APIVersion = "apps/v1"
	}
	if ref.APIVersion == "" {
		return fmt.Errorf(
			"invalid HPA ScaleTargetRef in CRD %q (hpa %q): missing apiVersion\n\n"+
				"Example:\n"+
				"  ScaleTargetRef:\n"+
				"    apiVersion: apps/v1\n"+
				"    kind: Deployment\n"+
				"    name: my-app",
			crdName, hpaName,
		)
	}

	if ref.Kind == "" {
		return fmt.Errorf(
			"invalid HPA ScaleTargetRef in CRD %q (hpa %q): missing kind\n\n"+
				"Example:\n"+
				"  ScaleTargetRef:\n"+
				"    apiVersion: apps/v1\n"+
				"    kind: ReplicaSet\n"+
				"    name: my-app",
			crdName, hpaName,
		)
	}

	if ref.Name == "" {
		return fmt.Errorf(
			"invalid HPA ScaleTargetRef in CRD %q (hpa %q): missing name\n\n"+
				"Example:\n"+
				"  ScaleTargetRef:\n"+
				"    apiVersion: apps/v1\n"+
				"    kind: StatefulSet\n"+
				"    name: my-app",
			crdName, hpaName,
		)
	}

	return nil
}

// validateStatusTypes ensures that all declarative status fields declare a valid
// type. StatusFieldSpec.Type controls how the resolved template value is cast
// before being written into the CR's /status subresource.
//
// Supported type names (case‑insensitive):
//   - "string", "str", ""      → string (default)
//   - "bool", "boolean"        → boolean
//   - "int", "integer"         → integer
//
// Any other value is rejected at katalog‑load time. This prevents silent
// mis‑casts in the status resolver and ensures that typed CRD fields (such as
// those required by the Kubernetes /scale subresource) receive correctly‑typed
// values.
//
// This validator mirrors the style of validateHPAReference, validateTimeDuration,
// and other fail‑fast katalog validators: the first invalid type aborts loading
// with a clear, actionable error message.
func (k *Katalog) validateStatusTypes() error {
	for name, crd := range k.enabledCRDs {
		// Skip CRDs without declarative status
		if crd.OperatorBox.Status == nil {
			continue
		}

		if crd.OperatorBox.Status.HasFields() {
			for _, f := range crd.OperatorBox.Status.Fields {
				switch strings.ToLower(f.Type) {
				case "", "string", "str", "default":
				case "int", "integer":
				case "bool", "boolean":
				case "float", "auto":
					// valid
				default:
					return fmt.Errorf(
						"invalid status field type %q in CRD %q (path: %q):\n"+
							"  must be one of: string, str, int, integer, bool, boolean, float, auto\n",
						f.Type, name, f.Path,
					)
				}
			}
		}
	}

	return nil
}

// validateTeams ensures that a team referenced under a notify: block
// (in onCreate, onReconcile, or rollback) was actually declared in
// notification.teams within this Katalog.
//
// This is a static validation step used by ork validate and ork run
// (the same validator is invoked in both paths). It prevents typos,
// misconfigured team names, or references to teams that do not exist
// in the platform-level notification configuration.
//
// Behavior:
//   - If the katalog has no notification block → no-op (notifications disabled)
//   - If the katalog has no teams declared → no-op (nothing to validate against)
//   - If teamName is not found in notification.teams → return an error
//
// This keeps notify: ["teamA", "teamB"] aligned with the declared
// notification.teams map and ensures that runtime dispatch never
// attempts to send to an undefined team.
func (k *Katalog) validateTeams() error {
	if !k.HasNotification() {
		return nil
	}
	if !k.HasTeams() {
		return nil
	}

	for name := range k.enabledCRDs {
		if _, ok := k.Notification.Teams[name]; !ok {
			return fmt.Errorf("%s team not found", name)
		}
	}
	return nil
}
