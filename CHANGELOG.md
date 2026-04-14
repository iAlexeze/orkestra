# Changelog — OperatorBox Rename, Security Refactor, Provider Credentials

## Breaking Changes

### `reconciler:` → `operatorBox:` (Katalog YAML schema)

The per-CRD runtime block has been renamed from `reconciler:` to `operatorBox:` in all Katalog YAML files.

**Before:**
```yaml
spec:
  crds:
    - apiTypes:
        kind: MyApp
      reconciler:
        workers: 3
        hooks: ...
```

**After:**
```yaml
spec:
  crds:
    - apiTypes:
        kind: MyApp
      operatorBox:
        workers: 3
        hooks: ...
```

**Affected Go types:**
- `ReconcilerConfig` → `OperatorBoxConfig`
- `CRDEntry.ReconcilerConfig` → `CRDEntry.OperatorBox`
- `ReconcilerInfo` → `OperatorBoxInfo`
- `ReconcilerSummary` → `OperatorBoxSummary`
- JSON response field `"reconciler"` → `"operatorBox"` in health/catalog endpoints
- Control Center UI: "Reconciler Configuration" → "OperatorBox Configuration"

---

### `security.webhooks.conversion` → `security.conversion` (Katalog YAML schema)

Conversion webhooks are a separate Kubernetes concept from admission webhooks and are now declared at the top level of the `security:` block.

**Before:**
```yaml
security:
  webhooks:
    conversion:
      enabled: true
      conversionWindow: 200
```

**After:**
```yaml
security:
  conversion:
    enabled: true
    conversionWindow: 200
```

---

### `spec.providers[]` → top-level `providers:` (Katalog YAML schema)

Provider requirements have been promoted from `spec.providers[]` to a top-level block alongside `spec:` and `security:`. Providers represent operational infrastructure dependencies — distinct state from the CRD definitions in `spec:`.

**Before:**
```yaml
spec:
  providers:
    - name: aws
      required: true
      auth:
        accessKeyId: "$AWS_ACCESS_KEY_ID"
        secretAccessKey: "$AWS_SECRET_ACCESS_KEY"
        region: "$AWS_REGION"
  crds:
    my-app: ...
```

**After:**
```yaml
providers:
  - name: aws
    required: true
    auth:
      accessKeyId: "$AWS_ACCESS_KEY_ID"
      secretAccessKey: "$AWS_SECRET_ACCESS_KEY"
      region: "$AWS_REGION"
  - name: mongodb
    required: true
    auth:
      mongoUri: "$MONGODB_URL"

spec:
  crds:
    my-app: ...

security:
  ...
```

The full top-level Katalog document shape is now:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: my-operator

providers:          # ← infrastructure dependencies (new top-level)
  - name: aws
    required: true
    auth:
      accessKeyId: "$AWS_ACCESS_KEY_ID"
      ...

spec:               # ← CRD definitions
  crds:
    my-app:
      operatorBox: ...

security:           # ← security settings
  deletionProtection:
    enabled: true
```

---

## New Features

### Provider credentials in `providers[].auth`

Top-level `providers[]` declarations accept an `auth:` map with `$ENV_VAR` expansion at startup:

```yaml
providers:
  - name: aws
    required: true
    auth:
      accessKeyId: "$AWS_ACCESS_KEY_ID"
      secretAccessKey: "$AWS_SECRET_ACCESS_KEY"
      region: "$AWS_REGION"
  - name: mongodb
    required: true
    auth:
      mongoUri: "$MONGODB_URL"
```

- Providers **not declared** at top level are never registered. Per-CRD `operatorBox.providers` blocks for unregistered providers are skipped with a warning log.
- `required: true` causes a fatal startup error if the provider fails to initialise.
- `required: false` logs a warning and the operator continues.
- Both `auth.mongoUri` and `auth.uri` are accepted for MongoDB.

### `crdEntry.conversion.updateCRD` — automatic caBundle patching

When a CRD uses conversion webhooks, set `updateCRD: true` to have Orkestra patch the CRD's `spec.conversion.webhook.clientConfig.caBundle` automatically at startup:

```yaml
spec:
  crds:
    my-app:
      conversion:
        storageVersion: v1
        updateCRD: true
        paths: [...]
```

### Unified TLS certificate handling

TLS certificates (required for deletion protection, admission webhooks, and conversion webhooks) are now stored in a single location (`SecurityConfig.Webhooks.TLSCert/TLSKey`) and shared across all three webhook types. The deprecated `webhookConfig` / `webhookRegistration` types have been removed.

### Deletion protection enabled by default when block is declared

When `security.deletionProtection:` is present in the Katalog YAML without an explicit `enabled:` field, deletion protection defaults to **enabled**. This prevents accidental omission from silently disabling protection.

### `NeedsCertificates()` on Katalog

A single method now determines whether TLS generation is required — it returns `true` when any of deletion protection, admission webhooks, or conversion is active.

---

## Internal Changes

- `pkg/types/katalog.go` — `Providers []KatalogProviderRequirement` removed from `KatalogSpec`; added as top-level field on `KatalogFile` and `KatalogForUI`.
- `pkg/katalog/type.go` — `Providers []KatalogProviderRequirement` added as top-level field on `Katalog` runtime struct.
- `pkg/merger/merger.go` — `providers` field added; `ToProviders()` method added.
- `pkg/merger/file.go` — `m.providers = doc.Providers` set in both `loadKatalog` and `loadKomposer`.
- `pkg/katalog/parser.go` — `k.Providers = m.ToProviders()` wired in `KomposeKatalogFromYaml`.
- `pkg/types/provider_katalog.go` — Package-level comment updated; `KatalogProviderRequirement` doc updated.
- `pkg/konfig/type.go` — Removed deprecated `webhookConfig`, `webhookRegistration` types and their accessor methods. TLS fields moved into `SecurityConfig.Webhooks`. Port constants exposed via `HTTPSPort()` / `HTTPSPortInt32()`.
- `pkg/katalog/security.go` — Full rewrite using `envSecurityReader` adapter pattern for ENV → YAML precedence without import cycles.
- `pkg/health/webhook_registration.go` — Added `admissionv1FailurePolicyType()` helper; removed dependency on `WebhookRegistration()`.
- `pkg/kubeclient/kubeclient.go` — Added `ApiextensionsClient()` method backed by `apiextclientset.Interface` for CRD patching.
- `pkg/provider/aws/provider.go` — Added `NewFromAuth()` constructor for credential injection from Katalog auth map.
- `pkg/provider/mongo/provider.go` — Now registered via Katalog-level declaration; `NewFromURI()` called with resolved auth.
- `cmd/internal/provider.go` — Rewritten to iterate `kat.Providers` (top-level) rather than `kat.Spec.Providers`.
- `cmd/internal/konstruct_security.go` — `ensureSecurity()` extended to cover admission + conversion; `patchConversionCRDs()` added.
- `docs/runtime-manual/concepts/provider.md` — Updated for `providers:` top-level, `operatorBox.providers` per-CRD.
- `docs/publications/provider-library.md` — Updated pattern YAML to use top-level `providers:`.
- All YAML katalog files — `reconciler:` → `operatorBox:` (53 occurrences).
