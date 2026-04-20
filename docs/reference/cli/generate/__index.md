# ork generate

Generate Orkestra components from one or more Katalog files.

```bash
ork generate <command> [flags]
```

The `generate` command group contains tools for producing CRDs, example CRs, runtime registries, documentation, dashboards, RBAC, ConfigMaps, installation bundles, and more.

Each generator reads one or more `katalog.yaml` files, merges them, and produces the requested artifacts.

---

## Available Commands

| Command     | Description                                                   |
|-------------|---------------------------------------------------------------|
| [crd](./crd.md)           | Generate Kubernetes CRDs from a Katalog                     |
| [cr](./cr.md)             | Generate example CustomResources for a CRD                  |
| [registry](./registry.md) | Generate zz_generated_runtime_registry.go for typed operators |
| [docs](./docs.md)         | Generate Markdown documentation for all CRDs *(in development)* |
| [dashboards](./dashboards.md) | Generate Grafana dashboards for all CRDs *(in development)* |
| [rbac](./rbac.md)         | Generate a minimal ClusterRole based on the Katalog        |
| [configmap](./configmap.md) | Generate a ConfigMap embedding a Katalog or Komposer       |
| [bundle](./bundle.md)     | Generate a complete installation bundle (RBAC + ConfigMap) |
| [all](./all.md)           | Run all generators (registry, docs, dashboards)            |

---

## Shared Flags

Most `generate` subcommands support:

| Flag | Description |
|------|-------------|
| `-k, --katalog <file>` | One or more Katalog files (comma‑separated or repeated) |
| `-o, --output <path>` | Write output to file or directory |
| `-n, --namespace <name>` | Namespace for generated manifests (default: `orkestra-system`) |
| `--dry-run` | Print output to stdout without writing files |
