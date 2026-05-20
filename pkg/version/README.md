# pkg/version

`version` exposes the build-time version string injected via `ldflags`. The three variables are always present — they default to `dev` / `none` / `unknown` for local builds.

```go
fmt.Println(version.String())  // "v1.2.3 (commit: abc1234, built: 2026-05-01T10:00:00Z)"
fmt.Println(version.Short())   // "v1.2.3"
```

## Setting version at build time

```sh
go build -ldflags "
  -X github.com/orkspace/orkestra/pkg/version.Version=v1.2.3
  -X github.com/orkspace/orkestra/pkg/version.Commit=$(git rev-parse --short HEAD)
  -X github.com/orkspace/orkestra/pkg/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)
" ./cmd/ork
```

The `Makefile` injects these automatically when building release binaries.
