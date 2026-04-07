# Integration Tests

Integration tests verify multi-package behaviour across the Orkestra system. They use
real file I/O, real in-process dependency graphs, and real validation pipelines — but
do **not** require a live Kubernetes cluster.

## Running

```bash
make test-integration
```

This is equivalent to:

```bash
go test ./tests/integration/... -v -tags=integration -count=1
```

The `integration` build tag keeps these tests out of `make test-unit`. They run
independently so CI can separate fast unit feedback from slower multi-package checks.

## Structure

```
tests/integration/
├── activation/          CRD health lifecycle — appears, disappears, reappears
├── dependency/          Topological sort and cycle detection across the katalog graph
├── komposer/            Merger loading Katalog/Komposer files, source composition, dedup
└── reconciler/          Validation rule pipelines for deployment, service, and secret CRDs
```

Each subdirectory is its own `package *_test` with the `//go:build integration` tag at
the top of every file.

## What counts as an integration test here

| Criterion | Integration | Unit |
|-----------|-------------|------|
| Spans multiple packages | Yes | No — stays in one package |
| Writes real temp files | Yes (komposer/) | No |
| Requires live Kubernetes | No | No |
| Requires network | No | No |

If a test needs a live cluster (watching informers, applying CRs, checking health
endpoints) it belongs in `tests/e2e/`, not here.

## Writing a new integration test

1. Pick the right subdirectory — or create one if the domain is new.
2. Add `//go:build integration` as the **first line** (before the package declaration).
3. Use `package <dir>_test` (black-box — import via the public API).
4. Avoid global state that leaks between tests. Use `t.Cleanup` to remove temp files.
5. Call `t.Helper()` on shared helper functions so failure lines point to the caller.

### Example

```go
//go:build integration

package komposer_test

import (
    "os"
    "testing"
    "github.com/ialexeze/orkestra/pkg/merger"
)

func TestMerger_LoadsKatalogFromFile(t *testing.T) {
    f, _ := os.CreateTemp("", "*.yaml")
    f.WriteString(`apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: my-katalog
spec:
  crds:
    - name: website
      enabled: true
`)
    f.Close()
    t.Cleanup(func() { os.Remove(f.Name()) })

    m := merger.New(f.Name())
    if err := m.Merge(); err != nil {
        t.Fatalf("Merge() error: %v", err)
    }
    if m.Count() != 1 {
        t.Errorf("expected 1 CRD, got %d", m.Count())
    }
}
```

## Test helpers and test exports

Some integration tests need access to unexported internals. The pattern used in
this project is a `test_exports.go` file inside the target package:

```go
// pkg/katalog/test_exports.go
package katalog

func NewKatalogForTest(crds []orktypes.CRDEntry) *Katalog { ... }
func DetectCyclesForTest(k *Katalog) error { ... }
```

This file is compiled into **all** builds (not just tests) but is tiny and adds no
runtime overhead. It follows the same pattern as `pkg/merger/test_exports.go` and
`pkg/health/test_exports.go`.

## CI

Integration tests run in the `test-integration` job after unit tests pass. They do not
require cluster credentials. The `//go:build integration` tag ensures they are never
accidentally included in the unit test run.
