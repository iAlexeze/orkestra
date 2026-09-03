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
| `--cluster <ctx>` | _(none)_ | Reuse an existing cluster context for the E2E gate. Skips cluster creation. |
| `--workers <n>` | `0` | Number of kind worker nodes to provision for the E2E gate cluster (0 = control-plane only). |
| `--add-intent <file>` | _(none)_ | Run `ork serve play` against this intent file and bake the result as an attestation in the artifact. |
| `--sign` | `false` | Sign the artifact with Cosign keyless after push. Same as running `ork pattern sign` immediately after push. |
| `--sign-local` | `false` | Push to ttl.sh and sign — for local signing tests. Skips the normal registry. |
| `--ttl <duration>` | `1h` | TTL for the ttl.sh artifact when using `--sign-local` (e.g. `1h`, `24h`). |

> **Note:** `--use-current` and `--cluster` skip cluster provisioning and accept whatever state the cluster is in. Use them only for local iteration — `ork push` is intended for production publishing against a clean cluster.

---

## Gate sequence

```text
Validate → Simulate → E2E → Intent play → Push → Sign
```

1. **Validate** — schema, types, required fields, no local file imports. Always runs.
2. **Simulate** — runs if `simulate.yaml` is present (unless `--no-simulate` or `--force`).
3. **E2E** — runs if `e2e.yaml` is present (unless `--no-e2e` or `--force`).
4. **Intent play** — runs if `--add-intent` is passed. Bakes the result as an attestation.
5. **Push** — OCI artifact published with quality annotations baked in.
6. **Sign** — if `--sign` is passed, calls `ork pattern sign` on the pushed digest.

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

## Examples (signing)

```bash
# Push and sign in one step (CI — push and sign credentials in same job)
ork push postgres:v14 ./ --sign

# Push then sign separately (different OIDC context for signing)
ork push postgres:v14 ./
ork pattern sign postgres:v14

# Local test: push to ttl.sh and sign
ork push postgres:v14 ./ --sign-local --ttl 24h
```

Output with `--sign-local`:

```text
✓ Pushed  oci://ttl.sh/ork-a3f9c2/postgres:24h  (expires in 24h)
✓ Signed

  Expires in   24h
  Verify:      ork pattern verify oci://ttl.sh/ork-a3f9c2/postgres:24h --no-tlog
  Inspect:     ork inspect oci://ttl.sh/ork-a3f9c2/postgres:24h
```

---

## Related

- [`ork pattern sign`](./12-pattern.md) — sign a pushed artifact
- [`ork inspect`](./11-inspect.md) — read quality annotations and signature status after push
- [`ork patterns`](./12-patterns.md) — browse the registry
- [Gate Mechanics](../../guides/registry/05-gate-mechanics.md) — full gate story
- [Artifact Signing](../../security/10-artifact-signing.md) — how keyless signing works
