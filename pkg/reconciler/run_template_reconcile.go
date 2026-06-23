// pkg/reconciler/run_template_reconcile.go
//
// Execution order (each step can reference all previous steps):
//
//  1. Recieve NewResolver        → .spec.*, .status.*, .metadata.*
//  2. r.readCross     			  → .cross.<crd>.status.* (informer cache, zero API calls)
//  3. runExternal        	      → .external.<n>.status, .body (HTTP calls)
//  4. forEach expand             → N sources from N-element list fields
//  5. onCreate groups            → deployments, services, secrets, configmaps, ...
//  6. onReconcile groups
//  7. runProviders               → aws:, mongodb:, ... (external infra)
package reconciler

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/children"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orklabels "github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// runTemplateReconcile interprets the Katalog's onCreate and onReconcile blocks.
// Returns the enriched resolver so callers (reconcileImpl) can pass cross/external
// data into patchStatusWithChildren for status field evaluation.
func (r *GenericReconciler[PTR]) runTemplateReconcile(ctx context.Context, resolver *orktmpl.Resolver, obj domain.Object) (*orktmpl.Resolver, error) {
	kube, ok := kubeclient.FromContext(ctx)
	if !ok {
		return resolver, fmt.Errorf("kubeclient not found in context")
	}

	// Step 1: We now receive a base resolver (already normalized) from reconcileImpl.
	// All subsequent steps (cross, git, external, docker, resources, providers)
	// enrich this resolver in-place.
	var err error

	// Step 2: cross-CRD observation
	// Reads from sibling CRD informer caches via r.katalogRegistry — zero API calls.
	// Must run first so git, docker, external calls, and resources can reference .cross.*
	if len(r.operatorBox.Cross) > 0 {
		crossData := r.readCross(ctx, obj, r.operatorBox.Cross, resolver)
		logger.FromContext(ctx).Debug().
			Str("observer", obj.GetName()).
			Int("cross_entries", len(crossData)).
			Interface("cross_keys", crossDataKeys(crossData)).
			Msg("cross: resolver enrichment")
		if len(crossData) > 0 {
			resolver = resolver.WithCross(crossData)
		}
	}

	// Step 3: Git hook
	// Runs before external calls so URLs, tokens, and payloads can reference .git.commit,
	// .git.changed, and .git.path. Git is a declarative precondition for pipelines.
	if t := r.operatorBox.OnReconcile; t != nil && t.Git != nil {
		resolver, err = runGit(ctx, r.crd.GVKString(), resolver, kube, obj, r.crd.GVR(), t.Git)
		if err != nil {
			return resolver, fmt.Errorf("git hook: %w", err)
		}
	}
	if t := r.operatorBox.OnCreate; t != nil && t.Git != nil {
		resolver, err = runGit(ctx, r.crd.GVKString(), resolver, kube, obj, r.crd.GVR(), t.Git)
		if err != nil {
			return resolver, fmt.Errorf("git hook: %w", err)
		}
	}

	// Step 4: external HTTP calls
	// Runs after Git so external URLs can embed commit hashes or paths.
	if t := r.operatorBox.OnReconcile; t != nil && len(t.External) > 0 {
		resolver, err = runExternal(ctx, r.crd.GVKString(), resolver, t.External)
		if err != nil {
			return resolver, fmt.Errorf("external calls: %w", err)
		}
	}
	if t := r.operatorBox.OnCreate; t != nil && len(t.External) > 0 {
		resolver, err = runExternal(ctx, r.crd.GVKString(), resolver, t.External)
		if err != nil {
			return resolver, fmt.Errorf("external calls: %w", err)
		}
	}

	// Step 5: Docker hook
	// Runs after external so build/push can use tokens or metadata from external calls.
	if t := r.operatorBox.OnReconcile; t != nil && t.Docker != nil {
		resolver, err = runDocker(ctx, r.crd.GVKString(), resolver, t.Docker)
		if err != nil {
			return resolver, fmt.Errorf("docker hook: %w", err)
		}
	}
	if t := r.operatorBox.OnCreate; t != nil && t.Docker != nil {
		resolver, err = runDocker(ctx, r.crd.GVKString(), resolver, t.Docker)
		if err != nil {
			return resolver, fmt.Errorf("docker hook: %w", err)
		}
	}

	// Step 6: onCreate resource groups (update=false)
	if t := r.operatorBox.OnCreate; t != nil {
		if err := r.runResourceGroup(ctx, kube, resolver, obj, t, false); err != nil {
			return resolver, err
		}
	}

	// Step 7: onReconcile resource groups (update=true)
	if t := r.operatorBox.OnReconcile; t != nil {
		if err := r.runResourceGroup(ctx, kube, resolver, obj, t, true); err != nil {
			return resolver, err
		}
	}

	// Step 8: provider dispatch
	if len(r.operatorBox.ProviderBlocks) > 0 && r.providerRegistry != nil && r.providerRegistry.Len() > 0 {
		kubeReader := &kubeReaderAdapter{kube: kube}
		if err := runProviders(ctx, obj, resolver, r.operatorBox.ProviderBlocks, r.providerRegistry, kubeReader, r.providerStats); err != nil {
			return resolver, fmt.Errorf("providers: %w", err)
		}
	}

	return resolver, nil
}

// runResourceGroup dispatches all resource types in one HookTemplates block.
// forEach expansion happens here — run_*.go receives already-expanded slices.
func (r *GenericReconciler[PTR]) runResourceGroup(
	ctx context.Context,
	kube kubeclient.KubeClient,
	resolver *orktmpl.Resolver,
	obj domain.Object,
	t *orktypes.HookTemplates,
	update bool,
) error {
	// Guard closure — captures r for access to CRD config.
	// nil-safe: if CRD has no restrictions, guard is a no-op.
	guard := r.namespaceGuardFunc(ctx, obj)

	labelMgr := orklabels.NewManager(orklabels.Config{
		Standalone:                r.kat.IsStandaloneGateway(),
		DeletionProtectionEnabled: r.kat.IsDeletionProtectionEnabled(),
	})

	// Create namespaces first
	if err := runNamespaces(ctx, kube, resolver, obj,
		children.ExpandForEachNamespaces(resolver, t.Namespaces), update); err != nil {
		return err
	}

	if err := runSecrets(ctx, kube, resolver, obj,
		children.ExpandForEachSecrets(resolver, t.Secrets), update, guard); err != nil {
		return err
	}
	if err := runConfigMaps(ctx, kube, resolver, obj,
		children.ExpandForEachConfigMaps(resolver, t.ConfigMaps), update, guard); err != nil {
		return err
	}
	if err := runServiceAccounts(ctx, kube, resolver, obj,
		children.ExpandForEachServiceAccounts(resolver, t.ServiceAccounts), update, guard); err != nil {
		return err
	}
	if err := runRoles(ctx, kube, resolver, obj,
		children.ExpandForEachRoles(resolver, t.Roles), update, guard); err != nil {
		return err
	}
	if err := runRoleBindings(ctx, kube, resolver, obj,
		children.ExpandForEachRoleBindings(resolver, t.RoleBindings), update, guard); err != nil {
		return err
	}
	if err := runCustomResources(ctx, kube, resolver, obj,
		children.ExpandForEachCustomResources(resolver, t.CustomResource), update, guard, labelMgr,
		r.kat.IsDeletionProtectionEnabled() && r.crd.ShouldProtectCRs()); err != nil {
		return err
	}
	if err := runReplicaSets(ctx, kube, resolver, obj,
		children.ExpandForEachReplicaSets(resolver, t.ReplicaSets), update, guard); err != nil {
		return err
	}
	if err := runDeployments(ctx, kube, resolver, obj,
		children.ExpandForEachDeployments(resolver, t.Deployments), update, guard); err != nil {
		return err
	}
	if err := runServices(ctx, kube, resolver, obj,
		children.ExpandForEachServices(resolver, t.Services), update, guard); err != nil {
		return err
	}
	if err := runJobs(ctx, kube, resolver, obj,
		children.ExpandForEachJobs(resolver, t.Jobs), guard); err != nil {
		return err
	}
	if err := runCronJobs(ctx, kube, resolver, obj,
		children.ExpandForEachCronJobs(resolver, t.CronJobs), update, guard); err != nil {
		return err
	}
	if err := runStatefulSets(ctx, kube, resolver, obj,
		children.ExpandForEachStatefulSets(resolver, t.StatefulSets), update, guard); err != nil {
		return err
	}
	if err := runPVs(ctx, kube, resolver, obj,
		children.ExpandForEachPVs(resolver, t.PersistentVolumes), update); err != nil {
		return err
	}
	if err := runPVCs(ctx, kube, resolver, obj,
		children.ExpandForEachPVCs(resolver, t.PersistentVolumeClaims), update, guard); err != nil {
		return err
	}
	if err := runIngresses(ctx, kube, resolver, obj,
		children.ExpandForEachIngresses(resolver, t.Ingresses), update, guard); err != nil {
		return err
	}
	if err := runHPAs(ctx, kube, resolver, obj,
		children.ExpandForEachHPAs(resolver, t.HorizontalPodAutoscalers), update, guard); err != nil {
		return err
	}
	if err := runPDBs(ctx, kube, resolver, obj,
		children.ExpandForEachPDBs(resolver, t.PodDisruptionBudgets), update, guard); err != nil {
		return err
	}
	if err := runPods(ctx, kube, resolver, obj,
		children.ExpandForEachPods(resolver, t.Pods), update, guard); err != nil {
		return err
	}
	return nil
}

// runTemplateOnDelete interprets the onDelete block.
func (r *GenericReconciler[PTR]) runTemplateOnDelete(ctx context.Context, resolver *orktmpl.Resolver, obj domain.Object) error {
	kube, ok := kubeclient.FromContext(ctx)
	if !ok {
		return fmt.Errorf("kubeclient not found in context")
	}

	guard := r.namespaceGuardFunc(ctx, obj)

	if t := r.operatorBox.OnDelete; t != nil {
		if t.Ordered {
			if err := r.runOrderedDelete(ctx, kube, resolver, obj, t, guard); err != nil {
				return err
			}
		} else {
			if err := runJobs(ctx, kube, resolver, obj,
				children.ExpandForEachJobs(resolver, t.Jobs), guard); err != nil {
				return err
			}
		}
	}

	if len(r.operatorBox.ProviderBlocks) > 0 && r.providerRegistry != nil {
		kubeReader := &kubeReaderAdapter{kube: kube}
		if err := runProviderDelete(ctx, obj, resolver, r.operatorBox.ProviderBlocks, r.providerRegistry, kubeReader, r.providerStats); err != nil {
			return fmt.Errorf("provider cleanup: %w", err)
		}
	}

	// Namespaces are cluster-scoped and cannot have namespace-scoped owners, so GC
	// never cleans them up. Always run explicit cleanup regardless of ordered/unordered path.
	if err := deleteOwnedNamespaces(ctx, kube, resolver, obj, r.operatorBox); err != nil {
		return fmt.Errorf("namespace cleanup: %w", err)
	}

	return nil
}

func crossDataKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// readCross reads cross-CRD observations for all declared cross: entries.
// Returns the map injected via resolver.WithCross().
//
// Resolution priority per declaration:
//  1. Informer cache via r.katalogRegistry — zero API calls, same-binary CRDs
//  2. HTTP endpoint (decl.Source.Endpoint) — cross-binary, cross-cluster
//  3. Empty not-found map — when neither path is available
func (r *GenericReconciler[PTR]) readCross(
	ctx context.Context,
	obj domain.Object,
	decls []orktypes.CrossCRDDeclaration,
	resolver *orktmpl.Resolver,
) map[string]interface{} {
	if len(decls) == 0 {
		return nil
	}

	log := logger.FromContext(ctx)
	result := make(map[string]interface{}, len(decls))

	registryNil := r.katalogRegistry == nil

	for _, decl := range decls {
		as := decl.As
		if as == "" {
			as = decl.Crd
		}

		name, _ := resolver.Resolve(decl.Selector.Name)
		namespace, _ := resolver.Resolve(decl.Selector.Namespace)
		if namespace == "" {
			namespace = obj.GetNamespace()
		}
		key := crossKey(namespace, name)

		// Path 1: informer cache — zero API calls.
		// katalogRegistry is threaded in from konstructRuntime via NewGenericReconciler.
		// Path 1a: label-based informer lookup
		if len(decl.LabelSelector) > 0 && r.katalogRegistry != nil {
			for labelKey, labelValue := range decl.LabelSelector {
				inf, found := r.katalogRegistry.GetInformerByLabelSelector(labelKey, labelValue)
				if found {
					data := ReadCrossFromInformerByLabel(inf.GetIndexer(), labelKey, labelValue)
					result[as] = data
					break
				}
				log.Warn().
					Str("label_key", labelKey).
					Str("label_val", labelValue).
					Str("crd", decl.Crd).
					Str("as", as).
					Msg("cross: no CRD matched label selector in registry")
			}
		}

		notFoundInBianry := false
		// Path 1b: name-based informer lookup
		if decl.Crd != "" && r.katalogRegistry != nil {
			inf, found := r.katalogRegistry.GetInformerByName(decl.Crd)
			if found {
				name, _ := resolver.Resolve(decl.Selector.Name)
				ns, _ := resolver.Resolve(decl.Selector.Namespace)
				if ns == "" {
					ns = obj.GetNamespace()
				}
				crossAccess := r.katalogRegistry.GetCrossAccessByName(decl.Crd)
				data := ReadCrossFromInformer(inf.GetIndexer(), crossKey(ns, name), crossAccess)
				result[as] = data
				continue
			}
			notFoundInBianry = true
		}

		notFoundCrossBinary := false
		// For cross-binary or cross-cluster. Uses Orkestra's CR detail endpoint.
		// Path 2: HTTP fallback (raw endpoint OR ONCOP host-based URL inference)
		if decl.Source != nil {

			// 2a: Raw endpoint takes precedence (non-Orkestra operators)
			if decl.Source.Endpoint != "" {
				endpointURL, _ := resolver.Resolve(decl.Source.Endpoint)
				token := expandEnv(decl.Source.Token)

				data := fetchCrossViaHTTP(ctx, endpointURL, token)
				if data != nil {
					result[as] = data
					log.Debug().
						Str("crd", decl.Crd).
						Str("as", as).
						Str("endpoint", endpointURL).
						Msg("cross: read via raw HTTP endpoint")
					continue
				}

				notFoundCrossBinary = true
				log.Warn().
					Str("crd", decl.Crd).
					Str("endpoint", endpointURL).
					Msg("cross: raw HTTP endpoint returned nil")
			}

			// 2b: ONCOP host-based URL inference (Orkestra-native operators)
			if decl.Source.Host != "" {
				// Build ONCOP URL from host + type + crd + ns + name
				url := orktypes.BuildONCOPURL(decl)

				token := expandEnv(decl.Source.Token)
				data := fetchCrossViaHTTP(ctx, url, token)
				if data != nil {
					result[as] = data
					log.Debug().
						Str("crd", decl.Crd).
						Str("as", as).
						Str("endpoint", url).
						Msg("cross: read via ONCOP host")
					continue
				}

				notFoundCrossBinary = true
				log.Warn().
					Str("crd", decl.Crd).
					Str("endpoint", url).
					Msg("cross: ONCOP endpoint returned nil")
			}
		}

		if notFoundInBianry && notFoundCrossBinary {
			log.Warn().
				Str("crd", decl.Crd).
				Str("as", as).
				Bool("registry_nil", registryNil).
				Msg("cross: no CRD matched name in registry")
		}

		// Path 3: not found.
		result[as] = map[string]interface{}{
			"found":     "false",
			"name":      name,
			"namespace": namespace,
			"status":    map[string]interface{}{},
			"spec":      map[string]interface{}{},
		}
		log.Debug().
			Str("crd", decl.Crd).
			Str("as", as).
			Str("key", key).
			Msg("cross: not found — empty result")
	}

	return result
}
