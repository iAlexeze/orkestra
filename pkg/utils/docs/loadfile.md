# Remote file loading and caching

`loadfile.go` and `file_cache.go` together make every remote file fetch in Orkestra cache-aware.

## How it works

```
LoadFile(path)                      ─┐
LoadFileWithAuth(path, auth)         ├─ delegates to LoadFileWithAuthRefresh(path, auth, refresh=false)
LoadFileWithAuthRefresh(path, auth, refresh)
```

For local paths (`/`, `./`, `../`, no `://`) the cache is bypassed — the file is always read from disk.

For `https://` or `http://` URLs:

1. Compute `SHA256(url)` → look up `~/.orkestra/files/<hex>/content`
2. If the file exists and `refresh` is `false`, return the cached bytes immediately.
3. Otherwise fetch from the network, write the response to the cache, and return the bytes.
4. If the cache write fails it is silently ignored — the bytes are still returned to the caller.

## Cache layout

```
~/.orkestra/
  files/
    <sha256hex>/
      content          ← raw file bytes, no metadata
```

One directory per URL. The directory name is the hex-encoded SHA256 of the full URL string.

## Invalidation

There is no TTL. Cached files persist until:

- `ork pull --refresh` is run (calls `InvalidateFileCache(url)` per URL before re-fetching)
- The user manually deletes `~/.orkestra/files/`

## Functions

| Function | Description |
|----------|-------------|
| `LoadFile(path)` | Load a local or remote file; remote hits use cache |
| `LoadFileWithAuth(path, auth)` | Same, with HTTP Basic or Bearer auth for remote fetches |
| `LoadFileWithAuthRefresh(path, auth, refresh)` | Full form; `refresh=true` bypasses and overwrites cache |
| `CachedFileBytes(url)` | Cache lookup — `([]byte, true)` on hit, `(nil, false)` on miss |
| `CacheFileBytes(url, data)` | Write bytes to cache |
| `InvalidateFileCache(url)` | Remove a cache entry |
