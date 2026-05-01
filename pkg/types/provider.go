// pkg/types/provider.go
//
// Provider interface — the extension point for external infrastructure.
//
// A provider handles a named YAML block in onCreate/onReconcile/onDelete.
// Orkestra dispatches the block to the registered provider after all
// Kubernetes resource groups have been reconciled.
//
// Registration (in cmd/operator/main.go or wherever komponents are wired):
//
//	registry := orktypes.NewProviderRegistry()
//	registry.Register(awsprovider.New(sess))
//	registry.Register(mongoprovider.New(uri))
//
// Katalog usage:
//
//	onCreate:
//	  aws:
//	    - rds:
//	        instanceClass: db.t3.micro
//	        engine: postgres
//	  database:
//	    - driver: mongo
//	      uri: "$MONGO_ATLAS_URI"
//	      createUser: "{{ .spec.dbUser }}"
//
// Orkestra calls provider.Reconcile for each block whose name matches a
// registered provider. Unregistered blocks are skipped with a warning.
// Provider errors fail the reconcile cycle and trigger backoff.
package types

import (
	"context"
	"fmt"
	"sync"

	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/rs/zerolog"
)

// ─────────────────────────────────────────────────────────────────────────────
// Core interface
// ─────────────────────────────────────────────────────────────────────────────

// Provider handles one named block in the Katalog's provider sections.
//
// Implement this interface to extend Orkestra with external resource management.
// Register your implementation at startup via ProviderRegistry.Register.
//
// Both Reconcile and Delete must be idempotent. They are called on every
// reconcile cycle and every finalizer execution respectively.
type Provider interface {
	// Name returns the YAML block key this provider handles.
	// Must be unique across all registered providers.
	// Convention: lowercase, no hyphens (e.g. "aws", "database", "stripe").
	Name() string

	// Reconcile is called after all Kubernetes resources are reconciled.
	// It should bring external resources into alignment with the declared state.
	// Must be idempotent — called on every resync, not just on first creation.
	//
	// Return nil when:
	//   - External resource already matches desired state (no-op)
	//   - External resource was successfully created or updated
	//   - Resource is still provisioning (check again next cycle)
	//   - Declaration kind is unknown (log a warning, skip, return nil)
	//
	// Return non-nil error when:
	//   - External API is unreachable
	//   - Credentials are invalid or expired
	//   - A hard quota is exceeded
	//   - Any condition that will not resolve on the next retry without intervention
	Reconcile(ctx context.Context, req ReconcileRequest) error

	// Delete is called during finalizer execution before the CR is removed.
	// It should delete all external resources created for this CR.
	// Must be idempotent — if the resource does not exist, return nil.
	//
	// The finalizer is NOT removed until Delete returns nil.
	// Returning an error leaves the CR stuck in deletion — ensure
	// transient errors are retried and permanent errors are surfaced clearly.
	Delete(ctx context.Context, req DeleteRequest) error
}

// ─────────────────────────────────────────────────────────────────────────────
// Request types
// ─────────────────────────────────────────────────────────────────────────────

// ReconcileRequest carries everything a provider needs for one reconcile cycle.
type ReconcileRequest struct {
	// Object is the full CR as an unstructured map.
	// Identical to the data available in Katalog template expressions.
	// Access patterns:
	//   spec:     obj["spec"].(map[string]interface{})["fieldName"]
	//   status:   obj["status"].(map[string]interface{})["phase"]
	//   children: obj["children"].(map[string]interface{})["deployment"]
	Object map[string]interface{}

	// Declarations are the pre-resolved provider block declarations.
	// Template expressions have already been evaluated against the CR.
	// Conditions have already been evaluated — only passing declarations arrive.
	// Each entry corresponds to one item in the YAML list.
	Declarations []ProviderDeclaration

	// Kube provides read-only access to cluster resources.
	// Use to read Secrets for credentials and ConfigMaps for configuration.
	Kube KubeReader

	// Logger is a structured logger pre-tagged with crd, resource, request_id.
	Logger zerolog.Logger

	// OwnerName is the CR's metadata.name.
	OwnerName string

	// OwnerNamespace is the CR's metadata.namespace.
	OwnerNamespace string
}

// DeleteRequest carries everything a provider needs for cleanup.
// Identical shape to ReconcileRequest — separate type for clarity at call sites.
type DeleteRequest struct {
	Object         map[string]interface{}
	Declarations   []ProviderDeclaration
	Kube           KubeReader
	Logger         zerolog.Logger
	OwnerName      string
	OwnerNamespace string
}

// ProviderDeclaration is one resolved declaration from the YAML block.
//
// For this Katalog block:
//
//	aws:
//	  - rds:
//	      instanceClass: "{{ .spec.instanceClass }}"
//	      engine: postgres
//	  - s3:
//	      bucket: "my-bucket"
//
// Orkestra produces two ProviderDeclarations:
//
//	{Kind: "rds", Fields: {"instanceClass": "db.t3.micro", "engine": "postgres"}}
//	{Kind: "s3",  Fields: {"bucket": "my-bucket"}}
//
// Template expressions have been resolved — Fields values are plain strings.
// Nested YAML maps are flattened with dot notation:
//
//	credentials:
//	  secretName: my-secret
//
// becomes Fields["credentials.secretName"] = "my-secret"
type ProviderDeclaration struct {
	// Kind is the first key in the YAML map entry: "rds", "s3", "product", "widget".
	Kind string

	// Fields are the resolved key-value pairs. Values are always strings —
	// template resolution produces strings; type conversion is the provider's
	// responsibility.
	Fields map[string]string
}

// Field returns a field value with a default if the key is absent.
func (d ProviderDeclaration) Field(key, defaultValue string) string {
	if v, ok := d.Fields[key]; ok && v != "" {
		return v
	}
	return defaultValue
}

// Require returns a field value or an error if the key is absent or empty.
// Use for required fields that must be present for the provider to proceed.
func (d ProviderDeclaration) Require(key string) (string, error) {
	v, ok := d.Fields[key]
	if !ok || v == "" {
		return "", fmt.Errorf("declaration %q: required field %q is missing or empty", d.Kind, key)
	}
	return v, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// KubeReader — narrow read interface for providers
// ─────────────────────────────────────────────────────────────────────────────

// KubeReader provides read-only access to cluster resources.
// Providers must not write Kubernetes resources — Orkestra owns cluster state.
type KubeReader interface {
	// GetSecret reads a Secret's data by name in the given namespace.
	// Returns the decoded data map (base64 already decoded).
	GetSecret(ctx context.Context, namespace, name string) (map[string][]byte, error)

	// GetConfigMap reads a ConfigMap's data by name in the given namespace.
	GetConfigMap(ctx context.Context, namespace, name string) (map[string]string, error)
}

// ─────────────────────────────────────────────────────────────────────────────
// ProviderRegistry
// ─────────────────────────────────────────────────────────────────────────────

// ProviderRegistry holds all registered providers.
// Thread-safe — providers are registered at startup and read concurrently
// during reconcile.
type ProviderRegistry interface {
	// Register adds a provider. Panics if a provider with the same name is
	// already registered — fail fast at startup rather than silently ignoring.
	Register(p Provider)

	// Get returns the provider for the given block name and true, or nil and
	// false if no provider is registered for that name.
	Get(name string) (Provider, bool)

	// Names returns all registered provider names.
	// Used by the validator to warn on unknown blocks in the Katalog.
	Names() []string

	// Len returns the number of registered providers.
	Len() int
}

// ─────────────────────────────────────────────────────────────────────────────
// Default implementation
// ─────────────────────────────────────────────────────────────────────────────

// providerRegistry is the default in-memory ProviderRegistry.
type providerRegistry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewProviderRegistry returns an empty ProviderRegistry.
func NewProviderRegistry() ProviderRegistry {
	return &providerRegistry{
		providers: make(map[string]Provider),
	}
}

// NoOpProviderRegistry returns a registry with no providers registered.
// Used in tests and in reconcilers that do not use provider blocks.
func NoOpProviderRegistry() ProviderRegistry {
	return NewProviderRegistry()
}

func (r *providerRegistry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[p.Name()]; exists {
		logger.Warn().
			Str("provider", p.Name()).
			Msg("duplicate registration skipping...")
	}
	r.providers[p.Name()] = p
}

func (r *providerRegistry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

func (r *providerRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

func (r *providerRegistry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers)
}
