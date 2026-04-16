---
title: "Provider Library"
weight: 72
---

# The Orkestra Provider Model

**Extending declarative operators to external infrastructure.**

---

## The problem with external resources

Kubernetes operators manage Kubernetes resources. The reconcile loop reads a
CR, creates Deployments and Services, writes status. This is the model
controller-runtime was designed for and the model Orkestra's declarative layer
covers completely.

Real operators manage more than Kubernetes resources. A `DatabaseCluster` CR
creates an RDS instance and a Route 53 record. A `MessageQueue` CR provisions
an SQS queue and attaches an IAM policy. A `MongoUser` CR creates a database
user in Atlas. These operations live outside the Kubernetes API. They require
credentials, SDKs, external state, and careful idempotency.

Before the provider model, every operator that needed external resources either:

1. Used a Go hook — correct but isolated. The hook is specific to one operator.
   It cannot be reused across katalogs. It cannot be published to the registry.
   It cannot be composed with other declarations.

2. Used Crossplane — correct but heavyweight. Crossplane is an infrastructure
   control plane. It requires its own operators, its own CRDs, its own
   composition machinery. It is the right answer for infrastructure teams. It
   is not the right answer for an application team that needs one S3 bucket
   alongside three Deployments.

The provider model is the third option: external resources declared in the
Katalog, executed by a registered provider library, composable with Kubernetes
resource declarations, publishable to the OrkestraRegistry.

---

## What a provider is

A provider is a Go library that handles a named block in the Katalog's
`onCreate`, `onReconcile`, or `onDelete` sections. AWS ships a provider that
handles `aws:` blocks. MongoDB ships a provider that handles `database:` blocks
with `driver: mongo`. Stripe ships a provider that handles `stripe:` blocks.

The provider receives:

- The resolved CR object — all spec fields, metadata, status accessible as
  `map[string]interface{}`
- The declarations from its YAML block — already template-resolved by Orkestra's
  resolver before the provider is called
- A context with cancellation, logger, and request ID
- The Kubernetes client — for reading Secrets, ConfigMaps, and other cluster
  resources the provider may need

The provider returns an error or nil. Orkestra handles retry, backoff, status
patching, and event emission. The provider handles exactly one thing: talking
to the external system.

---

## The Katalog syntax

```yaml
spec:
  crds:
    application-stack:
      apiTypes:
        group: platform.acme.io
        version: v1
        kind: ApplicationStack
        plural: applicationstacks

      reconciler:
        default: true

        onCreate:
          # ── Kubernetes resources (Orkestra core) ────────────────────────
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"

          services:
            - name: "{{ .metadata.name }}"
              port: "{{ .spec.port }}"

          # ── External resources (provider libraries) ─────────────────────
          aws:
            - s3:
                bucket: "{{ .metadata.name }}-{{ .metadata.namespace }}-assets"
                region: "{{ .spec.region }}"
                versioning: "{{ .spec.enableVersioning }}"
                when:
                  - field: spec.needsStorage
                    equals: "true"

            - rds:
                instanceClass: "{{ .spec.dbInstanceClass }}"
                engine: postgres
                version: "15"
                storage: "{{ .spec.dbStorageGb }}"
                multiAZ: "{{ .spec.production }}"
                credentials:
                  secretName: "{{ .metadata.name }}-db-credentials"
                when:
                  - field: spec.needsDatabase
                    equals: "true"

            - route53:
                zone: "{{ .spec.domain }}"
                record: "{{ .metadata.name }}.{{ .spec.domain }}"
                type: CNAME
                target: "{{ .children.service.status.loadBalancer.ingress }}"

          database:
            - driver: mongo
              uri: "$MONGO_ATLAS_URI"
              createUser: "{{ .spec.dbUser }}"
              database: "{{ .spec.dbName }}"
              roles: ["readWrite"]

          stripe:
            - product:
                name: "{{ .spec.productName }}"
                description: "{{ .spec.productDescription }}"
                priceUSD: "{{ .spec.priceUSD }}"
                interval: "{{ .spec.billingInterval }}"

        status:
          fields:
            - path: bucketName
              value: "{{ .spec.metadata.name }}-{{ .metadata.namespace }}-assets"
              when:
                - field: spec.needsStorage
                  equals: "true"

            - path: databaseEndpoint
              value: "{{ .children.rds.status.endpoint }}"
              when:
                - field: spec.needsDatabase
                  equals: "true"
```

The Katalog author writes YAML. The provider author writes Go. The operator
user writes neither — they apply a CR.

---

## The provider interface

```go
// pkg/types/provider.go

// Provider handles a named block in onCreate/onReconcile/onDelete.
// Implement this interface to register an external resource handler.
//
// The Name method returns the YAML block key. AWS registers "aws".
// MongoDB registers "database". Stripe registers "stripe".
// Multiple drivers under one key (driver: mongo, driver: mysql) are handled
// inside the provider — Orkestra routes by block name only.
type Provider interface {
    // Name is the YAML key this provider handles.
    // Must be unique across all registered providers.
    // Convention: lowercase, no hyphens — "aws", "gcp", "database", "stripe"
    Name() string

    // Reconcile executes the declarations for this block.
    // Called after all Kubernetes resource groups reconcile on every cycle.
    // Must be idempotent — called on every resync, not just on first create.
    // Declarations are pre-resolved: template expressions have been evaluated.
    Reconcile(ctx context.Context, req ReconcileRequest) error

    // Delete is called during finalizer execution.
    // Must clean up all external resources created by this provider for the CR.
    // Called once — not retried on failure (the finalizer remains until success).
    Delete(ctx context.Context, req DeleteRequest) error
}

// ReconcileRequest carries everything a provider needs for one reconcile cycle.
type ReconcileRequest struct {
    // Object is the full CR as an unstructured map.
    // Access spec, metadata, status, and children via standard dot paths.
    Object map[string]interface{}

    // Declarations are the resolved provider block declarations.
    // Each entry corresponds to one item in the YAML list under the provider key.
    // Template expressions have already been resolved — values are plain strings.
    Declarations []ProviderDeclaration

    // Kube provides read access to cluster resources.
    // Providers use this to read Secrets for credentials, ConfigMaps for config.
    // Write access is intentionally not provided — providers own external state,
    // not Kubernetes state. Status is written by Orkestra after Reconcile returns.
    Kube KubeReader

    // Logger is a structured logger pre-tagged with crd, resource, request_id.
    Logger zerolog.Logger

    // OwnerName is the CR's metadata.name — convenience for resource naming.
    OwnerName string

    // OwnerNamespace is the CR's metadata.namespace.
    OwnerNamespace string
}

// DeleteRequest carries everything a provider needs for cleanup.
type DeleteRequest struct {
    Object         map[string]interface{}
    Declarations   []ProviderDeclaration
    Kube           KubeReader
    Logger         zerolog.Logger
    OwnerName      string
    OwnerNamespace string
}

// ProviderDeclaration is one resolved declaration from the YAML block.
// The Kind identifies which resource type to act on (rds, s3, bucket, etc.).
// Fields are the resolved key-value pairs from the YAML.
// When holds the original conditions — already evaluated; if false, the
// declaration is skipped before Reconcile is called.
type ProviderDeclaration struct {
    Kind       string            // first key in the YAML map: "rds", "s3", "product"
    Fields     map[string]string // resolved field values
    Conditions []Condition       // from the when: block (already evaluated)
}

// KubeReader provides read-only access to cluster resources.
// Intentionally narrow — providers should not write Kubernetes resources.
type KubeReader interface {
    // GetSecret reads a Secret by name in the given namespace.
    GetSecret(ctx context.Context, namespace, name string) (map[string][]byte, error)

    // GetConfigMap reads a ConfigMap by name in the given namespace.
    GetConfigMap(ctx context.Context, namespace, name string) (map[string]string, error)
}
```

---

## The provider registry

```go
// pkg/types/provider_registry.go

// ProviderRegistry holds all registered providers.
// Thread-safe — providers are registered at startup and read during reconcile.
type ProviderRegistry interface {
    // Register adds a provider. Panics on name collision — fail fast at startup.
    Register(p Provider)

    // Get returns the provider for a given block name.
    Get(name string) (Provider, bool)

    // Names returns all registered provider names — used for validation.
    Names() []string
}
```

---

## How Orkestra calls providers

After the Kubernetes resource groups reconcile (`runDeployments`, `runServices`,
`runJobs`, etc.), the reconciler iterates over provider blocks:

```go
// In runTemplateReconcile, after all Kubernetes resource groups:

for blockName, rawDeclarations := range rc.ProviderBlocks {
    provider, ok := providerRegistry.Get(blockName)
    if !ok {
        logger.Warn().
            Str("block", blockName).
            Msg("no provider registered — block skipped. " +
                "Import and register the provider library.")
        continue
    }

    // Resolve template expressions in declarations
    resolved, err := resolveProviderDeclarations(resolver, rawDeclarations)
    if err != nil {
        return fmt.Errorf("provider %q: resolving declarations: %w", blockName, err)
    }

    // Evaluate when: conditions — skip declarations whose conditions fail
    active := filterByConditions(resolved, resolver.Data())

    if len(active) == 0 {
        continue // all declarations gated out — normal
    }

    req := orktypes.ReconcileRequest{
        Object:         resolver.Data(),
        Declarations:   active,
        Kube:           kubeReader,
        Logger:         logger.With().Str("provider", blockName).Logger(),
        OwnerName:      obj.GetName(),
        OwnerNamespace: obj.GetNamespace(),
    }

    if err := provider.Reconcile(ctx, req); err != nil {
        return fmt.Errorf("provider %q: %w", blockName, err)
    }
}
```

The call is synchronous. Providers that need async behavior (polling for RDS
instance availability) should use the phase model: return nil immediately,
write intermediate state, check completion on the next reconcile cycle. This
is the same model as the declarative state machine — the Kubernetes reconcile
loop is the async runtime.

---

## What providers must guarantee

**Idempotency.** Reconcile is called on every cycle — every resync, every watch
event. A provider that creates an RDS instance must check whether it exists
before calling `CreateDBInstance`. The pattern: read current state, compare to
desired state, apply delta. This is the same contract Orkestra's Kubernetes
reconcilers follow.

**Declarative intent, not imperative sequence.** The provider receives
declarations of desired state. It should not assume that because Reconcile is
being called, the resource does not exist. The correct question is always
"does the current external state match the declared desired state?" not
"should I create this?"

**Safe deletion.** Delete is called once during finalizer execution. If the
external resource does not exist, Delete should return nil — not an error.
Missing resource on deletion is a success condition, not a failure.

**No Kubernetes writes.** Providers read Secrets for credentials and ConfigMaps
for configuration. They do not create, update, or delete Kubernetes resources.
Orkestra owns cluster state. The provider owns external state. This boundary
is enforced by the `KubeReader` interface — no write methods are available.

**Error semantics.** Return a non-nil error only when the external system is
genuinely broken — credentials invalid, quota exceeded, network unreachable.
Do not return an error for "resource being provisioned" — return nil and check
again on the next cycle. Errors cause the reconcile to fail, emit a Warning
event, and back off. "Still provisioning" is not an error.

---

## Provider development guide

A minimal provider:

```go
package myprovider

import (
    "context"
    "fmt"

    orktypes "github.com/orkspace/orkestra/pkg/types"
)

// Provider implements orktypes.Provider for the "myprovider" block.
type Provider struct {
    client *MyExternalClient
}

func New(client *MyExternalClient) *Provider {
    return &Provider{client: client}
}

func (p *Provider) Name() string { return "myprovider" }

func (p *Provider) Reconcile(ctx context.Context, req orktypes.ReconcileRequest) error {
    for _, decl := range req.Declarations {
        switch decl.Kind {
        case "widget":
            if err := p.reconcileWidget(ctx, req, decl); err != nil {
                return fmt.Errorf("widget %q: %w", decl.Fields["name"], err)
            }
        default:
            req.Logger.Warn().Str("kind", decl.Kind).Msg("unknown declaration kind — skipped")
        }
    }
    return nil
}

func (p *Provider) Delete(ctx context.Context, req orktypes.DeleteRequest) error {
    for _, decl := range req.Declarations {
        if decl.Kind == "widget" {
            name := decl.Fields["name"]
            if err := p.client.DeleteWidget(ctx, name); err != nil {
                if !isNotFound(err) {
                    return fmt.Errorf("deleting widget %q: %w", name, err)
                }
                // Already gone — success
            }
            req.Logger.Info().Str("widget", name).Msg("widget deleted")
        }
    }
    return nil
}

func (p *Provider) reconcileWidget(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
    name := decl.Fields["name"]
    size := decl.Fields["size"]

    existing, err := p.client.GetWidget(ctx, name)
    if err != nil && !isNotFound(err) {
        return fmt.Errorf("getting widget: %w", err)
    }

    if isNotFound(err) {
        _, err = p.client.CreateWidget(ctx, name, size)
        if err != nil {
            return fmt.Errorf("creating widget: %w", err)
        }
        req.Logger.Info().Str("widget", name).Msg("widget created")
        return nil
    }

    // Drift correction
    if existing.Size != size {
        if err := p.client.UpdateWidget(ctx, name, size); err != nil {
            return fmt.Errorf("updating widget: %w", err)
        }
        req.Logger.Info().Str("widget", name).Str("size", size).Msg("widget updated")
    }

    return nil
}
```

Registration at startup:

```go
import (
    "github.com/myorg/orkestra-provider-myprovider"
    orktypes "github.com/orkspace/orkestra/pkg/types"
)

registry := orktypes.NewProviderRegistry()
registry.Register(myprovider.New(myClient))

// Pass registry to GenericReconciler or DependencyKordinator
```

---

## The registry ecosystem

The OrkestraRegistry distributes provider libraries as OCI artifacts alongside
Katalog patterns. A pattern that needs an AWS RDS instance declares the
dependency in `pattern.yaml`:

```yaml
apiVersion: registry.orkestra.io/v1
kind: Pattern
metadata:
  name: rds-backed-application
  version: 1.2.0
spec:
  providers:
    - name: aws
      library: oci://registry.orkestra.io/providers/aws:1.8.0
      required: true
    - name: database
      library: oci://registry.orkestra.io/providers/database:2.1.0
      required: false
```

The `ork` CLI handles provider installation:

```bash
ork provider install --pattern rds-backed-application
# → pulls aws provider OCI artifact
# → pulls database provider OCI artifact
# → generates import scaffolding for cmd/operator/main.go
```

Operators that want to contribute providers to the registry:

```bash
ork provider publish --name aws --version 1.9.0
# → packages the Go library as an OCI artifact
# → pushes to the OrkestraRegistry
# → updates pattern.yaml manifests that depend on it
```

---

## Comparison to alternatives

| | Orkestra Provider | Crossplane Provider | Terraform | Raw Go Hook |
|---|---|---|---|---|
| **Declared in** | Katalog YAML | Composition YAML | HCL | Go code |
| **Composable with K8s resources** | ✓ Same file | ✗ Separate CRDs | ✗ Separate state | ✓ Same reconciler |
| **Publishable to registry** | ✓ OCI artifact | ✓ OCI artifact | ✗ Modules only | ✗ Binary only |
| **Phase/state machine** | ✓ Declarative | ✗ Manual pipelines | ✗ Not a concept | ✓ Custom code |
| **Drift correction** | ✓ Every reconcile | ✓ Every reconcile | Partial | Optional |
| **Operator footprint** | Zero (same runtime) | Extra controllers | Separate process | Zero |
| **CRD footprint** | Zero | Extra CRDs per provider | None | Zero |
| **Language** | Go (provider), YAML (operator) | Go (provider), YAML (operator) | Go (provider), HCL (operator) | Go |

The key distinction from Crossplane: Orkestra providers live inside the
operator's reconcile loop. There are no extra controllers, no extra CRDs, no
extra deployments. The provider library is a dependency of the operator binary.
This is the zero-footprint principle applied to external resources.

---

## What providers cannot do

**Generate random values.** Random password generation must happen outside the
reconcile loop — in an init Job or a one-time hook — because reconciliation
is deterministic. The provider can read a pre-generated Secret but cannot
generate one idempotently within Reconcile.

**Manage cross-cluster resources natively.** A provider can call any external
API, but Orkestra's owner reference model is single-cluster. Cross-cluster
resource ownership requires the provider to implement its own tracking — for
example, storing the remote resource ID in a local ConfigMap.

**Express dynamic cardinality in YAML.** If the CR has `spec.databases: [a, b, c]`
and the provider needs one Atlas cluster per entry, the YAML cannot express
"for each element in this list." The provider Go code can iterate over the
resolved list field — but the YAML declaration is a fixed structure, not a
loop. This is the correct boundary: YAML for intent, Go for iteration.

**Replace Crossplane for infrastructure teams.** Crossplane is the right answer
for platform teams managing shared infrastructure with complex dependency graphs
across many teams. Orkestra providers are the right answer for application teams
that need a few external resources alongside their Kubernetes workloads. The use
cases are adjacent, not competing.

---

## Status propagation

Providers can report external resource state back to the CR status by returning
structured data alongside the error:

```go
// Future extension — not in v1 of the provider interface
type ReconcileResult struct {
    // StatusFields are written to the CR's status after successful reconcile.
    // Key is the dot-notation path: "rds.endpoint", "s3.bucket", "stripe.productId"
    StatusFields map[string]interface{}
}
```

In v1, providers write status by annotating the CR with Orkestra's label
convention, and Orkestra's status patching reads the annotation. In v2, the
provider interface returns a `ReconcileResult` that Orkestra merges into the
status patch alongside the declared status fields.

This closes the loop: the Katalog can declare `status.fields` that reference
`{{ .children.rds.status.endpoint }}` — the same way it references
`{{ .children.deployment.status.readyReplicas }}`. External and internal
children are addressed identically.

---

## Conclusion

The provider model is the natural endpoint of Orkestra's architecture. The
declarative layer handles Kubernetes resources. Providers handle external
resources. Notes handle data transformation. The boundary between them is clean
and permanent.

A team building a `DatabaseCluster` operator does not write a hook that knows
about both RDS and Kubernetes. They write a Katalog that declares what they
want, register the AWS provider library that knows about RDS, and Orkestra
assembles the full reconcile loop. The operator author declares intent. The
provider author handles external reality. Orkestra manages the Kubernetes
lifecycle between them.

When AWS ships `github.com/aws/orkestra-provider-aws`, every Orkestra operator
that registers it gains declarative RDS, S3, Route 53, and IAM — without
writing a line of AWS SDK code. When MongoDB ships
`github.com/mongodb/orkestra-provider-atlas`, every operator gains declarative
Atlas clusters and users. The ecosystem grows around the interface, not around
individual operators.

This is the vision: one runtime, many providers, operators as pure
declarations of intent.
