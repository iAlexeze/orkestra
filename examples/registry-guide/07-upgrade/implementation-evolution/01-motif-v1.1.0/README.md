# 07-01 — web-service v1.1.0

`web-service:v1.1.0` adds two optional inputs that make probes configurable:

| Input | Default | What it does |
|-------|---------|--------------|
| `probeProfile` | `standard` | Probe profile for readiness and liveness checks |
| `probePath` | `/health` | HTTP path hit by both probes |

Both default to values that preserve v1.0.0 behavior. No existing CR needs to change.

## Publish

```bash
export ORK_MOTIFS_REGISTRY=ghcr.io/myorg/motifs

ork validate -f motif.yaml

ork push .
# ✓ Pushed: oci://ghcr.io/myorg/motifs/web-service:v1.1.0
```

Motifs have no simulate or e2e gate — there is nothing to gate until the motif is imported into a katalog. Validate confirms the motif is structurally correct. The behavioral proof is produced in `02-api-team-upgrades/`, where the upgraded katalog runs simulate + e2e with the new version.

```bash
ork patterns --motifs
# NAME          LATEST   KIND   E2E
# web-service   v1.1.0   Motif  -
```

`ork patterns` shows only the latest per pattern. To see the full version history:

```bash
ork inspect web-service --motif --versions
# web-service  (2 versions)
#
#   v1.1.0   ✓ Simulate  -  E2E   ← latest
#   v1.0.0   ✓ Simulate  -  E2E
```

v1.0.0 is still in the registry. Any Katalog pinned to `web-service:v1.0.0` continues to resolve it unchanged. The upgrade is opt-in.

→ Next: [02-api-team-upgrades](../02-api-team-upgrades/README.md)
