# ork plan

Compare a local Katalog against the currently deployed one in the cluster. Shows what would change — added CRDs, removed CRDs, and field changes (workers, resync, crdFile, resource counts).

```bash
ork plan -f katalog.yaml
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | | Path to the local katalog.yaml |
| `--cm` | | `orkestra-katalog` | Name of the ConfigMap holding the deployed Katalog |
| `--namespace` | `-n` | `orkestra-system` | Namespace of the deployed Katalog ConfigMap |

## Examples

```bash
# Compare against the deployed Katalog in the default location
ork plan -f katalog.yaml

# Use a custom ConfigMap name and namespace
ork plan -f katalog.yaml --cm my-katalog --namespace my-ns
```

## Output

If no ConfigMap is found, `ork plan` prints a summary of what would be applied fresh.

If the ConfigMap exists, the diff shows:

```
Changes to apply:

  + CRD 'website'  (new)
  ~ CRD 'database'  (removed)

  CRD 'cache':
    ~ workers:  1 → 3
    ~ resync:  30s → 1m0s
```

`ork plan` exits `0` when there are no changes and there is no error.
