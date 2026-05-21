# 01 — Generators

`pkg/generate` contains one generator per artifact type. All generators follow the same shape: they accept `[]orktypes.CRDEntry` (or derived data), render output, and either write files or print to stdout when `dryRun` is true.

## Generator map

| Generator | File | Command | Output |
|-----------|------|---------|--------|
| TypeRegistry | `registry_generator.go` | `ork generate registry` | `pkg/typeregistry/zz_generated_typeregistry.go` |
| RBAC | `rbac_generator.go` | `ork generate rbac` | `rbac.yaml` — Namespace + ServiceAccounts + ClusterRoles + ClusterRoleBindings |
| ConfigMap | `configmap_generator.go` | `ork generate configmap` | `config.yaml` — Namespace + ConfigMap embedding the Katalog |
| Bundle | `bundle_generator.go` | `ork generate bundle` | `bundle.yaml` — RBAC + ConfigMap in document order |
| Dashboards | `dashboard_generator.go` | `ork generate dashboards` | `_generated/dashboards/<crd>.json` |
| Katalog scaffold | `katalog_generator.go` | `ork generate katalog` | `katalog.yaml` starter template |
| CRD + CR | `crd_generator.go` | `ork generate crd` | CRD manifest + sample CR YAML |

## Template rendering

Generators that produce text files (dashboards, katalog scaffold) use Go `text/template` via a shared helper:

```go
func renderTemplateToFile(tmpl *template.Template, data any, outPath string, gofmt bool, dryRun bool) error
```

Template files live in `templates/` and are embedded into the binary at compile time via `//go:embed templates/*` in `helper.go`. Adding a new template file requires no changes outside `helper.go`.

The `gofmt` flag runs `gofmt` on the output before writing — used only by the TypeRegistry generator since it produces Go source.

## Component selection: `BundleOptions`

RBAC and Bundle generators accept `BundleOptions` to control which Orkestra components are included:

```go
type BundleOptions struct {
    IncludeRuntime       bool  // orkestra SA + ClusterRole
    IncludeGateway       bool  // orkestra-gateway SA + ClusterRole
    IncludeControlCenter bool  // orkestra-cc SA (no ClusterRole)
}
```

The `--for` flag on `ork generate rbac` and `ork generate bundle` maps to this struct. When absent, `DefaultBundleOptions()` includes all three.

## Dashboard generator status

`dashboard_generator.go` produces a starting-point Grafana JSON file for each enabled CRD with four standard panels: queue depth, health status, p95 reconcile duration, and reconcile errors. The panels use Orkestra's Prometheus metric names.

The generated JSON is intentionally minimal — it provides a working starting point that you extend with CRD-specific panels, variables, and layout. Import the JSON into Grafana and customize from there.

→ Next: [02-bundle-rbac.md](02-bundle-rbac.md)
