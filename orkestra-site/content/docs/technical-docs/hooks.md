---
title: "Hooks"
weight: 158
---

# Hooks

Hooks are Go functions that run inside the reconcile loop. They give you type-safe access to CR fields, the ability to call external APIs, and complete control over reconcile logic — while Orkestra still handles the lifecycle scaffolding: finalizers, events, metrics, health tracking, and leader election.

---

## When to use hooks vs templates

**Use templates when:**
- You need to create Kubernetes resources (Deployments, Services, Secrets, etc.)
- The resource configuration is derivable from CR fields via template expressions
- You do not need to call external APIs

**Use hooks when:**
- You need to call external APIs (databases, cloud providers, DNS)
- You need type-safe struct field access (`obj.Spec.Image` instead of `"{{ .spec.image }}"`)
- You need conditional logic that cannot be expressed with `when:` conditions
- You need to read state from other Kubernetes resources before deciding what to create

**Hooks and templates can coexist.** A CRD can declare both `hooks` and `onCreate` templates. Hooks run first; templates run for resources not covered by the hook. This lets you handle the complex 20% in Go while the declarative 80% is handled by Orkestra.

---

## Hook interface

```go
// domain/hooks.go

// ReconcileHooks declares the three lifecycle hook functions.
// All are optional — implement only the phases you need.
type ReconcileHooks[T Object] struct {
    OnCreate    func(ctx context.Context, obj T) error
    OnReconcile func(ctx context.Context, obj T) error
    OnDelete    func(ctx context.Context, obj T) error
}

// AnyReconcileHooks is the type-erased version stored in the hook registry.
// Your hook constructor must return this type.
type AnyReconcileHooks interface {
    // internal — implemented by ReconcileHooks[T]
}
```

Your hook function must have this signature:

```go
func MyHooks() domain.AnyReconcileHooks
```

This function is called once at CRD startup to produce the hooks for that CRD. It is registered in the hook registry and looked up by GVK when the reconciler initialises.

---

## Writing a typed hook

A typed hook operates on a concrete Go struct — the most common pattern when you have generated API types.

### Step 1: API types

Your API package must have the generated types. Typically produced by `controller-gen`:

```go
// api/v1alpha1/website_types.go
package v1alpha1

type Website struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec   WebsiteSpec   `json:"spec,omitempty"`
    Status WebsiteStatus `json:"status,omitempty"`
}

type WebsiteSpec struct {
    Image    string `json:"image"`
    Replicas int32  `json:"replicas"`
    Port     int32  `json:"port"`
}
```

### Step 2: Write the hook function

```go
// pkg/hooks/website_hooks.go
package hooks

import (
    "context"
    "fmt"

    "github.com/orkspace/orkestra/domain"
    orkdeploy "github.com/orkspace/orkestra/pkg/orkestra-registry/deployments"
    orksvc    "github.com/orkspace/orkestra/pkg/orkestra-registry/services"
    orktmpl   "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
    "github.com/orkspace/orkestra/pkg/kubeclient"
    apiv1 "github.com/myorg/api/website/v1alpha1"
)

// WebsiteHooks returns the reconcile hooks for the Website CRD.
// Called once at CRD startup — the returned hooks handle all reconcile events.
func WebsiteHooks() domain.AnyReconcileHooks {
    return domain.ReconcileHooks[*apiv1.Website]{
        OnReconcile: reconcileWebsite,
        OnDelete:    deleteWebsite,
    }
}

// reconcileWebsite handles CREATE and UPDATE events.
// obj is fully typed — access spec fields directly.
func reconcileWebsite(ctx context.Context, obj *apiv1.Website) error {
    kube, ok := kubeclient.FromContext(ctx)
    if !ok {
        return fmt.Errorf("no kube client in context")
    }

    // Type-safe field access — no template expressions needed
    depSpec := orkdeploy.ResolvedDeploymentSpec{
        Name:      obj.Name,
        Namespace: obj.Namespace,
        Image:     obj.Spec.Image,     // direct struct field access
        Replicas:  obj.Spec.Replicas,
        Port:      fmt.Sprintf("%d", obj.Spec.Port),
    }

    // OrkestraRegistry handles create/update/owner references/system labels
    if err := orkdeploy.Update(ctx, kube, obj, depSpec); err != nil {
        return fmt.Errorf("deployment: %w", err)
    }

    svcSpec := orksvc.ResolvedServiceSpec{
        Name:       obj.Name + "-svc",
        Namespace:  obj.Namespace,
        Port:       80,
        TargetPort: int32(obj.Spec.Port),
    }

    if err := orksvc.Update(ctx, kube, obj, svcSpec); err != nil {
        return fmt.Errorf("service: %w", err)
    }

    return nil
}

// deleteWebsite handles DELETE events — runs before finalizers are removed.
func deleteWebsite(ctx context.Context, obj *apiv1.Website) error {
    // Call external cleanup APIs here
    // e.g. notify DNS, remove cloud resources, etc.
    // OrkestraRegistry resources are cleaned up automatically via owner references.
    return nil
}
```

### Step 3: Declare in Katalog

```yaml
- name: website
  apiTypes:
    group: demo.orkestra.io
    version: v1alpha1
    kind: Website
    plural: websites
    location: github.com/myorg/api/website/v1alpha1  # required for typed hooks

  operatorBox:
    default: true
    hooks:
      location: github.com/myorg/hooks
      function: WebsiteHooks
```

### Step 4: Generate the runtime registry

```bash
ork generate registry --katalog katalog.yaml
```

This produces `zz_generated_runtime_registry.go` which registers your hook function in `HookRegistry` keyed by GVK. The runtime reads this registry at startup.

---

## Writing a dynamic hook

A dynamic hook operates on `*unstructured.Unstructured` — no generated types needed. Fields are accessed via the template resolver or direct map access.

```go
package hooks

import (
    "context"
    "fmt"

    "github.com/orkspace/orkestra/domain"
    orkdeploy "github.com/orkspace/orkestra/pkg/orkestra-registry/deployments"
    orktmpl   "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
    "github.com/orkspace/orkestra/pkg/kubeclient"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// WebsiteDynamicHooks returns hooks for dynamic mode (no apiTypes.location needed).
func WebsiteDynamicHooks() domain.AnyReconcileHooks {
    return domain.ReconcileHooks[*unstructured.Unstructured]{
        OnReconcile: func(ctx context.Context, obj *unstructured.Unstructured) error {
            kube, _ := kubeclient.FromContext(ctx)

            // Use the template resolver for field extraction
            resolver, err := orktmpl.NewResolver(ctx, obj)
            if err != nil {
                return fmt.Errorf("building resolver: %w", err)
            }

            // Resolve fields from the unstructured map
            image, _ := resolver.Resolve("{{ .spec.image }}")
            replicas, _ := resolver.Resolve("{{ .spec.replicas }}")

            spec := orkdeploy.ResolvedDeploymentSpec{
                Name:      obj.GetName(),
                Namespace: obj.GetNamespace(),
                Image:     image,
                Replicas:  1, // parse replicas from string if needed
            }

            return orkdeploy.Update(ctx, kube, obj, spec)
        },
    }
}
```

Dynamic hooks do not need `apiTypes.location` in the Katalog. They also do not need `ork generate registry` — register them directly in `BuildKatalogFromGo` or `konstructOrkestra`.

{{< callout type="note" title="When to use dynamic hooks" >}}
Dynamic hooks are the right choice when you have no generated API types
and do not want to add `controller-gen` to your build pipeline. They are
slightly less ergonomic than typed hooks (no struct field completion) but
fully capable.
{{< /callout >}}

---

## Using OrkestraRegistry from hooks

The OrkestraRegistry provides production-ready Create, Update, and Delete functions for every common resource type. Always use these instead of raw API calls — they handle owner references, system labels, and idempotency.

```go
import (
    orkdeploy "github.com/orkspace/orkestra/pkg/orkestra-registry/deployments"
    orksvc    "github.com/orkspace/orkestra/pkg/orkestra-registry/services"
    orksecret "github.com/orkspace/orkestra/pkg/orkestra-registry/secrets"
    orkcm     "github.com/orkspace/orkestra/pkg/orkestra-registry/configmaps"
    orkjobs   "github.com/orkspace/orkestra/pkg/orkestra-registry/jobs"
)

// Create: idempotent — safe to call on every reconcile
orkdeploy.Create(ctx, kube, obj, depSpec)

// Update: applies desired state — creates if missing, updates if present
// Preferred for onReconcile (drift correction)
orkdeploy.Update(ctx, kube, obj, depSpec)

// Delete: idempotent — safe even if resource does not exist
orkdeploy.Delete(ctx, kube, obj, depSpec)
```

The `obj` parameter is the owner CR. All registry functions set owner references on the created resource pointing to `obj`. This means:
- Deleting `obj` automatically deletes all resources created from it
- You do not need `onDelete` hooks for registry-managed resources in most cases

---

## The kubeclient context

The Kubernetes client is injected via the context. Always extract it at the start of your hook:

```go
kube, ok := kubeclient.FromContext(ctx)
if !ok {
    return fmt.Errorf("kubeclient not in context — this is a bug in Orkestra startup")
}
```

`kubeclient.FromContext` is never expected to fail in a properly initialised runtime. The check exists to surface misconfiguration early with a clear message rather than a nil pointer panic.

---

## Accessing other Kubernetes resources

To read resources other than the CR being reconciled, use the dynamic client from `kube`:

```go
// Read a Secret from another namespace
secret, err := kube.DynamicClient().
    Resource(schema.GroupVersionResource{Version: "v1", Resource: "secrets"}).
    Namespace("platform").
    Get(ctx, "database-credentials", metav1.GetOptions{})
```

Or use the informer cache from another CRD's informer if it is managed by the same Orkestra instance. Access via the informer registry is faster (no API call) but requires knowing the CRD name.

---

## Testing hooks

Hooks should be tested with a real cluster (kind or minikube is sufficient). The pattern:

1. Apply the CRD: `kubectl apply -f crd.yaml`
2. Start Orkestra with your Katalog: `ork run --katalog katalog.yaml`
3. Apply a CR: `kubectl apply -f cr.yaml`
4. Verify the reconcile: `kubectl get deployments`, `ork events website my-cr`
5. Verify deletion: `kubectl delete -f cr.yaml`, confirm Deployment is gone

For unit tests, mock the `kubeclient.Kubeclient` interface and pass a fake dynamic client. The OrkestraRegistry functions accept a `*kubeclient.Kubeclient` — build a test instance with a `dynamicfake.NewSimpleDynamicClient`.

---

## Common patterns

### Conditional resource creation

```go
OnReconcile: func(ctx context.Context, obj *apiv1.Website) error {
    kube, _ := kubeclient.FromContext(ctx)

    // Create LoadBalancer Service only in production
    if obj.Spec.Environment == "production" {
        spec := orksvc.ResolvedServiceSpec{
            Name:      obj.Name + "-lb",
            Namespace: obj.Namespace,
            Type:      "LoadBalancer",
            Port:      80,
            TargetPort: int32(obj.Spec.Port),
        }
        if err := orksvc.Update(ctx, kube, obj, spec); err != nil {
            return err
        }
    }

    // Always create internal ClusterIP Service
    return orksvc.Update(ctx, kube, obj, internalSpec)
},
```

### External API call

```go
OnReconcile: func(ctx context.Context, obj *apiv1.Database) error {
    kube, _ := kubeclient.FromContext(ctx)

    // Create the Kubernetes resources first
    if err := createDatabaseResources(ctx, kube, obj); err != nil {
        return err
    }

    // Then call the external API
    client := newDatabaseClient(obj.Spec.Host)
    if err := client.CreateDatabase(ctx, obj.Spec.DatabaseName); err != nil {
        // External call failed — the Kubernetes resources exist but the database does not
        // Returning an error requeues the reconcile — it will retry
        return fmt.Errorf("creating database %q: %w", obj.Spec.DatabaseName, err)
    }

    return nil
},
```

### Reading a Secret from another namespace

```go
OnReconcile: func(ctx context.Context, obj *apiv1.Application) error {
    kube, _ := kubeclient.FromContext(ctx)

    // Read credentials from platform namespace
    secretGVR := schema.GroupVersionResource{Version: "v1", Resource: "secrets"}
    raw, err := kube.DynamicClient().Resource(secretGVR).
        Namespace("platform").
        Get(ctx, "db-credentials", metav1.GetOptions{})
    if err != nil {
        return fmt.Errorf("reading db credentials: %w", err)
    }

    // Copy to the application namespace
    spec := orksecret.ResolvedSecretSpec{
        Name:          obj.Name + "-db-creds",
        Namespace:     obj.Namespace,
        FromSecret:    "db-credentials",
        FromNamespace: "platform",
    }
    return orksecret.Update(ctx, kube, obj, spec)
},
```
