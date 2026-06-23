# ork migrate

Rewrite a controller-runtime `Reconcile` method to the Orkestra constructor signature and generate the full operator scaffolding — `katalog.yaml`, `simulate.yaml`, `e2e.yaml`, `go.mod`, `Makefile`, and `Dockerfile` — as a starting point.

```bash
ork migrate <file> [flags]
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | *(none)* | Write all output to this directory (non-destructive; skips confirmation prompt) |
| `--module` | | *(derived)* | Go module path for the migrated operator (e.g. `github.com/myorg/my-operator`) |
| `--name` | | *(derived)* | Operator name in kebab-case (e.g. `my-operator`). Derived from receiver type if omitted. |

## Examples

```bash
# Non-destructive: write to a new directory
ork migrate ./controller/webapp_controller.go -o ./my-operator

# Specify module path for go.mod and katalog.yaml location hints
ork migrate ./controller/webapp_controller.go \
  --module github.com/myorg/webapp-operator \
  -o ./webapp-operator

# Interactive: replace the file in place after confirmation
ork migrate ./controller/webapp_controller.go
```

## What it rewrites

| Before | After |
|--------|-------|
| `Reconcile(ctx, req ctrl.Request) (ctrl.Result, error)` | `Reconcile(ctx context.Context, key string) error` |
| `return ctrl.Result{}, err` | `return err` |
| `return ctrl.Result{}, nil` | `return nil` |
| `req.NamespacedName` | `client.ObjectKey{Namespace: namespace, Name: name}` |
| `req.String()` | `key` |
| `r.Status().Update(...)` | flagged `// TODO(ork migrate):` |
| `ctrl.Result{RequeueAfter: X}` | flagged `// TODO(ork migrate):` |
| `SetupWithManager` method | removed with explanation comment |
| `ctrl` import | removed |

## Output files

When `-o` is provided:

```text
my-operator/
  webapp_controller.go   rewritten reconciler
  katalog.yaml           constructor Katalog stub
  simulate.yaml          simulation stub
  e2e.yaml               end-to-end test stub
  go.mod                 module file with Orkestra pinned to this CLI version
  Makefile               registry, build, build-runtime, docker, release targets
  Dockerfile             distroless production image
```

## After migration

Search for `TODO(ork migrate)` in the output:

```bash
grep -rn "TODO(ork migrate)" ./my-operator/
```

Items that need manual attention:

1. Set `group`, `kind`, `plural`, `location` in `katalog.yaml`
2. Replace the embedded `client.Client` struct field with `kube kubeclient.KubeClient`
3. Update the constructor function to match the Orkestra signature
4. Replace `r.Status().Update()` with `r.kube.PatchStatus()`
5. Fill in resource assertions in `simulate.yaml` and `e2e.yaml`
6. Delete `main.go`, scheme registration, and manager setup
7. Run `go mod tidy`

→ Full review checklist: [pkg/migrate/README.md](../../../pkg/migrate/README.md)

## Context

`ork migrate` is the last step in the `from-controller-runtime` migration pack. The pack shows the same operator expressed five ways — from the controller-runtime baseline through declarative, hybrid, hooks-only, and constructor options — and `ork migrate` automates the constructor path for an existing operator.

```bash
ork init --pack from-controller-runtime
```
