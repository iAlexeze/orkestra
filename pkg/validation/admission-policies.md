# Admission Policies in Orkestra

Orkestra can automatically translate your validation and mutation rules into
Kubernetes [ValidatingAdmissionPolicy](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#validatingadmissionpolicy) (VAP) and [MutatingAdmissionPolicy](https://kubernetes.io/docs/reference/access-authn-authz/mutating-admission-policy) (MAP)
for fast, API‑server‑level enforcement.

This is an **optional enhancement** that runs on top of Orkestra's continuous
validation and mutation. If your cluster does not support VAP/MAP (Kubernetes 1.28+),
Orkestra falls back to continuous enforcement only.

---

## How It Works

When you define rules in your Katalog, Orkestra does two things:

| Concern | Admission‑Time (VAP/MAP) | Continuous (Orkestra) |
|---------|--------------------------|-----------------------|
| Validation | ✅ Fast rejection | ✅ Continuous enforcement |
| Mutation | ✅ Apply defaults at admission | ✅ Drift correction |

The user writes one rule. Orkestra handles both.

---

## Mutation Rules

Mutation rules define how to transform resources at admission time.
Orkestra supports the following mutation types:

| Type | Description | Example |
|------|-------------|---------|
| `default` | Apply default if field is empty | `replicas: 1` |
| `set` | Unconditionally set field | `environment: production` |
| `merge` | Merge maps | `labels: {team: platform}` |
| `add` | Add unique values to lists | `regions: [us-east-1]` |

### Example

```yaml
mutation:
  - field: spec.replicas
    type: default
    default: 1
    message: "replicas defaulted to 1"
    phase: admission   # applies at admission and continuously

  - field: spec.environment
    type: set
    value: production
    when: "object.spec.environment == ''"
    message: "environment set to production"
```

---

## Cluster Requirements

- Kubernetes 1.28 or later
- The `admissionregistration.k8s.io/v1` API enabled
- Sufficient RBAC to create VAP and MAP resources

Orkestra checks for VAP/MAP support at startup. If the APIs are not available,
it logs a warning and continues with continuous validation/mutation only.

---

## Disabling Admission Policies

To disable admission policy generation for a specific rule, set `phase: continuous`:

```yaml
validation:
  - field: spec.image
    operator: regex
    pattern: "^myregistry\\.com/.*@sha256:[a-f0-9]{64}$"
    message: "image must be pinned to a digest"
    action: error
    phase: continuous   # only runs in controller
```

To disable for the entire CRD, omit rules from the Katalog.
