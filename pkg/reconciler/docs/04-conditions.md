# 04 — Conditions, when:, anyOf:, and the activeNames Guard

## How conditions are declared in YAML

Every template source supports two optional condition blocks:

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}-prod"
      image: nginx:latest
      # AND: all must pass
      when:
        - field: spec.environment
          equals: production
        - field: spec.enabled
          equals: "true"
      # OR: at least one must pass
      anyOf:
        - field: spec.tier
          equals: premium
        - field: spec.tier
          equals: enterprise
```

`when:` conditions are AND'd. `anyOf:` conditions are OR'd. When both are declared, both must pass.

## EvaluateWhen

```go
orktypes.EvaluateWhen(data map[string]interface{}, allOf []Condition, anyOf []Condition) bool
```

`data` is `resolver.Data()` — the full CR data map including injected fields:

| Prefix | Source |
|--------|--------|
| `.spec.*` | CR spec |
| `.status.*` | CR status |
| `.metadata.*` | CR metadata |
| `.children.*` | Owned child resource summaries |
| `.external.<n>.*` | HTTP call responses |
| `.cross.<kind>.*` | Sibling CRD status |

Do NOT pass `owner` (the `domain.Object`) directly. The injected fields are only on `resolver.Data()`.

## Condition operators

| Shorthand field | Operator constant | Semantics |
|-----------------|-------------------|-----------|
| `equals:` | `ConditionEquals` | Exact string match |
| `notEquals:` | `ConditionNotEquals` | String mismatch |
| `contains:` | `ConditionContains` | Substring |
| `prefix:` | `ConditionPrefix` | String starts with |
| `suffix:` | `ConditionSuffix` | String ends with |
| `greaterThan:` | `ConditionGt` | Numeric > |
| `lessThan:` | `ConditionLt` | Numeric < |
| `operator: exists` | `ConditionExists` | Field present and non-empty |
| `operator: notExists` | `ConditionNotExists` | Field absent or empty |
| `operator: in` | `ConditionIn` | Field is one of comma-separated list |
| `operator: typeOf` | `ConditionTypeOf` | Runtime type of field value |

The `value:` field on a `Condition` struct supports template expressions:

```yaml
when:
  - field: metadata.name
    equals: "{{ .spec.expectedName }}"
```

Absent numeric fields are treated as `0` (Kubernetes omits zero-value integers from JSON).

## The typeOf operator

`typeOf` compares the **runtime Go type** of a field value — not its string representation — against the expected string. Useful for distinguishing list vs scalar fields:

```yaml
when:
  - field: spec.replicas
    operator: typeOf
    value: float64   # Kubernetes numbers unmarshal as float64
```

This is evaluated via `note.TypeOf(raw)` which uses `NavigateRawPath` (returns `interface{}`) not `NavigateDotPath` (returns `string`).

## The activeNames pre-pass

### The problem it solves

Consider two declarations that share a resolved name but have mutually exclusive conditions:

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}-worker"
      when:
        - field: spec.mode
          operator: typeOf
          value: string          # mode is a plain string
      reconcile: true

    - name: "{{ .metadata.name }}-worker"
      when:
        - field: spec.mode
          operator: typeOf
          value: "[]interface {}" # mode is a list
      reconcile: true
```

On every reconcile, declaration [0] passes (mode is a string) and declaration [1] fails. Without the pre-pass, declaration [1]'s condition-failure path calls `DeleteIfOwned` — which deletes the deployment that declaration [0] just created. The next reconcile re-creates it. The result is a create/delete loop.

### How the pre-pass fixes it

Before the main loop, build a set of every `(ns/name)` that has **at least one passing condition**:

```go
activeNames := make(map[string]bool, len(srcs))
for _, s := range srcs {
    if !orktypes.EvaluateWhen(resolver.Data(), s.Conditions, s.AnyOf) {
        continue
    }
    n, _   := resolver.Resolve(s.Name)
    nsp, _ := resolver.Resolve(s.Namespace)
    if nsp == "" {
        nsp = owner.GetNamespace()
    }
    activeNames[nsp+"/"+n] = true
}
```

In the main loop, guard `DeleteIfOwned` with:

```go
if !activeNames[ns+"/"+name] {
    orkwidget.DeleteIfOwned(...)
}
```

If `(ns/name)` is in `activeNames`, at least one passing declaration will create or keep it — the failing declaration must not delete it.

### When to omit the pre-pass

- The resource type is create-only and never calls `DeleteIfOwned` in the failure path (e.g. `runJobs` — jobs run to completion and are not cleaned up on condition failure).
- The resource type can never have two declarations with the same resolved name (uncommon — but if the type enforces unique names you can omit it).

In all other cases, include the pre-pass. The cost is one extra loop over the source slice — negligible.

## The Conditions and AnyOf fields on template source types

Every `XxxTemplateSource` struct in `pkg/types/` must carry:

```go
type IngressTemplateSource struct {
    Name       string      `yaml:"name"`
    Namespace  string      `yaml:"namespace,omitempty"`
    Conditions []Condition `yaml:"when,omitempty"`
    AnyOf      []Condition `yaml:"anyOf,omitempty"`
    Reconcile  bool        `yaml:"reconcile,omitempty"`
    ForEach    *ForEachSpec `yaml:"forEach,omitempty"`
    // ... resource-specific fields
}
```

The `Conditions` field maps to `when:` and the `AnyOf` field maps to `anyOf:`. Both are optional — empty slices always pass.

---

**Next →** [05 — forEach Expansion](05-foreach.md)
