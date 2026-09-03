# 29 — Docker Notes

## In Development

Parse and construct Docker image references, and transform the `.docker.*` context injected by the `docker:` block. These notes make it easy to build commit-tagged images, pin to a digest after push, and inspect reference components.

## Docker context

When `docker:` runs, it injects these fields into the template context:

| Field | Value |
|-------|-------|
| `.docker.image` | fully qualified image reference (`registry/repo:tag`) |
| `.docker.digest` | image digest after push (`sha256:abc...`) |
| `.docker.buildSucceeded` | `"true"` / `"false"` |
| `.docker.error` | non-empty on failure (when `continueOnError: true`) |
| `.docker.called` | `"true"` when the block ran |

## Reference

### `dockerRegistry`

Extract the registry hostname from an image reference. Returns `""` for images that implicitly use Docker Hub.

Keywords: docker, image, registry, hostname, parse, string

```yaml
# "ghcr.io/myorg/app:v1" → "ghcr.io"
# "myorg/app:v1"          → "" (Docker Hub)
metadata:
  annotations:
    myorg.io/registry: "{{ dockerRegistry .spec.image }}"
```

---

### `dockerRepo`

Extract the repository path from an image reference, without the registry hostname or tag.

Keywords: docker, image, repo, path, parse, string

```yaml
# "ghcr.io/myorg/app:v1" → "myorg/app"
# "myorg/app:v1"          → "myorg/app"
value: "{{ dockerRepo .spec.image }}"
```

---

### `dockerTag`

Extract the tag from an image reference. Returns `"latest"` when no tag is present.

Keywords: docker, image, tag, parse, string, version

```yaml
# "ghcr.io/myorg/app:v1.2.3" → "v1.2.3"
# "ghcr.io/myorg/app"         → "latest"
status:
  fields:
    - path: imageTag
      value: "{{ dockerTag .spec.image }}"
```

---

### `dockerNoTag`

Return the image reference without the tag or digest. Use to construct a new reference with a different tag.

Keywords: docker, image, tag, strip, remove, string

```yaml
# "ghcr.io/myorg/app:v1" → "ghcr.io/myorg/app"
image: "{{ dockerNoTag .spec.image }}:{{ gitShortCommit .git.commit }}"
```

---

### `dockerName`

Extract just the image name — the last path segment of the repository, without registry or tag.

Keywords: docker, image, name, parse, string

```yaml
# "ghcr.io/myorg/app:v1" → "app"
name: "{{ dockerName .spec.image }}-deployment"
```

---

### `dockerWithTag`

Return the image reference with the tag replaced or added. The canonical way to build a tagged reference from a base image.

Keywords: docker, image, tag, replace, add, string, construct

```yaml
# Tag with the git short commit
image: "{{ dockerWithTag .spec.image (gitShortCommit .git.commit) }}"
# "ghcr.io/myorg/app" + "a3f5c2b" → "ghcr.io/myorg/app:a3f5c2b"

# Tag with the CR version field
image: "{{ dockerWithTag .spec.image .spec.version }}"
```

---

### `dockerWithDigest`

Append a digest to an image reference for an immutable pin. Use after a successful docker push to lock the deployed image to a specific layer digest. Returns the image unchanged when the digest is empty.

Keywords: docker, image, digest, immutable, pin, sha256, string

```yaml
# Pin the deployment to the pushed digest
image: "{{ dockerWithDigest .docker.image .docker.digest }}"
# "ghcr.io/myorg/app:v1" + "sha256:abc..." → "ghcr.io/myorg/app:v1@sha256:abc..."

# Gate on digest presence before using it
when:
  - field: "{{ dockerHasDigest .docker.digest }}"
    equals: "true"
```

---

### `dockerCommitTag`

Build a commit-tagged image reference from registry, repository path, and commit SHA. The canonical way to produce a mutable-but-traceable image tag in one step.

Keywords: docker, image, commit, tag, construct, string, git

```yaml
# registry + repo + commit → "registry.io/myorg/app:a3f5c2b"
image: "{{ dockerCommitTag .spec.registry .spec.repo .git.commit }}"

# Equivalent using other notes:
image: "{{ dockerWithTag (dockerNoTag .spec.image) (gitShortCommit .git.commit) }}"
```

---

### `dockerBuildSucceeded`

Normalize `.docker.buildSucceeded` to `"true"` or `"false"`. Use in `when:` conditions to gate deployment on a successful build.

Keywords: docker, build, succeeded, boolean, condition, when, string

```yaml
# Only deploy when the build succeeded
when:
  - field: "{{ dockerBuildSucceeded .docker.buildSucceeded }}"
    equals: "true"
```

---

### `dockerHasDigest`

Return `true` when a digest is present and starts with `sha256:`. Use to gate deployment on a verified, pushed image.

Keywords: docker, digest, present, boolean, sha256, check

```yaml
when:
  - field: "{{ dockerHasDigest .docker.digest }}"
    equals: "true"
```

---

## Quick reference

| Note | Accepts | Returns | Notes |
|------|---------|---------|-------|
| `dockerRegistry` | `image string` | `string` | `""` for Docker Hub |
| `dockerRepo` | `image string` | `string` | no registry, no tag |
| `dockerTag` | `image string` | `string` | `"latest"` when absent |
| `dockerNoTag` | `image string` | `string` | strips tag and digest |
| `dockerName` | `image string` | `string` | last path segment |
| `dockerWithTag` | `image, tag string` | `string` | replaces or adds tag |
| `dockerWithDigest` | `image, digest string` | `string` | unchanged when digest is `""` |
| `dockerCommitTag` | `registry, repo, commit string` | `string` | short commit SHA as tag |
| `dockerBuildSucceeded` | `s string` | `string` | normalized `"true"` / `"false"` |
| `dockerHasDigest` | `digest string` | `bool` | requires `sha256:` prefix |
