// pkg/reconciler/run_template_reconcile.go
//
// Execution order (each step can reference all previous steps):
//
//  1. NewResolver        → .spec.*, .status.*, .metadata.*
//  2. r.readCross        → .cross.<kind>.status.* (informer cache, zero API calls)
//  3. runExternal        → .external.<n>.status, .body (HTTP calls)
//  4. forEach expand     → N sources from N-element list fields
//  5. onCreate groups    → deployments, services, secrets, configmaps, ...
//  6. onReconcile groups
//  7. runProviders       → aws:, mongodb:, ... (external infra)
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// runTemplateReconcile interprets the Katalog's onCreate and onReconcile blocks.
func (r *GenericReconciler[T]) runTemplateReconcile(ctx context.Context, obj domain.Object) error {
	kube, ok := kubeclient.FromContext(ctx)
	if !ok {
		return fmt.Errorf("kubeclient not found in context")
	}

	// Step 1: base resolver
	resolver, err := orktmpl.NewResolver(ctx, obj)
	if err != nil {
		return fmt.Errorf("building resolver: %w", err)
	}

	// Step 2: cross-CRD observation
	// Reads from sibling CRD informer caches via r.katalogRegistry — zero API calls.
	// Must run first so external calls and resources can reference .cross.*
	if len(r.rc.Cross) > 0 {
		crossData := r.readCross(ctx, obj, r.rc.Cross, resolver)
		if len(crossData) > 0 {
			resolver = resolver.WithCross(crossData)
		}
	}

	// Step 3: external HTTP calls
	if t := r.rc.OnReconcile; t != nil && len(t.External) > 0 {
		resolver, err = runExternal(ctx, resolver, t.External)
		if err != nil {
			return fmt.Errorf("external calls: %w", err)
		}
	}

	// Steps 4+5: onCreate resource groups (update=false)
	if t := r.rc.OnCreate; t != nil {
		if err := r.runResourceGroup(ctx, kube, resolver, obj, t, false); err != nil {
			return err
		}
	}

	// Step 6: onReconcile resource groups (update=true)
	if t := r.rc.OnReconcile; t != nil {
		if err := r.runResourceGroup(ctx, kube, resolver, obj, t, true); err != nil {
			return err
		}
	}

	// Step 7: provider dispatch
	if len(r.rc.ProviderBlocks) > 0 && r.providerRegistry != nil && r.providerRegistry.Len() > 0 {
		kubeReader := &kubeReaderAdapter{kube: kube}
		if err := runProviders(ctx, obj, resolver, r.rc.ProviderBlocks, r.providerRegistry, kubeReader); err != nil {
			return fmt.Errorf("providers: %w", err)
		}
	}

	return nil
}

// runResourceGroup dispatches all resource types in one HookTemplates block.
// forEach expansion happens here — run_*.go receives already-expanded slices.
func (r *GenericReconciler[T]) runResourceGroup(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	resolver *orktmpl.Resolver,
	obj domain.Object,
	t *orktypes.HookTemplates,
	update bool,
) error {
	if err := runDeployments(ctx, kube, resolver, obj,
		expandForEachDeployments(resolver, t.Deployments), update); err != nil {
		return err
	}
	if err := runServices(ctx, kube, resolver, obj,
		expandForEachServices(resolver, t.Services), update); err != nil {
		return err
	}
	if err := runSecrets(ctx, kube, resolver, obj,
		expandForEachSecrets(resolver, t.Secrets), update); err != nil {
		return err
	}
	if err := runConfigMaps(ctx, kube, resolver, obj,
		expandForEachConfigMaps(resolver, t.ConfigMaps), update); err != nil {
		return err
	}
	if err := runServiceAccounts(ctx, kube, resolver, obj,
		expandForEachServiceAccounts(resolver, t.ServiceAccounts), update); err != nil {
		return err
	}
	if err := runJobs(ctx, kube, resolver, obj,
		expandForEachJobs(resolver, t.Jobs)); err != nil {
		return err
	}
	if err := runCronJobs(ctx, kube, resolver, obj,
		expandForEachCronJobs(resolver, t.CronJobs), update); err != nil {
		return err
	}
	return nil
}

// runTemplateOnDelete interprets the onDelete block.
func (r *GenericReconciler[T]) runTemplateOnDelete(ctx context.Context, obj domain.Object) error {
	kube, ok := kubeclient.FromContext(ctx)
	if !ok {
		return fmt.Errorf("kubeclient not found in context")
	}

	resolver, err := orktmpl.NewResolver(ctx, obj)
	if err != nil {
		return fmt.Errorf("building resolver: %w", err)
	}

	if t := r.rc.OnDelete; t != nil {
		if t.Ordered {
			return r.runOrderedDelete(ctx, kube, resolver, obj, t)
		}
		if err := runJobs(ctx, kube, resolver, obj,
			expandForEachJobs(resolver, t.Jobs)); err != nil {
			return err
		}
	}

	if len(r.rc.ProviderBlocks) > 0 && r.providerRegistry != nil {
		kubeReader := &kubeReaderAdapter{kube: kube}
		if err := runProviderDelete(ctx, obj, resolver, r.rc.ProviderBlocks, r.providerRegistry, kubeReader); err != nil {
			return fmt.Errorf("provider cleanup: %w", err)
		}
	}

	return nil
}

// runOrderedDelete deletes resource groups sequentially with verification.
func (r *GenericReconciler[T]) runOrderedDelete(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	resolver *orktmpl.Resolver,
	obj domain.Object,
	t *orktypes.HookTemplates,
) error {
	log := logger.FromContext(ctx)
	log.Info().Str("name", obj.GetName()).Msg("ordered delete: starting sequential cleanup")

	if len(t.Jobs) > 0 {
		jobs := expandForEachJobs(resolver, t.Jobs)
		if err := runJobs(ctx, kube, resolver, obj, jobs); err != nil {
			return fmt.Errorf("ordered delete: cleanup jobs: %w", err)
		}
	}

	return nil
}

// readCross reads cross-CRD observations for all declared cross: entries.
// Returns the map injected via resolver.WithCross().
//
// Resolution priority per declaration:
//  1. Informer cache via r.katalogRegistry — zero API calls, same-binary CRDs
//  2. HTTP endpoint (decl.Source.Endpoint) — cross-binary, cross-cluster
//  3. Empty not-found map — when neither path is available
func (r *GenericReconciler[T]) readCross(
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

	for _, decl := range decls {
		as := decl.As
		if as == "" {
			as = decl.Kind
		}

		name, _ := resolver.Resolve(decl.Selector.Name)
		namespace, _ := resolver.Resolve(decl.Selector.Namespace)
		if namespace == "" {
			namespace = obj.GetNamespace()
		}
		key := crossKey(namespace, name)

		// Path 1: informer cache — zero API calls.
		// katalogRegistry is threaded in from konstructOrkestra via NewGenericReconciler.
		// GetInformerByName returns the live SharedIndexInformer for the target CRD.
		if r.katalogRegistry != nil {
			if inf, ok := r.katalogRegistry.GetInformerByName(decl.Kind); ok {
				data := ReadCrossFromInformer(inf.GetIndexer(), key)
				result[as] = data
				log.Debug().
					Str("kind", decl.Kind).
					Str("as", as).
					Str("key", key).
					Bool("found", data["found"] == "true").
					Msg("cross: read from informer cache")
				continue
			}
		}

		// Path 2: HTTP endpoint fallback.
		// For cross-binary or cross-cluster. Uses Orkestra's CR detail endpoint.
		if decl.Source != nil && decl.Source.Endpoint != "" {
			endpointURL, _ := resolver.Resolve(decl.Source.Endpoint)
			token := expandEnv(decl.Source.Token)
			data := fetchCrossViaHTTP(ctx, endpointURL, token)
			if data != nil {
				result[as] = data
				log.Debug().
					Str("kind", decl.Kind).
					Str("as", as).
					Str("endpoint", endpointURL).
					Msg("cross: read via HTTP endpoint")
				continue
			}
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
			Str("kind", decl.Kind).
			Str("as", as).
			Str("key", key).
			Msg("cross: not found in registry or HTTP source")
	}

	return result
}
