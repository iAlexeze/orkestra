# pkg/plan

Computes the structured diff between two parsed Katalogs. Used by `ork plan` to show what would change if a local Katalog were applied to a cluster.

## Usage

```go
from, _ := katalog.ParseBytes(deployedYAML, ".")
to, _ := katalog.ParseFile("katalog.yaml")

diff := plan.ComputeKatalogDiff(from, to)
if diff.Empty() {
    fmt.Println("No changes.")
    return
}
diff.Print()
```

## What the diff covers

| Field | Description |
|-------|-------------|
| Added CRDs | CRDs present in the local Katalog but not in the deployed one |
| Removed CRDs | CRDs present in the deployed Katalog but not in the local one |
| Changed workers | Per-CRD worker count change |
| Changed resync | Per-CRD resync interval change |
| Changed crdFile | crdFile path change |
| operatorBox resource counts | Count changes per resource type in `onCreate` / `onReconcile` |

The diff does not compare template internals — only the structural fields that affect operator behaviour.
