# pkg/migrate

`migrate` rewrites a controller-runtime `Reconcile` method to the Orkestra constructor signature. It is invoked by `ork migrate` and produces a rewritten Go file plus the full Orkestra scaffolding — `katalog.yaml`, `simulate.yaml`, `e2e.yaml`, `go.mod`, `Makefile`, and `Dockerfile` — as a starting point.

```sh
ork migrate ./controller/webapp_controller.go -o ./my-operator
ork migrate ./controller/webapp_controller.go --module github.com/myorg/my-operator -o ./out
ork migrate ./controller/webapp_controller.go   # prompts before replacing in place
```

## What it rewrites

| Before | After |
|--------|-------|
| `Reconcile(ctx, req ctrl.Request) (ctrl.Result, error)` | `Reconcile(ctx context.Context, key string) error` |
| `return ctrl.Result{}, err` | `return err` |
| `return ctrl.Result{}, nil` | `return nil` |
| `req.NamespacedName` | `client.ObjectKey{Namespace: namespace, Name: name}` |
| `req.String()` | `key` |
| `r.Status().Update(...)` | flagged with `// TODO(ork migrate):` |
| `ctrl.Result{RequeueAfter: X}` | flagged with `// TODO(ork migrate):` |
| `SetupWithManager` | removed with explanation comment |
| `ctrl` import | removed |
| logging imports | left untouched — users keep their logger |

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

## Review checklist

After running `ork migrate`, search for `TODO(ork migrate)` in the output directory:

- Set `group`, `kind`, `plural`, `location` in `katalog.yaml`
- Replace `r.Status().Update()` with `r.kube.PatchStatus()`
- Replace the embedded `client.Client` struct field with `kube kubeclient.KubeClient`
- Update `NewXxx` constructor to accept `(kube kubeclient.KubeClient, informer cache.SharedIndexInformer, ev event.Recorder)`
- Add `github.com/orkspace/orkestra/domain` and `pkg/kubeclient` imports
- Fill in resource assertions in `simulate.yaml` and `e2e.yaml`
- Delete `main.go`, scheme registration, and manager setup

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand the full signature change and what the output looks like | [docs/01-output.md](docs/01-output.md) |
| See a before/after of the generated files | [docs/02-generated-files.md](docs/02-generated-files.md) |
| Understand what the tool cannot auto-fix | [docs/03-limitations.md](docs/03-limitations.md) |
