# 03 — The Registry Layer

Each resource type has a dedicated package under `pkg/resources/<kind>/`. The registry package is the only place that touches the Kubernetes API for that resource type. The `run_*.go` file orchestrates; the registry package executes.

## Package structure

```
pkg/resources/
    deployments/
        types.go       — DeploymentTemplateSource, ResolvedDeploymentSpec
        deployment.go  — Create, Update, Delete, DeleteIfOwned, Resolve
    services/
        types.go
        services.go
    secrets/
        types.go
        secret.go
    configmaps/
        types.go
        configmap.go
    serviceaccounts/
        serviceaccount.go
    roles/
        role.go            — Create, Update, Delete, DeleteIfOwned, Resolve
    rolebindings/
        rolebinding.go     — Create, Update, Delete, DeleteIfOwned, Resolve
    jobs/
        types.go
        job.go
    cronjobs/
        cronjob.go
```

> **Roles and RoleBindings** use `rbacv1` from `k8s.io/api/rbac/v1`. Their template source types (`RoleTemplateSource`, `RoleBindingTemplateSource`) live in `pkg/types/types.go` alongside all other source types. The `RoleBindingTemplateSource.RoleRef.Kind` defaults to `"Role"` — set it to `"ClusterRole"` to bind to a cluster-scoped role. RoleBinding's `Update` path handles the Kubernetes immutability constraint on `roleRef` by deleting and recreating the binding when the ref changes.

## The four functions every registry package exposes

### 1. `Create`

Creates the resource if it does not already exist. Idempotent — a second call is a no-op and returns nil.

```go
func Create(ctx context.Context, kube kubeclient.KubeClient,
    owner domain.Object, spec ResolvedXxxSpec) error
```

Sets an owner reference so the resource is garbage-collected when the CR is deleted.

Uses a direct typed client `Get` call (not an informer cache) to check existence:

```go
_, err := kube.Clientset().NetworkingV1().Ingresses(ns).Get(ctx, spec.Name, metav1.GetOptions{})
if err == nil {
    return nil // already exists
}
if !errors.IsNotFound(err) {
    return fmt.Errorf("ingress.Create: checking existence: %w", err)
}
```

### 2. `Update`

Reconciles an existing resource to match the spec. If the resource does not exist, creates it. Checks each reconcilable field for drift before patching.

```go
func Update(ctx context.Context, kube kubeclient.KubeClient,
    owner domain.Object, spec ResolvedXxxSpec) error
```

Only patch when drift is detected. A no-drift path logs at `Debug` and returns nil without an API write.

### 3. `DeleteIfOwned`

Deletes the resource only if the Orkestra owner label matches. This prevents cascading deletion of resources created by a different CR or operator.

```go
func DeleteIfOwned(ctx context.Context, kube kubeclient.KubeClient,
    owner domain.Object, name, namespace string) error
```

```go
// Canonical implementation
existing, err := kube.Clientset().NetworkingV1().Ingresses(ns).Get(ctx, name, metav1.GetOptions{})
if errors.IsNotFound(err) {
    return nil
}
if err != nil {
    return err
}
if existing.Labels[labels.OrkestraOwner] != owner.GetName() {
    return nil // not ours — do not touch
}
return kube.Clientset().NetworkingV1().Ingresses(ns).Delete(ctx, name, metav1.DeleteOptions{})
```

### 4. `Resolve`

Converts a `XxxTemplateSource` (already template-evaluated) into a `ResolvedXxxSpec`. No Kubernetes API calls — pure data transformation.

```go
func Resolve(src orktypes.XxxTemplateSource, ownerName string) ResolvedXxxSpec
```

Always injects the two mandatory Orkestra system labels:

```go
spec.Labels[labels.ManagedKey]      = labels.ManagedValue
spec.Labels[labels.OrkestraOwner] = ownerName
```

## The owner reference

Every resource created by Orkestra carries an owner reference back to the CR. This is what causes Kubernetes to garbage-collect child resources when the CR is deleted — no explicit deletion logic needed for the happy path.

```go
OwnerReferences: []metav1.OwnerReference{
    {
        APIVersion:         owner.GetObjectKind().GroupVersionKind().GroupVersion().String(),
        Kind:               owner.GetObjectKind().GroupVersionKind().Kind,
        Name:               owner.GetName(),
        UID:                owner.GetUID(),
        Controller:         utils.BoolPtr(true),
        BlockOwnerDeletion: utils.BoolPtr(true),
    },
},
```

Exception: Jobs created under `onDelete` do NOT set owner references, because the owner CR is already being deleted — the Job must complete independently after the CR is gone.

## The ResolvedXxxSpec type

Declared in `types.go` alongside the template source. Must include at minimum:

```go
type ResolvedIngressSpec struct {
    Name        string
    Namespace   string
    Labels      map[string]string
    Annotations map[string]string
    // ... resource-specific fields
}
```

The `Namespace` field is optional — if empty, the registry functions fall back to `owner.GetNamespace()`.

## Naming the template source type

Template source types live in `pkg/types/`, not in the registry package. The registry package imports `orktypes` for the source type, and the template resolver package imports `orktypes` for the same type. This keeps the dependency graph acyclic:

```
pkg/types             ← source structs, condition types
pkg/resources ← imports pkg/types
pkg/reconciler        ← imports both
```

Never put source struct types in the registry package.

---

**Next →** [04 — Conditions and the activeNames Guard](04-conditions.md)
