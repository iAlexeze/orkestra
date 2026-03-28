# Mutation

Orkestra mutation applies defaults and normalises CR fields automatically.
Rules are evaluated before validation — defaults are set first so validation
sees a complete object.

Like validation, mutation runs at two points from one declaration.

---

## Two enforcement points, one declaration

### Admission time

When `ENABLE_WEBHOOKS=true`, Orkestra registers a `MutatingWebhookConfiguration`.
The API server calls `/mutate` before `/validate` — this is the standard
Kubernetes ordering. Defaults are applied to the object before it reaches
validation or storage.

```
kubectl apply -f website.yaml
    │
    ▼
API server → POST /mutate (Orkestra)            ← mutation fires first
    │             applies defaults and overrides
    │             returns JSON patch
    │
    ▼
API server → POST /validate (Orkestra)
    │             evaluates rules on the mutated object
    │
    ▼
Object stored (with defaults already applied)
```

The user sees the mutated object when they run `kubectl get website my-site -o yaml`.
The default values are visible in the stored spec.

### Reconcile time

The same mutation rules run in the reconcile loop. This applies defaults to:

- CRs that existed before the webhook was enabled
- CRs whose mutated fields were changed after admission
- CRs where `override` rules should always enforce a value

At reconcile time, when mutation rules produce changes, Orkestra patches the
CR via merge patch and requeues it. The next reconcile cycle sees the mutated
values and proceeds with template reconciliation.

---

## Declaring mutation rules

Mutation rules live under `reconciler.mutation` in the Katalog:

```yaml
- name: website
  reconciler:
    mutation:
      - field: spec.replicas
        default: "2"          # set to 2 when not declared

      - field: spec.logLevel
        default: "info"       # set to info when not declared

      - field: spec.image
        override: "myorg/{{ .metadata.name }}:latest"  # always set
```

---

## Rule types

### `default`

Sets the field **only when it is absent or empty**. If the user has provided
a value, the rule is a no-op.

```yaml
- field: spec.replicas
  default: "2"
```

This is the most common mutation pattern. Use it to supply required-but-optional
fields — fields the operator needs but the user might not declare.

### `override`

**Always sets the field**, regardless of what the user declared. Use with
caution — this overwrites user-provided values.

```yaml
- field: spec.image
  override: "registry.myorg.io/{{ .metadata.name }}:latest"
```

Override is for normalisation rules — ensuring a field is always in a canonical
format regardless of what the user wrote.

!!! warning "Override overwrites user intent"
    A field with `override` cannot be set by the user to a different value
    — Orkestra will reset it on every reconcile and at every admission. Document
    this clearly in your CRD's README. Only use override when the platform
    must control the value unconditionally.

---

## Template expressions

Both `default` and `override` support Go template expressions resolved against
the CR object:

```yaml
- field: spec.image
  override: "myorg/{{ .metadata.name }}:{{ .spec.version | default \"latest\" }}"

- field: metadata.labels.managed-by
  override: "orkestra"   # plain string — no template needed
```

The resolver has access to the full CR object: `.metadata.name`,
`.metadata.namespace`, `.metadata.labels.*`, `.spec.*`.

Plain strings without `{{` are returned as-is — the fast path requires no
template parsing.

---

## Ordering: mutation before validation

At admission time, the Kubernetes API server calls mutating webhooks before
validating webhooks. Orkestra follows this ordering.

At reconcile time, `reconciler.mutation.mutateFirst: false` (default) means
validate first, then mutate valid objects. Set `mutateFirst: true` when a
default value is needed to satisfy a validation rule:

```yaml
reconciler:
  validation:
    - field: spec.replicas
      min: "1"
      message: "replicas must be at least 1"
      action: deny

  mutation:
    - field: spec.replicas
      default: "1"

  mutateFirst: true  # apply default before validation sees the empty field
```

Without `mutateFirst: true`, a CR with no `spec.replicas` would fail the
`min: "1"` validation rule before the `default: "1"` mutation could run.

!!! note "mutateFirst only affects reconcile ordering"
    At admission time, mutation always fires before validation regardless of
    `mutateFirst`. This field controls only the reconcile-time ordering.

---

## Enabling admission-time mutation

Same requirements as validation:

```bash
ENABLE_WEBHOOKS=true     # starts the HTTPS server on :8443 and registers /validate and /mutate
TLS_CERT=/tls/tls.crt
TLS_KEY=/tls/tls.key
```

Orkestra automatically creates the `MutatingWebhookConfiguration` at startup,
covering only CRDs with mutation rules declared. The webhook uses
`ReinvocationPolicy: IfNeeded` — if another webhook modifies the object after
Orkestra's mutation, Orkestra is re-invoked to ensure its defaults are still
applied to the final object.

!!! warning "RBAC requirement"
    ```yaml
    - apiGroups: ["admissionregistration.k8s.io"]
      resources:
        - mutatingwebhookconfigurations
      verbs: ["get", "create", "update", "patch"]
    ```

---

## How the patch is built

Orkestra computes a JSON patch (RFC 6902) from the mutation changes:

```json
[
  { "op": "add",     "path": "/spec/replicas", "value": "2"    },
  { "op": "replace", "path": "/spec/logLevel",  "value": "info" }
]
```

`add` is used for fields that were absent. `replace` is used for fields that
existed but are being overridden. The patch is returned in the webhook response
and applied by the API server before the object is stored.

The patch targets only the fields that mutation rules changed — it is minimal
and precise. Fields the user declared correctly are not included in the patch.

---

## Observability

```
controller_mutation_total{crd}
```
Reconciles where at least one mutation rule was applied. High rate means
many CRs arrive without required fields.

```
controller_mutation_applied_total{crd, field, type="default|override"}
```
Which specific fields are being defaulted or overridden. `default` with high
count = users are not setting this field. `override` with high count = users
are setting it but the platform is normalising it.

Admission-time mutations are also counted — every admission call triggers a
reconcile via the watch event, and the reconcile emits the metrics.
