# Source caching

The merger caches git repositories and remote Helm repository charts so that repeated `ork template`, `ork validate`, and `ork generate` calls don't clone or re-download anything after the first run.

## Cache namespaces

```
~/.orkestra/
  helm/
    git/<sha256>/    ← git-sourced charts (repo + ref + chart path)
    repo/<sha256>/   ← remote Helm repository charts (repo + chart + version)
  files/
    <sha256>/        ← remote HTTPS files (handled by pkg/utils)
```

## Cache keys

| Namespace | Key components |
|-----------|----------------|
| `helm/git` | `SHA256(repo + ref + subpath)` |
| `helm/repo` | `SHA256(repo + chart + version)` |

The key is the hex-encoded SHA256 of the tuple joined with `\x00`. All git refs (branch names, tags, commit SHAs) are treated equally — a `main` ref that advances is not automatically invalidated; use `--refresh` to re-fetch.

## Sentinel file

A cache entry is considered complete when `Chart.yaml` exists inside the cached directory. A partial copy (interrupted write) will be missing the sentinel and treated as a miss.

## Invalidation

There is no TTL. Cached entries persist until `ork pull --refresh` is run against the same komposer file or the user manually removes `~/.orkestra/helm/`.

## How caching threads through the merger

`Merger.Refresh bool` controls whether the cache is bypassed. It is set to `true` when `--refresh` is passed at the CLI level.

```
Merger.loadHelmSource(src)
  └── resolveChartPath(src, m.Refresh)
        ├── resolveGitChart(src, refresh)   — cache-first git clone
        └── resolveRemoteChart(src, refresh) — cache-first Helm pull
```

On a cache miss the resolved chart directory is copied into the cache before being returned. On a cache write failure the temp directory is returned directly and the error is logged at debug level — caching is always best-effort.

## Pre-warming with `ork pull`

`WarmHelmSource(src, refresh)` is exported so that `ork pull -f komposer.yaml` can pre-warm all helm sources before the user runs any other command:

```go
// cmd/cli/pull.go
for _, src := range helmImports.HelmSources {
    merger.WarmHelmSource(src, refresh)
}
```

This is equivalent to what the merger does on first use, run eagerly so subsequent commands are instant.
