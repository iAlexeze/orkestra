# **Extending Orkestra**  
### *How to Add New CRDs, Reconcilers, and Runtime Behavior*

Orkestra is designed so that extending it is **fast**, **predictable**, and **boilerplate‑free**.  
Whether you're adding a new CRD in Go or defining one dynamically through YAML, the workflow is intentionally simple:

> **You write API types and optionally hooks.  
> Orkestra builds the entire operator runtime around them.**

This guide walks you through the full process.

---

# **Overview: What You Need to Provide**

To add a new CRD to Orkestra, you only need to supply:

1. **API Types**  
2. **Optional Hooks** (for custom business logic)  
3. **Katalog Entry** (Go or YAML)  

Everything else — clients, informers, workers, resync intervals, finalizers, events, metrics, health APIs, dependency ordering, lifecycle orchestration — is **generated automatically**.

---

# Step 1 — Create API Types

Create a new directory for your CRD:

```
api/types/yourcrd/v1alpha1/
├── groupversion_info.go
├── yourcrd_types.go
└── zz_generated.deepcopy.go
```

### `groupversion_info.go`

```go
package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/schema"
)

var (
    Group      = "yourgroup.example.com"
    Version    = "v1alpha1"
    APIPath    = "/apis"
    Kind       = "YourCRD"
    NamePlural = "yourcrds"

    GroupVersion = schema.GroupVersion{
        Group:   Group,
        Version: Version,
    }

    SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
    AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
    scheme.AddKnownTypes(GroupVersion,
        &YourCRD{},
        &YourCRDList{},
    )
    metav1.AddToGroupVersion(scheme, GroupVersion)
    return nil
}
```

### `yourcrd_types.go`

```go
package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type YourCRD struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              YourCRDSpec   `json:"spec"`
    Status            YourCRDStatus `json:"status,omitempty"`
}

type YourCRDSpec struct {
    Replicas int `json:"replicas"`
}

type YourCRDStatus struct {
    Phase string `json:"phase,omitempty"`
}

type YourCRDList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []YourCRD `json:"items"`
}
```

### Generate deepcopy code

```bash
controller-gen object paths=./api/types/yourcrd/...
```

---

# Step 2 — Choose Your Reconciler Path

Orkestra offers **three levels** of involvement, from zero code to full control.

## 🟢 **Option 1: Zero‑Code (reconciler.default: true)**

Set `reconciler.default: true` in your Katalog and Orkestra provides a **fully‑featured reconciler** with **zero Go code**:

```yaml
reconciler:
  default: true  # No code required!
```

**What you get for free:**
- ✅ List/Watch from informer cache
- ✅ Finalizer management (add/remove automatically)
- ✅ Kubernetes events for all operations
- ✅ Deep copy safety — never mutate cache
- ✅ Deletion handling
- ✅ NotFound handling
- ✅ Metrics and health tracking
- ✅ Per‑CRD workers and resync intervals

---

## 🟡 **Option 2: Add Hooks for Business Logic**

When you need custom behavior, implement **only the hooks you need**:

```go
type ReconcileHooks[T Object] struct {
    OnReconcile func(ctx context.Context, obj T) error  // Create/Update
    OnDelete    func(ctx context.Context, obj T) error  // Deletion cleanup
    OnNotFound  func(ctx context.Context, key string) error // Object missing
}
```

All hooks are **optional**. Implement one, two, or none — the GenericReconciler handles everything else.

**Example: Project hooks**
```go
package hooks

import (
    "context"
    "fmt"

    "github.com/ialexeze/orkestra/domain"
    projectv1 "github.com/ialexeze/orkestra/example-crds/api/types/project/v1alpha1"
)

func ProjectHooks() domain.ReconcileHooks[domain.Object] {
    return domain.ReconcileHooks[domain.Object]{
        OnReconcile: func(ctx context.Context, obj domain.Object) error {
            project := obj.(*projectv1.Project)  // Type-safe after assertion
            // Your business logic here
            return nil
        },
        OnDelete: func(ctx context.Context, obj domain.Object) error {
            project := obj.(*projectv1.Project)
            // Cleanup external resources
            return nil
        },
        // OnNotFound is optional – skip if not needed
    }
}
```

**Reference in your Katalog:**

```yaml
reconciler:
  default: true
  hooks:
    location: github.com/yourorg/your-hooks
    package: hooks.ProjectHooks
```

Orkestra fetches the hooks at generation time — no manual wiring.

---

## 🔴 **Option 3: Full Custom Reconciler**

For complete control, implement the full `domain.Reconciler` interface:

```go
type YourCRDReconciler struct {
    informer cache.SharedIndexInformer
}

func NewYourCRDReconciler(inf cache.SharedIndexInformer) *YourCRDReconciler {
    return &YourCRDReconciler{informer: inf}
}

func (r *YourCRDReconciler) Reconcile(ctx context.Context, key string) error {
    namespace, name, err := cache.SplitMetaNamespaceKey(key)
    if err != nil {
        return err
    }

    obj, exists, err := r.informer.GetStore().GetByKey(key)
    if err != nil || !exists {
        return nil
    }

    cr := obj.(*v1alpha1.YourCRD)
    // Your business logic here
    return nil
}
```

**Reference in your Katalog:**

```yaml
reconciler:
  default: false
  constructor: "reconciler.NewYourCRDReconciler"
```

---

# Step 3 — Register Your CRD in the Katalog

You can register CRDs in **Go mode** or **YAML mode**.

---

## Option A — Go Mode (Typed)

Add your CRD directly to the Go katalog: [crd-katalog.go](../initialize/crd-katalog.go)

```go
{
    Name:        "yourcrd",
    Object:      &yourcrdv1.YourCRD{},
    ListObject:  &yourcrdv1.YourCRDList{},
    Group:       yourcrdv1.Group,
    Version:     yourcrdv1.Version,
    Kind:        yourcrdv1.Kind,
    Plural:      yourcrdv1.NamePlural,
    Namespace:   "default",
    Namespaced:  true,
    Workers:     3,
    Resync:      10 * time.Minute,
    // Scheme is automatic – ork generate reads your API types
    ReconcilerConfig: reconciler.Config{
        Default: true,  // or hook reference
    },
}
```

### Go Mode Benefits
- Full type safety  
- IDE autocompletion  
- Compile‑time validation  

---

## Option B — YAML Mode (Dynamic)

Enable YAML mode:

```bash
export KATALOG_MODE=YAML
export KATALOG_PATH=initialize/crd-katalog.yaml    # or remote URL
```

Add your CRD to the YAML Katalog:

```yaml
crds:
  - name: yourcrd
    enabled: true
    group: yourgroup.example.com
    version: v1alpha1
    kind: YourCRD
    plural: yourcrds
    namespace: default
    namespaced: true
    workers: 3
    resync: 10m
    dependsOn: ["project"]

    apiTypes:
      location: github.com/yourorg/your-api/types/yourcrd/v1alpha1  # Go package path
      # Orkestra fetches this at generation time

    reconciler:
      default: true
      # Optional hooks:
      # hooks:
      #   location: github.com/yourorg/your-hooks
      #   package: hooks.YourCRDHooks
```

### YAML Mode Benefits
- No recompilation  
- GitOps‑friendly  
- Remote registries  
- Multi‑cluster orchestration  
- Canary rollouts  

---

# Step 4 — Run `ork generate registry`

This is the **magic step**. Orkestra reads your Katalog and generates all the Go bindings:

```bash
ork generate registry --katalog initialize/crd-katalog.yaml
```

**What `ork generate` does:**
1. Fetches API types from `apiTypes.location` (via `go get`)
2. Fetches hook packages from `hooks.location` (if specified)
3. Generates `initialize/registry.go` with:
   - `RegisterRuntimeObjects()` – object factories for all CRDs
   - `RegisterScheme()` – scheme registration for all CRDs
4. Wires everything together

**If `go mod tidy` is needed, run it once. Done.**

---

# Step 5 — Start Orkestra

```bash
go run ./cmd/
```

Orkestra now automatically:

- ✅ builds REST clients via SharedClientFactory  
- ✅ creates informers with per‑CRD resync  
- ✅ assigns worker pools per CRD  
- ✅ wires event handlers  
- ✅ adds finalizers automatically (if configured)  
- ✅ emits Kubernetes events  
- ✅ dispatches by GVK  
- ✅ enforces dependency ordering  
- ✅ exposes per‑CRD metrics  
- ✅ provides health endpoints  
- ✅ integrates with leader election  
- ✅ handles graceful shutdown  

**You wrote ~50 lines of API types and optional hooks.**  
**Orkestra generated ~500 lines of runtime behavior.**

---

# Testing Your CRD

You can test your reconciler in isolation:

- use fake informer stores  
- use fake kubeclient  
- simulate events  
- assert state transitions  

Orkestra's clean interfaces make this trivial.

---

# Summary

Adding a new CRD to Orkestra is:

1. **Write API types** (controller-gen)  
2. **Write optional hooks** (your business logic)  
3. **Add Katalog entry** (Go or YAML)  
4. **Run `ork generate registry`**  
5. **Done**  

Orkestra handles:

| Area | Generated |
|------|-----------|
| **Clients** | ✅ SharedClientFactory |
| **Informers** | ✅ With per‑CRD resync |
| **Workqueues** | ✅ Per‑CRD or shared |
| **Workers** | ✅ Per‑CRD pools |
| **Finalizers** | ✅ Automatic |
| **Events** | ✅ Kubernetes events |
| **Metrics** | ✅ 5 production metrics |
| **Health APIs** | ✅ /katalog/* endpoints |
| **Dependencies** | ✅ Topological order |
| **Lifecycle** | ✅ Graceful shutdown |
| **HA** | ✅ Leader election |

**You focus on business logic. Orkestra handles the rest.** 🚀