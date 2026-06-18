# 07-02 — API team upgrades

The API team owns `webapp-operator` under the `api-team` namespace. They follow the motif to v1.1.0: one import line changes, two new CRD fields appear, the gate passes, and they publish `webapp-operator:v1.1.0`.

The platform team does not need to do anything. Their katalog still imports `web-service:v1.0.0` and continues to pass its own gate. The two teams are independent.

> **Before you begin:** Update `author:` in [katalog.yaml](katalog.yaml) to your org. Then get the exact OCI ref for the motif you just published in `01-motif-v1.1.0/`:
>
> ```bash
> ork inspect web-service:v1.1.0 --motif
> ```
>
> Copy the import ref from the output and update the motif import in [katalog.yaml](katalog.yaml) before proceeding.

## What changed in katalog.yaml

```yaml
metadata:
  name: webapp-operator
  namespace: api-team      # ← each team scopes their katalog
  version: v1.1.0

spec:
  crds:
    webapp:
      imports:
        - motif: oci://ghcr.io/myorg/motifs/web-service:v1.1.0   # bumped
          with:
            image: "{{ .spec.image }}"
            port: "{{ .spec.port }}"
            replicas: "{{ .spec.replicas }}"
            probeProfile: "{{ .spec.probeProfile }}"   # new
            probePath: "{{ .spec.probePath }}"          # new
```

The CRD also gains two optional fields in `crd.yaml` (`probeProfile`, `probePath`), both with backwards-compatible defaults. Existing CRs that omit them get the motif defaults — no migration needed.

## Validate and simulate

```bash
ork pull -f katalog.yaml

ork validate

ork template
# The expanded Deployment now includes readinessProbe and livenessProbe.

ork simulate
# ✓ 2/2 assertions passed
```

## E2E coverage

The `e2e.yaml` adds test coverage for both new features. After the Deployment is running, it checks that `probePath` and `probeProfile` were actually wired in — not just that the operator started, but that the values reached the container spec:

```bash
ork e2e
# ✓ WebApp CR created
# ✓ Deployment running
# ✓ Service exists
# ✓ Ingress absent when host empty
# ✓ probePath wired as HTTP GET path         (.readinessProbe.httpGet.path = /ready)
# ✓ standard probeProfile expands correctly  (periodSeconds=20 failureThreshold=3)
# ✓ WebApp CR deleted
# ✓ Deployment removed
```

The simulate gate proves the resources are created. The e2e gate proves the probe configuration expanded to its correct runtime values against a real Kubernetes API.

## Publish

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs

ork push ./
# ✓ Simulate: passed (2 assertions)
# ✓ E2E: passed
# ✓ Pushed: oci://ghcr.io/myorg/katalogs/webapp-operator:v1.1.0
```

```bash
ork inspect webapp-operator --versions
# webapp-operator  (2 versions)
#
#   v1.1.0   ✓ Simulate  ✓ E2E   ← latest
#   v1.0.0   ✓ Simulate  ✓ E2E
```

Both versions are in the registry. `v1.0.0` is unchanged. Any consumer pinned to it sees nothing different.

→ Next: [implementation-evolution](../README.md#compose-and-deploy) — compose both teams into one runtime
