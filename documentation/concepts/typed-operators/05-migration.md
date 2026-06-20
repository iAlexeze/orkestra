# Migrating from controller-runtime

You have a working Kubernetes operator. The question is not whether controller-runtime works — it does. The question is what it costs: informers, workqueues, worker pools, leader election, status, finalizers, events, metrics — written from scratch for every CRD.

Orkestra removes the machinery. You keep the business logic.

---

## The migration pack

The `from-controller-runtime` pack shows the same WebApp operator expressed five ways so you can see what you are choosing between:

```bash
ork init --pack from-controller-runtime
```

| Option | Go required | What you own |
|--------|-------------|--------------|
| **00 — controller-runtime baseline** | Yes — full | Everything: informers, manager, scheme, main.go |
| **01 — declarative** | No | Nothing — pure YAML |
| **02 — hybrid** | Yes — hook only | The 10% templates can't express |
| **03 — hooks only** | Yes — all resources | All child resource specs in Go |
| **04 — constructor: lift and change** | Yes — full reconciler | Reconcile logic; manager removed |
| **05 — constructor: Orkestra resources** | Yes — full reconciler | Reconcile logic; resource ops simplified |

Start at `00` and follow the READMEs — each step removes one layer of machinery.

---

## The mechanical path: `ork migrate`

`ork migrate` automates option 04 — it takes your reconciler file, rewrites the signature, and generates the scaffolding:

```bash
ork migrate ./controller/webapp_controller.go -o ./my-operator
```

Output:

```
my-operator/
  webapp_controller.go   rewritten — signature changed, ctrl.Result collapsed
  katalog.yaml           constructor Katalog stub — fill in group, kind, location
  simulate.yaml          simulation stub — fill in expected resources
  e2e.yaml               end-to-end stub — fill in CR assertions
  go.mod                 module file with Orkestra pinned to this CLI version
```

### What the rewrite does

**Signature change — the only mechanical step:**

```go
// Before
func (r *WebAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)

// After
func (r *WebAppReconciler) Reconcile(ctx context.Context, key string) error
```

`key` is `namespace/name` — the same as `req.String()`. Orkestra calls this from its worker pool.

**Return values collapsed:**

```go
return ctrl.Result{}, err   →   return err
return ctrl.Result{}, nil   →   return nil
```

**Struct and constructor rewritten:**

The embedded `client.Client` is replaced with Orkestra's interfaces, and a constructor function is generated:

```go
// Before
type WebAppReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

// After
type WebAppReconciler struct {
    informer cache.SharedIndexInformer
    kube     kubeclient.KubeClient
    ev       event.Recorder
}

func NewWebAppReconciler(
    kube kubeclient.KubeClient,
    informer cache.SharedIndexInformer,
    ev event.Recorder,
) domain.Reconciler {
    return &WebAppReconciler{kube: kube, informer: informer, ev: ev}
}
```

**SetupWithManager removed:**

```go
// Replaced with a comment:
// SetupWithManager removed — Orkestra provides the informer, workqueue,
// worker pool, leader election, panic recovery, and metrics.
```

**What still needs manual review:**

- `r.Get`, `r.Create`, `r.Patch` in sub-methods — change to `r.kube.*` calls
- `r.Status().Update()` — flagged `// TODO(ork migrate):` — replace with `r.kube.PatchStatus()`
- `ctrl.Result{RequeueAfter: X}` — flagged — Orkestra uses exponential backoff; return an error to retry
- Fill in `group`, `kind`, `location` in `katalog.yaml`
- Delete `main.go`, scheme registration, and manager setup

Search after migration:

```bash
grep -rn "TODO(ork migrate)" ./my-operator/
```

---

## Option 05: Orkestra resources

After completing option 04, you can go one step further and replace the manual Get / IsNotFound / Create / Patch pattern with `pkg/resources`:

```go
// Before — manual (option 04)
existing := &appsv1.Deployment{}
err := r.kube.Get(ctx, namespace, name, existing)
if errors.IsNotFound(err) { return r.kube.Create(ctx, desired) }
patch := sigs.MergeFrom(existing.DeepCopy())
existing.Spec = desired.Spec
return r.kube.Patch(ctx, existing, patch)

// After — Orkestra resources (option 05)
return orkdeploy.Update(ctx, r.kube, webapp, spec)
```

`Update` handles create-if-absent, drift correction, owner references, and system labels. `DeleteIfOwned` removes a resource only if this CR owns it.

---

## See also

- `from-controller-runtime` pack — `ork init --pack from-controller-runtime`
- [ork migrate reference](../../reference/cli/migrate.md)
- [Constructor concept](./02-constructor.md) — deep dive on the constructor pattern
