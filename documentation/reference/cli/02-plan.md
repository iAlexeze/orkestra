# ork plan

Show what would change if you applied your local Katalog — without applying anything. `ork plan` computes the diff between your local definition and a deployed reference, and prints it. Nothing is created, updated, or deleted.

A source for the deployed state must be provided explicitly — `ork plan` never silently reaches out to the cluster.

---

## Two sources for the deployed state

### Local bundle file (`--bundle`) — no cluster needed

Reads the deployed Katalog from a local bundle YAML. A bundle is the output of `ork generate bundle` — a multi-document YAML that includes the Katalog ConfigMap alongside CRDs and other manifests. No cluster connection required.

```bash
ork plan --bundle bundle.yaml
ork plan -f katalog.yaml --bundle bundle.yaml
```

This is the recommended form for CI: store the bundle artifact from the last deploy, then run `ork plan --bundle` without cluster credentials.

### Cluster ConfigMap (`--cm`) — requires cluster access

Reads the deployed Katalog from a ConfigMap in the cluster.

```bash
ork plan --cm orkestra-katalog
ork plan -f katalog.yaml --cm orkestra-katalog --namespace orkestra-system
```

`--namespace` defaults to `orkestra-system` when `--cm` is used.

---

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | `katalog.yaml` | Path to the local Katalog to compare |
| `--bundle` | `-b` | | Path to a bundle YAML — reads deployed state from it (no cluster needed) |
| `--cm` | | | ConfigMap name holding the deployed Katalog — requires cluster access |
| `--namespace` | `-n` | `orkestra-system` | Namespace of the deployed Katalog ConfigMap |

If neither `--bundle` nor `--cm` is given, `ork plan` exits with an error and prints usage guidance.

---

## Output

```text
Changes to apply:

  + CRD 'website'  (new)
  ~ CRD 'database'  (modified)
  - CRD 'legacy'  (removed)

  CRD 'cache':
    ~ workers:  1 → 3
    ~ resync:  30s → 1m0s
```

`+` new, `~` modified, `-` removed. If no ConfigMap is found (first deploy), `ork plan` prints what would be applied fresh.

`ork plan` exits `0` when there are no changes and no error.

---

## Cluster access

`ork plan --bundle` is fully offline — no cluster credentials needed. `ork plan --cm` reads a ConfigMap from the cluster but never writes anything.

`ork run` and `ork gate` are the only commands that actively manage cluster state. `ork plan --cm` reads only.

---

## Examples

```bash
# Offline — compare against a bundle from the last deploy (recommended for CI)
ork plan --bundle bundle.yaml

# Compare against a Katalog in a live cluster
ork plan --cm orkestra-katalog

# Compare against a Katalog in a non-default namespace
ork plan --cm team-a-katalog --namespace platform-system

# Use a non-standard local Katalog file
ork plan -f my-katalog.yaml --bundle bundle.yaml
```
