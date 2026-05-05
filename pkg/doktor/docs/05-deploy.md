# 05 — Cluster Operations

The files in this area handle external operations: detecting cluster state, runtime health, persistent deploy state, and multi-project tracking. Image building and pushing are handled by [`pkg/buildx`](../../buildx/README.md).

## Image tag

```go
image := doktor.ImageTag("ghcr.io/myorg", "my-app", "a3f5c2b")
// → "ghcr.io/myorg/my-app:a3f5c2b"
```

The tag is the git commit short SHA by default. `ork deploy --tag v1.2.0` overrides it. When there is no git repository, the tag falls back to `"latest"`.

## Build and push

Image operations now live in `pkg/buildx`, which auto-selects Docker, Podman, or Buildah:

```go
err := buildx.BuildImage(".", "ghcr.io/myorg/my-app:a3f5c2b",
    buildx.ComposeBuild{Dockerfile: "Dockerfile.prod"}, os.Stdout)

err = buildx.PushImage("ghcr.io/myorg/my-app:a3f5c2b", os.Stdout)
```

See [`pkg/buildx`](../../buildx/README.md) for builder selection, compose-based builds, and the `--dockerfile` flag.

## Cluster connectivity

### ClusterReachable

```go
if !doktor.ClusterReachable() {
    // no cluster reachable on current kubeconfig context
}
```

Runs `kubectl cluster-info --request-timeout=5s` with a 5-second context timeout. Returns false when kubectl is missing, the kubeconfig is invalid, or the API server is unreachable.

`ork deploy` calls this before any cluster operations. When the check fails and `--dev` was not passed, the user is told to use `--dev`.

### KubectlAvailable

```go
if !doktor.KubectlAvailable() {
    return fmt.Errorf("kubectl not found in PATH")
}
```

### GoInstalled

```go
if !doktor.GoInstalled() {
    return fmt.Errorf("Go required to install kind")
}
```

Required before attempting to install kind via `go install`.

## kind cluster (--dev)

`kind.go` manages a local development cluster named `orkestra-playground`.

```go
err := doktor.EnsureKindCluster(doktor.KindClusterName)
```

- Creates the cluster if it does not already exist.
- Switches kubectl to the kind context (`kind-orkestra-playground`).
- Installs kind via `go install sigs.k8s.io/kind@v0.31.0` if not found in PATH or GOBIN.

`ork deploy --dev` calls `GoInstalled()` first; if Go is absent it returns an error with install instructions before attempting anything else.

## Orkestra installation

```go
if !doktor.OrkestraInstalled() || upgradeOrkestra {
    doktor.InstallOrUpgradeOrkestra(version, valuesFile, upgrade)
}
```

`helm.go` runs `helm install` or `helm upgrade --install` against `https://orkspace.github.io/orkestra`. `ork deploy` auto-detects `.orkestra/values.yaml` and passes it via `--values` when the `--values` flag is not set explicitly.

## Runtime health

`runtime.go` verifies the operator is healthy before patching any workload state.

```go
health := doktor.CheckRuntimeHealth()
// health.Running bool
// health.Reason  string  — set when Running is false
```

Checks:
1. Deployment `orkestra-runtime` exists in `orkestra-system`
2. At least one ready replica
3. No pod in `CrashLoopBackOff`

When unhealthy, `FetchRuntimeLogs()` saves the last 100 lines to `/tmp/orkestra/runtime.log` (and control-center logs to `/tmp/orkestra/controlcenter.log`) and returns the last 10 lines for inline display.

### Katalog change detection

```go
if doktor.KatalogChanged(dir) {
    doktor.RestartOrkestra()
}
```

`KatalogChanged` runs `git diff HEAD -- .orkestra/katalog.yaml` and `git diff HEAD~1 HEAD -- ...`. When either diff is non-empty, `RestartOrkestra` issues a rollout restart and waits up to 3 minutes.

## Ingress detection and auto-install

`DetectIngressController` probes for known ingress controller pods:

```go
ic := doktor.DetectIngressController()
// IngressNginx | IngressTraefik | IngressNone
```

`ork deploy` calls `ensureIngressController()` automatically when the project has a frontend (`HasFrontend == true`). It:
1. Calls `DetectIngressController()` — returns early if already present.
2. Detects a kind context and applies the kind-specific nginx manifest.
3. Otherwise installs via `helm install ingress-nginx ingress-nginx/ingress-nginx`.

## Persistent deploy state

`state.go` maintains `~/.orkestra/deploy/state.json` — the authoritative record of every deployed project on this machine.

```go
state, _ := doktor.LoadState()
state.RecordDeploy(appName, ns, katalogPath, newImage)  // captures previous image first
state.Save()

prev := state.PreviousImage(appName)  // "" when no previous deploy
```

`RecordDeploy` must be called **before** patching the cluster so `previousImage` always reflects what is currently live.

`CurrentContext()` returns the active kubectl context name (used to record which cluster was deployed to).

## Multi-project tracking (global Komposer)

`komposer.go` manages `~/.orkestra/deploy/komposer.yaml`, which aggregates Katalog paths from all projects deployed from this machine.

```go
komposer, _ := doktor.LoadGlobalKomposer()
komposer.RegisterKatalog("/abs/path/to/.orkestra/katalog.yaml")
komposer.Save()

names := komposer.DeployedProjects()  // ["my-app", "my-api"]
```

`ork deploy` registers the current project's Katalog path on every deploy and then attempts to run `ork kompose` to merge all Katalogs into a single `~/.orkestra/deploy/merged-katalog.yaml` that Orkestra can watch. This step is non-fatal — if `ork kompose` is not yet available, a warning is printed and the deploy continues normally.

## The full deploy sequence

`ork deploy` orchestrates everything in order:

```
 0. Cluster check         → --dev: EnsureKindCluster / else: ClusterReachable()
    Show context          → CurrentContext() + komposer.DeployedProjects()
 1. Build image           → doktor.Build(dir, image, w)
 2. Push image            → doktor.Push(image, w)
 3. Generate env bundle   → doktor.GenerateBundle(name, ns, secrets, config, bundleDir)
 4. Generate Katalog bundle → ork generate bundle -k katalog.yaml -w <ns> -o bundle/
 5. Apply bundle          → kubectl apply -f bundle/bundle.yaml
 6. Apply env files       → kubectl apply -f bundle/app-config.yaml, app-secrets.yaml
 7. Apply app.yaml        → kubectl apply -f .orkestra/app.yaml
 8. Register Komposer     → komposer.RegisterKatalog + applyKomposer (non-fatal)
 9. Install Orkestra      → helm install/upgrade (skipped when already installed)
10. Auto-install ingress  → ensureIngressController() when HasFrontend (non-fatal)
11. Health check          → CheckRuntimeHealth() + FetchRuntimeLogs() on failure
12. Katalog diff check    → KatalogChanged() → RestartOrkestra() if true
13. Record previous image → state.RecordDeploy() + annotation on ConfigMap
14. Patch image           → kubectl patch configmap <name> -n <ns> '{"data":{"image":...}}'
15. Wait for rollout      → kubectl rollout status deployment/<name> -n <ns> --timeout=5m
16. Print summary         → URL, image, Control Center link, log command
```

Steps 1–16 are skipped with `--dry-run`.

## Rollback

```bash
ork deploy rollback              # restore previous image from state.json
ork deploy rollback --image ghcr.io/myorg/app:a1b2c3d   # explicit image
```

Rollback reads `~/.orkestra/deploy/state.json` first and falls back to the `orkestra.io/previous-image` annotation on the ConfigMap for backward compatibility. It swaps `currentImage` ↔ `previousImage` in state before patching, so a second rollback call re-rolls-forward.

## Control Center URL

After a successful deploy, `ork deploy` prints:

```
  Control Center → https://control.mycompany.com
```

When `controlCenterHost` in `.orkestra/app.yaml` is empty, it falls back to:

```
  Control Center → http://orkestra-cc.orkestra-system.svc.cluster.local:8081
                   set controlCenterHost in .orkestra/app.yaml for external access
```

Set `controlCenterHost` (and configure the ingress via `.orkestra/values.yaml`) to expose the Control Center externally.
