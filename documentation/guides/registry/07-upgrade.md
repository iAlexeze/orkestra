# Upgrading Patterns

Patterns are immutable at a version. Upgrading means publishing a new version and consumers opting in — on their own schedule, one by one.

---

## Layer 1: Upgrading a Motif

A new Motif version adds inputs, changes behavior, or fixes a bug. Backwards-compatible changes use defaults so existing consumers don't need to update their CRs.

```yaml
# web-service v1.1.0 — adds probeProfile and probePath
inputs:
  - name: probeProfile
    default: "standard"    # same behavior as before — existing CRs unaffected
  - name: probePath
    default: "/health"
```

Push alongside the old version:

```bash
ork push web-service:v1.1.0 ./01-motifs/web-service/
```

```bash
ork inspect web-service:v1.1.0 --motifs
# web-service  v1.1.0  Motif   ← new
# web-service  v1.0.0  Motif   ← still there, unchanged
```

Old and new coexist. No Katalog is broken. No consumer is forced to upgrade.

---

## Layer 2: A Katalog follows the Motif

The Katalog bumps one import line — `web-service:v1.0.0` → `web-service:v1.1.0` — and exposes the new inputs in its CRD spec:

```yaml
# webapp-operator v1.1.0 — wires probeProfile and probePath through from spec
imports:
  - motif: oci://ghcr.io/myorg/motifs/web-service:v1.1.0
    with:
      probeProfile: "{{ .spec.probeProfile }}"    # new optional field
      probePath: "{{ .spec.probePath }}"          # new optional field
```

The simulate gate confirms the upgrade is backwards-compatible:

```bash
ork simulate
# ✓ 2/2 assertions passed — same resources, probe is configuration not a new resource
```

E2E confirms it works in a real cluster:

```bash
ork e2e
# ✓ PASS
```

Push:

```bash
ork push webapp-operator:v1.1.0 ./
ork inspect webapp-operator:v1.1.0
# Simulate: ✓ Verified  E2E: ✓ Verified
```

---

## Deploy the upgrade

Bump the import in the Komposer:

```yaml
imports:
  registry:
    - oci://ghcr.io/myorg/katalogs/webapp-operator:v1.1.0   # was v1.0.0
```

```bash
ork generate bundle -f komposer.yaml -o bundle-v1.1.0.yaml
kubectl apply -f bundle-v1.1.0.yaml

kubectl rollout restart deployment/orkestra-runtime -n orkestra-system
kubectl rollout status deployment/orkestra-runtime -n orkestra-system
```

Existing CRs continue reconciling — the new optional inputs default to the same behavior as before. No CR migration.

---

## Rollback

```bash
# Revert komposer.yaml to webapp-operator:v1.0.0
ork generate bundle -f komposer.yaml -o bundle-rollback.yaml
kubectl apply -f bundle-rollback.yaml
kubectl rollout restart deployment/orkestra-runtime -n orkestra-system
```

v1.1.0 is still in the registry. Untouched. Rollback doesn't delete it.

---

## Version isolation: two Katalogs, same cluster

Platform teams choose their upgrade schedule. One team may stay on `v1.0.0` while another runs `v1.1.0` — in the same Orkestra runtime, managing the same CRD.

```text
webapp-operator:v1.0.0  → web-service:v1.0.0  (platform team — pinned)
webapp-operator:v1.1.0  → web-service:v1.1.0  (your team — upgraded)
```

Each Katalog's configuration lives in its own ConfigMap slice. Bumping one does not touch the other. When the platform team is ready, they upgrade on their own timeline.

This is the distribution model: the registry stores both versions permanently. Upgrading is a deliberate act in each consumer.

---

## Rollback safety

Before upgrading in production, `ork plan` shows what would change:

```bash
ork plan -f komposer.yaml --bundle bundle-v1.1.0.yaml
# ~ deployment/my-webapp: readinessProbe added (http /health profile standard)
# No deletions. No CRD schema changes.
```

Non-destructive changes (adding probes, updating images) are safe to apply without cluster disruption. Destructive changes (removing fields, changing CRD groups) require more care — check what downstream operators watch before upgrading.

---

## Try it

```bash
ork init --pack registry-guide
cd 07-upgrade

# Follow the steps in the README
```

→ Next: [API Evolution](08-api-evolution.md)
