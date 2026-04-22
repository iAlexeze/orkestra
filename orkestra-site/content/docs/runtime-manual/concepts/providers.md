---
title: "Providers"
weight: 130
---

# Providers

Providers extend Orkestra's declarative layer to external infrastructure.
They handle named YAML blocks in `onCreate`, `onReconcile`, and `onDelete`
that Orkestra's core cannot handle — AWS resources, databases, SaaS APIs.

---

## Mental model

```
Katalog YAML
  └── Kubernetes blocks  →  Orkestra core reconciler
  └── Provider blocks    →  Registered provider library  →  External API
```

The Katalog author writes both. Orkestra dispatches each block to its handler.
The operator user sees one CR, one status, one set of conditions.

---

## Registration

Providers are registered at binary startup before `o.Start()`:

```go
registry := orktypes.NewProviderRegistry()
registry.Register(awsprovider.New(sess))
registry.Register(mongoprovider.New(uri))
```

Pass the registry to `NewDependencyKordinator`. It is threaded into every
`GenericReconciler` that runs template reconciliation.

A provider name that is not registered is skipped with a warning — it does
not fail the reconcile. This allows operators to load a Katalog that declares
provider blocks without having that provider registered locally, which is
useful for `ork validate` and `ork run` in development.

---

## Lifecycle

For each reconcile cycle:

1. Kubernetes resource groups reconcile (deployments, services, jobs, etc.)
2. `ReadChildren` populates `.children.*`
3. For each provider block in the Katalog:
   a. Template expressions in declarations are resolved
   b. `when:` conditions are evaluated — declarations failing conditions are dropped
   c. `provider.Reconcile(ctx, req)` is called with the surviving declarations
4. Status fields are patched
5. Ready condition is written

On CR deletion (finalizer):

1. `provider.Delete(ctx, req)` is called for each registered provider block
2. Only after all providers return nil is the finalizer removed
3. Kubernetes resources are cleaned up via owner references (garbage collection)

---

## The `when:` contract

Conditions on provider declarations work identically to conditions on Kubernetes
resource declarations. The resolved condition is evaluated by Orkestra before
the provider is called — the provider never sees declarations whose conditions
did not pass.

```yaml
aws:
  - rds:
      instanceClass: db.t3.micro
      when:
        - field: spec.needsDatabase
          equals: "true"
```

If `spec.needsDatabase` is not `"true"`, the `rds` declaration is removed from
the list before `Reconcile` is called. The provider sees an empty or reduced
declaration list and returns nil immediately — no external API call made.

---

## Idempotency patterns

**Read-before-write (standard pattern):**
```go
existing, err := client.GetResource(ctx, name)
if isNotFound(err) {
    return client.CreateResource(ctx, spec)
}
if err != nil {
    return err
}
// Drift: compare existing to spec, update if different
```

**Tag-based ownership:**
For providers that cannot use labels, use tags on external resources:
```
orkestra-owner: <cr-name>
orkestra-namespace: <cr-namespace>
orkestra-catalog: <catalog-name>
```
Delete only resources with these tags. Never delete resources you did not
create.

**Intermediate states:**
Do not poll for external resource availability inside Reconcile. Return nil
immediately after initiating creation. On the next reconcile, check status.
Write phase via status fields. This is the state machine pattern applied to
external resources:

```go
func (p *Provider) Reconcile(ctx context.Context, req ReconcileRequest) error {
    phase := req.Object["status"]["phase"].(string)

    switch decl.Kind {
    case "rds":
        switch phase {
        case "", "Pending":
            return p.initiateRDSCreation(ctx, req, decl)
            // returns nil — next reconcile checks status
        case "Provisioning":
            return p.checkRDSStatus(ctx, req, decl)
            // returns nil if still provisioning
            // writes "Ready" to status when done (via StatusFields in result)
        }
    }
    return nil
}
```

---

## KubeReader — credentials pattern

Providers read credentials from Secrets, not from environment variables:

```yaml
aws:
  - rds:
      credentials:
        secretName: "{{ .metadata.name }}-aws-credentials"
```

In the provider:

```go
secret, err := req.Kube.GetSecret(ctx, req.OwnerNamespace, decl.Fields["credentials.secretName"])
if err != nil {
    return fmt.Errorf("reading AWS credentials: %w", err)
}
accessKey := string(secret["AWS_ACCESS_KEY_ID"])
secretKey := string(secret["AWS_SECRET_ACCESS_KEY"])
```

This keeps credentials out of the Katalog YAML (which may be committed to Git)
and in Kubernetes Secrets (which have RBAC controls and can be sealed/encrypted).

---

## Children — provider status in templates

In v1, providers cannot write to `.children` directly. Status is written
through declared `status.fields` using known paths:

```yaml
status:
  fields:
    - path: databaseEndpoint
      value: "{{ .children.rds.endpoint }}"
```

The `.children.rds` path is populated when the provider writes the endpoint
to a well-known annotation on the CR, which Orkestra reads into the children
map. The exact mechanism is defined per provider.

In v2 (planned), providers return a `ReconcileResult` with `StatusFields` that
Orkestra merges into the status patch directly — no annotation intermediary.

---

## Error handling

| Situation | Return |
|---|---|
| External API unreachable | `error` — reconcile fails, backs off |
| Credentials invalid | `error` — reconcile fails, emits Warning event |
| Resource not found on delete | `nil` — idempotent, success |
| Resource still provisioning | `nil` — check again next cycle |
| Quota exceeded | `error` — reconcile fails, operator should surface this |
| Declaration kind unknown | Log warn, skip, `nil` — do not fail reconcile |

---

## Testing providers

Providers should be tested independently of Orkestra using standard Go testing.
The `ReconcileRequest` struct can be constructed directly in tests:

```go
func TestReconcileWidget(t *testing.T) {
    client := &mockWidgetClient{}
    p := myprovider.New(client)

    req := orktypes.ReconcileRequest{
        OwnerName:      "my-app",
        OwnerNamespace: "default",
        Declarations: []orktypes.ProviderDeclaration{
            {
                Kind: "widget",
                Fields: map[string]string{
                    "name": "my-app-widget",
                    "size": "large",
                },
            },
        },
        Logger: zerolog.Nop(),
    }

    err := p.Reconcile(context.Background(), req)
    require.NoError(t, err)
    assert.True(t, client.createCalled)
}
```

Integration tests should run against a local or test-tier instance of the
external system — not production, not mocked at the HTTP level. The provider
interface is thin enough that unit tests with mock clients cover the decision
logic, and integration tests with real clients cover the API interaction.

---

## Provider naming conventions

| Block name | Convention | Example |
|---|---|---|
| Cloud provider | Cloud brand | `aws`, `gcp`, `azure` |
| Service category with drivers | Category | `database`, `cache`, `queue` |
| Specific SaaS | Product name | `stripe`, `twilio`, `sendgrid` |
| Internal platform | Short name | `vault`, `consul`, `kafka` |

Driver discrimination within a block:

```yaml
database:
  - driver: mongo    # handled by mongo sub-handler inside database provider
    ...
  - driver: mysql    # handled by mysql sub-handler
    ...
```

The provider's `Reconcile` switches on `decl.Fields["driver"]`. This avoids
registering one provider per database engine and keeps the YAML readable.
