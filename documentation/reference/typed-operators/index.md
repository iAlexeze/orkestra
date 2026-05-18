#

If an operatorBox entry sets `version` and `fetch: true`, the CLI will attempt a best-effort
`go get <location>@<version>` after ensuring the project is a valid Go module. This may update
go.mod/go.sum. For private modules, set GOPRIVATE and ensure credentials are available.

```yaml
operatorBox:
  hooks:
    location: github.com/myorg/hooks
    version: v2.1.0
    fetch: true                     # Orkestra will try to fetch the module at: <location>@<version>
    function: ProjectHooks
    alias: projecthooks

  constructor:
    # If @version in location differs from explicit declaration:
    # Orkestra will warn, but use the explicit version (v0.3.1) instead
    location: github.com/myorg/constructors@v0.3.0
    version: v0.3.1
    fetch: false
    function: NewManagedNamespaceReconciler
    alias: nsrec
```