# Mutation Schema

Complete schema reference for `reconciler.mutation` in a Katalog CRD entry.

---

## `reconciler.mutation`

```yaml
reconciler:
  mutation:
    rules:
      - field: string       # required
        default: string     # set when absent or empty
        override: string    # always set

    mutateFirst: bool       # default: false — validate before mutate at reconcile
```

---

## `MutationConfig`

| Field | Type | Default | Description |
|---|---|---|---|
| `rules` | `[]MutationRule` | `[]` | Ordered list of mutation rules. Applied in declaration order. |
| `mutateFirst` | bool | `false` | When true, mutation runs before validation at reconcile time. Does not affect admission-time ordering (mutation always fires before validation at admission). |

---

## `MutationRule`

| Field | Type | Required | Description |
|---|---|---|---|
| `field` | string | yes | Dot-notation path to the CR field to mutate: `spec.replicas`, `metadata.labels.managed-by` |
| `default` | string | one of `default` or `override` | Set only when field is absent or empty. Supports template expressions. |
| `override` | string | one of `default` or `override` | Always set, regardless of current value. Supports template expressions. |

!!! warning "Declare exactly one of `default` or `override`"
    A rule with both `default` and `override` set will use `override` (it takes
    priority). A rule with neither declared is a no-op — it produces no patch.

---

## Template expressions

Both `default` and `override` support Go `text/template` expressions resolved
against the CR object being admitted or reconciled.

| Expression | Resolves to |
|---|---|
| `{{ .metadata.name }}` | CR name |
| `{{ .metadata.namespace }}` | CR namespace |
| `{{ .spec.version }}` | `spec.version` field value |
| `{{ .spec.version \| default "latest" }}` | `spec.version` or "latest" if empty |
| `"plain string"` | Returned as-is (fast path, no template parsing) |

The resolver has access to the full CR object map. Missing fields resolve to
empty string — `missingkey=zero` — no error on absent optional fields.

---

## `reconciler.webhooks.mutation` — per-CRD admission control

Controls whether this CRD participates in the `MutatingWebhookConfiguration`
when `ENABLE_WEBHOOKS=true`.

```yaml
- name: website
  webhooks:
    mutation: true         # default: true when rules are declared
    operations:
      - CREATE
      - UPDATE
```

| Field | Type | Default | Description |
|---|---|---|---|
| `mutation` | bool | `true` | Include in `MutatingWebhookConfiguration` |
| `operations` | `[]string` | `["CREATE", "UPDATE"]` | Which operations trigger mutation |

!!! tip "Mutation on CREATE only"
    If you only want defaults applied on first creation — not on every update —
    declare `operations: ["CREATE"]`. This prevents Orkestra from patching the
    object on every `kubectl apply`, which may interfere with GitOps workflows
    that expect a clean diff.

---

## Patch format

Orkestra returns a JSON patch (RFC 6902). Each mutation rule that produces a
change becomes one patch operation:

| Field was | Operation | Example |
|---|---|---|
| Absent (not in spec) | `add` | `{"op":"add","path":"/spec/replicas","value":"2"}` |
| Present but overridden | `replace` | `{"op":"replace","path":"/spec/logLevel","value":"info"}` |

Path conversion: dot-notation → JSON Pointer. `spec.replicas` → `/spec/replicas`.
`metadata.labels.team` → `/metadata/labels/team`.

---

## Error reference

### Startup errors

```
error: ENABLE_WEBHOOKS requires ENABLE_CONVERSION — set ENABLE_CONVERSION=true
```

```
error: webhook registration: reading CA bundle: TLS_CERT is required
```

```
error: webhook registration: mutating: failed to create MutatingWebhookConfiguration:
  ... admissionregistration.k8s.io/mutatingwebhookconfigurations requires get, create, update
```

### Template resolution errors (logged, not fatal at admission)

```
admission/mutate: error applying rules — allowing without mutation
  kind=Website name=my-site
  err: mutation rule override for field "spec.image": resolving template "{{ .spec.badField }}": ...
```

!!! note "Mutation failures are non-fatal"
    If a mutation rule fails (template error, patch error, API conflict),
    Orkestra logs the error and allows the operation without mutation. Mutation
    must never block an admission operation — only validation can reject.

### Reconcile-time events (kubectl describe)

```
Events:
  Type    Reason           Message
  ----    ------           -------
  Normal  WebsiteMutation  defaults applied: spec.replicas="2", spec.logLevel="info"
```

---

## Complete example

```yaml
- name: website
  apiTypes:
    group: demo.orkestra.io
    version: v1alpha1
    kind: Website
    plural: websites

  webhooks:
    mutation: true
    operations: ["CREATE", "UPDATE"]

  reconciler:
    mutation:
      # Defaults — applied when field is absent
      - field: spec.replicas
        default: "2"

      - field: spec.logLevel
        default: "info"

      - field: spec.port
        default: "8080"

      # Override — always normalised to internal registry
      - field: spec.image
        override: "registry.myorg.io/{{ .metadata.name }}:{{ .spec.version | default \"latest\" }}"

      # Override — always set the managed-by label
      - field: metadata.labels.managed-by
        override: "orkestra"

    mutateFirst: false   # default — validate first, then mutate valid objects
                         # set true when defaults are needed to pass validation
```

### With mutateFirst: true

```yaml
reconciler:
  validation:
    - field: spec.replicas
      min: "1"
      message: "replicas must be at least 1"
      action: deny

  mutation:
    - field: spec.replicas
      default: "1"    # must run BEFORE validation or this will be denied

  mutateFirst: true   # mutation runs before validation at reconcile time
                      # at admission time, mutation always fires first regardless
```
