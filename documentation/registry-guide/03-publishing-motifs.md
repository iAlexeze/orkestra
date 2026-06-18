# Publishing Motifs

A Motif is published separately from the Katalogs that import it. Push it once; every Katalog that references it by OCI ref gets the same immutable artifact.

---

## The motif directory

```text
01-motifs/web-service/
  motif.yaml      ← required
  README.md       ← optional — rendered in ork patterns and ork inspect
```

`motif.yaml` is the only required file. A README makes the pattern discoverable and tells consumers which inputs are available and what the motif produces.

---

## Writing a motif

A Motif declares named inputs and the resources it contributes. Everything a CRD needs to deploy a stateless web service — Deployment, Service, optional Ingress — in one file:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Motif
metadata:
  name: web-service
  version: v1.0.0
  description: Deployment + ClusterIP Service + optional Ingress
  author: myorg
  license: Apache-2.0
  tags: [web, deployment, stateless]

inputs:
  - name: image
    required: true
  - name: port
    default: "9999"
  - name: replicas
    default: "1"
  - name: host
    default: ""          # Ingress only created when non-empty

resources:
  deployments:
    - name: "{{ .inputs.name }}"
      image: "{{ .inputs.image }}"
      port: "{{ .inputs.port }}"
      replicas: "{{ .inputs.replicas }}"
      reconcile: true

  services:
    - name: "{{ .inputs.name }}-svc"
      port: "80"
      targetPort: "{{ .inputs.port }}"
      reconcile: true

  ingresses:
    - name: "{{ .inputs.name }}-ingress"
      host: "{{ .inputs.host }}"
      when:
        - field: "{{ .inputs.host }}"
          exists: true
      reconcile: true
```

**Design rules:**
- All inputs are strings. Defaults make every input optional at import time.
- `required: true` inputs without a `default` are a startup error if the Katalog omits them from `with:` — caught before any CR is applied.
- Resources with `reconcile: true` are re-applied on every reconcile cycle. Without it, they are created once and not corrected if something external changes them.

---

## Validate before pushing

```bash
ork validate -f motif.yaml
```

Catches schema errors, missing required fields, and template syntax issues. Motifs have no simulate or E2E gate — validation is the only automated check before publish.

The behavioral proof lives in the Katalog that imports the Motif. When that Katalog's simulate gate passes, it proves the Motif produces the right resources for that CRD's inputs. See [Gate Mechanics](05-gate-mechanics.md).

---

## Push

```bash
export ORK_MOTIFS_REGISTRY=ghcr.io/myorg/motifs

ork push ./web-service/
# ✓ Pushed: oci://ghcr.io/myorg/motifs/web-service:v1.0.0
#   Digest: sha256:...
```

The version comes from `metadata.version` in `motif.yaml`. To override without editing the file:

```bash
ork push web-service:v1.1.0 ./web-service/
```

If the tag and `metadata.version` differ, `ork push` errors unless you pass `--force` or `--update-meta`:

```bash
# Persist the new version back to motif.yaml
ork push web-service:v1.1.0 ./web-service/ --update-meta
```

---

## Both versions coexist

```bash
ork patterns --motifs
# NAME         LATEST  KIND   TAGS
# web-service  v1.1.0  Motif  web, deployment, stateless
# web-service  v1.0.0  Motif  web, deployment, stateless
```

Old and new versions are in the registry simultaneously. Katalogs that import `web-service:v1.0.0` are not affected by the new version. Upgrading is a deliberate act in each consumer, on its own schedule. See [Upgrading Patterns](07-upgrade.md).

---

## After pushing — update the Katalog import

During local development, Katalogs reference the Motif by local file path:

```yaml
imports:
  - motif: ../01-motifs/web-service/motif.yaml
```

After pushing the Motif, update the import to the OCI ref before pushing the Katalog:

```yaml
imports:
  - motif: oci://ghcr.io/myorg/motifs/web-service:v1.0.0
```

`ork push` on a Katalog blocks if it finds local file imports and tells you exactly which fields to update. This guard exists because local paths cannot be resolved by consumers after the artifact is published.

---

## Try it

```bash
ork init --pack registry-guide
cd 01-motifs/web-service

export ORK_MOTIFS_REGISTRY=ghcr.io/myorg/motifs
ork validate -f motif.yaml
ork push ./

ork patterns --motifs
ork inspect web-service:v1.0.0 --motif
```

→ Next: [Publishing Katalogs](04-publishing-katalogs.md)
