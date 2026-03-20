# pkg/reconciler

The reconciler package is the heart of Orkestra's runtime. It is where a
Kubernetes event — "this CR changed" — becomes a real-world outcome: a
Deployment was created, a Secret was copied, a Job ran cleanup.

This document explains how the reconciler works, why it is structured the
way it is, and how to extend it.

---

## The journey of a reconcile

A CR event enters the system and travels a fixed path before any resource
is touched. Understanding this path is the foundation for everything else.

```
Kubernetes API server emits event (Add / Update / Delete)
        │
        ▼
Informer receives the event and places the object key into the workqueue
        │
        ▼
Worker goroutine dequeues the key and calls Reconcile(ctx, key)
        │
        ▼
generic.go reads the object from the informer cache
  ├── not found in cache → deleted, call OnNotFound hook if set, done
  ├── DeletionTimestamp set → handleDeletion()
  └── exists and live → ensureFinalizers() → reconcile()
        │
        ▼
reconcile() dispatches to the correct implementation
  ├── Go hooks declared → r.hooks.OnReconcile(ctx, obj)
  ├── Templates declared → runTemplateReconcile(ctx, obj)
  └── Neither → no-op (finalizers and events still handled)
        │
        ▼
runTemplateReconcile() calls each resource runner in order
  ├── runDeployments()
  ├── runServices()
  ├── runSecrets()
  ├── runConfigMaps()
  ├── runServiceAccounts()
  └── runCronJobs()
        │
        ▼
Each runner resolves template expressions then calls the OrkestraRegistry
  ├── resolver.ResolveXxxTemplate(src) → literal values
  └── orkxxx.Create() / orkxxx.Update() → Kubernetes API write
```

Every step is logged. Every state transition fires a Kubernetes event.
Every reconcile is measured by Prometheus. None of that is the concern
of the resource runners — it is handled once in `generic.go`.

---

## File structure

```
pkg/reconciler/
  generic.go              Pure dispatcher — owns the reconcile lifecycle
  run_deployments.go      Deployment create/update
  run_services.go         Service create/update
  run_secrets.go          Secret create/copy/sync + toNamespaces
  run_configmaps.go       ConfigMap create/copy/sync + toNamespaces
  run_serviceaccounts.go  ServiceAccount create (no update — nothing drifts)
  run_jobs.go             Job create (onDelete cleanup)
  run_cronjobs.go         CronJob create/update
```

This separation is intentional and enforced. `generic.go` is a dispatcher.
It contains zero resource-specific logic. When you add a new resource type,
`generic.go` gets one new line. The resource logic lives entirely in its
own file.

---

## generic.go — the dispatcher

`generic.go` is responsible for the reconcile lifecycle. It does not know
what a Deployment or a Secret is. It knows three things:

**1. How to read safely from the cache**

```go
raw, exists, err := r.informer.GetIndexer().GetByKey(key)
```

All reads go through the informer's local store. The API server is never
called for reads. This is fundamental to Kubernetes controller correctness —
the informer cache is the source of truth during reconciliation.

**2. How to route the reconcile**

```go
switch {
case r.hooks.OnReconcile != nil:
    // Go hooks — user-provided, full type-safe access
case r.rc.OnCreate != nil || r.rc.OnReconcile != nil:
    // Declarative templates — interpreted at runtime
default:
    // No-op — finalizers and events still handled
}
```

The switch is the entire reconcile dispatch. Nothing else belongs here.

**3. How to manage finalizers**

Finalizers are the mechanism that lets Orkestra run cleanup before
Kubernetes garbage collects a CR. `generic.go` owns finalizer add and
remove. Resource runners never touch finalizers.

The deletion path is strict:

```
DeletionTimestamp set
      │
      ▼
run OnDelete hook OR onDelete templates (cleanup)
      │
      ├── cleanup failed → return error, retry on next reconcile
      │                    finalizers NOT removed, object stays protected
      │
      └── cleanup succeeded → remove our finalizers → Kubernetes GC proceeds
```

Finalizers are never removed on error. The object stays protected until
cleanup succeeds and returns nil.

---

## runTemplateReconcile — the resource dispatch

`runTemplateReconcile` is the bridge between `generic.go` and the
individual resource runners. It reads the Katalog's template declarations
and calls the appropriate function for each resource type.

```go
func (r *GenericReconciler[T]) runTemplateReconcile(ctx context.Context, obj domain.Object) error {
    // ...
    if t := r.rc.OnCreate; t != nil {
        runDeployments(ctx, kube, resolver, obj, t.Deployments, false)
        runServices(ctx, kube, resolver, obj, t.Services, false)
        runSecrets(ctx, kube, resolver, obj, t.Secrets, false)
        runConfigMaps(ctx, kube, resolver, obj, t.ConfigMaps, false)
        runServiceAccounts(ctx, kube, resolver, obj, t.ServiceAccounts)
        runCronJobs(ctx, kube, resolver, obj, t.CronJobs, false)
    }
    // onReconcile follows the same pattern with update=true
}
```

This function is the only place that knows which resource types exist.
Adding a new resource type adds one call here. Nothing else in `generic.go`
changes.

---

## The resource runners

Each `run_xxx.go` file owns one resource type end to end.

**Anatomy of a resource runner:**

```go
func runDeployments(
    ctx context.Context,
    kube *kubeclient.Kubeclient,
    resolver *orktmpl.Resolver,
    owner domain.Object,
    srcs []orktypes.DeploymentTemplateSource,
    update bool,
) error {
    for i, src := range srcs {
        // 1. Resolve template expressions against the live CR
        resolved, err := resolver.ResolveDeploymentTemplate(src)

        // 2. Build the registry spec from resolved values
        spec := orkdeploy.Resolve(resolved, staticReplicas, resolver.OwnerName())

        // 3. Call the registry — create or update based on the path
        if update {
            orkdeploy.Update(ctx, kube, owner, spec)
        } else {
            orkdeploy.Create(ctx, kube, owner, spec)
            if src.Reconcile {
                orkdeploy.Update(ctx, kube, owner, spec) // reconcile: true shorthand
            }
        }
    }
}
```

Three steps, always in this order:
1. Resolve — evaluate `{{ .spec.field }}` expressions against the CR
2. Build spec — translate resolved values into the registry's spec type
3. Call registry — the registry handles idempotency, owner references, and error wrapping

The runner never handles owner references, never patches finalizers, never
fires events. Those are the dispatcher's concern.

---

## The `update` flag and `reconcile: true`

Every resource runner that supports drift correction takes an `update bool`
parameter.

`update=false` is the **onCreate path** — idempotent create. If the resource
already exists, skip it. If it does not exist, create it.

`update=true` is the **onReconcile path** — drift correction. Always apply
the desired state. If the resource has been manually modified, reconcile it
back.

`reconcile: true` on an onCreate declaration is a shorthand. When set,
the runner calls both Create and Update in the same reconcile loop. The
user declares the resource once under `onCreate` and gets drift correction
without writing a separate `onReconcile` block.

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
      reconcile: true    # ← create it AND keep it in sync
```

ServiceAccounts do not take an `update` parameter — there is nothing
meaningful to update on a ServiceAccount after creation. Jobs do not take
an `update` parameter — they are fire-and-forget, always creates.

---

## Template resolution

Before any registry function is called, all `{{ .spec.field }}` expressions
in the template declaration must be evaluated. This is the resolver's job.

```go
resolver, err := orktmpl.NewResolver(ctx, obj)
resolved, err := resolver.ResolveDeploymentTemplate(src)
```

The resolver builds a `map[string]interface{}` from the CR object and
evaluates Go `text/template` expressions against it. For
`*unstructured.Unstructured` objects (the common case) the full CR
including all spec fields is accessible. The resolver uses `missingkey=zero`
— missing fields resolve to empty string rather than erroring.

After resolution, `resolved` contains only literal values:

```
Before: image: "{{ .spec.image }}"     replicas: "{{ .spec.replicas }}"
After:  image: "nginx:1.25"            replicas: "2"
```

The registry functions never see template expressions. They only receive
literal values. This is a hard boundary — template evaluation happens
entirely in the reconciler, registry functions are pure Kubernetes API calls.

---

## The OrkestraRegistry

Each resource runner calls functions from the OrkestraRegistry — the
package under `pkg/orkestra-registry/`. These functions are the
implementations: they build Kubernetes objects, call the API server,
handle idempotency, and set owner references.

```
pkg/orkestra-registry/
  deployments/    Create, Update, Delete, Resolve
  services/       Create, Update, Delete, Resolve
  secrets/        Create, Update, Delete, Resolve, CopyToNamespaces
  configmaps/     Create, Update, Delete, Resolve, CopyToNamespaces
  serviceaccounts/ Create, Delete, Resolve
  jobs/           Create, Delete, Resolve
  cronjobs/       Create, Update, Delete, Resolve
  template/       Resolver — evaluates template expressions
```

Every registry implementation follows the same contract:

```go
// Create — idempotent. If already exists, return nil (skip, no error).
func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedXxxSpec) error

// Update — reconcile desired state. If not found, create it (drift from deletion).
func Update(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedXxxSpec) error

// Delete — if not found, return nil (idempotent).
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedXxxSpec) error

// Resolve — build a ResolvedXxxSpec from a template source.
// Called by the runner after template expressions are already evaluated.
func Resolve(src orktypes.XxxTemplateSource, ownerName string) ResolvedXxxSpec
```

Owner references are set by every Create call. This means cascade deletion
is automatic — when a CR is deleted, all child resources are garbage
collected by Kubernetes without any explicit onDelete declarations.

---

## Go hooks vs declarative templates

The reconciler supports two reconcile implementations. Understanding when
to use each is important.

**Declarative templates** — the default for new operators.

```yaml
reconciler:
  default: true
  onCreate:
    deployments:
      - image: "{{ .spec.image }}"
        reconcile: true
```

Orkestra interprets these declarations at runtime. No Go code. No
`ork generate runtime`. Works with `*unstructured.Unstructured` — full
access to all CR spec fields via template expressions.

Use this when your operator creates and manages Kubernetes resources
based on CR spec values. This covers the majority of operator use cases.

**Go hooks** — for operators that need full programmatic control.

```yaml
reconciler:
  default: true
  hooks:
    location: github.com/myorg/hooks
    function: WebsiteHooks
```

```go
// hooks/website.go
func WebsiteHooks() domain.AnyReconcileHooks {
    return domain.ReconcileHooks[*apiv1.Website]{
        OnReconcile: func(ctx context.Context, obj *apiv1.Website) error {
            // full type-safe access to obj.Spec.Image, obj.Spec.Replicas etc.
            // call OrkestraRegistry functions directly
            // call external APIs
            // complex conditional logic
        },
    }
}
```

Requires `ork generate runtime` to register the hook function.
Use this when you need type-safe access to compiled Go structs, complex
conditional logic, or calls to external APIs.

**Priority:** Go hooks always run if declared. Templates only run when no
Go hooks are declared. The two are mutually exclusive for a given CRD.

---

## `ork generate runtime` — what it does and when it is needed

`ork generate runtime` reads the Katalog and emits two files:

```
pkg/runtime/__generated_runtime_registry.go   — ObjectRegistry, ListRegistry, RegisterScheme
pkg/runtime/__generated_runtime_hooks.go      — HookRegistry entries for Go hooks
```

It is needed when:

| Situation | generate runtime needed? |
|-----------|--------------------------|
| Dynamic CRD with declarative templates only | **No** |
| Dynamic CRD with Go hooks declared | **Yes** — registers hook in HookRegistry |
| Typed CRD (`apiTypes.location` set) | **Yes** — ObjectRegistry + RegisterScheme |
| Custom constructor (`reconciler.default: false`) | **Yes** — registers constructor |

For pure dynamic operators using only `onCreate`/`onReconcile` template
declarations, `ork generate runtime` is never needed. The reconciler reads
the Katalog directly and interprets the declarations at runtime.

---

## Contributing a new resource type

Adding a new resource type to the reconciler is a four-step process.
No existing files change except `generic.go` (one line) and the
`HookTemplates` struct (one field).

**Step 1 — Add to `orktypes.HookTemplates`**

```go
// pkg/types/types.go
type HookTemplates struct {
    Deployments     []DeploymentTemplateSource
    Services        []ServiceTemplateSource
    Secrets         []SecretTemplateSource
    ConfigMaps      []ConfigMapTemplateSource
    ServiceAccounts []ServiceAccountTemplateSource
    Jobs            []JobTemplateSource
    CronJobs        []CronJobTemplateSource
    Ingresses       []IngressTemplateSource   // ← new
}
```

**Step 2 — Add the template source type**

```go
// pkg/types/types.go
type IngressTemplateSource struct {
    Version    string          `yaml:"version"    validate:"omitempty"`
    Name       string          `yaml:"name"       validate:"omitempty"`
    Namespace  string          `yaml:"namespace"  validate:"omitempty"`
    Host       string          `yaml:"host"       validate:"required"`
    Path       string          `yaml:"path"       validate:"omitempty"`
    ServiceName string         `yaml:"serviceName" validate:"required"`
    ServicePort string         `yaml:"servicePort" validate:"required"`
    TLSSecret  string          `yaml:"tlsSecret"  validate:"omitempty"`
    Labels     []ResourceLabel `yaml:"labels"     validate:"omitempty"`
    Reconcile  bool            `yaml:"reconcile"  validate:"omitempty"`
}
```

**Step 3 — Add resolver method**

```go
// pkg/orkestra-registry/template/resolver.go
func (r *Resolver) ResolveIngressTemplate(src orktypes.IngressTemplateSource) (orktypes.IngressTemplateSource, error) {
    resolved := orktypes.IngressTemplateSource{Version: src.Version}
    var err error

    if resolved.Name, err = r.Resolve(src.Name); err != nil {
        return resolved, fmt.Errorf("ingress.name: %w", err)
    }
    if resolved.Host, err = r.Resolve(src.Host); err != nil {
        return resolved, fmt.Errorf("ingress.host: %w", err)
    }
    if resolved.ServiceName, err = r.Resolve(src.ServiceName); err != nil {
        return resolved, fmt.Errorf("ingress.serviceName: %w", err)
    }
    if resolved.ServicePort, err = r.Resolve(src.ServicePort); err != nil {
        return resolved, fmt.Errorf("ingress.servicePort: %w", err)
    }

    ns := src.Namespace
    if ns == "" {
        ns = "{{ .metadata.namespace }}"
    }
    if resolved.Namespace, err = r.Resolve(ns); err != nil {
        return resolved, fmt.Errorf("ingress.namespace: %w", err)
    }
    if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
        return resolved, fmt.Errorf("ingress.labels: %w", err)
    }

    return resolved, nil
}
```

**Step 4 — Add the registry implementation**

```
pkg/orkestra-registry/ingresses/
  types.go       ResolvedIngressSpec
  ingress.go     Create, Update, Delete, Resolve
```

Following the same contract as every other registry implementation:
- `Create` is idempotent — skip if exists
- `Update` recreates if not found (drift from deletion)
- `Delete` is idempotent — skip if not found
- `Resolve` builds `ResolvedIngressSpec` from the template source
- All `Create` calls set owner references for cascade deletion

**Step 5 — Add the runner**

```go
// pkg/reconciler/run_ingresses.go
package reconciler

func runIngresses(
    ctx context.Context,
    kube *kubeclient.Kubeclient,
    resolver *orktmpl.Resolver,
    owner domain.Object,
    srcs []orktypes.IngressTemplateSource,
    update bool,
) error {
    for i, src := range srcs {
        resolved, err := resolver.ResolveIngressTemplate(src)
        if err != nil {
            return fmt.Errorf("ingresses[%d]: %w", i, err)
        }

        spec := orkingress.Resolve(resolved, resolver.OwnerName())

        if update {
            if err := orkingress.Update(ctx, kube, owner, spec); err != nil {
                return fmt.Errorf("ingresses[%d].update: %w", i, err)
            }
        } else {
            if err := orkingress.Create(ctx, kube, owner, spec); err != nil {
                return fmt.Errorf("ingresses[%d].create: %w", i, err)
            }
            if src.Reconcile {
                if err := orkingress.Update(ctx, kube, owner, spec); err != nil {
                    return fmt.Errorf("ingresses[%d].reconcile: %w", i, err)
                }
            }
        }
    }
    return nil
}
```

**Step 6 — Call it from `runTemplateReconcile` in `generic.go`**

```go
// generic.go — runTemplateReconcile
if t := r.rc.OnCreate; t != nil {
    // ... existing resource runners ...
    if err := runIngresses(ctx, kube, resolver, obj, t.Ingresses, false); err != nil {
        return err
    }
}

if t := r.rc.OnReconcile; t != nil {
    // ... existing resource runners ...
    if err := runIngresses(ctx, kube, resolver, obj, t.Ingresses, true); err != nil {
        return err
    }
}
```

That is the complete contribution. Six focused changes, all contained,
nothing broken.

The Katalog immediately supports the new resource type:

```yaml
onCreate:
  ingresses:
    - name: "{{ .metadata.name }}-ingress"
      host: "{{ .spec.hostname }}"
      serviceName: "{{ .metadata.name }}-svc"
      servicePort: "80"
      tlsSecret: "{{ .spec.tlsSecret }}"
      reconcile: true
```

---

## What the reconciler does not do

Understanding the boundaries is as important as understanding the implementation.

**The reconciler does not read from the API server.** All reads go through
the informer cache. `GetIndexer().GetByKey()` is the only read path.

**The reconciler does not set status.** Status updates are the responsibility
of the registry implementation or the Go hook. The generic reconciler has
no knowledge of the CR's status subresource.

**The reconciler does not retry.** The workqueue handles retries with
exponential backoff. If `Reconcile()` returns an error, the key is
re-queued automatically. The reconciler returns errors and trusts the
workqueue.

**The reconciler does not own event firing for business logic.** Generic
lifecycle events (reconciled, deleted, finalizer added) are fired in
`generic.go`. Business logic events (e.g. "image updated") belong in Go
hooks or registry implementations.

**The reconciler does not handle its own panics.** `safeReconcile` in the
worker goroutine wraps every `Reconcile()` call in a panic recovery. The
reconciler can panic safely — it will be caught, logged, and the worker
will continue.