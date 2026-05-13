---
title: "Extending Orkestra"
weight: 5
description: "Adding a new CRD to Orkestra means adding a **Katalog entry**."
---

Adding a new CRD to Orkestra means adding a **Katalog entry**.  
Orkestra builds the operator around it — informer, workqueue, workers, health API, metrics, finalizers, and lifecycle.  
You declare the CRD. Orkestra manages it.

:::tip
The fastest way to extend Orkestra is to start with a minimal Katalog entry and iterate.  
You only add Go code when you *need* typed access or complex custom logic.
:::

---

## What You Provide

The minimum required for any CRD:

```yaml
apiVersion: orkestra.orkspace.io/v1
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
      operatorBox:
        default: true
```

This is a complete operator entry.

- Apply the CRD to your cluster  
- Run `ork run --file katalog.yaml`  
- Orkestra begins managing `MyResource` immediately  

Everything else — typed Go structs, hooks, constructors — is optional.

---

## Step 1 — Declare Your CRD in the Katalog

Every CRD entry needs:

- `name`
- `apiTypes` (group, version, kind, plural)
- `reconciler.default`

Example:

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

  operatorBox:
    default: true

  queue:
    maxQueueDepth: 500
    degradeThreshold: 5
```

### Dynamic Mode vs Typed Mode

#### Dynamic Mode (no `apiTypes.location`)
- No Go types required  
- Orkestra uses the dynamic client  
- Templates resolve fields at runtime  
- Perfect for rapid development and simple CRDs  

#### Typed Mode (`apiTypes.location` set)
- Requires compiled Go structs  
- Enables type‑safe access (`obj.Spec.Field`)  
- Required for Go hooks and custom constructors  

:::note
Most users start in **dynamic mode** and move to **typed mode** only when they need compile‑time safety or custom logic.
:::

---

## Step 2 — Choose Your Reconcile Path

Orkestra supports three reconciliation styles.

### Path A — Declarative Templates (Zero Code)

This is the simplest and most common path.

You declare what resources should exist. Orkestra creates and reconciles them.

```yaml
operatorBox:
  default: true
  finalizers:
    - finalizer.myorg.io/myresource
  onCreate:
    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        reconcile: true
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

#### Why Declarative Templates?

- No Go code  
- No compilation  
- No controller logic  
- Drift correction built‑in  
- Fastest way to build operators  

:::tip
`reconcile: true` means the resource is also corrected on every reconcile — you declare it once and get both create and drift‑correction behavior.
:::
---

### Path B — Go Hooks

Use Go hooks when you need:

- Type‑safe access to `obj.Spec`  
- Complex conditional logic  
- External API calls  
- Complex orchestration  

Katalog:

```yaml
operatorBox:
  default: true
  hooks:
    location: github.com/myorg/hooks
    function: MyResourceHooks
```

Hook implementation:

```go
func MyResourceHooks() domain.AnyReconcileHooks {
    return domain.ReconcileHooks[*apiv1.MyResource]{
        OnReconcile: func(ctx context.Context, obj *apiv1.MyResource) error {
            kube, _ := kubeclient.FromContext(ctx)

            spec := orkdeploy.ResolvedDeploymentSpec{
                Name:      obj.Name,
                Namespace: obj.Namespace,
                Image:     obj.Spec.Image,
                Replicas:  int32(obj.Spec.Replicas),
            }

            // All above can be declared in a katalog
            // However, you may want to add additional logic 

            return orkdeploy.Create(ctx, kube, obj, spec)
        },
        OnDelete: func(ctx context.Context, obj *apiv1.MyResource) error {
          // Cleanup external resources
            return nil
        },
    }
}
```

:::note
Go hooks require `apiTypes.location` and `ork generate registry` command to register the hook.
:::
---

### Path C — Custom Constructor

Use this when you need:

- Full control of the reconcile loop  
- Custom retry strategies  
- Complex state machines  
- Integration with existing controllers  

Katalog:

```yaml
operatorBox:
  default: false
  constructor:
    location: github.com/myorg/reconcilers
    function: NewMyResourceReconciler
```

Constructor:

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
```

:::warning
Custom constructors give you full power — but also full responsibility.  
Use them only when declarative templates or hooks cannot express your logic.
:::

---

## Step 3 — Typed Mode: Create API Types

Skip this if you’re using dynamic mode.

Directory structure:

```
api/types/myresource/v1alpha1/
  groupversion_info.go
  myresource_types.go
  zz_generated.deepcopy.go
```

Generate deepcopy:

```bash
controller-gen object paths=./api/types/myresource/...
```

---

## Step 4 — Generate Runtime Registry (Typed + Hooks Only)

Dynamic CRDs skip this step.

```bash
ork generate registry --file katalog.yaml
go mod tidy
```

What gets generated:

| Mode | Generated |
|------|-----------|
| Dynamic | Nothing |
| Typed | Scheme registration |
| Hooks | HookRegistry entry |
| Constructor | ReconcilerRegistry entry |

---

## Step 5 — Apply the CRD and Run

```bash
kubectl apply -f myresource-crd.yaml
ork validate --file katalog.yaml
ork template --file katalog.yaml --graph
ork run --file katalog.yaml
```

Create a CR:

```bash
kubectl apply -f myresource-cr.yaml
```

Check health:

```bash
curl localhost:8080/katalog/myresource/health | jq
```

---

## Summary

| You Provide | Orkestra Provides |
|------------|-------------------|
| Katalog entry | Informer + workqueue |
| Templates (optional) | Runtime template engine |
| Go types (optional) | Scheme + typed client |
| Go hooks (optional) | GenericReconciler lifecycle |
| Custom constructor (optional) | Everything except reconcile logic |
| | Health API |
| | Metrics |
| | Finalizers |
| | Drift correction |
| | Dependency ordering |
| | Leader election |
| | Graceful shutdown |
