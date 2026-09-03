# ork run

Start the Orkestra Runtime.

```bash
ork run [<name>:<version>]
ork run --file <path>
```

Merges all provided katalog files, validates them, then starts the operator runtime. Without a positional ref or `-f` flag, looks for `katalog.yaml` or `komposer.yaml` in the current directory.

---

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--file`, `-f` | | Path(s) to katalog.yaml (repeatable). Takes precedence over the positional ref. |
| `--dev` | `false` | Create a local Kind cluster if none is reachable. |
| `--apply-cr` | `false` | Apply `crd.yaml` and `cr.yaml` from the pattern directory before starting the runtime. Only applies to the example resources shipped with the pattern — not to your own CRDs. |
| `--use-komposer` | `false` | When running from a positional OCI ref, use `komposer.yaml` from the pulled pattern instead of `katalog.yaml`. |
| `--refresh` | `false` | Re-pull the pattern from the registry even if already cached. |
| `--dev-server` | `false` | Start the mock dev server for `external:` examples. |
| `--dev-server-port` | `8090` | Port for the mock dev server. |

---

## Running a pattern from the registry

```bash
ork run postgres:v1.0.0
```

If the pattern is not in the local cache, it is pulled first. The short name `postgres:v1.0.0` resolves against `ORK_REGISTRY` (default: `ghcr.io/orkspace/orkestra-registry/patterns/katalogs`). A full OCI ref (`oci://ghcr.io/...`) is used as-is.

```bash
# Pull + start on a local cluster — bring your own CR
ork run postgres:v1.0.0 --dev

# Pull + apply example CR + start runtime — full batteries-included
ork run postgres:v1.0.0 --dev --apply-cr

# Run via the pattern's komposer instead of its katalog
ork run postgres:v1.0.0 --dev --use-komposer

# Force re-pull before running
ork run postgres:v1.0.0 --dev --refresh
```

`--apply-cr` applies `crd.yaml` then waits for the CRD to be established, then applies `cr.yaml`. The runtime starts after this sequence. The operator will have a CR waiting to reconcile on its first cycle.

---

## Running a local katalog

```bash
# Current directory (katalog.yaml or komposer.yaml auto-detected)
ork run

# Explicit path
ork run -f ./my-operator/katalog.yaml

# Multiple katalogs merged
ork run -f ./service-a/katalog.yaml -f ./service-b/katalog.yaml
```

---

## Registry resolution

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
```

Set `ORK_REGISTRY` to pull from a private registry. Unset, the default Orkestra registry is used. Authentication reads `~/.docker/config.json` — run `docker login <registry>` first.

---

## Endpoints

Once running, the runtime exposes:

```text
/health
/ready
/metrics
/katalog
/katalog/{crd}
/katalog/{crd}/cr
/katalog/{crd}/cr/<ns>/<name>
/katalog/{crd}/health
```

---

## Related

- [`ork pull`](./10-pull.md) — pre-warm the cache without running
- [`ork inspect`](./11-inspect.md) — browse quality signals before pulling
- [Running Patterns](../../getting-started/07-running-patterns.md) — full workflow walkthrough
