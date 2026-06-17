# ork push

Publish a pattern directory to the OCI registry. Runs the simulate gate (if `simulate.yaml` is present) and the E2E gate (if `e2e.yaml` is present) before uploading. Blocks if either gate fails unless `--force` is passed.

```bash
ork push [<name>:<version>] [<dir>]
```

`<dir>` defaults to current directory. `<name>:<version>` defaults to the name and version in the primary file (`motif.yaml` or `katalog.yaml`). Passing an explicit tag overrides the version from the file — the command errors if they differ unless `--force` or `--update-meta` is set.

---

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Push even if gates fail or versions differ. Gates are recorded as skipped in the artifact. |
| `--no-e2e` | `false` | Skip the E2E gate even if `e2e.yaml` is present. Simulate still runs. |
| `--no-simulate` | `false` | Skip the simulate gate even if `simulate.yaml` is present. E2E still runs. |
| `--e2e <file>` | `e2e.yaml` | Path to an alternative E2E spec file. |
| `--update-meta` | `false` | Persist the overridden version tag back into the primary file. |
| `--use-current` | `false` | Use the current kubeconfig context for the E2E gate. Skips cluster creation — significantly faster for local iteration. |
| `--cluster <ctx>` | _(none)_ | Reuse an existing kind cluster context for the E2E gate. Skips cluster creation. |

> **Note:** `--use-current` and `--cluster` skip cluster provisioning and accept whatever state the cluster is in. Use them only for local iteration — `ork push` is intended for production publishing against a clean cluster.

---

## Gate sequence

```text
Validate → Simulate → E2E → Push
```

1. **Validate** — schema, types, required fields, no local file imports. Always runs.
2. **Simulate** — runs if `simulate.yaml` is present (unless `--no-simulate` or `--force`).
3. **E2E** — runs if `e2e.yaml` is present (unless `--no-e2e` or `--force`).
4. **Push** — OCI artifact published with quality annotations baked in.

---

## Registry resolution

The target registry is resolved from the environment:

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs       # katalogs and komposers
export ORK_MOTIFS_REGISTRY=ghcr.io/myorg/motifs  # motifs
```

Authentication uses `~/.docker/config.json`. Run `docker login <registry>` before pushing.

---

## Examples

```bash
# Standard push — name and version from katalog.yaml
ork push ./

# Explicit version override
ork push webapp-operator:v1.0.0 ./

# Push as draft — both gates skipped, recorded in artifact
ork push --force ./

# Simulate passes but skip E2E (CI without Docker-in-Docker)
ork push --no-e2e ./

# Update katalog.yaml with the new version before pushing
ork push webapp-operator:v1.2.0 ./ --update-meta

# Non-standard directory
ork push webapp-operator:v1.0.0 ./patterns/webapp/
```

---

## Output

```text
Pushing webapp-operator:v1.0.0 (Katalog) to ghcr.io/myorg/katalogs...
  ✓ katalog.yaml         valid
  ✓ crd.yaml             valid
  ✓ cr.yaml              (512 B)
  ✓ simulate.yaml        (872 B)
  ✓ e2e.yaml             (1.4 KB)
  ✓ README.md            (3.1 KB)

Running simulate gate (simulate.yaml)...
  ✓ Simulate passed (2 assertions · 14ms)

Running E2E gate (e2e.yaml)...
  ✓ E2E passed (48s)

✓ Pushed: oci://ghcr.io/myorg/katalogs/webapp-operator:v1.0.0
  Digest: sha256:a3f1c8d20e4b7f9c...

To import:
  imports:
    registry:
      - oci://ghcr.io/myorg/katalogs/webapp-operator:v1.0.0
```

---

## Error: local file imports

```text
✗ Push blocked: local file imports in katalog.yaml

  spec.crds.webapp.imports[0]: "../01-motifs/web-service/motif.yaml"

  Local imports work for ork simulate and ork template, but cannot
  be resolved by consumers after the katalog is published.

  Before publishing:
    1. Push the motif:  ork push <motif-dir>/
    2. Replace the local path with the OCI ref:
       motif: oci://<your-registry>/motifs/<name>:<version>
```

---

## Error: gate failure

```text
✗ Simulate gate failed — push blocked
  Run 'ork simulate' to see the failures
  Use --force to override (recorded in the artifact)
```

```text
✗ E2E gate failed — push blocked
  Run 'ork e2e' to see the failures
  Use --force to override (recorded in the artifact)
```

---

## Related

- [`ork inspect`](./11-inspect.md) — read quality annotations after push
- [`ork patterns`](./12-patterns.md) — browse the registry
- [Gate Mechanics](../../registry-guide/05-gate-mechanics.md) — full gate story
