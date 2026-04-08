---
title: "Conversion Validation Mutation"
weight: 155
---

# Conversion, Validation, and Mutation

This document explains how Orkestra enforces policy on custom resources — what each mechanism does, where it runs, which code handles it, and how they connect.

---

## The big picture

Orkestra can intercept a custom resource at two distinct moments:

1. **Admission time** — while `kubectl apply` is in flight, before the object is stored in etcd. Kubernetes calls Orkestra's webhook endpoints synchronously. The request can be rejected before it ever lands.

2. **Reconcile time** — after the object is already stored. The operator's reconcile loop checks the same rules every time it processes the object.

Both moments read from the same Katalog declaration. You declare rules once; Orkestra enforces them everywhere.

There is a third mechanism, **conversion**, which is separate: it translates an object from one API version to another when multiple `apiVersion` values exist for the same CRD.

---

## Where the rules live

Rules are declared in the Katalog YAML inside each CRD entry:

```yaml
spec:
  crds:
    website:
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites

      validation:           # ← who: admission webhook + reconcile loop
        rules:
          - field: spec.image
            prefix: "myorg/"
            message: "image must come from the myorg registry"

      mutation:             # ← who: admission webhook only (reconcile also has runMutation)
        rules:
          - field: spec.replicas
            default: "2"

      conversion:           # ← who: conversion webhook only
        paths:
          - from: v1alpha1
            to: v1
            spec:
              image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"

      webhooks:
        validation: true    # register /validate for this CRD (default: true if rules exist)
        mutation: true      # register /mutate for this CRD (default: true if rules exist)
```

At startup, `RegisterAdmissionRulesFromEntry` in [pkg/katalog/admission_registry.go](https://github.com/iAlexeze/orkestra/blob/main/pkg/katalog/admission_registry.go) reads each CRD entry and loads its rules into the `InMemoryAdmissionRegistry`, keyed by the CRD's GVR string (`"group/version/resource"`). The conversion rules go into a separate `ConversionRegistry`.

---

## Mutation

### What it does

Mutation sets default values on fields that are absent or empty, or overrides fields unconditionally. It runs **before** validation so that a rule like `min: "1"` on a field that would otherwise be missing can still pass after mutation fills in its default.

### Two execution paths

**Path 1 — Admission webhook (`/mutate`)**

This runs synchronously when `kubectl apply` is called. Kubernetes sends a `POST /mutate` request carrying the full object in a `MutationAdmissionReview` JSON body.

The flow in [pkg/health/admission_handlers.go](https://github.com/iAlexeze/orkestra/blob/main/pkg/health/admission_handlers.go):

```
POST /mutate  (HealthServer.mutationHandler)
  │
  ├─ decode AdmissionReview from request body
  ├─ look up MutationConfig from InMemoryAdmissionRegistry (keyed by GVR)
  │    → GetMutationRules(gvrKey)
  ├─ deep-copy the incoming object (original stays untouched for diffing)
  ├─ applyMutationRules(copy, cfg)          ← pkg/health/admission_evaluation.go
  │    for each rule:
  │      if rule.Override → always set the field (resolved via Go template)
  │      if rule.Default  → set only when field is absent or empty
  │    returns: list of {Field, OldValue, NewValue, ChangeType}
  ├─ buildJSONPatch(changes)                ← RFC 6902 patch document
  │    "spec.replicas" → JSON Pointer "/spec/replicas"
  │    absent field    → op: "add"
  │    present field   → op: "replace"
  └─ return AdmissionReview with patch attached
```

The API server applies the patch to the object **in memory** before passing it on to validation and then writing it to etcd. The user never sees a raw 422; the object arrives already corrected.

**Path 2 — Reconcile loop (`runMutation`)**

This runs inside the operator's reconcile cycle. It is in [pkg/reconciler/run_mutations.go](https://github.com/orkestra-sh/orkestra/pkg/reconciler/run_mutations.go):

```
runMutation(ctx, kube, obj, cfg, gvr, crdName)
  │
  ├─ same rule evaluation logic as Path 1
  ├─ if any fields changed: builds a merge patch (not JSON patch)
  └─ patches the object via kube.DynamicClient().Resource(gvr).Patch(...)
```

The reconciler applies the patch back to the live object via the Kubernetes API. The caller should re-read the object from the informer cache after this returns with `Applied > 0`.

### Template expressions

Both paths resolve template expressions in `Override` and `Default` values using `orktmpl.NewResolver`. That means you can write:

```yaml
mutation:
  rules:
    - field: spec.owner
      default: "{{ .metadata.namespace }}/{{ .metadata.name }}"
```

and the resolved value will be the actual namespace and name of the incoming CR.

---

## Validation

### What it does

Validation checks that fields on a CR satisfy declared constraints. A failed rule with `action: deny` blocks the operation. A failed rule with `action: warn` surfaces a warning but allows it through.

### Two execution paths

**Path 1 — Admission webhook (`/validate`)**

```
POST /validate  (HealthServer.validationHandler)
  │
  ├─ decode AdmissionReview
  ├─ look up ValidationConfig from InMemoryAdmissionRegistry
  │    → GetValidationRules(gvrKey)
  ├─ evaluateValidationRules(obj, cfg, kindName) ← pkg/health/admission_evaluation.go
  │    runs every rule — does NOT stop on first failure
  │    splits results into: denials (action: deny) and warnings (action: warn)
  ├─ if warnings: add them to AdmissionResponse.Warnings
  └─ if denials:
       set Allowed = false
       set Status.Message = "[orkestra] validation failed: field X: message (got: Y)"
       HTTP 400 visible to kubectl apply caller
```

**Path 2 — Reconcile loop (`runValidation`)**

```
runValidation(obj, cfg, crdName)  ← pkg/reconciler/run_validations.go
  │
  ├─ same evaluation logic as Path 1
  ├─ returns ValidationResult{Passed, Violations, Warnings}
  └─ caller (generic.go / hooks) decides what to do:
       Passed == false → halt reconciliation, emit event
```

### Operators available

| YAML shorthand  | Operator constant         | Meaning                              |
|-----------------|---------------------------|--------------------------------------|
| `equals:`       | `ConditionEquals`         | field must exactly match             |
| `notEquals:`    | `ConditionNotEquals`      | field must not equal                 |
| `prefix:`       | `ConditionPrefix`         | field must start with                |
| `suffix:`       | `ConditionSuffix`         | field must end with                  |
| `contains:`     | `ConditionContains`       | field must contain substring         |
| `min:`          | `ConditionGt` (>=)        | numeric field must be >= value       |
| `max:`          | `ConditionLt` (<=)        | numeric field must be <= value       |
| `operator: exists`    | `ConditionExists`   | field must be present and non-empty  |
| `operator: notExists` | `ConditionNotExists`| field must be absent or empty        |

All rules for a CRD are evaluated before returning — the user gets a complete list of violations in a single rejection, not one error per apply.

### The `action:` flag

```yaml
- field: metadata.labels.team
  operator: exists
  message: "all resources should declare a team label"
  action: warn    # advisory — does not block
```

`action: deny` is the default when the field is omitted. Every new rule blocks unless you explicitly mark it as advisory.

---

## Conversion

### What it does

Conversion translates a CR from one `apiVersion` to another. It runs only when multiple versions of a CRD are registered and Kubernetes needs to convert a stored object to a different version — for example, when you upgrade from `v1alpha1` to `v1` and existing objects need to be served in the new format.

### Execution path

```
POST /convert  (HealthServer.conversionHandler)  ← pkg/health/conversion.go
  │
  ├─ decode ConversionReview (apiextensions.k8s.io/v1)
  ├─ for each object in review.Request.Objects:
  │    ├─ extract sourceVersion from object.apiVersion
  │    ├─ extract targetVersion from review.Request.DesiredAPIVersion
  │    ├─ if source == target: copy object, update apiVersion, done
  │    ├─ look up rules from ConversionRegistry by kind
  │    │    → GetConversionRules(kind)
  │    ├─ applyConversion(obj, rules, targetAPIVersion) ← pkg/health/conversion_logic.go
  │    │    ├─ FindPath(sourceVersion, targetVersion)  ← pkg/types/conversion.go
  │    │    ├─ build orktmpl.Resolver from source object
  │    │    └─ resolveMap(resolver, path.Spec)
  │    │         walks the declared `spec:` block, resolves any {{ .spec.X }} expressions
  │    └─ emit converted object with updated apiVersion
  └─ return ConversionReview with convertedObjects
```

### Declaring a conversion path

```yaml
conversion:
  paths:
    - from: v1alpha1
      to: v1
      spec:
        # Declare every field the new version has.
        # Reference the source object's fields with Go template syntax.
        image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        # Computed or renamed fields:
        fullName: "{{ .metadata.namespace }}/{{ .metadata.name }}"
```

If no path exists for a given `(from, to)` pair, conversion fails with a clear error message that shows exactly which path to add.

---

## The ordering — when each runs

The Kubernetes admission chain for a `CREATE` or `UPDATE` is fixed:

```
kubectl apply
    │
    ▼
Kubernetes API server
    │
    ├─ 1. MutatingAdmissionWebhook   → POST /mutate   (Orkestra applies defaults)
    │
    ├─ 2. Object written to etcd (with patch applied)
    │
    ├─ 3. ValidatingAdmissionWebhook → POST /validate  (Orkestra checks rules)
    │        denied  → HTTP 400, kubectl sees error, object NOT stored
    │        allowed → object confirmed in etcd
    │
    └─ 4. Reconcile loop picks up the object
              runMutation(...)      ← catches anything missed at admission
              runValidation(...)    ← halts reconcile on violation
              runTemplateReconcile  ← creates Deployments, Services, etc.
```

Mutation runs first so that defaults are already in place when validation evaluates them. Conversion is separate and runs only when `GET`/`LIST` requests ask for a different API version than what is stored.

---

## Where the code lives

| What                          | File                                                              |
|-------------------------------|-------------------------------------------------------------------|
| Rule declarations (types)     | [pkg/types/admission.go](https://github.com/iAlexeze/orkestra/blob/main/pkg/types/admission.go)            |
| Conversion types              | [pkg/types/conversion.go](https://github.com/iAlexeze/orkestra/blob/main/pkg/types/conversion.go)          |
| Registry interface + impl     | [pkg/katalog/admission_registry.go](https://github.com/iAlexeze/orkestra/blob/main/pkg/katalog/admission_registry.go) |
| Conversion registry           | [pkg/katalog/conversion_registry.go](https://github.com/iAlexeze/orkestra/blob/main/pkg/katalog/conversion_registry.go) |
| Webhook HTTP handlers         | [pkg/health/admission_handlers.go](https://github.com/iAlexeze/orkestra/blob/main/pkg/health/admission_handlers.go) |
| Rule evaluation (webhook)     | [pkg/health/admission_evaluation.go](https://github.com/iAlexeze/orkestra/blob/main/pkg/health/admission_evaluation.go) |
| Conversion HTTP handler       | [pkg/health/conversion.go](https://github.com/iAlexeze/orkestra/blob/main/pkg/health/conversion.go)        |
| Conversion logic              | [pkg/health/conversion_logic.go](https://github.com/iAlexeze/orkestra/blob/main/pkg/health/conversion_logic.go) |
| Webhook server startup        | [pkg/health/health.go](https://github.com/iAlexeze/orkestra/blob/main/pkg/health/health.go)                |
| runValidation (reconcile)     | [pkg/reconciler/run_validations.go](https://github.com/iAlexeze/orkestra/blob/main/pkg/reconciler/run_validations.go) |
| runMutation (reconcile)       | [pkg/reconciler/run_mutations.go](https://github.com/iAlexeze/orkestra/blob/main/pkg/reconciler/run_mutations.go) |
| Reconcile dispatcher          | [pkg/reconciler/generic.go](https://github.com/iAlexeze/orkestra/blob/main/pkg/reconciler/generic.go)      |

---

## The shared evaluation logic — and the duplication

The evaluation logic in `admission_evaluation.go` (webhook path) and `run_validations.go` / `run_mutations.go` (reconcile path) is intentionally parallel: same operators, same field resolver, same rule structure. They are not shared via a common function today because the webhook path works on a raw `map[string]interface{}` decoded from JSON, while the reconcile path works on `*unstructured.Unstructured`. The underlying field traversal (`resolveFieldPath` vs `resolveField`) and operator switch are the same pattern implemented twice.

If you add a new operator, add it in both places.

---

## Enabling webhooks

Webhooks are opt-in. They require TLS certificates and are disabled by default.

```bash
export ENABLE_ADMISSION_WEBHOOK=true
export TLS_CERT=/path/to/tls.crt
export TLS_KEY=/path/to/tls.key
```

At startup, `health.go` registers `/validate` and `/mutate` on the HTTPS server (`:8443`), then calls `RegisterWebhooks` to create the `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` objects in the cluster. Without this, the API server does not know to call Orkestra and admission interception does not happen.

Conversion is enabled separately:

```bash
export ENABLE_CONVERSION=true
```

This registers `/convert` on the same HTTPS server. The `ConversionWebhook` field on the CRD definition must point to Orkestra's service for the API server to route conversion requests here.
