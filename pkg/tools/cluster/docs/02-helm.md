# 02 — Helm Installation

`pkg/tools/cluster` wraps the Helm operations Orkestra needs to install and configure itself on a cluster. These are thin wrappers around `helm upgrade --install` — not a general-purpose Helm client.

---

## Installing or upgrading Orkestra

```go
// Latest version, no extra flags
err := ork.InstallOrUpgradeOrkestra("", nil)

// Specific version with --set
err = ork.InstallOrUpgradeOrkestra("0.5.0", nil,
    "--set", "gateway.enabled=true",
    "--atomic",
)

// Values file
err = ork.InstallOrUpgradeOrkestra("", []string{"./prod-values.yaml"},
    "--set", "runtime.replicaCount=3",
)
```

`InstallOrUpgradeOrkestra` always runs `helm upgrade --install`, making it safe whether Orkestra is already present or not. Before every call it runs `helm repo add` and `helm repo update` for the Orkestra chart repo so the index is always current.

The chart is installed into `OrkestraNamespace` (`orkestra-system`) with `--create-namespace`. The namespace is created if it does not exist.

**Additional args** are appended verbatim to the Helm command. Pass any Helm flag here: `--atomic`, `--wait`, `--timeout`, `--set`, `--set-string`.

### Constants

| Constant | Value |
|----------|-------|
| `Orkestra` | `"orkestra"` — Helm release name |
| `OrkestraNamespace` | `"orkestra-system"` |
| `OrkestraChartRepo` | `"https://orkspace.github.io/orkestra"` |
| `OrkestraChartName` | `"orkestra"` |

---

## Building a Control Center values file

When deploying with a custom ingress host for the Control Center:

```go
valuesFile, err := ork.BuildControlCenterValues("control-center.myorg.io")
if err != nil {
    return err
}
defer os.Remove(valuesFile)

err = ork.InstallOrUpgradeOrkestra("", []string{valuesFile})
```

`BuildControlCenterValues` writes a temporary YAML file that enables the Control Center ingress for the given hostname. The caller is responsible for deleting it when done (`defer os.Remove(valuesFile)`).

The generated file:

```yaml
controlCenter:
  ingress:
    enabled: true
    hosts:
      - host: <host>
        paths:
          - path: /
            pathType: Prefix
```

---

## Where this is used

| Caller | What it does |
|--------|-------------|
| `pkg/registry/e2e/runner.go` | Installs Orkestra before running E2E test expectations |
| `cmd/cli/deploy.go` | Installs/upgrades Orkestra during `ork doctor deploy` |
