# pkg/utils

`utils` is a general-purpose utility package for the Orkestra control plane. It groups helpers that are too small to warrant their own packages but are used across multiple layers of the system.

## File organisation

| File | Purpose |
|------|---------|
| `utils.go` | `Sleep`, `BoolPtr`, `Retry`, `IsRunningInCluster`, status constants, HTTP content-type constants |
| `yaml.go` | YAML marshal/unmarshal helpers, multi-document YAML splitting |
| `rawmap.go` | `map[string]interface{}` navigation helpers (`Get`, `Set`, `Delete`, `Navigate`) |
| `http.go` | HTTP client helpers used by external call integrations |
| `loadfile.go` | File loading helpers with path normalisation |
| `orkestra.go` | Orkestra-specific utilities (version strings, resource name helpers) |
| `printer.go` | CLI output formatting helpers |
| `prune_yaml.go` | YAML pruning — removes zero-value fields before applying to the cluster |
| `gvk.go` | GroupVersionKind parsing and formatting helpers |
| `crd.go` | CRD discovery utilities — REST mapper construction, GVR lookup by kind |

## Key utilities

**`Retry(fn func() error, opts RetryOptions) error`** — retries `fn` up to `opts.Attempts` times with exponential backoff capped at `opts.Delay`. Returns the last error when all attempts are exhausted.

**`IsRunningInCluster() bool`** — detects in-cluster execution by checking for the service account token file at `/var/run/secrets/kubernetes.io/serviceaccount/token`. Used by `pkg/konfig` for namespace resolution.

**`Navigate(m map[string]interface{}, path string) (interface{}, bool)`** — walks a dot-separated path through nested maps. Powers the `{{ .spec.someField }}` template resolver.

## Developer documentation

| Topic | Go to |
|-------|-------|
| How the REST mapper and CRD discovery client work | [docs/crd.md](docs/crd.md) |
| How dot-path navigation works | [pkg/types/docs/navigate-dot-path.md](../types/docs/navigate-dot-path.md) |
