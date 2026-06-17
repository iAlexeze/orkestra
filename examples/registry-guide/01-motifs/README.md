# 01 — Motifs

A Motif is the smallest reusable primitive in Orkestra. It declares named inputs and resource templates. It cannot run alone — it must be imported by a Katalog that wires its inputs to CR spec fields via a `with:` block.

This directory contains three motifs that the later examples build on.

---

## Motifs in this directory

| Motif | What it provides |
|-------|-----------------|
| [web-service/motif.yaml](./web-service/motif.yaml) | Deployment + Service + optional Ingress |
| [data-store/motif.yaml](./data-store/motif.yaml) | StatefulSet + PVC + Services + auto-rotating credentials |
| [platform-admission/motif.yaml](./platform-admission/motif.yaml) | Image registry policy + replica bounds + team label advisory |

---

## Validate

```bash
ork validate -f web-service/motif.yaml
ork validate -f data-store/motif.yaml
ork validate -f platform-admission/motif.yaml
```

Validation is offline — no cluster, no registry pull required.

---

## Push to the registry

Authenticate to your registry _(any OCI-compatible registry)_ and set the registry URLs. Motifs and Katalogs use separate registries so each can be versioned and scaled independently:

```bash
# GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
export ORK_MOTIFS_REGISTRY=ghcr.io/myorg/motifs
export ORK_REGISTRY=ghcr.io/myorg/katalogs
```

```bash
# Docker Hub — multi-level paths are not supported; use docker.io/<username> directly
echo $DOCKER_TOKEN | docker login docker.io -u YOUR_USERNAME --password-stdin
export ORK_MOTIFS_REGISTRY=docker.io/myusername
export ORK_REGISTRY=docker.io/myusername
```

The pattern name becomes the image name in Docker Hub (e.g. `docker.io/myusername/web-service:v1.0.0`). With ghcr.io you get the full path (`ghcr.io/myorg/motifs/web-service:v1.0.0`), which keeps motifs and katalogs in separate namespaces.

The motif files use `author: myorg` as a placeholder. Optionally update it to your GitHub username or organisation name before pushing — `ork inspect` and `ork patterns` surface the author field to consumers.

```bash
# Optional — replace myorg with your name in each motif file
sed -i 's/author: myorg/author: yourname/' web-service/motif.yaml data-store/motif.yaml platform-admission/motif.yaml
```

If the tag (`2.0.0`) you pass doesn't match the version declared in `motif.yaml` (`1.0.0`), the push fails:

```bash
ork push web-service:2.0.0 ./web-service/
# motif.yaml: metadata.version "1.0.0" does not match provided tag "2.0.0"; use '--force' to override
# Exit 1 — nothing was pushed
```

The tag and the metadata version must agree. Fix `metadata.version` in the motif file, or use the matching tag. The version can be any string — `v1.0.0`, `1.0.0`, `2024-06`, `stable` — as long as the tag and the file match.

If you need to override (not recommended for published patterns):

```bash
ork push web-service:2.0.0 ./web-service/ --force
# Warning: motif.yaml: metadata.version "1.0.0" does not match provided tag "2.0.0" (continuing due to --force)
```

Push each motif at its declared version. If `metadata.version` is set in `motif.yaml`, the tag is optional — `ork` reads it from the file:

```bash
ork push ./web-service/
ork push ./data-store/
ork push ./platform-admission/
```

Confirm they are visible:

```bash
ork patterns
```

---

## How motifs are imported

During development, reference the motif by local path:

```yaml
spec:
  crds:
    webapp:
      imports:
        - motif: ./web-service/motif.yaml
          with:
            image: "{{ .spec.image }}"
            port: "{{ .spec.port }}"
            host: "{{ .spec.host }}"
```

After publish, other teams can import the motif by OCI reference:

```yaml
spec:
  crds:
    webapp:
      imports:
        - motif: ghcr.io/myorg/registry/motifs/web-service:v1.0.0
          with:
            image: "{{ .spec.image }}"
            port: "{{ .spec.port }}"
            host: "{{ .spec.host }}"
```
The Katalog does not change — only the source.

---
## Tags

The `tags:` list under `metadata:` is indexed by the registry. Use `ork patterns --tag web` to find motifs by capability. Always tag before publishing — it is the primary discovery surface for other teams.

---

## Next step

→ [02-katalog-api](../02-katalog-api/README.md) — build a Katalog from the web-service motif
