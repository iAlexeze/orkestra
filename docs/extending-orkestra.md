# 🎼 **Extending Orkestra**  
### *How to Add New CRDs, Reconcilers, and Runtime Behavior*

Orkestra is designed so that extending it is **fast**, **predictable**, and **boilerplate‑free**.  
Whether you’re adding a new CRD in Go or defining one dynamically through YAML, the workflow is intentionally simple:

> **You write API types and a reconciler.  
Orkestra builds the entire operator runtime around them.**

This guide walks you through the full process.

---

# 🧩 **Overview: What You Need to Provide**

To add a new CRD to Orkestra, you only need to supply:

1. **API Types**  
2. **Reconciler**  
3. **Registry Entry** (Go or YAML)  
4. *(YAML mode only)* Scheme registration

Everything else — clients, informers, workers, resync intervals, dependency ordering, lifecycle orchestration — is generated automatically.

---

# 🏗️ Step 1 — Create API Types

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

# 🧠 Step 2 — Write Your Reconciler

Your reconciler is the **only Go logic** you write.

```go
package reconciler

import (
    "context"
    "k8s.io/client-go/tools/cache"
)

type YourCRDReconciler struct{}

func NewYourCRDReconciler(inf cache.SharedIndexInformer) *YourCRDReconciler {
    return &YourCRDReconciler{}
}

func (r *YourCRDReconciler) Reconcile(ctx context.Context, key string) error {
    namespace, name, err := cache.SplitMetaNamespaceKey(key)
    if err != nil {
        return err
    }

    // Fetch object from informer store
    obj, exists, err := r.informer.GetStore().GetByKey(key)
    if err != nil || !exists {
        return nil // handle deletion if needed
    }

    cr := obj.(*v1alpha1.YourCRD)

    // Your business logic here
    return nil
}
```

---

# 🧭 Step 3 — Register Your CRD

You can register CRDs in **Go mode** or **YAML mode**.

---

## 🟦 Option A — Go Mode (Typed)

Add your CRD to the Go registry: [crd-registry](../initialize/crd_registry.go)

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
    Scheme:      yourcrdv1.AddToScheme,
    Reconciler: func(kube *kubeclient.Kubeclient, inf cache.SharedIndexInformer) domain.Reconciler {
        return reconciler.NewYourCRDReconciler(inf)
    },
}
```

### Go Mode Benefits
- Full type safety  
- Automatic scheme registration  
- IDE autocompletion  
- Compile‑time validation  

---

## 🟩 Option B — YAML Mode (Dynamic)

Enable YAML mode:

```bash
export CRD_REGISTRY_MODE=YAML
export CRD_REGISTRY=initialize/crd-registry.yaml    # or remote URL
```

Add your CRD:

```yaml
crds:
  - name: yourcrd
    group: yourgroup.example.com
    version: v1alpha1
    kind: YourCRD
    plural: yourcrds
    namespace: default
    namespaced: true
    workers: 3
    resync: 10m
    dependsOn: ["project"]
```

### YAML Mode Benefits
- No recompilation  
- GitOps‑friendly  
- Remote registries  
- Multi‑cluster orchestration  
- Canary rollouts  

---

# 🎼 Step 4 — Register Your Reconciler (YAML Mode Only)

YAML mode uses a name‑based reconciler registry: [reconciler-registry](../pkg/reconciler/reconcile.go)

```go
func RegisterReconcilers() map[string]NewReconcilerFunc {
    return map[string]NewReconcilerFunc{
        "yourcrd": func(k *kubeclient.Kubeclient, inf cache.SharedIndexInformer) domain.Reconciler {
            return NewYourCRDReconciler(inf)
        },
    }
}
```

The key (`yourcrd`) must match the YAML `name:` field.

---

# 🎻 Step 5 — Register Your Scheme (YAML Mode Only)
YAML mode scheme registry: [scheme-registry](../initialize/scheme_registry.go)

```go
func RegisterScheme(scheme *runtime.Scheme) (*runtime.Scheme, error) {
    if err := yourcrdv1.AddToScheme(scheme); err != nil {
        return nil, err
    }
    return scheme, nil
}
```

---

# 🎉 Step 6 — Done!

Orkestra now automatically:

- builds REST clients  
- creates informers  
- applies your resync interval  
- assigns worker pools  
- wires event handlers  
- dispatches by GVK  
- enforces dependency ordering  
- exposes metrics  
- integrates with leader election  
- handles graceful shutdown  

You wrote **~50 lines of code**.  
Orkestra generated **~500 lines of runtime behavior**.

---

# 🧪 Testing Your CRD

You can test your reconciler in isolation:

- use fake informer stores  
- use fake kubeclient  
- simulate events  
- assert state transitions  

Orkestra’s clean interfaces make this trivial.

---

# 🏁 Summary

Adding a new CRD to Orkestra is:

1. Write API types  
2. Write reconciler  
3. Register CRD (Go or YAML)  
4. *(YAML mode)* Register scheme and reconciler 
5. Done  

Orkestra handles:

- clients  
- informers  
- workqueues  
- dispatch  
- workers  
- resync  
- dependencies  
- lifecycle  
- metrics  
- HA  

So you can focus entirely on **business logic**, not plumbing.