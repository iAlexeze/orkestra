# 05 — Docker and Cluster Operations

The final two files in the package handle external operations: building and pushing a Docker image, and detecting the state of the target cluster.

## Docker operations

`docker.go` wraps the Docker CLI. There is no Docker API dependency — it shells out to `docker`, which is always available when someone is deploying an image.

### ImageTag

```go
image := doktor.ImageTag("ghcr.io/myorg", "my-app", "a3f5c2b")
// → "ghcr.io/myorg/my-app:a3f5c2b"
```

The tag is the git commit short SHA by default. `ork deploy --tag v1.2.0` overrides it. When there is no git repository, the tag falls back to `"latest"`.

### Build

```go
err := doktor.Build(".", "ghcr.io/myorg/my-app:a3f5c2b", os.Stdout)
// runs: docker build -t ghcr.io/myorg/my-app:a3f5c2b .
```

Stdout and stderr are streamed to the provided `io.Writer` — `ork deploy` passes `os.Stdout` so the developer sees the build output live.

### Push

```go
err := doktor.Push("ghcr.io/myorg/my-app:a3f5c2b", os.Stdout)
// runs: docker push ghcr.io/myorg/my-app:a3f5c2b
```

Same streaming behaviour. The caller is responsible for ensuring the user is already authenticated to the registry.

## Cluster detection

`ingress.go` probes the cluster via `kubectl` before applying resources.

### KubectlAvailable

```go
if !doktor.KubectlAvailable() {
    return fmt.Errorf("kubectl not found in PATH")
}
```

Uses `exec.LookPath` — no network call, no cluster required.

### OrkestraInstalled

```go
if !doktor.OrkestraInstalled() {
    // install via helm
}
```

Checks for the `orkestra-system` namespace. If missing, `ork deploy` runs `helm install orkestra` before applying the bundle.

### DetectIngressController

```go
ic := doktor.DetectIngressController()
switch ic {
case doktor.IngressNginx:
    // ingress-nginx
case doktor.IngressTraefik:
    // traefik
case doktor.IngressNone:
    // no known ingress controller found
}
```

Probes for known ingress controller pods across all namespaces using label selectors. Returns `IngressNone` if `kubectl` is not available or no known controller is found.

This is used by `ork doktor` to report what the cluster has, but it does not block generation — the developer can add an ingress controller after the initial deploy.

## The full deploy sequence

`ork deploy` orchestrates everything in order:

```
1. Detect project          → doktor.Detect(".")
2. Build image             → doktor.Build(dir, image, w)
3. Push image              → doktor.Push(image, w)
4. Generate bundle         → doktor.GenerateBundle(...)
5. Generate Katalog bundle → ork generate bundle
6. Check Orkestra          → doktor.OrkestraInstalled()
7. Install if missing      → helm install orkestra ...
8. Apply bundle            → kubectl apply -f .orkestra/bundle/
9. Patch image in CR       → kubectl patch configmap <name> ...
10. Watch until ready      → poll CR status.phase until "Ready"
```

Steps 2–3 are skipped with `--dry-run`. The developer sees progress output at each step.
