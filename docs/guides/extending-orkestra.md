# Extending Orkestra

Adding a new CRD to Orkestra means adding a Katalog entry. Orkestra builds
the operator around it — informer, workqueue, workers, health API, metrics,
finalizers, and lifecycle. You declare the CRD. Orkestra manages it.

---

## What you provide

The minimum for any CRD:

```yaml
# in your Katalog
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
spec:
  crds:
    - name: myresource
      enabled: true
      apiTypes:
        group: myorg.io
        version: v1alpha1
        kind: MyResource
        plural: myresources
      reconciler:
        default: true
```

That is a complete operator entry. Apply the CRD definition to your cluster,
run `ork run --katalog katalog.yaml`, and Orkestra manages `MyResource`
from that moment forward.

Everything else — compiled types, Go hooks, custom constructors — is
optional and added only when you need it.

---

## Step 1 — Declare your CRD in the Katalog

Every CRD entry needs at minimum: `name`, `apiTypes` (group, version, kind,
plural), and `reconciler.default`.

```yaml
- name: myresource
  enabled: true
  namespaced: true
  workers: 2
  resync: 30s
  dependsOn: []

  apiTypes:
    group: myorg.io
    version: v1alpha1
    kind: MyResource
    plural: myresources
    # location omitted → dynamic mode
    # location: github.com/myorg/apis/v1alpha1 → typed mode

  reconciler:
    default: true

  queue:
    maxQueueDepth: 500
    degradeThreshold: 5
```

**Dynamic mode** (no `apiTypes.location`) — no Go types needed. Orkestra
uses the dynamic client. Template expressions in `onCreate`/`onReconcile`/
`onDelete` resolve against the live CR at reconcile time.

**Typed mode** (`apiTypes.location` set) — compiled Go types are registered
at startup. The REST client decodes API server responses into your structs.
Required for Go hooks that access `obj.Spec` fields with compile-time safety.

---

## Step 2 — Choose your reconcile path

### Path A — Declarative templates (zero code)

Declare what resources to create in the Katalog. No Go. No code generation.
No build step. Just `ork run`.

```yaml
reconciler:
  default: true
  finalizers:
    - finalizer.myorg.io/myresource
  onCreate:
    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        reconcile: true   # drift correction — no separate onReconcile needed
    services:
      - name: "{{ .metadata.name }}-svc"
        port: "80"
        targetPort: "{{ .spec.port }}"
        type: "{{ .spec.serviceType }}"
        reconcile: true
    configMaps:
      - name: "{{ .metadata.name }}-config"
        data:
          LOG_LEVEL: "{{ .spec.logLevel }}"
        reconcile: true
  onDelete:
    jobs:
      - name: "{{ .metadata.name }}-cleanup"
        image: busybox
        command: ["sh", "-c", "cleanup.sh {{ .metadata.name }}"]
```

GenericReconciler interprets these declarations at runtime. `reconcile: true`
means the same resource is also drift-corrected on every reconcile — you
declare it once and get both create and correction behaviour.

### Path B — Go hooks

Use when you need: type-safe `obj.Spec` access, conditional resource creation,
external API calls, or status subresource writes.

```yaml
reconciler:
  default: true
  hooks:
    location: github.com/myorg/hooks
    function: MyResourceHooks
```

```go
// pkg/hooks/myresource.go
func MyResourceHooks() domain.AnyReconcileHooks {
    return domain.ReconcileHooks[*apiv1.MyResource]{
        OnReconcile: func(ctx context.Context, obj *apiv1.MyResource) error {
            kube, _ := kubeclient.FromContext(ctx)

            // type-safe spec access
            spec := orkdeploy.ResolvedDeploymentSpec{
                Name:      obj.Name,
                Namespace: obj.Namespace,
                Image:     obj.Spec.Image,     // compiled field access
                Replicas:  int32(obj.Spec.Replicas),
            }
            return orkdeploy.Create(ctx, kube, obj, spec)
        },
        OnDelete: func(ctx context.Context, obj *apiv1.MyResource) error {
            // cleanup logic
            return nil
        },
    }
}
```

Requires `ork generate runtime` to register the hook in `HookRegistry`.
Also requires a compiled Go type package — set `apiTypes.location`.

### Path C — Custom constructor

Use when you need the full reconcile loop: complex state machines, custom
retry strategies, or wrapping an existing reconciler implementation.

```yaml
reconciler:
  default: false
  constructor:
    location: github.com/myorg/reconcilers
    function: NewMyResourceReconciler
```

```go
type MyResourceReconciler struct {
    informer cache.SharedIndexInformer
    kube     *kubeclient.Kubeclient
    event    *event.Event
}

func NewMyResourceReconciler(
    kube *kubeclient.Kubeclient,
    inf cache.SharedIndexInformer,
    ev *event.Event,
) domain.Reconciler {
    return &MyResourceReconciler{informer: inf, kube: kube, event: ev}
}

func (r *MyResourceReconciler) Reconcile(ctx context.Context, key string) error {
    raw, exists, err := r.informer.GetIndexer().GetByKey(key)
    if err != nil || !exists {
        return err
    }
    obj := raw.(*apiv1.MyResource).DeepCopy()
    // your reconcile logic
    return nil
}
```

Orkestra still owns the informer, workqueue, metrics, health API, and leader
election. You own the reconcile function.

Requires `ork generate runtime` to register the constructor.

---

## Step 3 — For typed mode: create API types

Only needed when `apiTypes.location` is set. Skip for dynamic CRDs.

```
api/types/myresource/v1alpha1/
  groupversion_info.go
  myresource_types.go
  zz_generated.deepcopy.go   ← generated by controller-gen
```

**`groupversion_info.go`:**

```go
package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/schema"
)

var (
    Group      = "myorg.io"
    Version    = "v1alpha1"
    Kind       = "MyResource"
    NamePlural = "myresources"

    GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
    SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
    AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
    scheme.AddKnownTypes(GroupVersion, &MyResource{}, &MyResourceList{})
    metav1.AddToGroupVersion(scheme, GroupVersion)
    return nil
}
```

**`myresource_types.go`:**

```go
package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type MyResource struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              MyResourceSpec   `json:"spec"`
    Status            MyResourceStatus `json:"status,omitempty"`
}

type MyResourceSpec struct {
    Image     string `json:"image"`
    Replicas  int    `json:"replicas,omitempty"`
    LogLevel  string `json:"logLevel,omitempty"`
}

type MyResourceStatus struct {
    Phase   string `json:"phase,omitempty"`
    Message string `json:"message,omitempty"`
}

type MyResourceList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []MyResource `json:"items"`
}
```

Generate deepcopy:

```bash
controller-gen object paths=./api/types/myresource/...
```

---

## Step 4 — Generate runtime wiring (typed and hooks only)

Dynamic template CRDs skip this entirely.

```bash
# Typed CRDs — registers Go types + scheme
# Go hooks — registers hook factory in HookRegistry
# Custom constructors — registers constructor in ReconcilerRegistry
ork generate runtime --katalog katalog.yaml

# Preview without writing
ork generate runtime --katalog katalog.yaml --dry-run

# Then
go mod tidy
```

What gets generated for each case:

| Case | Generated |
|---|---|
| Typed CRD | `ObjectRegistry` + `ListRegistry` entries + `RegisterTypedScheme` |
| Go hooks | `HookRegistry` entry |
| Custom constructor | `ReconcilerRegistry` entry |
| Dynamic templates only | **Nothing — generation skipped** |

---

## Step 5 — Apply the CRD and run

```bash
# Apply the Kubernetes CRD definition
kubectl apply -f myresource-crd.yaml

# Validate the Katalog before running
ork validate --katalog katalog.yaml

# Preview the merged state
ork template --katalog katalog.yaml --graph

# Start the operator
ork run --katalog katalog.yaml
```

Orkestra starts, the informer syncs, and your CRD is now managed. Create a
CR and watch the resources appear:

```bash
kubectl apply -f myresource-cr.yaml
kubectl get deployments
curl localhost:8080/katalog/myresource/health | jq
```

---

## Summary

| What you provide | What Orkestra provides |
|---|---|
| Katalog entry | Informer + workqueue |
| Template declarations (optional) | Runtime template interpretation |
| Go types (optional, typed mode) | Scheme registration + REST client |
| Go hooks (optional) | GenericReconciler lifecycle |
| Custom constructor (optional) | Everything except reconcile function |
| | Health API (`/katalog/myresource`) |
| | Prometheus metrics |
| | Dependency ordering |
| | Leader election |
| | Graceful shutdown |
| | Drift correction |
| | Cascade deletion via owner references |
| | Finalizer management |
| | Kubernetes events |