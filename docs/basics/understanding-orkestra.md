# Understanding Orkestra

Orkestra is a declarative operator runtime. It takes the problem described in
[Kubernetes Basics](./kubernetes-basics.md) — operators are hard to write — and
removes it. Not by making the code easier to write. By removing the need to write
code at all.

---

## The shift

Every operator framework before Orkestra accepted a premise: operators are software
projects. You scaffold a project, write Go, compile a binary, build an image, deploy
it. The frameworks made this faster. None made it unnecessary.

Orkestra starts from a different question: **if the CRD already describes the schema,
what else does the cluster actually need to manage it?**

The answer is a runtime and a declaration. The code that would have been written
does not need to exist.

---

## What you write

You write a **Katalog** — a YAML file that declares what a CRD should do:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: website-operator

spec:
  crds:
    website:
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites

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

You run:

```bash
ork run --katalog katalog.yaml
```

That is the operator.

---

## What Orkestra provides automatically

For every CRD entry in a Katalog, Orkestra creates a complete, isolated operator stack:

**Informer** — watches the exact GVK you declared. The API server notifies Orkestra
the instant a `Website` object is created, updated, or deleted.

**Workqueue** — buffers and deduplicates events. Three rapid updates to the same
`Website` produce one reconcile, not three.

**Worker pool** — multiple goroutines process the queue concurrently. Configure with
`workers: 4`. Isolated — another CRD's workload cannot consume this CRD's workers.

**Template resolver** — evaluates `{{ .spec.image }}` against the live CR object at
reconcile time. The image value is whatever the user wrote in their CR spec.

**OrkestraRegistry calls** — creates the Deployment and Service via production-ready
functions that set owner references, system labels, and handle idempotency.

**Owner references** — every child resource (Deployment, Service) has an owner
reference pointing to the CR. When the CR is deleted, Kubernetes automatically deletes
all its children.

**Finalizer management** — Orkestra adds a finalizer to every CR it manages. This
prevents dirty deletion — the CR will not be removed until Orkestra has completed its
cleanup.

**Kubernetes events** — every reconcile emits a `Normal` event on the CR. Failures
emit a `Warning` event. `kubectl describe website my-site` shows what happened.

**Prometheus metrics** — `controller_reconcile_total`, `controller_reconcile_duration_seconds`,
`controller_queue_depth`, and more — all labeled by GVK. Out of the box.

**Health API** — `/katalog/website/health` returns 200 when healthy and 503 when
degraded. `/katalog/website` returns live statistics: queue depth, worker count,
reconcile totals, error rate.

**Leader election** — only one Orkestra instance actively reconciles at any time.
Followers maintain warm caches and take over within seconds if the leader fails.

**Drift correction** — `reconcile: true` on a template resource means it is
re-applied on every reconcile cycle, not just on creation. If someone manually deletes
the Deployment, Orkestra recreates it.

None of this is written by you. All of it is provided by declaring a Katalog entry.

---

## The template expressions

The `{{ .spec.image }}` syntax is standard Go `text/template` evaluated against the
live CR object. Every field in the CR is accessible:

```
{{ .metadata.name }}        CR name
{{ .metadata.namespace }}   CR namespace
{{ .spec.image }}           spec field
{{ .spec.replicas }}        another spec field
{{ .spec.db.host }}         nested spec field
```

A plain string without `{{` is a static value — no evaluation, returned as-is.
The fast path means there is no performance cost to declaring static values alongside
dynamic ones.

---

## What happens when you apply a CR

```bash
kubectl apply -f website.yaml
```

1. The API server stores the `Website` CR in etcd
2. Orkestra's informer receives the watch event
3. The key `default/my-website` is added to the workqueue
4. A worker dequeues the item
5. The worker reads the CR from the informer cache — no API call
6. Template expressions are resolved against the CR's fields
7. The Deployment and Service are created via the OrkestraRegistry
8. Owner references are set on both child resources
9. A `Reconciled` event is emitted on the CR
10. Metrics are updated: `controller_reconcile_total{result="success"} +1`

The entire path — from `kubectl apply` to Deployment existing — takes under a second
on a healthy cluster.

---

## Multiple CRDs, one runtime

Add more CRD entries to the Katalog:

```yaml
spec:
  crds:
    website:
      # ...
    application:
      dependsOn: [website]
      # ...
    platform-namespace:
      # ...
```

Each CRD gets its own complete operator stack. The `application` CRD waits for
`website` to be ready before starting its workers — because of `dependsOn`. This is
dependency ordering: if `application` depends on `website`, Orkestra starts `website`
first and waits for its informer to sync before starting `application`.

All three run in one Orkestra process. One deployment to manage. One health endpoint
to monitor. One upgrade to perform.

---

## When you do need Go

Orkestra's declarative layer covers the common case — creating and drift-correcting
Kubernetes resources from CR fields. Some use cases need more:

- Calling an external API (create a database user inside PostgreSQL)
- Complex conditional logic beyond `when:` conditions
- Type-safe struct field access for performance-critical reconcilers

For these, **hooks** are the answer. A hook is a Go function you write that runs
inside the reconcile loop. Orkestra still provides the informer, queue, workers,
finalizers, events, and metrics. You provide the logic.

```go
func WebsiteHooks() domain.AnyReconcileHooks {
    return domain.ReconcileHooks[*apiv1.Website]{
        OnReconcile: func(ctx context.Context, obj *apiv1.Website) error {
            // type-safe access: obj.Spec.Image, obj.Spec.Replicas
            // call external APIs
            // return nil on success, error to requeue
        },
    }
}
```

Hooks are declared in the Katalog:

```yaml
operatorBox:
  default: true
  hooks:
    location: github.com/myorg/hooks
    function: WebsiteHooks
```

Hooks and declarative templates coexist. Most operators use templates for the
standard resources and hooks only for the parts that genuinely need Go.

---

## Validating and defaulting CRs

Orkestra can validate and mutate CRs at two points:

**At admission time** — when `ENABLE_ADMISSION_WEBHOOK=true`, Orkestra intercepts `kubectl apply`
and either rejects the CR (deny) or applies defaults (mutation) before it is stored:

```yaml
validation:
  - field: spec.image
    prefix: "myorg/"
    message: "image must be from myorg registry"
    action: deny

mutation:
  - field: spec.replicas
    default: "2"
```

**At reconcile time** — the same rules run on every reconcile cycle, catching
violations on CRs that existed before the rules were added.

Declare once. Enforce at both points.

---

## The mental model

```
Your CRD   — defines the schema, describes your domain
Katalog    — declares how Orkestra should manage the CRD
Orkestra   — runs the operator, continuously reconciling
Kubernetes — stores the CRs and the resources Orkestra creates
```

The CRD is the API. The Katalog is the behavior. Orkestra is the runtime. Kubernetes
is the platform.

You own the CRD. Orkestra handles everything else.

---

## Ready to build

If this makes sense, you are ready to write your first operator.

[Getting Started →](../getting-started/index.md)