---
title: "Orkestra Registry"
weight: 162
---

# OrkestraRegistry — Technical Documentation

`pkg/orkestra-registry` is the standard library of Kubernetes resource implementations for Orkestra. It is the layer between the reconciler and the Kubernetes API. When a Katalog declares `onCreate: deployments: - image: "{{ .spec.image }}"`, the template resolver produces a `ResolvedDeploymentSpec`, and `orkdeploy.Create` turns that spec into an actual Kubernetes Deployment.

---

## The contract

Every resource package exports the same four functions and follows the same rules:

```go
// Create — idempotent. If the resource already exists, return nil.
// Never errors on "already exists". Safe to call on every reconcile.
func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedXxxSpec) error

// Update — drift correction. Apply desired state regardless of current state.
// If the resource does not exist, creates it. If it exists but has drifted, corrects it.
// Preferred in onReconcile. Also handles the "create if missing" case.
func Update(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedXxxSpec) error

// Delete — idempotent. If the resource does not exist, return nil.
// Never errors on "not found".
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedXxxSpec) error

// Resolve — build a ResolvedXxxSpec from a template source.
// Template expressions must already be evaluated before calling.
// Applies defaults, parses types, sets system labels.
func Resolve(src orktypes.XxxTemplateSource, ownerName string) ResolvedXxxSpec
```

**The `owner` parameter is always the CR.** Every function that creates a Kubernetes resource sets owner references from the created resource back to `owner`. This means cascade deletion is automatic — when the CR is deleted, all resources created from it are garbage collected by Kubernetes.

**System labels** are set on every created resource and cannot be overridden:

```
managed-by: orkestra
orkestra-owner: <cr-name>
```

`managed-by` identifies Orkestra-managed resources globally. `orkestra-owner` is used as the pod selector on Services — it ensures a Service routes only to pods owned by the same CR, not to pods from a different CR of the same Kind.

---

## The resolver

```go
// pkg/orkestra-registry/template/resolver.go
type Resolver struct {
    data    map[string]interface{}  // the CR object map
    owner   string                  // CR name, used for defaults
}
```

The Resolver evaluates Go `text/template` expressions against the CR's unstructured map. It is built once per reconcile cycle and shared across all template evaluations for that reconcile:

```go
// Built from an unstructured object (dynamic mode)
resolver, err := orktmpl.NewResolver(ctx, obj)

// Built from a plain map (used by admission mutation handler)
resolver := orktmpl.NewResolverFromMap(obj.Object)
```

### Fast path

Any string without `{{` is returned as-is without template parsing. This is the fast path for static values — the majority of fields in most Katalogs.

```go
func (r *Resolver) Resolve(tmpl string) (string, error) {
    if !strings.Contains(tmpl, "{{") {
        return tmpl, nil  // static value — no parsing
    }
    return r.evaluate(tmpl)
}
```

### Template context

All CR fields are accessible in templates:

```
.metadata.name          CR name
.metadata.namespace     CR namespace
.metadata.labels        CR labels (map)
.metadata.annotations   CR annotations (map)
.spec.*                 any spec field (unstructured only)
.status.*               any status field
```

Missing fields resolve to empty string (`missingkey=zero`). No error on absent optional fields.

### Per-type resolve methods

Each resource type has a dedicated resolve method that resolves all its fields in one call and applies namespace defaulting:

```go
func (r *Resolver) ResolveDeploymentTemplate(src orktypes.DeploymentTemplateSource) (orktypes.DeploymentTemplateSource, error)
func (r *Resolver) ResolveServiceTemplate(src orktypes.ServiceTemplateSource) (orktypes.ServiceTemplateSource, error)
// ... one per resource type
```

If `src.Namespace` is empty after resolution, the method defaults it to `{{ .metadata.namespace }}`. You almost never need to declare namespace explicitly for namespaced CRDs.

---

## Packages

### `deployments/`

Manages Deployment lifecycle.

```go
type ResolvedDeploymentSpec struct {
    Name        string
    Namespace   string
    Image       string
    Replicas    int32
    Port        int32              // optional
    Labels      map[string]string
    Annotations map[string]string
    Resources   *ResourceRequirements
}
```

**Create:** Checks if a Deployment with the same name exists in the same namespace. If it exists, returns nil (idempotent). If it does not, creates it with owner references, system labels, and all declared labels.

**Update:** Gets the current Deployment. Compares image and replica count. If either has drifted, patches the Deployment. Declared labels and annotations are merged — system labels cannot be removed.

**Delete:** Deletes the Deployment by name and namespace. If not found, returns nil.

**Drift detection fields:** image and replicas. Other fields (resource limits, annotations beyond system ones) are not drift-corrected by default unless declared under `onReconcile` separately.

---

### `services/`

Manages Service lifecycle.

```go
type ResolvedServiceSpec struct {
    Name       string
    Namespace  string
    Type       string   // ClusterIP, NodePort, LoadBalancer
    Port       int32
    TargetPort int32
    Labels     map[string]string
}
```

The selector is always `orkestra-owner: <cr-name>` — it cannot be overridden. This selector is set by `Resolve()` before the spec reaches `Create`. It ensures the Service routes to pods created by the Deployment registry for the same CR, even if the pod labels differ.

**Drift detection fields:** port and targetPort. Service type is not drift-corrected after creation — changing Service type requires delete-and-recreate, which would briefly disrupt traffic. If you need to change Service type, delete the CR and recreate it.

---

### `secrets/`

Manages Secret lifecycle. Supports three patterns:

**Pattern 1 — Static data:**
```yaml
secrets:
  - name: "{{ .metadata.name }}-config"
    data:
      API_KEY: my-key
```
Creates a Secret with literal data. Values in `data` are not template-evaluated — they are static strings.

**Pattern 2 — Copy from source:**
```yaml
secrets:
  - name: db-creds
    fromSecret: master-db-creds
    fromNamespace: platform
    reconcile: true
```
`Create` / `Update` reads the source Secret from `fromNamespace` and creates a copy in the target namespace. Owner references point to the CR — when the CR is deleted, the copy is deleted. The source is not affected.

**Pattern 3 — Distribute to multiple namespaces:**

```go
orksecrets.CopyToNamespaces(ctx, kube, obj, spec, []string{
  "ns-a", 
  "ns-b", 
  "ns-c",
  "ns-d",
})
```

Reads the source once. Creates copies in each namespace. More efficient than calling `Create` per namespace.

---

### `configmaps/`

Same three patterns as secrets. Additionally supports the merge pattern:

```yaml
configMaps:
  - name: app-config
    fromConfigMap: base-config
    fromNamespace: platform
    data:
      LOG_LEVEL: "{{ .spec.logLevel }}"  # overrides the base value
    reconcile: true
```

Source keys are copied first. Declared `data` keys override matching source keys. The result is a merged ConfigMap that inherits platform defaults and applies CR-specific overrides.

---

### `jobs/`

Manages Job lifecycle. Jobs are fire-and-forget — `Create` creates, there is no `Update`.

**Critical behaviour for `onDelete` jobs:**

Owner references on `onDelete` Jobs have `blockOwnerDeletion: true`. This means Kubernetes will not proceed with deleting the CR until the Job completes. This is the mechanism that ensures cleanup runs before the CR is removed.

```
kubectl delete website my-site
  │
  ▼
DeletionTimestamp set on CR
  │
  ▼
Reconciler detects deletion → runs onDelete templates
  │
  ▼
Job created with owner reference (blockOwnerDeletion: true)
  │
  ▼
Kubernetes holds CR in terminating state
  │
  ▼
Job completes → finalizers removed → CR deleted
```

---

### `cronjobs/`, `pods/`, `serviceaccounts/`

Follow the same four-function contract. CronJobs support drift detection on schedule and image. Pods are immutable after creation — image drift triggers delete-and-recreate. ServiceAccounts have no meaningful drift after creation.

---

## How the reconciler calls the registry

The `runTemplateReconcile` function in `pkg/reconciler/generic.go` is the bridge:

```go
func (r *GenericReconciler[T]) runTemplateReconcile(ctx context.Context, obj T) error {
    resolver, err := orktmpl.NewResolver(ctx, obj)
    if err != nil { return err }

    templates := r.crdInfo.ReconcilerConfig.OnCreate  // or OnReconcile or OnDelete

    // For each deployment template:
    for i, src := range templates.Deployments {
        // Evaluate conditions — skip if any fails
        if !evaluateConditions(obj, src.Conditions) { continue }

        // Resolve template expressions
        resolved, err := resolver.ResolveDeploymentTemplate(src)
        if err != nil { return fmt.Errorf("deployments[%d]: %w", i, err) }

        // Build the resolved spec
        spec := orkdeploy.Resolve(resolved, resolver.OwnerName())

        // Call the registry
        if update {
            if err := orkdeploy.Update(ctx, r.kube, obj, spec); err != nil {
                return fmt.Errorf("deployments[%d]: update: %w", i, err)
            }
        } else {
            if err := orkdeploy.Create(ctx, r.kube, obj, spec); err != nil {
                return fmt.Errorf("deployments[%d]: create: %w", i, err)
            }
        }
    }
    // Same pattern for services, secrets, configmaps, etc.
}
```

The `update` flag is `true` for `onReconcile` — drift correction always uses `Update`. It is `false` for `onCreate` unless `reconcile: true` is set on the template, in which case `Update` is also called from `onCreate`.

---

## Adding a new resource type

See [Contributing — Adding a resource type](../technical-docs/CONTRIBUTING.md#adding-a-resource-type-to-orkestraregistry) for the complete step-by-step.

{{< callout type="note" title="The short version" >}}
To add the `XxxTemplateSource` type,

- Write `Create/Update/Delete/Resolve` functions following the contract above

- Add a `ResolveXxxTemplate` method to the Resolver

- Write a `runXxxs` function following the pattern in `run_deployments.go`, and call it from `runTemplateReconcile`.
{{< /callout >}}

    - Write unit test for the new resource type.
