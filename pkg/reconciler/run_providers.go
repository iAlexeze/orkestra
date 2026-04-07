// pkg/reconciler/run_providers.go
//
// Provider dispatch — the engine that connects Katalog declarations to
// registered provider libraries.
//
// This file is called from runTemplateReconcile after all Kubernetes resource
// groups (deployments, services, jobs, etc.) have been reconciled. It:
//
//  1. Skips immediately if no provider blocks are declared or no providers registered
//  2. Resolves template expressions in every declaration field
//  3. Evaluates when: conditions — drops declarations that do not pass
//  4. Calls provider.Reconcile(ctx, req) for each active block
//
// On CR deletion (finalizer path), runProviderDelete is called instead.
// It calls provider.Delete for all declarations — no condition filtering.
//
// Integration in runTemplateReconcile (after all Kubernetes resource groups):
//
//	if err := runProviders(ctx, obj, resolver, rc.ProviderBlocks, providerRegistry, kubeReader); err != nil {
//	    return fmt.Errorf("providers: %w", err)
//	}
//
// Integration in the finalizer handler:
//
//	if err := runProviderDelete(ctx, obj, resolver, rc.ProviderBlocks, providerRegistry, kubeReader); err != nil {
//	    return fmt.Errorf("provider cleanup: %w", err)
//	}
package reconciler

import (
	"context"
	"fmt"
	"strings"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// runProviders dispatches provider blocks to registered provider libraries.
// Called after all Kubernetes resource reconciliation is complete.
// Blocks are dispatched in declaration order.
func runProviders(
	ctx context.Context,
	obj domain.Object,
	resolver *orktmpl.Resolver,
	blocks []orktypes.ProviderBlock,
	registry orktypes.ProviderRegistry,
	kube orktypes.KubeReader,
) error {
	if len(blocks) == 0 {
		return nil
	}
	if registry == nil || registry.Len() == 0 {
		logger.FromContext(ctx).Debug().
			Msg("provider blocks declared but no providers registered — all blocks skipped")
		return nil
	}

	log := logger.FromContext(ctx)

	for _, block := range blocks {
		provider, ok := registry.Get(block.Name)
		if !ok {
			log.Warn().
				Str("block", block.Name).
				Strs("registered", registry.Names()).
				Msg("provider block skipped — no provider registered with this name. " +
					"Register it in loadProviders() in internal/providers.go")
			continue
		}

		// Resolve template expressions in all declaration fields
		resolved, err := resolveProviderBlock(resolver, block)
		if err != nil {
			return fmt.Errorf("provider %q: resolving declarations: %w", block.Name, err)
		}

		// Evaluate when: conditions — drop declarations that do not pass.
		// resolver.Data() includes .spec.*, .status.*, .children.*
		active := filterProviderDeclarations(obj, resolved)
		if len(active) == 0 {
			log.Debug().
				Str("provider", block.Name).
				Msg("all declarations gated by when: conditions — provider not called")
			continue
		}

		req := orktypes.ReconcileRequest{
			Object:       resolver.Data(),
			Declarations: active,
			Kube:         kube,
			Logger: log.With().
				Str("provider", block.Name).
				Logger(),
			OwnerName:      obj.GetName(),
			OwnerNamespace: obj.GetNamespace(),
		}

		log.Debug().
			Str("provider", block.Name).
			Int("total", len(resolved)).
			Int("active", len(active)).
			Msg("calling provider.Reconcile")

		if err := provider.Reconcile(ctx, req); err != nil {
			return fmt.Errorf("provider %q: %w", block.Name, err)
		}
	}

	return nil
}

// runProviderDelete dispatches deletion to all registered providers.
// Called during finalizer execution. All declarations are passed to Delete
// regardless of when: conditions — deletion is attempted for everything
// that might have been created. All errors collected before returning.
func runProviderDelete(
	ctx context.Context,
	obj domain.Object,
	resolver *orktmpl.Resolver,
	blocks []orktypes.ProviderBlock,
	registry orktypes.ProviderRegistry,
	kube orktypes.KubeReader,
) error {
	if len(blocks) == 0 || registry == nil {
		return nil
	}

	log := logger.FromContext(ctx)
	var errs []string

	for _, block := range blocks {
		provider, ok := registry.Get(block.Name)
		if !ok {
			log.Warn().
				Str("block", block.Name).
				Msg("provider block skipped during deletion — provider not registered. " +
					"External resources for this block may require manual cleanup.")
			continue
		}

		// Resolve fields — conditions NOT evaluated on delete
		resolved, err := resolveProviderBlock(resolver, block)
		if err != nil {
			errs = append(errs, fmt.Sprintf("provider %q: resolve: %v", block.Name, err))
			continue
		}

		req := orktypes.DeleteRequest{
			Object:       resolver.Data(),
			Declarations: toProviderDeclarations(resolved),
			Kube:         kube,
			Logger: log.With().
				Str("provider", block.Name).
				Logger(),
			OwnerName:      obj.GetName(),
			OwnerNamespace: obj.GetNamespace(),
		}

		log.Debug().
			Str("provider", block.Name).
			Int("declarations", len(req.Declarations)).
			Msg("calling provider.Delete")

		if err := provider.Delete(ctx, req); err != nil {
			// Collect — do not return. Try all providers before surfacing errors.
			errs = append(errs, fmt.Sprintf("provider %q: %v", block.Name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("provider deletion errors:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

// resolvedDeclaration holds a declaration with template-resolved fields.
type resolvedDeclaration struct {
	Kind       string
	Fields     map[string]string
	Conditions []orktypes.Condition
}

// resolveProviderBlock resolves template expressions in all field values.
func resolveProviderBlock(
	resolver *orktmpl.Resolver,
	block orktypes.ProviderBlock,
) ([]resolvedDeclaration, error) {
	result := make([]resolvedDeclaration, 0, len(block.Declarations))

	for _, raw := range block.Declarations {
		resolved := resolvedDeclaration{
			Kind:       raw.Kind,
			Fields:     make(map[string]string, len(raw.Fields)),
			Conditions: raw.Conditions,
		}
		for key, tmplVal := range raw.Fields {
			val, err := resolver.Resolve(tmplVal)
			if err != nil {
				return nil, fmt.Errorf("declaration %q, field %q: %w", raw.Kind, key, err)
			}
			resolved.Fields[key] = val
		}
		result = append(result, resolved)
	}

	return result, nil
}

// filterProviderDeclarations removes declarations whose when: conditions fail.
// Uses EvaluateConditions from resolver_conditions.go (same package).
func filterProviderDeclarations(
	obj domain.Object,
	declarations []resolvedDeclaration,
) []orktypes.ProviderDeclaration {
	result := make([]orktypes.ProviderDeclaration, 0, len(declarations))
	for _, decl := range declarations {
		if len(decl.Conditions) > 0 && !evaluateConditions(obj, decl.Conditions) {
			continue
		}
		result = append(result, orktypes.ProviderDeclaration{
			Kind:   decl.Kind,
			Fields: decl.Fields,
		})
	}
	return result
}

// toProviderDeclarations converts resolved declarations to ProviderDeclaration
// without condition filtering — used for the Delete path.
func toProviderDeclarations(decls []resolvedDeclaration) []orktypes.ProviderDeclaration {
	result := make([]orktypes.ProviderDeclaration, len(decls))
	for i, d := range decls {
		result[i] = orktypes.ProviderDeclaration{Kind: d.Kind, Fields: d.Fields}
	}
	return result
}
