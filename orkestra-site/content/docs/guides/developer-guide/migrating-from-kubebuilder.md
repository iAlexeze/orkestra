---
title: "Migrating From Kubebuilder"
weight: 29
---

# Migrating from Kubebuilder

This guide walks through migrating a real Kubebuilder operator to Orkestra.
The example is a `Website` operator — simple enough to understand completely,
complex enough to demonstrate every meaningful migration step.

The migration is incremental. You run both the existing Kubebuilder operator
and Orkestra simultaneously during transition. There is no cutover moment.

---

## The existing operator

A typical Kubebuilder `Website` operator has this structure:

```
website-operator/
  api/
    v1alpha1/
      website_types.go    # CRD types
      zz_generated.deepcopy.go
  controllers/
    website_controller.go # reconcile loop
  config/
    crd/bases/...         # generated CRD manifests
    rbac/...
    manager/...
  main.go
  Makefile
  Dockerfile
  go.mod
```

The controller looks like this (simplified):

```go
// controllers/website_controller.go
func (r *WebsiteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    website := &demov1.Website{}
    if err := r.Get(ctx, req.NamespacedName, website); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Create or update Deployment
    dep := r.deploymentForWebsite(website)
    if err := r.reconcileDeployment(ctx, website, dep); err != nil {
        return ctrl.Result{}, err
    }

    // Create or update Service
    svc := r.serviceForWebsite(website)
    if err := r.reconcileService(ctx, website, svc); err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

This is the 80% case — creating a Deployment and a Service from CR fields.
All of this becomes a Katalog entry.

---

## Step 1: Write the Katalog

Do not touch the Kubebuilder operator. Do not stop it. Write the Katalog
alongside the existing operator and let both run simultaneously during migration.

```yaml
# website-katalog.yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: website-operator-v2

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
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              port: "{{ .spec.port }}"
              reconcile: true      # was reconcileDeployment
          services:
            - name: "{{ .metadata.name }}-svc"
              port: "80"
              targetPort: "{{ .spec.port }}"
              reconcile: true      # was reconcileService
```

The template expressions map directly to the `deploymentForWebsite` helper
fields in the existing controller. No logic is being reimplemented — the
declarations replace the construction functions.

**Validate it without a cluster:**

```bash
ork validate --katalog website-katalog.yaml
```

---

## Step 2: Run Orkestra alongside the existing operator

Do not stop the Kubebuilder operator. Run Orkestra simultaneously:

```bash
ork run --katalog website-katalog.yaml
```

Both operators now reconcile the same CRDs. This is safe because both use
idempotent operations — `Create` checks for existence, `Update` applies desired
state. If the Kubebuilder operator creates a Deployment and Orkestra's reconcile
cycle runs next, Orkestra finds the Deployment already exists and skips creation.

The `managed-by: orkestra` label that Orkestra adds to child resources identifies
what Orkestra created. Resources created by the Kubebuilder operator do not have
this label. You can see the ownership split:

```bash
kubectl get deployments -l managed-by=orkestra
kubectl get deployments -l 'managed-by!=orkestra'
```

---

## Step 3: Transfer ownership

Apply new CRs and let Orkestra manage them. Existing CRs remain under the
Kubebuilder operator until you explicitly transfer them.

To transfer an existing CR to Orkestra, add the managed label:

```bash
kubectl label website my-site orkestra.konductor.io/managed=true
```

Orkestra picks up the label on the next watch event and adds its finalizer.
From this point, the Kubebuilder operator and Orkestra both reconcile the same
CR — which is fine, both are idempotent — but Orkestra's finalizer now protects
deletion.

To stop the Kubebuilder operator from reconciling specific CRs, you can use
its `--namespace` flag to scope it to a different namespace, or scale its
Deployment to zero once all CRs are transferred.

---

## Step 4: Migrate validation logic

The Kubebuilder operator likely has a `ValidatingWebhookConfiguration` and a
`defaulting` webhook for validation and mutation. These become Katalog declarations.

**Original webhook handler:**

```go
func (w *WebsiteValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
    website := obj.(*demov1.Website)
    if !strings.HasPrefix(website.Spec.Image, "myorg/") {
        return fmt.Errorf("image must be from myorg registry")
    }
    return nil
}
```

**Orkestra equivalent:**

```yaml
validation:
  - field: spec.image
    prefix: "myorg/"
    message: "image must be from myorg registry"
    action: deny
```

**Original defaulting webhook:**

```go
func (w *WebsiteDefaulter) Default(ctx context.Context, obj runtime.Object) error {
    website := obj.(*demov1.Website)
    if website.Spec.Replicas == 0 {
        website.Spec.Replicas = 2
    }
    return nil
}
```

**Orkestra equivalent:**

```yaml
mutation:
  - field: spec.replicas
    default: "2"
```

After migrating, delete the `ValidatingWebhookConfiguration` and
`MutatingWebhookConfiguration` created by the Kubebuilder operator. Orkestra
registers its own when `ENABLE_ADMISSION_WEBHOOK=true`.

---

## Step 5: Migrate status updates

The Kubebuilder operator likely updates status using the status subresource:

```go
website.Status.Phase = "Running"
website.Status.ObservedReplicas = website.Spec.Replicas
if err := r.Status().Update(ctx, website); err != nil {
    return ctrl.Result{}, err
}
```

This becomes a Katalog declaration:

```yaml
status:
  fields:
    - path: phase
      value: "Running"
    - path: observedReplicas
      value: "{{ .spec.replicas }}"
    - path: readyReplicas
      value: "{{ .children.deployment.status.readyReplicas }}"
```

The Ready condition is written automatically by Orkestra — no need to replicate
the condition management logic from the existing controller.

---

## Step 6: Migrate conversion webhooks

If the CRD has multiple versions and a conversion webhook, that webhook becomes
declarative conversion paths in the Katalog.

**Original conversion webhook (Go):**

```go
func (w *WebsiteConversionWebhook) ConvertTo(dstRaw conversion.Hub) error {
    dst := dstRaw.(*v1.Website)
    src := w.(*v1alpha1.Website)
    dst.Spec.Image = src.Spec.Image
    dst.Spec.Replicas = src.Spec.Replicas
    dst.Spec.SEO.Enabled = false  // new field — supply default
    return nil
}
```

**Orkestra equivalent:**

```yaml
conversion:
  storageVersion: v1
  paths:
    - from: v1alpha1
      to: v1
      spec:
        image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        seo:
          enabled: false
```

Delete the conversion webhook Deployment and its `ValidatingWebhookConfiguration`.
Orkestra's built-in conversion webhook handles this on the same HTTPS server
as admission.

---

## Step 7: Remove the Kubebuilder operator

Once all CRs are managed by Orkestra and all webhook configurations have been
migrated, scale down and remove the Kubebuilder operator:

```bash
# Scale down
kubectl scale deployment website-controller-manager -n website-system --replicas=0

# Verify Orkestra is handling all reconciles
ork status
curl localhost:8080/katalog/website

# Remove after a soak period
kubectl delete deployment website-controller-manager -n website-system
kubectl delete namespace website-system
```

The CRD stays — Orkestra watches it directly. No changes to the CRD definition
are required.

---

## What about complex business logic?

The migration path above handles the common case: resource factory operators
that create Kubernetes resources from CR fields. If the existing operator has
complex business logic — external API calls, state machine transitions, complex
conditional creation — that logic moves to a Go hook rather than to templates.

The hook is registered in the Katalog:

```yaml
operatorBox:
  default: true
  hooks:
    location: github.com/myorg/website-hooks
    function: WebsiteHooks
```

The hook implementation calls the OrkestraRegistry for the Kubernetes resources
and handles the business logic directly. The reconciler signature changes from
`ctrl.Result, error` to `error`:

```go
func WebsiteHooks() domain.AnyReconcileHooks {
    return domain.ReconcileHooks[*apiv1.Website]{
        OnReconcile: func(ctx context.Context, obj *apiv1.Website) error {
            // all your existing business logic here
            // use OrkestraRegistry for Kubernetes resources
            return nil
        },
    }
}
```

The migration is: extract the business logic into the hook, remove the
Kubernetes resource construction code (OrkestraRegistry handles it), and
delete everything else (informer setup, scheme registration, controller-manager
boilerplate). What remains is the logic that actually matters.

---

## The net result

After migration:

- The CRD is unchanged — no impact on existing CRs or users
- The Kubebuilder binary, its image, its Helm chart, and its RBAC are gone
- The Orkestra Katalog is 20–50 lines of YAML
- Validation, mutation, conversion, and status all live in the Katalog
- `ork status` shows the operator health — no separate monitoring setup needed
- The next CRD you add to the platform takes an hour, not weeks
