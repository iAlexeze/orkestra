# 02 — `.env` Parsing

`ParseEnvFile` reads a `.env` file and returns a slice of `EnvVar` values. `SplitEnvVars` then partitions them into secrets and config based on a single inline comment convention.

## EnvVar

```go
type EnvVar struct {
    Key    string
    Value  string
    IsCfg  bool  // true when line carries "# ork:cfg"
}
```

## The `# ork:cfg` convention

The only thing a developer needs to learn is one comment tag:

```bash
# .env

DATABASE_URL=postgres://user:pass@host/db   # → Secret (default)
JWT_SECRET=abc123xyz                         # → Secret (default)
STRIPE_KEY=sk_live_...                       # → Secret (default)

PORT=8080        # ork:cfg → ConfigMap
LOG_LEVEL=info   # ork:cfg → ConfigMap
ENVIRONMENT=production  # ork:cfg → ConfigMap
```

Without `# ork:cfg` the variable goes into a Kubernetes Secret (base64-encoded). With `# ork:cfg` it goes into a Kubernetes ConfigMap (plaintext). The choice is not about sensitivity alone — it is about whether the value should be readable in plain text from the cluster.

## Parsing rules

- Blank lines and lines that start with `#` (standalone comments) are skipped.
- Inline comments are stripped before the key=value is parsed. The comment is checked for `ork:cfg` before stripping.
- Surrounding quotes (`"..."` or `'...'`) are removed from values.
- Lines without `=` are skipped.
- The constant `configMapMarker = "ork:cfg"` is the single lookup string — changing it here changes the convention everywhere.

## SplitEnvVars

```go
secrets, config := doktor.SplitEnvVars(vars)
// secrets — IsCfg == false
// config  — IsCfg == true
```

`Detect()` calls `SplitEnvVars` automatically and stores the results in `info.Secrets` and `info.Config`. Most callers never need to call `SplitEnvVars` directly.

## Usage

```go
vars, err := doktor.ParseEnvFile(".env")
if err != nil {
    return err
}

secrets, config := doktor.SplitEnvVars(vars)
fmt.Println(len(secrets), "secrets")
fmt.Println(len(config),  "config vars")
```

## What happens to the values

`ParseEnvFile` returns values as plain strings. They are never written to disk or sent anywhere by the parser itself.

`GenerateBundle` (see [04-bundle.md](04-bundle.md)) is the only place values are serialised to disk — into a Secret (base64-encoded) and a ConfigMap (quoted string). The bundle directory is excluded from git via `.gitignore` — `.orkestra/bundle/` is added by `ork doktor init`.

→ Next: [03-generation.md](03-generation.md)
