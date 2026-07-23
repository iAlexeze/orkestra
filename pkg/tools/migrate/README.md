# pkg/tools/migrate

`migrate` rewrites a controller-runtime `Reconcile` method to the Orkestra constructor signature. It is invoked by `ork migrate` and produces a rewritten Go file plus the full Orkestra scaffolding — `katalog.yaml`, `simulate.yaml`, `e2e.yaml`, `go.mod`, `Makefile`, and `Dockerfile` — as a starting point.

---

## Try it first

Before reading further, pull the migration pack and explore the full before/after:

```bash
ork init --pack from-controller-runtime
```

This gives you eight progressive examples — from the raw controller-runtime baseline (`00-controller-runtime-baseline`) through five migration options to the automated `ork migrate` output (`06-ork-migrate`). The step-by-step narrative is in `documentation/guides/migration/`.

---

## Usage

```bash
ork migrate ./controller/webapp_controller.go -o ./my-operator
ork migrate ./controller/webapp_controller.go --module github.com/myorg/my-operator -o ./out
ork migrate ./controller/webapp_controller.go   # prompts before replacing in place
```

---

## What it rewrites

| Before (controller-runtime) | After (Orkestra constructor) |
|-----------------------------|------------------------------|
| `Reconcile(ctx, req ctrl.Request) (ctrl.Result, error)` | `Reconcile(ctx context.Context, key string) error` |
| `return ctrl.Result{}, err` | `return err` |
| `return ctrl.Result{}, nil` | `return nil` |
| `req.NamespacedName` | `client.ObjectKey{Namespace: namespace, Name: name}` |
| `req.String()` | `key` |
| `r.Status().Update(...)` | flagged with `// TODO(ork migrate):` |
| `ctrl.Result{RequeueAfter: X}` | flagged with `// TODO(ork migrate):` |
| `SetupWithManager` | removed with explanation comment |
| `ctrl` import | removed |
| logging imports | left untouched — keep your logger |

`r.client.Patch(ctx, obj, client.MergeFrom(...))` lines pass through unchanged and compile as-is — `kubeclient.Patch` is a type alias for `sigs.k8s.io/controller-runtime/pkg/client.Patch`, so existing patch calls work without modification. The only change needed is the method receiver: `r.client` → `r.kube`.

---

## What it generates

| File | Description |
|------|-------------|
| `<reconciler>.go` | Rewritten source with `TODO(ork migrate):` markers for manual review |
| `katalog.yaml` | Constructor Katalog stub — fill in group, kind, location |
| `simulate.yaml` | Simulation stub — fill in expected resource kinds |
| `e2e.yaml` | E2E test stub — fill in CR name, resource assertions |
| `go.mod` | Module file with Orkestra dependency pinned to the CLI version |
| `Makefile` | Standard typed operator Makefile — registry, build, build-runtime, docker, release |
| `Dockerfile` | Distroless production image — same as all typed examples |

---

## Review checklist

After running `ork migrate`, search for `TODO(ork migrate)` in the output directory:

```bash
grep -rn "TODO(ork migrate)" .
```

Work through each marker in order:

- [ ] Set `group`, `kind`, `plural`, `location` in `katalog.yaml`
- [ ] Replace the embedded `client.Client` struct field with `kube kubeclient.KubeClient`
- [ ] Update `NewXxx` constructor to accept `(kube kubeclient.KubeClient, informer cache.SharedIndexInformer, ev event.Recorder)`
- [ ] Rename `r.client` → `r.kube` at all call sites (patch lines compile unchanged — only the receiver name changes)
- [ ] Replace `r.Status().Update()` with `r.kube.PatchStatus(ctx, obj, gvr, map[string]interface{}{...})`
- [ ] Add `github.com/orkspace/orkestra/domain` and `pkg/kubeclient` imports
- [ ] Fill in resource assertions in `simulate.yaml` and `e2e.yaml`
- [ ] Delete `main.go`, scheme registration, and manager setup — Orkestra provides the informer, workqueue, and worker pool

---

## What Orkestra hands you for free

When a constructor reconciler runs inside Orkestra, you keep your existing logic and gain:

| Concern | Orkestra |
|---------|----------|
| Informer watching your CRD | ✓ |
| Workqueue with dedup and backoff | ✓ |
| Worker pool | ✓ |
| Panic recovery (`safeReconcile`) | ✓ |
| Leader election | ✓ |
| Prometheus metrics | ✓ |
| Health tracking | ✓ |
| `ork control` UI | ✓ |

---

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand the full signature change and what the output looks like | [docs/01-output.md](docs/01-output.md) |
| See a before/after of the generated files | [docs/02-generated-files.md](docs/02-generated-files.md) |
| Understand what the tool cannot auto-fix | [docs/03-limitations.md](docs/03-limitations.md) |
