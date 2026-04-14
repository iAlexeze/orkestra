# Providers

Providers extend Orkestra's declarative layer to external systems — AWS,
MongoDB, Stripe, Vault, or any API a Kubernetes operator needs to call.

---

## The two layers

**Layer 1 — Katalog-level manifest** (`providers`)

Declares which provider libraries this Katalog requires. Top-level alongside
`spec:` and `security:` — providers represent operational infrastructure
dependencies, distinct state from CRD definitions. Used by `ork validate`
to warn when a required provider is not registered, and by `ork provider install`
to pull OCI artifacts.

```yaml
providers:
  - name: aws
    required: true
    auth:
      accessKeyId: "$AWS_ACCESS_KEY_ID"
      secretAccessKey: "$AWS_SECRET_ACCESS_KEY"
      region: "$AWS_REGION"
  - name: mongodb
    required: false
    auth:
      mongoUri: "$MONGODB_URL"
```

**Layer 2 — Per-CRD declarations** (`spec.crds[].operatorBox.providers`)

The actual resource declarations — what to create, update, or delete for each
CR instance. These are dispatched to the registered provider library at
reconcile time.

```yaml
spec:
  crds:
    my-app:
      operatorBox:
        providers:
          aws:
            - s3:
                bucket: "{{ .metadata.name }}-assets"
                region: "{{ .spec.region }}"
          mongodb:
            - database:
                name: "{{ .metadata.name }}"
            - user:
                name: "{{ .spec.dbUser }}"
                database: "{{ .metadata.name }}"
```

---

## How providers are dispatched

```
Katalog parsed → OperatorBoxConfig.ProviderBlocks populated
                      ↓
GenericReconciler.reconcileImpl
                      ↓
runTemplateReconcile (after Kubernetes resources)
                      ↓
runProviders(ctx, obj, resolver, blocks, registry, kube)
    │
    ├── For each block in ProviderBlocks:
    │     │
    │     ├── registry.Get(block.Name)           — lookup by YAML key
    │     ├── resolveProviderBlock(...)           — template expressions resolved
    │     ├── filterProviderDeclarations(...)     — when: conditions evaluated
    │     └── provider.Reconcile(ctx, req)        — provider code runs
    │
    └── On finalizer: runProviderDelete(...)      — provider.Delete(ctx, req)
```

The registry is built once in `loadProviders(ctx, kat)` at startup and captured
in each reconciler factory closure. It never passes through
`DependencyKordinator` or `ReconcilerFactory`.

---

## The YAML structure

A provider block is a named map under `operatorBox.providers`. The key is the
provider's `Name()` return value. The value is a list of declarations.

Each declaration is a single-key map where the key is the resource kind and
the value is the field map. `when:` is a special key for conditions.

```yaml
operatorBox:
  providers:
    aws:                                     # block name → registry.Get("aws")
      - s3:                                  # declaration kind
          bucket: "{{ .metadata.name }}"    # template expression
          region: us-east-1                 # static value
          when:
            - field: spec.enableStorage
              equals: "true"

      - rds:
          instanceClass: "{{ .spec.dbInstanceClass }}"
          engine: postgres
          credentials:
            secretName: "{{ .metadata.name }}-aws-creds"
          when:
            - field: spec.needsDatabase
              equals: "true"
```

---

## Field resolution

Template expressions in field values are resolved before the provider is called.
The provider always receives plain strings — never raw template expressions.

Nested YAML maps are flattened with dot notation:

```yaml
credentials:
  secretName: my-secret
```

Becomes `decl.Fields["credentials.secretName"] = "my-secret"`.

---

## `when:` conditions

Conditions work identically to `when:` on Kubernetes resource blocks.
The full CR state is available including `.spec.*`, `.status.*`, `.children.*`.

On deletion (finalizer), conditions are **not** evaluated — all declarations
are passed to `provider.Delete` regardless.

---

## Reading credentials

Providers must read credentials from Kubernetes Secrets:

```yaml
- rds:
    credentials:
      secretName: "{{ .metadata.name }}-aws-creds"
```

```go
data, err := req.Kube.GetSecret(ctx, req.OwnerNamespace, decl.Field("credentials.secretName", ""))
accessKey := string(data["AWS_ACCESS_KEY_ID"])
```

---

## Error semantics

| Return | When |
|---|---|
| `nil` | Resource already matches desired state |
| `nil` | Resource still provisioning — check next cycle |
| `nil` | Resource not found on delete |
| `error` | API unreachable, credentials invalid, quota exceeded |

---

## Current providers

| Block name | Package | Kinds |
|---|---|---|
| `aws` | `pkg/providers/aws` | `s3`, `rds`, `route53` |
| `mongodb` | `pkg/providers/mongodb` | `database`, `user`, `collection` |

To add a new provider: see `docs/concepts/extending-providers.md`.