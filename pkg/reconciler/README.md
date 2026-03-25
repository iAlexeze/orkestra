# **Reconciler Architecture**

The reconciler is the heart of Orkestra’s runtime. It is where a Kubernetes
event — *“this CR changed”* — becomes real‑world action: a Deployment is
created, a Secret is synced, a Job runs cleanup.

This document explains how the reconciler works, why it is structured this
way, and how to extend it safely.

---

## The Reconcile Journey

Every reconcile follows the same deterministic path:

```
Kubernetes event (Add / Update / Delete)
        │
        ▼
Informer enqueues the key
        │
        ▼
Worker dequeues → Reconcile(ctx, key)
        │
        ▼
generic.go reads from informer cache
  ├── not found → deleted → OnNotFound hook
  ├── deletion timestamp → handleDeletion()
  └── exists → ensureFinalizers() → reconcile()
        │
        ▼
reconcile() dispatches:
  ├── Go hooks (if declared)
  ├── Declarative templates (onCreate/onReconcile)
  └── No-op
        │
        ▼
runTemplateReconcile() calls resource runners:
  deployments, services, secrets, configmaps,
  serviceaccounts, cronjobs, jobs (onDelete)
        │
        ▼
Each runner:
  1. Evaluate conditions (when:)
  2. Resolve template expressions
  3. Build registry spec
  4. Create/Update via OrkestraRegistry
```

Everything else — logging, events, metrics, retries — is handled centrally.

---

## Conditional Provisioning (`when:`)

Resource templates may declare conditions:

```yaml
when:
  - field: spec.enabled
    equals: "true"
  - field: spec.environment
    prefix: "prod"
```

All conditions must pass (AND semantics).  
If any condition fails, the resource is skipped for this reconcile cycle.

### Where conditions are evaluated

Conditions are evaluated **inside each resource runner**, before template
resolution:

```
runDeployments:
    if conditionsPass(src.Conditions, obj):
        resolve template
        apply resource
    else:
        skip
```

### Why typed CRDs always return true

Conditions rely on dot‑notation field access:

```
spec.replicas → obj.Object["spec"]["replicas"]
```

This only works for **unstructured CRDs**.

Typed CRDs are Go structs, not maps. They cannot be evaluated safely using
dot‑notation. Rather than silently skipping resources, Orkestra returns
`true` for typed CRDs and logs a warning:

> “typed object cannot be evaluated — skipping conditions, resource will be created”

Typed controllers should use **Go hooks** for conditional logic.

---

## Declarative Templates vs Go Hooks

Orkestra supports two reconcile modes:

### **Declarative templates**  
For dynamic CRDs and operators without Go code.

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
      reconcile: true
```

Templates are interpreted at runtime.  
Full access to CR fields via `{{ .spec.field }}`.

### **Go hooks**  
For typed CRDs or complex logic.

```go
func WebsiteHooks() domain.AnyReconcileHooks {
    return domain.ReconcileHooks[*apiv1.Website]{
        OnReconcile: func(ctx context.Context, obj *apiv1.Website) error {
            if obj.Spec.Enabled {
                ...
            }
        },
    }
}
```

Go hooks always take priority over templates.

---

## Template Resolution

Before any registry call, all template expressions are resolved:

```
"{{ .spec.image }}" → "nginx:1.25"
"{{ .metadata.name }}-svc" → "frontend-svc"
```

Missing fields resolve to empty string (`missingkey=zero`).

Registry functions never see template expressions — only literal values.

---

## Resource Runners

Each resource type has its own runner:

```
run_deployments.go
run_services.go
run_secrets.go
run_configmaps.go
run_serviceaccounts.go
run_jobs.go
run_cronjobs.go
```

Each runner follows the same pattern:

1. Evaluate conditions  
2. Resolve template  
3. Build registry spec  
4. Create/Update/Delete via OrkestraRegistry  

This keeps the reconciler clean and extensible.

---

## Finalizers and Deletion

Finalizers ensure cleanup happens before Kubernetes garbage collects the CR.

Deletion flow:

```
DeletionTimestamp set
      │
      ▼
run OnDelete hook OR onDelete templates
      │
      ├── cleanup failed → retry, finalizers stay
      └── cleanup succeeded → remove finalizers → GC proceeds
```

Finalizers are never removed on error.

---

## Adding a New Resource Type

To add a new resource type:

1. Add template source type  
2. Add resolver method  
3. Add registry implementation  
4. Add runner (`run_xxx.go`) 
5. Add test in `reconciler_test` directory
5. Add one line to `runTemplateReconcile`  

No other files change.