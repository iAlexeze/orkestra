# Orkestra Templating Engine

Orkestra's templating engine is the subsystem that evaluates declarative
resource declarations in the Katalog and applies them to a live Kubernetes
cluster. It is how a Katalog entry like:

```yaml
onCreate:
  deployments:
    - image: "{{ .spec.image }}"
      replicas: "{{ .spec.replicas }}"
```

becomes a real Deployment with `nginx:1.25` and `2` replicas when your
Website CR says `spec.image: nginx:1.25` and `spec.replicas: 2`.

---

## How it works

The templating engine runs entirely at reconcile time. There is no code
generation step for dynamic template CRDs. When a CR event fires, the
engine reads the Katalog's template declarations, evaluates them against
the live CR, and calls the OrkestraRegistry to apply the results.

```
CR event
    │
    ▼
GenericReconciler.Reconcile()
    │
    ├── reads ReconcilerConfig.OnCreate from the Katalog
    ├── builds a Resolver from the CR object
    │
    ▼
runTemplateReconcile()
    │
    ├── runDeployments() → resolver.ResolveDeploymentTemplate() → orkdeploy.Create()
    ├── runServices()    → resolver.ResolveServiceTemplate()    → orksvc.Create()
    ├── runSecrets()     → resolver.ResolveSecretTemplate()     → orksecrets.Create()
    ├── runConfigMaps()  → resolver.ResolveConfigMapTemplate()  → orkcm.Create()
    └── ...
```

No generated file involved. No `ork generate runtime`. Just `ork run`.

---

## Template expressions

Any string field in a template declaration that contains `{{` is treated
as a Go `text/template` expression and evaluated against the live CR object
at reconcile time. Any string without `{{` is a static value used as-is.

```yaml
# Static — same for every CR of this type
image: nginx:1.25

# Dynamic — resolved from the CR spec at reconcile time
image: "{{ .spec.image }}"

# Mixed — CR name with a static suffix
name: "{{ .metadata.name }}-api"
```

The same field can hold either a static or a dynamic value without any additional YAML structure. Orkestra determines which it is at evaluation time.

---

## Template context

The template context is the full CR object as `map[string]interface{}`.
For dynamic CRDs (no `apiTypes.location`) this is the raw object from the
API server — all fields accessible including the complete `spec`.

```
.metadata.name        CR name
.metadata.namespace   CR namespace
.metadata.labels      CR labels map
.metadata.annotations CR annotations map
.spec.*               any spec field
.status.*             any status field
```

Missing fields resolve to empty string — `missingkey=zero` is set. This
prevents reconcile failures when optional spec fields are omitted.

**Example — Website CR:**

```yaml
# CR
spec:
  image: nginx:1.25
  replicas: 2
  port: 80
  serviceType: LoadBalancer
```

```yaml
# Katalog template declarations
deployments:
  - name: "{{ .metadata.name }}"           # → my-website
    image: "{{ .spec.image }}"             # → nginx:1.25
    replicas: "{{ .spec.replicas }}"       # → 2
    port: "{{ .spec.port }}"              # → 80
    namespace: "{{ .metadata.namespace }}" # → default

services:
  - name: "{{ .metadata.name }}-svc"       # → my-website-svc
    type: "{{ .spec.serviceType }}"        # → LoadBalancer
    port: "80"                             # static
    targetPort: "{{ .spec.port }}"         # → 80
```

---

## The Resolver

`pkg/orkestra-registry/template/resolver.go`

The Resolver is created once per reconcile from the CR object. It holds
the CR data as a `map[string]interface{}` and evaluates template expressions
against it.

```go
resolver, err := orktmpl.NewResolver(ctx, obj)
```

For `*unstructured.Unstructured` — the fast path. The object already has
a native map. Full spec accessible at zero cost.

For typed objects — only metadata fields are accessible. Users with typed
CRDs should use Go hooks for spec field access.

**Per-resource resolve methods:**

```go
resolver.ResolveDeploymentTemplate(src)    // all string fields in DeploymentTemplateSource
resolver.ResolveServiceTemplate(src)       // all string fields in ServiceTemplateSource
resolver.ResolveSecretTemplate(src)        // name, namespace, fromSecret, fromNamespace, toNamespaces
resolver.ResolveConfigMapTemplate(src)     // name, namespace, fromConfigMap, fromNamespace, toNamespaces
resolver.ResolveServiceAccountTemplate(src)
resolver.ResolveJobTemplate(src)           // name, image, namespace, command elements
resolver.ResolveCronJobTemplate(src)       // name, image, schedule, namespace
resolver.ResolveLabels(labels)             // label values only — keys are never templates
resolver.ResolveStringSlice(slice)         // each element resolved independently
```

Each method returns a new struct of the same type with all template
expressions replaced by their evaluated values. The original source struct
is never mutated.

---

## The three-step pipeline

Every resource runner follows the same three steps:

**Step 1 — Resolve**

Evaluate all template expressions in the source declaration. After this
step every field is a literal value.

```go
resolved, err := resolver.ResolveDeploymentTemplate(src)
// resolved.Image is now "nginx:1.25", not "{{ .spec.image }}"
```

**Step 2 — Build spec**

Translate the resolved template source into the registry's spec type.
This applies defaults, parses strings to int, builds label maps, and sets
the owner name for system labels.

```go
spec := orkdeploy.Resolve(resolved, staticReplicas, resolver.OwnerName())
```

**Step 3 — Apply**

Call the registry to apply the spec to the cluster. The registry handles
idempotency, owner references, and the actual API server call.

```go
orkdeploy.Create(ctx, kube, obj, spec)  // onCreate path
orkdeploy.Update(ctx, kube, obj, spec)  // onReconcile path
```

---

## onCreate, onReconcile, and `reconcile: true`

**`onCreate`** resources are created idempotently on every reconcile. If the
resource already exists it is skipped without error. Think of it as "ensure
this exists".

**`onReconcile`** resources are updated on every reconcile. If the resource
has been manually modified, it is reconciled back to the declared state. If it
was deleted, it is recreated. Think of it as "keep this in sync".

**`reconcile: true`** is a shorthand that combines both. Declaring it on
an `onCreate` resource means "create it and keep it in sync" without writing
a separate `onReconcile` block.

```yaml
# Without reconcile: true — two separate declarations needed
onCreate:
  deployments:
    - image: "{{ .spec.image }}"
onReconcile:
  deployments:
    - image: "{{ .spec.image }}"   # same thing written twice

# With reconcile: true — declared once, both behaviours
onCreate:
  deployments:
    - image: "{{ .spec.image }}"
      reconcile: true              # create + drift correction
```

The runner implements this as:

```go
// update=false means onCreate path
if err := orkdeploy.Create(ctx, kube, owner, spec); err != nil { ... }
if src.Reconcile {
    if err := orkdeploy.Update(ctx, kube, owner, spec); err != nil { ... }
}
```

---

## onDelete

`onDelete` declarations run after the CR's `DeletionTimestamp` is set,
before Orkestra removes its finalizers. For most resources this is not
needed — owner references cause cascade deletion automatically when the CR
is deleted. Declare `onDelete` only for resources that need explicit cleanup:

- Jobs that must complete before the CR is considered deleted
- External resources not in Kubernetes
- Resources in other namespaces that aren't covered by owner references

```yaml
onDelete:
  jobs:
    - name: "{{ .metadata.name }}-cleanup"
      image: busybox
      command: ["sh", "-c", "drain-queue.sh {{ .metadata.name }}"]
      backoffLimit: 3
```

Owner references handle everything else. Keep `onDelete` minimal.

---

## Secrets and ConfigMaps — additional patterns

Secrets and ConfigMaps support two patterns beyond simple creation.

**`fromSecret` / `fromConfigMap`** — copy data from an existing resource
in another namespace rather than declaring it inline:

```yaml
secrets:
  - name: db-credentials
    fromSecret: master-db-creds      # source secret name
    fromNamespace: platform          # where the source lives
    namespace: "{{ .metadata.namespace }}"
    reconcile: true                  # re-sync if source changes
```

Orkestra reads the source at reconcile time and copies its data. When
the source Secret rotates, the copy is updated on the next reconcile loop.

**`toNamespaces`** — create one copy in each listed namespace:

```yaml
secrets:
  - name: registry-pull-secret
    fromSecret: docker-registry-creds
    fromNamespace: platform
    toNamespaces:
      - "{{ .metadata.namespace }}"
      - "{{ .metadata.namespace }}-staging"
      - monitoring
```

Orkestra reads the source once and writes copies to every namespace.
Each copy gets an owner reference back to the CR.

**ConfigMap merge** — combine a base ConfigMap with inline overrides:

```yaml
configMaps:
  - name: app-config
    fromConfigMap: base-app-config   # source ConfigMap
    fromNamespace: platform
    namespace: "{{ .metadata.namespace }}"
    data:
      LOG_LEVEL: "{{ .spec.logLevel }}"  # override — wins over source value
    reconcile: true
```

Source keys are copied first, then `data` keys override matching ones.
The result is a merged ConfigMap in the target namespace.

---

## Conditional Provisioning
Orkestra supports conditional provisioning with the `when` block.

```yaml
services:
  - name: "{{ .metadata.name }}-svc"
    type: "{{ .spec.serviceType }}"
    port: "80"
    targetPort: "{{ .spec.port }}"
    namespace: "{{ .metadata.namespace }}"
    reconcile: true
    when:
      - field: spec.exposePublicly
        equals: "true"
```

The result is that a service is only created when `spec.exposePublicly` is
`true`.

All conditions must pass (AND semantics).  
If any condition fails, the resource is skipped for this reconcile cycle.

> For more details on conditions, see  
> 👉 **[Conditional Provisioning](../docs/conditional-provisioning.md)**.

---

## Default namespace

When `namespace` is omitted from any template declaration, the resolver
defaults it to `{{ .metadata.namespace }}` — the same namespace as the CR.
This means you almost never need to declare `namespace` explicitly for
namespaced CRDs.

```yaml
# These two are equivalent for a namespaced CRD
deployments:
  - image: nginx:1.25
    # namespace omitted — defaults to CR namespace

deployments:
  - image: nginx:1.25
    namespace: "{{ .metadata.namespace }}"  # explicit, same result
```

---

## Default naming

When `name` is omitted, each resource type has a default naming pattern:

| Resource | Default name pattern |
|---|---|
| Deployment | `{{ .metadata.name }}-deployment` |
| Service | `{{ .metadata.name }}-svc` |
| Secret | `{{ .metadata.name }}-secret` |
| ConfigMap | `{{ .metadata.name }}-config` |
| ServiceAccount | `{{ .metadata.name }}-sa` |
| Job | `{{ .metadata.name }}-job` |
| CronJob | `{{ .metadata.name }}-cronjob` |

---

## System labels and annotations

Orkestra always adds two labels to every resource it creates. These are
not overridable:

### Labels
```bash
orkestra.konductor.io/managed=true
```

### Annotations
```bash
orkestra.konductor.io/managed-by: <cr-name>
orkestra.konductor.io/managed-since: <timestamp>
```

`orkestra.konductor.io/managed` identifies the resource as Orkestra-managed.
This is used as the pod selector by Services — it ensures a Service created for
a CR only routes to pods owned by that CR.

`orkestra.konductor.io/managed-by` identifies the particular CR that created
the resource. Name comes from `katalog.metadata.name`. or `komposer.metadata.name`

Additional labels declared in the template are merged alongside these:

```yaml
labels:
  - key: app
    value: "{{ .metadata.name }}"
  - key: environment
    value: "{{ .spec.environment }}"
  # managed, managed-by and managed-since are always added automatically
```

---

## Owner references

Every resource created by the templating engine has an owner reference
pointing to the CR that triggered its creation. This means cascade deletion
is automatic — when the CR is deleted, all child resources are garbage
collected by Kubernetes without any `onDelete` declarations.

```yaml
ownerReferences:
  - apiVersion: demo.orkestra.io/v1alpha1
    kind: Website
    name: my-website
    uid: fb0b6ae7-...
    controller: true
    blockOwnerDeletion: true
```

The only exception is `onDelete` Jobs — they cannot have owner references
because the CR is being deleted when they run. They must complete
independently after the CR is gone.

---

## When to use Go hooks instead

Declarative templates handle the majority of operator use cases. Use Go
hooks when you need:

- **Type-safe spec access** — templates only see field names as strings.
  Go hooks have compiled access to `obj.Spec.Image`, `obj.Spec.Replicas` etc.
- **External API calls** — provision cloud resources, call DNS APIs, notify
  external systems
- **Status updates** — write back to `obj.Status` with computed values

Declare Go hooks in the Katalog:

```yaml
reconciler:
  default: true
  hooks:
    location: github.com/myorg/hooks
    function: WebsiteHooks
```

Then run `ork generate runtime` to register the hook function. The reconciler
will use your Go hook instead of the template path.

---

## When `ork generate runtime` is needed

```
CRD type                                    generate runtime needed?
──────────────────────────────────────────────────────────────────────
Dynamic CRD, templates only                 NO
Dynamic CRD with reconciler.hooks           YES — registers Go hook
Typed CRD (apiTypes.location set)           YES — registers Go type + scheme
reconciler.default: false (custom)          YES — registers constructor
──────────────────────────────────────────────────────────────────────
```

For pure dynamic operators — the most common case — `ork generate runtime`
is never needed. The complete workflow is:

```bash
ork init my-operator
cd my-operator
kubectl apply -f examples/website/website-crd.yaml
ork run --katalog examples/website/website-katalog.yaml
```

**Whats Next?**
  - [Orkestra Use Cases](../docs/use-cases.md)
  - [What is a Katalog](../docs/katalog.md)
  - [What is a Komposer](../docs/komposer.md)
  - [How to decide which Orkestra input model is right for your operator](../docs/choosing-katalog-vs-komposer.md)