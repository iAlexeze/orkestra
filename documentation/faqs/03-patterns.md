# Patterns

---

## Can Orkestra manage built-in Kubernetes resources?

Yes. `kind: Deployment`, `kind: Pod`, `kind: Service`, and 30+ other built-in
Kubernetes kinds are supported without declaring group, version, or plural —
Orkestra enriches them automatically from its internal registry:

```yaml
- name: deployment-governance
  apiTypes:
    kind: Deployment   # ← only field needed for built-in kinds
  validation:
    - field: metadata.labels.team
      operator: exists
      message: "all deployments must declare a team owner"
      action: warn
```

!!! tip "Governance without a separate policy engine"
    This is how you apply governance to Kubernetes built-in resources without
    OPA, Kyverno, or a separate admission controller. Orkestra watches the resource,
    validates it at reconcile time, and optionally intercepts at admission time
    when `ENABLE_ADMISSION_WEBHOOK=true` or `security.webhooks.admission.enabled=true`.

Run `ork validate --full` to see exactly what Orkestra resolves for a kind-only declaration.

---

## What is the difference between validation and mutation?

**Validation** evaluates rules against a CR and either blocks it (`action: deny`)
or surfaces an advisory (`action: warn`).

**Mutation** applies defaults and normalisations to a CR before it is stored. Fields
declared with `default:` are set only when absent. Fields declared with `override:`
are always set.

Both run at two points:

- **Admission time** — when `ENABLE_ADMISSION_WEBHOOK=true`, synchronously during `kubectl apply`
- **Reconcile time** — every reconcile cycle, regardless of webhook configuration

Declare once, enforced at both points.

!!! tip "Roll out rules safely"
    Deploy new validation rules with `action: warn` first. Observe
    `controller_admission_validation_violations_total` in Prometheus to understand
    which CRs would be affected. When you are confident, change to `action: deny`.
    The Katalog change takes effect on the next Orkestra restart.

---

## Does `ENABLE_ADMISSION_WEBHOOK=true` block the API server if Orkestra is down?

No — by design. The webhook configuration uses `FailurePolicy: Ignore` by default.
If Orkestra is unreachable when the API server calls `/validate` or `/mutate`, the
operation is allowed through. Validation catches violations at reconcile time when
Orkestra restarts.

```yaml
# To change to blocking behaviour (requires high-availability Orkestra deployment):
webhooks:
  failurePolicy: Fail    # default: Ignore
```

!!! warning "Before setting Fail"
    `FailurePolicy: Fail` means Orkestra's availability directly gates all CR
    deployments. Set it only with multiple Orkestra replicas, a PodDisruptionBudget,
    and confidence that your admission rules are correct. Start with `Ignore`.

---

## How do I use `when:` conditions?

`when:` creates a resource only when a condition is met:

```yaml
services:
  - name: "{{ .metadata.name }}-public"
    port: "80"
    targetPort: "{{ .spec.port }}"
    when:
      - field: spec.exposePublicly
        equals: "true"
```

Multiple conditions in the same `when:` block are ANDed. To express OR, create two
separate resource entries with different `when:` conditions and the same name — only
one will be created, whichever condition is met first.

See [When Conditions](../reference/schema/06-when-conditions.md) for the full reference.

---

## How does `dependsOn` ordering work?

`dependsOn` tells Orkestra to hold one CRD's workers until another reaches a condition:

```yaml
spec:
  crds:
    database:
      workers: 8
    application:
      dependsOn:
        database:
          condition: healthy   # wait until database workers are running and failure-free
    analytics:
      dependsOn:
        database:
          condition: started   # wait only until database workers are running
```

`condition: healthy` waits until the dependency's workers are running and its consecutive failure count is zero. `condition: started` waits only until workers are running — useful when you want ordering without blocking on health.

The bare list form defaults to `started`:

```yaml
dependsOn:
  - database
```

See [CRD Entry Schema](../reference/schema/02-crd-entry.md) for the full reference.

---

## Next

- **[Running](./02-running.md)** — setup, configuration, and operations
- **[Ecosystem](./04-ecosystem.md)** — comparisons and the Kubernetes roadmap
