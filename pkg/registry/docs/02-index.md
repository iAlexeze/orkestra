# 02 — Index and List

## The index artifact

`ork registry list` does not crawl the registry (GHCR does not expose a usable `_catalog` API for arbitrary namespaces). Instead it reads a single index artifact:

```
ghcr.io/orkspace/orkestra-registry/patterns/index:latest
```

The artifact is an OCI manifest wrapping one JSON blob. The blob is a `PatternIndex`:

```json
{
  "updatedAt": "2026-04-29T12:00:00Z",
  "patterns": [
    {
      "name": "postgres",
      "latestVersion": "v14",
      "description": "Production-ready PostgreSQL operator",
      "tags": ["database", "stateful"],
      "author": "orkspace"
    }
  ]
}
```

The index is stored using OCI artifact media type `application/vnd.orkestra.index.v1+json`.

## Auto-update on push

Every `client.Push` call ends with `updateIndex`:

```
fetchIndex(ctx, indexRef)     — pull current index:latest (empty struct if not found)
    ↓ upsert pattern entry
pushIndex(ctx, indexRef, index) — push updated index:latest
```

There is no separate reindex step. The index is always current as of the last push.

`updateIndex` is best-effort: if it fails (e.g. a concurrent push wrote a newer index, or a network error occurs), the failure is printed to stderr and the pattern push is considered successful. The index will be corrected by the next `push`.

## Index ref derivation

Given a pattern ref like `ghcr.io/orkspace/orkestra-registry/patterns/website:0.1.0`, the index ref is derived by replacing the last path segment with `index` and pinning the tag to `latest`:

```
Repository: orkspace/orkestra-registry/patterns/website
                                                ↑ last segment
Index repo: orkspace/orkestra-registry/patterns/index:latest
```

```go
func indexRefFrom(ref *Ref) (*Ref, error) {
    lastSlash := strings.LastIndex(ref.Repository, "/")
    namespace := ref.Repository[:lastSlash]
    return parseRef(ref.Registry + "/" + namespace + "/index:latest")
}
```

## List behaviour

`client.List` fetches the index and returns it. If the index does not yet exist (first ever push hasn't happened, or the registry is new), it returns an empty `PatternIndex` — not an error. The CLI displays "0 patterns" in that case.

```go
index, err := client.List(ctx, "")  // "" = use default registry
```

Passing a custom registry URL overrides the default:

```go
index, err := client.List(ctx, "oci://myregistry.internal/patterns")
```

## Custom registries

Any OCI-compliant registry works. The index artifact lives at `<base>/index:latest` where `<base>` is the registry URL without `oci://` and without a trailing slash.

```bash
export ORK_REGISTRY=oci://myregistry.internal/patterns
ork registry push my-operator:v1.0.0 ./my-operator/
ork registry list   # reads myregistry.internal/patterns/index:latest
```

→ Next: [03-resolve-cache.md](03-resolve-cache.md)
