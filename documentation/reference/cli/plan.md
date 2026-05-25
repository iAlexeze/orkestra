# ork plan

Show what would change if you applied your local Katalog — without applying anything. `ork plan` computes the diff between your local definition and the currently deployed one, and prints it. Nothing is created, updated, or deleted.

```bash
ork plan
```

`--file` defaults to `katalog.yaml` — only required for non-standard filenames.

---

## Three sources for the deployed state

`ork plan` compares your local Katalog against one of three sources:

### Live cluster (default)

Reads the deployed Katalog from the `orkestra-katalog` ConfigMap in `orkestra-system`. Requires a cluster connection.

```bash
ork plan
ork plan -f katalog.yaml
ork plan -f katalog.yaml --cm my-katalog --namespace my-ns
```

### Local bundle file (`--bundle`)

Reads the ConfigMap from a local bundle YAML instead of the cluster. No cluster connection required. A bundle is the output of `ork generate bundle` — a multi-document YAML file that includes the Katalog ConfigMap alongside CRDs and other manifests.

```bash
ork plan --bundle bundle.yaml
ork plan -f katalog.yaml --bundle bundle.yaml
```

This is the offline form: both the local Katalog and the deployed reference come from local files. CI pipelines can run `ork plan` without cluster credentials by storing a bundle artifact from the last deploy.

---

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | `katalog.yaml` | Path to the local Katalog to compare |
| `--bundle` | `-b` | | Path to a bundle YAML — reads deployed state from it instead of the cluster |
| `--cm` | | `orkestra-katalog` | ConfigMap name holding the deployed Katalog |
| `--namespace` | `-n` | `orkestra-system` | Namespace of the deployed Katalog ConfigMap |

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

`ork plan` without `--bundle` reads from the cluster — it connects to Kubernetes to fetch the ConfigMap. With `--bundle`, it is fully offline and requires no cluster credentials.

`ork run` and `ork gate` are the only commands that write to or actively manage a cluster. `ork plan` reads only.

---

## Examples

```bash
# Compare local katalog.yaml against the deployed Katalog
ork plan

# Offline — compare against a bundle from the last deploy (no cluster needed)
ork plan --bundle bundle.yaml

# Compare against a Katalog in a non-default ConfigMap
ork plan --cm team-a-katalog --namespace platform-system
```
