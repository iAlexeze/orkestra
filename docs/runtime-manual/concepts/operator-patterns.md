# Operator Patterns

Not all operators are the same. Before writing a Katalog, it helps to identify
which pattern your use case belongs to — each maps to a different set of Orkestra
features, and understanding the pattern tells you exactly which features to reach for.

There are five distinct operator patterns. Most real operators are a combination
of two or three, but one pattern is always dominant.

---

## Pattern 1 — Resource factory

**What it is:** A CR declares desired state. The operator creates child Kubernetes
resources that implement that state and keeps them in sync.

**The signature:** "When a `Website` CR exists, there should be a Deployment and a
Service."

**Real examples:** Website operator, application operator, namespace provisioner,
certificate requestor, PVC provisioner.

**Orkestra fit:** Perfect. This is the pattern declarative templates were designed for.

```yaml
operatorBox:
  default: true
  onCreate:
    deployments:
      - image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        reconcile: true
    services:
      - port: "80"
        targetPort: "{{ .spec.port }}"
        reconcile: true
```

`reconcile: true` is the drift correction declaration. The operator does not just
create — it continuously enforces. Delete the Deployment manually; it reappears
on the next reconcile cycle.

**When to add hooks:** When the child resource configuration depends on data not in
the CR spec — querying another resource, computing a value, calling an API.

---

## Pattern 2 — External provisioner

**What it is:** A CR triggers the creation of something outside Kubernetes — a
cloud database, a DNS record, an S3 bucket, an external service account.

**The signature:** "When a `Database` CR exists, there should be a PostgreSQL database
running on RDS."

**Real examples:** AWS RDS operator, Route53 operator, Vault secret operator,
Datadog monitor operator.

**Orkestra fit:** Requires hooks. The external API call cannot be expressed in templates.
Orkestra handles the Kubernetes side (child resources, status, events, finalizers).
Hooks handle the external side.

```go
func DatabaseHooks() domain.AnyReconcileHooks {
    return domain.ReconcileHooks[*apiv1.Database]{
        OnReconcile: func(ctx context.Context, obj *apiv1.Database) error {
            // Create or update the RDS instance
            if err := rds.EnsureInstance(ctx, obj.Spec); err != nil {
                return err
            }
            // Create the Kubernetes Secret with connection details
            // using OrkestraRegistry — owner references set automatically
            return orksecrets.Update(ctx, kube, obj, secretSpec)
        },
        OnDelete: func(ctx context.Context, obj *apiv1.Database) error {
            return rds.DeleteInstance(ctx, obj.Spec.InstanceID)
        },
    }
}
```

**Status pattern:** Set `status.phase` and `status.endpoint` from the external
resource's response. Layer 3 child status propagation is not needed here —
the hook writes status directly.

---

## Pattern 3 — Governance observer

**What it is:** The operator watches existing Kubernetes resources — ones it did
not create — and enforces policy, emits warnings, or creates companion resources.

**The signature:** "Every Deployment in the cluster must have a `team` label."

**Real examples:** Label enforcement, resource limit enforcement, image policy,
companion sidecar injection, cost attribution tagging.

**Orkestra fit:** This is where built-in kind enrichment matters. Declare `kind:
Deployment` with no other `apiTypes` fields — Orkestra enriches group, version,
and plural automatically from the discovery API. Then declare validation and
mutation rules that apply to every Deployment.

```yaml
- name: deployment-governance
  apiTypes:
    kind: Deployment        # built-in — enriched automatically
  
  validation:
    - field: metadata.labels.team
      operator: exists
      message: "all Deployments must declare a team owner"
      action: warn

    - field: spec.template.spec.containers.0.resources.limits.memory
      operator: exists
      message: "memory limits are required"
      action: deny

  mutation:
    - field: metadata.labels.managed-by
      override: "platform-team"
```

**The key insight:** The governance observer does not own the resources it watches.
It imposes policy on resources owned by application teams. Validation with
`action: warn` is advisory — the Deployment is stored but the platform team sees
the violation in `/katalog/deployment-governance`. Validation with `action: deny`
blocks apply-time if `ENABLE_ADMISSION_WEBHOOK=true`.

This pattern replaces a significant portion of what OPA or Kyverno is used for —
at least for resources Orkestra manages — without a separate admission controller.

---

## Pattern 4 — State machine

**What it is:** A CR moves through explicit phases. The operator reads the current
phase from status and drives transitions based on what has been achieved.

**The signature:** "A `Pipeline` CR moves from `Pending` → `Running` → `Succeeded`
or `Failed`. Each transition has conditions and produces different child resources."

**Real examples:** CI/CD pipeline operator, database migration operator,
multi-step provisioning operator, approval workflow operator.

**Orkestra fit:** Requires hooks. The `when:` condition system handles simple
phase gates (create resource X only when `status.phase == Running`). Full state
machine logic — with transition guards, rollback, and conditional next states —
requires a hook.

```go
OnReconcile: func(ctx context.Context, obj *apiv1.Pipeline) error {
    switch obj.Status.Phase {
    case "":
        return r.handlePending(ctx, obj)
    case "Pending":
        return r.handleStartup(ctx, obj)
    case "Running":
        return r.handleRunning(ctx, obj)
    case "Succeeded", "Failed":
        return nil // terminal — nothing to do
    }
}
```

**Status pattern:** The phase string itself is managed by the hook. Layer 2
declarative status fields handle derived fields: `observedSteps`, `startTime`,
`completionTime` — resolved from the CR spec.

**Declarative `when:` for phase gates:**

```yaml
onCreate:
  jobs:
    - name: "{{ .metadata.name }}-migrate"
      image: "{{ .spec.migrationImage }}"
      when:
        - field: status.phase
          equals: "Running"    # only create the Job when the phase is Running
```

---

## Pattern 5 — Coordinator

**What it is:** A CR describes a higher-level intent that decomposes into multiple
other CRs, possibly of different types, in a specific dependency order.

**The signature:** "An `Application` CR creates a `Namespace` CR, a `Database` CR,
and a `Website` CR — in that order, each depending on the previous."

**Real examples:** Application platform operator, environment provisioner,
multi-tier application operator.

**Orkestra fit:** `dependsOn` handles the ordering. Each CRD in the Katalog manages
its own resources. The coordinator CRD creates CRs of the dependent types using
the Secret and ConfigMap patterns (to pass configuration between levels), or hooks
for complex orchestration.

```yaml
spec:
  crds:
    namespace:
      # creates the namespace, sets up RBAC
    
    database:
      dependsOn: [namespace]
      # creates the database after the namespace exists
    
    application:
      dependsOn: [database, namespace]
      # creates the application after database and namespace are ready
```

**The distinction from resource factory:** The coordinator creates CRs, not
Kubernetes resources directly. The child operators then create the Kubernetes
resources from those CRs.

---

## Choosing the right pattern

| Symptom | Pattern |
|---|---|
| Creating Kubernetes resources from CR fields | Resource factory |
| Creating something outside Kubernetes | External provisioner |
| Watching resources you didn't create | Governance observer |
| CR transitions through named phases | State machine |
| CR creates other CRs | Coordinator |

Most production operators combine patterns. A `Database` CR might be an external
provisioner (creates the RDS instance) and a resource factory (creates the
connection Secret and the Service). Identify the dominant pattern first, then
layer the others.

The dominant pattern tells you the primary Orkestra feature to reach for:
templates for factory, hooks for external, built-in enrichment for governance,
`when:` conditions for state machine gates, `dependsOn` for coordination.
