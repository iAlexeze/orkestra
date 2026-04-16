# OrkestraRegistry Technical Documentation

OrkestraRegistry is the standard library of Kubernetes resource implementations
for Orkestra operators. It provides production-ready Create, Update, Delete, and
Resolve functions for every common resource type — with idempotency, owner
references, drift detection, and consistent error handling built in.

You do not write Kubernetes API calls. You declare what you want in a Katalog
or call a registry function from a Go hook. OrkestraRegistry handles the rest.

!!! note
    New to Orkestra? Check out [Getting Started](../../getting-started/index.md) and the [Example Workflows](../../examples/index.md). This document is a reference for the registry implementation.

## What it provides

```
pkg/orkestra-registry/
  deployments/       Deployment create/update/delete
  services/          Service create/update/delete
  secrets/           Secret create/update/delete + copy across namespaces
  configmaps/        ConfigMap create/update/delete + copy and merge
  serviceaccounts/   ServiceAccount create/delete
  jobs/              Job create/delete
  cronjobs/          CronJob create/update/delete
  pods/              Pod create/update/delete
  template/          Resolver — evaluates template expressions against live CRs
```
## How it connects to Orkestra

OrkestraRegistry sits between the reconciler and the Kubernetes API. When a
CR event fires, the reconciler resolves template expressions against the live
CR object and then calls the registry to apply the result.

```
CR event
    │
    ▼
Resolver — evaluates "{{ .spec.image }}" → "nginx:1.25"
    │
    ▼
Registry — orkdeploy.Create(ctx, kube, owner, spec)
    │
    ▼
Kubernetes API — Deployment created with owner reference
```

!!! tip
    The reconciler never constructs Kubernetes objects directly. 
    The registry never evaluates template expressions. The boundary is clean.

## The contract

Every resource package exports the same four functions:

```go
// Create — idempotent. If the resource already exists, return nil.
// Never errors on "already exists". Safe to call on every reconcile.
func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedXxxSpec) error

// Update — drift correction. Apply desired state regardless of current state.
// If the resource does not exist, creates it.
// If the resource exists but has drifted, corrects it.
func Update(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedXxxSpec) error

// Delete — idempotent. If the resource does not exist, return nil.
// Never errors on "not found".
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedXxxSpec) error

// Resolve — build a ResolvedXxxSpec from a template source.
// Template expressions must already be evaluated before calling.
// Applies defaults, parses types, sets system labels.
func Resolve(src orktypes.XxxTemplateSource, ownerName string) ResolvedXxxSpec
```

**Owner references** are set on every resource created by the registry. This
means cascade deletion is automatic — when the CR is deleted, all child
resources are garbage collected by Kubernetes without any explicit `onDelete`
declarations needed.

**System labels** and **Annotations** are always added and cannot be overridden:

```
labels:
  orkestra.konductor.io/managed: "true"

annotations:
  orkestra.konductor.io/managed-by: website-katalog
  orkestra.konductor.io/managed-since: 2026-03-25T10:30:45Z
```

## Deployments

`pkg/orkestra-registry/deployments`

Manages Deployment lifecycle. The most commonly used registry package.

```go
import orkdeploy "github.com/orkspace/orkestra/pkg/orkestra-registry/deployments"
```

**Katalog declaration:**

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
      replicas: "{{ .spec.replicas }}"
      port: "{{ .spec.port }}"
      namespace: "{{ .metadata.namespace }}"
      reconcile: true
      labels:
        - key: app
          value: "{{ .metadata.name }}"
      resources:
        requests:
          cpu: 100m
          memory: 128Mi
        limits:
          cpu: 500m
          memory: 512Mi
```

**Go hook usage:**

```go
resolved, err := resolver.ResolveDeploymentTemplate(src)
spec := orkdeploy.Resolve(resolved, staticReplicas, resolver.OwnerName())
orkdeploy.Create(ctx, kube, obj, spec)   // onCreate
orkdeploy.Update(ctx, kube, obj, spec)   // onReconcile — drift correction
```

**Drift detection:** image and replica count. When either drifts, the
Deployment is patched. Other fields (resource limits, labels) are not
drift-corrected by default — declare them under `onReconcile` if needed.

**`ResolvedDeploymentSpec` fields:**

| Field | Type | Description |
|---|---|---|
| `Name` | string | Deployment name |
| `Namespace` | string | Target namespace |
| `Image` | string | Container image |
| `Replicas` | int32 | Number of pod replicas |
| `Port` | int32 | Container port (optional) |
| `Labels` | map[string]string | Applied to Deployment and pod template |
| `Annotations` | map[string]string | Applied to Deployment metadata |
| `Resources` | *ResourceRequirements | CPU and memory requests/limits |

## Services

`pkg/orkestra-registry/services`

Manages Service lifecycle. Selector is always `orkestra-owner: <cr-name>` —
automatically routes to pods created by the Deployment registry for the same CR.

```go
import orksvc "github.com/orkspace/orkestra/pkg/orkestra-registry/services"
```

**Katalog declaration:**

```yaml
onCreate:
  services:
    - name: "{{ .metadata.name }}-svc"
      type: "{{ .spec.serviceType }}"
      port: "80"
      targetPort: "{{ .spec.port }}"
      namespace: "{{ .metadata.namespace }}"
      reconcile: true
      labels:
        - key: app
          value: "{{ .metadata.name }}"
```

**Go hook usage:**

```go
resolved, err := resolver.ResolveServiceTemplate(src)
spec := orksvc.Resolve(resolved, resolver.OwnerName())
orksvc.Create(ctx, kube, obj, spec)
orksvc.Update(ctx, kube, obj, spec)
```

**Drift detection:** port and target port. Service type is not drift-corrected
after creation — Kubernetes treats type changes as breaking and requires delete
and recreate.

**Supported service types:** `ClusterIP` (default), `NodePort`, `LoadBalancer`.

**`ResolvedServiceSpec` fields:**

| Field | Type | Description |
|---|---|---|
| `Name` | string | Service name |
| `Namespace` | string | Target namespace |
| `Type` | string | ClusterIP / NodePort / LoadBalancer |
| `Port` | int32 | Service port |
| `TargetPort` | int32 | Container port to route to |
| `Labels` | map[string]string | Applied to Service metadata |

## Secrets

`pkg/orkestra-registry/secrets`

Manages Secret lifecycle. Supports three patterns beyond simple creation:
copy from an existing Secret, sync from a source on every reconcile, and
distribute to multiple namespaces at once.

```go
import orksecrets "github.com/orkspace/orkestra/pkg/orkestra-registry/secrets"
```

**Pattern 1 — static data:**

```yaml
secrets:
  - name: "{{ .metadata.name }}-config"
    namespace: "{{ .metadata.namespace }}"
    data:
      API_KEY: my-api-key
      REGION: us-east-1
```

**Pattern 2 — copy from existing Secret:**

```yaml
secrets:
  - name: db-credentials
    fromSecret: master-db-creds
    fromNamespace: platform
    namespace: "{{ .metadata.namespace }}"
    reconcile: true     # re-sync if source changes
```

**Pattern 3 — distribute to multiple namespaces:**

```yaml
secrets:
  - name: registry-pull-secret
    fromSecret: docker-registry-creds
    fromNamespace: platform
    toNamespaces:
      - "{{ .metadata.namespace }}"
      - monitoring
      - staging
    reconcile: true
```

**Go hook usage:**

```go
resolved, err := resolver.ResolveSecretTemplate(src)
spec := orksecrets.Resolve(resolved, resolver.OwnerName())

// Single namespace
orksecrets.Create(ctx, kube, obj, spec)
orksecrets.Update(ctx, kube, obj, spec)  // re-reads source, syncs data

// Multiple namespaces — source read once, written N times
namespaces, _ := resolver.ResolveStringSlice(src.ToNamespaces)
orksecrets.CopyToNamespaces(ctx, kube, obj, spec, namespaces)
```

**`CopyToNamespaces`** reads the source Secret once and creates copies in each
listed namespace. All copies get owner references back to the CR and are cleaned
up when the CR is deleted.

**`ResolvedSecretSpec` fields:**

| Field | Type | Description |
|---|---|---|
| `Name` | string | Secret name in target namespace |
| `Namespace` | string | Primary target namespace |
| `FromSecret` | string | Source Secret name to copy from |
| `FromNamespace` | string | Source Secret namespace |
| `Data` | map[string][]byte | Raw secret data |
| `StringData` | map[string]string | String secret data (auto-encoded) |
| `Type` | string | Kubernetes Secret type (default: Opaque) |
| `Labels` | map[string]string | Applied to Secret metadata |

## ConfigMaps

`pkg/orkestra-registry/configmaps`

Manages ConfigMap lifecycle. Supports the same distribution patterns as
Secrets, plus a merge pattern for environment-specific configuration.

```go
import orkcm "github.com/orkspace/orkestra/pkg/orkestra-registry/configmaps"
```

**Pattern 1 — static data:**

```yaml
configMaps:
  - name: "{{ .metadata.name }}-config"
    namespace: "{{ .metadata.namespace }}"
    data:
      LOG_LEVEL: info
      MAX_CONNECTIONS: "100"
```

**Pattern 2 — copy from source:**

```yaml
configMaps:
  - name: app-config
    fromConfigMap: base-app-config
    fromNamespace: platform
    namespace: "{{ .metadata.namespace }}"
    reconcile: true
```

**Pattern 3 — copy and override specific keys:**

```yaml
configMaps:
  - name: app-config
    fromConfigMap: base-app-config
    fromNamespace: platform
    namespace: "{{ .metadata.namespace }}"
    data:
      LOG_LEVEL: "{{ .spec.logLevel }}"    # overrides the base value
    reconcile: true
```

Source keys are copied first. Declared `data` keys override matching source
keys. The result is a merged ConfigMap in the target namespace.

**Pattern 4 — distribute to multiple namespaces:**

```yaml
configMaps:
  - name: monitoring-config
    fromConfigMap: prometheus-scrape-config
    fromNamespace: monitoring
    toNamespaces:
      - "{{ .metadata.namespace }}"
      - staging
```

**Go hook usage:**

```go
resolved, err := resolver.ResolveConfigMapTemplate(src)
spec := orkcm.Resolve(resolved, resolver.OwnerName())
orkcm.Create(ctx, kube, obj, spec)
orkcm.Update(ctx, kube, obj, spec)

namespaces, _ := resolver.ResolveStringSlice(src.ToNamespaces)
orkcm.CopyToNamespaces(ctx, kube, obj, spec, namespaces)
```

**`ResolvedConfigMapSpec` fields:**

| Field | Type | Description |
|---|---|---|
| `Name` | string | ConfigMap name |
| `Namespace` | string | Target namespace |
| `Data` | map[string]string | Configuration data |
| `FromConfigMap` | string | Source ConfigMap to copy from |
| `FromNamespace` | string | Source ConfigMap namespace |
| `Labels` | map[string]string | Applied to ConfigMap metadata |

## ServiceAccounts

`pkg/orkestra-registry/serviceaccounts`

Manages ServiceAccount lifecycle. ServiceAccounts have no meaningful spec
fields that can drift after creation — the registry provides Create and
Delete only. Create is always idempotent.

```go
import orksa "github.com/orkspace/orkestra/pkg/orkestra-registry/serviceaccounts"
```

**Katalog declaration:**

```yaml
onCreate:
  serviceAccounts:
    - name: "{{ .spec.team }}-sa"
      namespace: "{{ .spec.targetNamespace }}"
      labels:
        - key: team
          value: "{{ .spec.team }}"
```

**Go hook usage:**

```go
resolved, err := resolver.ResolveServiceAccountTemplate(src)
spec := orksa.Resolve(resolved, resolver.OwnerName())
orksa.Create(ctx, kube, obj, spec)
```

**`ResolvedServiceAccountSpec` fields:**

| Field | Type | Description |
|---|---|---|
| `Name` | string | ServiceAccount name |
| `Namespace` | string | Target namespace |
| `Labels` | map[string]string | Applied to ServiceAccount metadata |

## Jobs

`pkg/orkestra-registry/jobs`

Manages Job lifecycle. Jobs are fire-and-forget — created once, not updated.
Most commonly used under `onDelete` for cleanup tasks that must complete
before Orkestra removes finalizers from the CR.

```go
import orkjobs "github.com/orkspace/orkestra/pkg/orkestra-registry/jobs"
```

**Common use cases under `onDelete`:**
- Draining a message queue before a consumer CR is deleted
- Archiving state to external storage
- Notifying external systems of deletion
- Running database migrations before removing a schema CR

**Katalog declaration:**

```yaml
onDelete:
  jobs:
    - name: "{{ .metadata.name }}-cleanup"
      image: busybox
      command: ["sh", "-c", "drain-queue.sh {{ .metadata.name }}"]
      backoffLimit: 3
      namespace: "{{ .metadata.namespace }}"
```

**Go hook usage:**

```go
resolved, err := resolver.ResolveJobTemplate(src)
spec := orkjobs.Resolve(resolved, resolved.BackoffLimit, resolver.OwnerName())
orkjobs.Create(ctx, kube, obj, spec)
```

**Owner reference behaviour:** Jobs created under `onDelete` have owner
references set. Because the CR's `blockOwnerDeletion: true` is set on the
owner reference, Kubernetes holds the CR in terminating state until the Job
completes. This is intentional — it is how Orkestra ensures cleanup runs
before the CR is fully deleted.

Jobs created under `onCreate` have owner references and are garbage collected
when the CR is deleted.

**`ResolvedJobSpec` fields:**

| Field | Type | Description |
|---|---|---|
| `Name` | string | Job name |
| `Namespace` | string | Target namespace |
| `Image` | string | Container image |
| `Command` | []string | Container entrypoint |
| `Args` | []string | Container arguments |
| `BackoffLimit` | int | Retry count before Job is marked Failed (default: 3) |
| `Labels` | map[string]string | Applied to Job metadata |

## CronJobs

`pkg/orkestra-registry/cronjobs`

Manages CronJob lifecycle. Created under `onCreate` and drift-corrected under
`onReconcile` or with `reconcile: true`. Useful for scheduled workloads that
should exist for the lifetime of a CR.

```go
import orkcron "github.com/orkspace/orkestra/pkg/orkestra-registry/cronjobs"
```

**Common use cases:**
- Periodic sync jobs (cache warming, data replication)
- Scheduled backup jobs owned by a database CR
- Recurring cleanup tasks
- Audit or health check jobs on a schedule

**Katalog declaration:**

```yaml
onCreate:
  cronJobs:
    - name: "{{ .metadata.name }}-sync"
      schedule: "{{ .spec.syncSchedule }}"
      image: "{{ .spec.syncImage }}"
      command: ["sh", "-c", "sync.sh"]
      namespace: "{{ .metadata.namespace }}"
      reconcile: true
```

**Go hook usage:**

```go
resolved, err := resolver.ResolveCronJobTemplate(src)
spec := orkcron.Resolve(resolved, resolver.OwnerName())
orkcron.Create(ctx, kube, obj, spec)
orkcron.Update(ctx, kube, obj, spec)  // corrects schedule and image drift
```

**Drift detection:** schedule expression and container image. When either
drifts the CronJob is updated in place.

**`ResolvedCronJobSpec` fields:**

| Field | Type | Description |
|---|---|---|
| `Name` | string | CronJob name |
| `Namespace` | string | Target namespace |
| `Schedule` | string | Cron expression e.g. `"0 * * * *"` |
| `Image` | string | Container image |
| `Command` | []string | Container entrypoint |
| `Args` | []string | Container arguments |
| `Labels` | map[string]string | Applied to CronJob metadata |

## Pods

`pkg/orkestra-registry/pods`

Manages Pod lifecycle directly. Prefer Deployments for long-running workloads
— Deployments manage Pod restarts, rolling updates, and replica sets
automatically. Use Pods only when you need direct, single-instance Pod control.

```go
import orkpods "github.com/orkspace/orkestra/pkg/orkestra-registry/pods"
```

**Katalog declaration:**

```yaml
onCreate:
  pods:
    - name: "{{ .metadata.name }}-worker"
      image: "{{ .spec.workerImage }}"
      port: "9090"
      namespace: "{{ .metadata.namespace }}"
```

**Go hook usage:**

```go
resolved, err := resolver.ResolvePodTemplate(src)
spec := orkpods.Resolve(resolved, resolver.OwnerName())
orkpods.Create(ctx, kube, obj, spec)
orkpods.Update(ctx, kube, obj, spec)  // image drift → delete and recreate
```

**Drift detection:** Pods are largely immutable. Image drift triggers a delete
and recreate since most Pod spec fields cannot be updated in place.

**`ResolvedPodSpec` fields:**

| Field | Type | Description |
|---|---|---|
| `Name` | string | Pod name |
| `Namespace` | string | Target namespace |
| `Image` | string | Container image |
| `Port` | int | Container port (optional) |
| `Labels` | map[string]string | Applied to Pod metadata |
| `Annotations` | map[string]string | Applied to Pod metadata |
| `Resources` | *ResourceRequirements | CPU and memory requests/limits |

## Template resolver

`pkg/orkestra-registry/template`

The Resolver evaluates Go `text/template` expressions against a live CR object.
It is the bridge between a Katalog template declaration and the literal values
that registry functions receive.

```go
import orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
```

### Creating a resolver

```go
resolver, err := orktmpl.NewResolver(ctx, obj)
```

The resolver builds a `map[string]interface{}` from the CR object. For
`*unstructured.Unstructured` — the common case — the full CR including all
spec fields is accessible. For typed objects, only metadata fields are
available via template expressions.

### Template context

```
.metadata.name          CR name
.metadata.namespace     CR namespace
.metadata.labels        CR labels map
.metadata.annotations   CR annotations map
.spec.*                 any spec field (unstructured CRDs only)
.status.*               any status field
```

Missing fields resolve to empty string — `missingkey=zero`. No error on
absent optional fields.

### Resolve methods

```go
// Single field — fast path for static values (no "{{" → returned as-is)
resolver.Resolve("{{ .spec.image }}")      // → "nginx:1.25"
resolver.Resolve("nginx:1.25")            // → "nginx:1.25" (no evaluation)

// Per-resource methods — resolve all fields in one call
resolver.ResolveDeploymentTemplate(src)
resolver.ResolveServiceTemplate(src)
resolver.ResolveSecretTemplate(src)
resolver.ResolveConfigMapTemplate(src)
resolver.ResolveServiceAccountTemplate(src)
resolver.ResolveJobTemplate(src)
resolver.ResolveCronJobTemplate(src)
resolver.ResolvePodTemplate(src)

// Slice — each element resolved independently, empty results dropped
resolver.ResolveStringSlice([]string{"{{ .metadata.namespace }}", "monitoring"})
// → ["my-namespace", "monitoring"]

// Labels — values resolved, keys are never templates
resolver.ResolveLabels([]orktypes.ResourceLabel{
    {Key: "app", Value: "{{ .metadata.name }}"},
    {Key: "env", Value: "production"},
})
// → [{Key: "app", Value: "my-website"}, {Key: "env", Value: "production"}]
```

### Namespace defaulting

When a `namespace` field is empty, all resolver methods default it to
`{{ .metadata.namespace }}` — the same namespace as the CR. You almost
never need to declare `namespace` explicitly for namespaced CRDs.

### Owner helpers

```go
resolver.OwnerName()       // CR name — used by Resolve() for default naming
resolver.OwnerNamespace()  // CR namespace
```

## Adding a new resource type

OrkestraRegistry is designed to grow. Every resource type follows the same
four-function pattern. Adding a new type — Ingress, HorizontalPodAutoscaler,
NetworkPolicy — is consistent and contained.

**1. Define the template source type in `orktypes.HookTemplates`:**

```go
type HookTemplates struct {
    // existing fields ...
    Ingresses []IngressTemplateSource `yaml:"ingresses" validate:"omitempty"`
}

type IngressTemplateSource struct {
    Version     string          `yaml:"version"      validate:"omitempty"`
    Name        string          `yaml:"name"         validate:"omitempty"`
    Namespace   string          `yaml:"namespace"    validate:"omitempty"`
    Host        string          `yaml:"host"         validate:"required"`
    ServiceName string          `yaml:"serviceName"  validate:"required"`
    ServicePort string          `yaml:"servicePort"  validate:"required"`
    TLSSecret   string          `yaml:"tlsSecret"    validate:"omitempty"`
    Labels      []ResourceLabel `yaml:"labels"       validate:"omitempty"`
    Reconcile   bool            `yaml:"reconcile"    validate:"omitempty"`
}
```

**2. Add a resolver method:**

```go
// pkg/orkestra-registry/template/resolver.go
func (r *Resolver) ResolveIngressTemplate(src orktypes.IngressTemplateSource) (orktypes.IngressTemplateSource, error) {
    resolved := orktypes.IngressTemplateSource{Version: src.Version}
    var err error
    if resolved.Name, err = r.Resolve(src.Name); err != nil { ... }
    if resolved.Host, err = r.Resolve(src.Host); err != nil { ... }
    if resolved.ServiceName, err = r.Resolve(src.ServiceName); err != nil { ... }
    if resolved.ServicePort, err = r.Resolve(src.ServicePort); err != nil { ... }
    // namespace default ...
    return resolved, nil
}
```

**3. Add the registry package:**

```
pkg/orkestra-registry/ingresses/
  ingress.go     — ResolvedIngressSpec, Create, Update, Delete, Resolve
```

Following the same contract as every other package. `Create` is idempotent.
`Update` recreates if not found. `Delete` is idempotent. `Resolve` applies
defaults and sets system labels.

**4. Add the runner in `pkg/reconciler/run_ingresses.go`:**

```go
func runIngresses(ctx context.Context, kube *kubeclient.Kubeclient,
    resolver *orktmpl.Resolver, owner domain.Object,
    srcs []orktypes.IngressTemplateSource, update bool) error {
    for i, src := range srcs {
        resolved, err := resolver.ResolveIngressTemplate(src)
        if err != nil { return fmt.Errorf("ingresses[%d]: %w", i, err) }
        spec := orkingress.Resolve(resolved, resolver.OwnerName())
        if update {
            orkingress.Update(ctx, kube, owner, spec)
        } else {
            orkingress.Create(ctx, kube, owner, spec)
            if src.Reconcile { orkingress.Update(ctx, kube, owner, spec) }
        }
    }
    return nil
}
```

**5. Call it from `runTemplateReconcile` in `generic.go`:**

```go
if err := runIngresses(ctx, kube, resolver, obj, t.Ingresses, false); err != nil {
    return err
}
```

That is the complete contribution. The Katalog immediately accepts:

```yaml
onCreate:
  ingresses:
    - name: "{{ .metadata.name }}-ingress"
      host: "{{ .spec.hostname }}"
      serviceName: "{{ .metadata.name }}-svc"
      servicePort: "80"
      reconcile: true
```

## Using OrkestraRegistry from Go hooks

OrkestraRegistry is designed to be called directly from Go hooks. Import
whichever packages you need and call the functions with resolved specs.

```go
// pkg/hooks/website.go
package hooks

import (
    "context"
    "fmt"

    "github.com/orkspace/orkestra/domain"
    orkdeploy "github.com/orkspace/orkestra/pkg/orkestra-registry/deployments"
    orksvc "github.com/orkspace/orkestra/pkg/orkestra-registry/services"
    orksecrets "github.com/orkspace/orkestra/pkg/orkestra-registry/secrets"
    orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
    "github.com/orkspace/orkestra/pkg/kubeclient"
    apiv1 "github.com/myorg/apis/website/v1alpha1"
)

func WebsiteHooks() domain.AnyReconcileHooks {
    return domain.ReconcileHooks[*apiv1.Website]{
        OnReconcile: func(ctx context.Context, obj *apiv1.Website) error {
            kube, _ := kubeclient.FromContext(ctx)
            resolver, _ := orktmpl.NewResolver(ctx, obj)

            // Deployment — using typed spec fields directly
            depSpec := orkdeploy.ResolvedDeploymentSpec{
                Name:      obj.Name,
                Namespace: obj.Namespace,
                Image:     obj.Spec.Image,      // type-safe access
                Replicas:  int32(obj.Spec.Replicas),
            }
            if err := orkdeploy.Create(ctx, kube, obj, depSpec); err != nil {
                return fmt.Errorf("deployment: %w", err)
            }

            // Secret — copy from platform namespace
            secretSpec := orksecrets.ResolvedSecretSpec{
                Name:          obj.Name + "-db",
                Namespace:     obj.Namespace,
                FromSecret:    "master-db-creds",
                FromNamespace: "platform",
            }
            if err := orksecrets.Create(ctx, kube, obj, secretSpec); err != nil {
                return fmt.Errorf("secret: %w", err)
            }

            return nil
        },
    }
}
```

## Default naming

When `name` is omitted from any template declaration, the registry applies
a default based on the owner CR name:

| Resource | Default |
|---|---|
| Deployment | `<cr-name>-deployment` |
| Service | `<cr-name>-svc` |
| Secret | `<cr-name>-secret` |
| ConfigMap | `<cr-name>-config` |
| ServiceAccount | `<cr-name>-sa` |
| Job | `<cr-name>-job` |
| CronJob | `<cr-name>-cronjob` |
| Pod | `<cr-name>-pod` |
