# 27 — Data Notes

Data encoding, hashing, and name-safety utilities. Used for Secret data encoding, content-addressed naming, JSON embedding, and generating Kubernetes-safe resource name segments from arbitrary string inputs.

## Reference

### `toBase64`

Base64-encode a string using standard encoding. The standard format for Kubernetes `Secret.data` values.

Keywords: data, base64, encode, secret, string, kubernetes

```yaml
onCreate:
  secrets:
    - name: "{{ .metadata.name }}-config"
      data:
        endpoint: "{{ toBase64 .spec.endpoint }}"
        token: "{{ toBase64 .spec.token }}"
```

---

### `fromBase64`

Decode a base64 string. Use to read Secret data values back into status fields or cross-CRD templates where the raw value is needed.

Keywords: data, base64, decode, secret, string, read

```yaml
status:
  fields:
    - path: endpoint
      value: "{{ fromBase64 (index .children.secret.data \"endpoint\") }}"
```

---

### `toJSON`

Marshal any value to a JSON string. Useful for embedding structured data in annotations, ConfigMap values, or status fields.

Keywords: data, json, marshal, encode, serialize, annotation, configmap, string

```yaml
onCreate:
  configMaps:
    - name: "{{ .metadata.name }}-config"
      data:
        settings: "{{ toJSON .spec.settings }}"
        labels: "{{ toJSON .metadata.labels }}"
```

---

### `sha256sum`

Return the first 8 hex characters of the SHA256 hash of a string. The canonical tool for content-addressed naming and change detection in Orkestra — produces a stable, short, collision-resistant fingerprint.

Keywords: data, hash, sha256, checksum, fingerprint, content, addressed, string

```yaml
# Content-addressed ConfigMap name — changes when content changes
onCreate:
  configMaps:
    - name: "config-{{ sha256sum .spec.config }}"
      data:
        content: "{{ .spec.config }}"

# Change-detection annotation
metadata:
  annotations:
    myorg.io/config-hash: "{{ sha256sum .spec.config }}"
```

---

### `truncateName`

Hard-truncate a string to at most `maxLen` characters with no suffix. For generating valid Kubernetes resource names where the total length must stay under the 63-character DNS label limit. Unlike `truncate` (strings domain), this note adds no `...` — illegal characters in Kubernetes names.

Keywords: data, truncate, name, kubernetes, length, limit, string, dns

```yaml
# Keep the name under 50 chars before appending a fixed suffix
name: "{{ truncateName .spec.projectName 50 }}-deployment"

# Combine with slugify for safe, length-bounded names
name: "{{ truncateName (slugify .spec.teamName) 40 }}-operator"
```

---

### `slugify`

Convert a string to a Kubernetes-safe name segment: lowercase, non-alphanumeric characters replaced with dashes, consecutive dashes collapsed, leading and trailing dashes trimmed.

Keywords: data, slug, slugify, name, safe, kubernetes, string, lowercase

```yaml
# Derive a safe resource name from a human-readable field
name: "{{ slugify .spec.teamName }}-operator"
# "My Team / Platform" → "my-team-platform-operator"

# Use with truncateName for long values
name: "{{ truncateName (slugify .spec.projectName) 52 }}-svc"
```

---

## Quick reference

| Note | Accepts | Returns | Notes |
|------|---------|---------|-------|
| `toBase64` | `s string` | `string` | standard encoding |
| `fromBase64` | `s string` | `string` | `""` on decode error |
| `toJSON` | `v any` | `string` | `"{}"` on marshal error |
| `sha256sum` | `s string` | `string` | first 8 hex chars of SHA256 |
| `truncateName` | `s string, maxLen int` | `string` | no suffix — use for Kubernetes names |
| `slugify` | `s string` | `string` | lowercase, dashes, trimmed |
