# 01 — Cluster Operations

The cluster operations in `pkg/tools/cluster` cover three concerns: detecting whether Orkestra is installed and healthy, synchronising it after a Katalog change, and managing kind clusters for local development and E2E tests.

---

## Dependency checks

Before any cluster operation, verify that the required tools are present.

```go
if !ork.KubectlAvailable() {
    // kubectl not in PATH
}
if !ork.HelmAvailable() {
    // helm not in PATH
}
if !ork.ClusterReachable() {
    // no reachable cluster in current kubeconfig context
}

// Or install missing tools automatically:
if err := ork.EnsureDependencies(); err != nil {
    return err
}
```

`ClusterReachable` runs `kubectl cluster-info --request-timeout=5s` and returns `true` on a zero exit code. It times out after 5 seconds so it is safe to call during startup.

---

## Checking whether Orkestra is installed

```go
if ork.OrkestraInstalled() {
    // runtime Deployment exists in orkestra-system
}

if ork.RuntimeDeployed() {
    // same check, with a 10s context deadline
}
```

`OrkestraInstalled` calls `kubectl get deploy orkestra-runtime -n orkestra-system` and returns true when the Deployment is found. `RuntimeDeployed` is identical but uses a short context deadline — use it when you need a guaranteed timeout.

---

## Health checking

```go
status := ork.CheckRuntimeHealth()
if !status.Running {
    fmt.Println("Not ready:", status.Reason)
}
```

`CheckRuntimeHealth` polls every 2 seconds for up to 200 seconds. It returns `Running: true` as soon as `readyReplicas > 0`.

It returns immediately with a non-empty `Reason` when:
- The Deployment does not exist: `"deployment orkestra-runtime not found in orkestra-system"`
- A pod is in CrashLoopBackOff: `"pod orkestra-runtime-xyz is in CrashLoopBackOff"`
- The poll timeout expires: `"timeout (3m20s) waiting for Orkestra runtime to become ready"`

`RuntimeStatus.Reason` is always set when `Running` is false.

---

## Syncing the runtime after a Katalog change

When a Katalog ConfigMap is updated, the runtime must restart to reload it.

```go
if ork.KatalogChanged(dir) {
    if err := ork.SyncRuntime(); err != nil {
        return err
    }
}
```

`SyncRuntime` runs `kubectl rollout restart deploy/orkestra-runtime -n orkestra-system` followed by `kubectl rollout status` with a 3 minute timeout. It writes kubectl output to stdout so progress is visible.

`KatalogChanged` checks whether `.orkestra/katalog.yaml` has uncommitted changes or was touched by the most recent git commit. Returns `true` when the katalog has changed since the last deploy.

---

## Fetching runtime logs

```go
tail, err := ork.FetchRuntimeLogs()
if err != nil {
    // disk write failure
}
fmt.Println("Last 10 lines:", tail)
// full log at /tmp/orkestra/runtime.log
```

`FetchRuntimeLogs` saves the last 100 log lines from the runtime to `/tmp/orkestra/runtime.log`. If the Control Center Deployment exists but has no ready replicas, its logs are also saved to `/tmp/orkestra/controlcenter.log`. The return value is the last 10 lines of the runtime log for inline display.

---

## Kind cluster management

```go
// Create a cluster (downloads kind binary if not in PATH)
if err := ork.EnsureKindCluster("ork-e2e"); err != nil {
    return err
}

// Delete when done
if err := ork.DeleteKindCluster("ork-e2e"); err != nil {
    return err
}
```

`EnsureKindCluster` is idempotent — if the cluster already exists, it switches kubectl context to it and returns. For new clusters it creates them, waits for nodes to be Ready, and sets the kubectl context.

The kind binary is resolved from PATH first, then from `~/.orkestra/bin/kind`. If neither exists, the binary for the current OS/arch is downloaded from GitHub releases (`v0.27.0`).

`KindClusterName = "orkestra-playground"` is the default name used by `ork run --dev`.
