# Normalize Reference

## Declaration syntax

```yaml
normalize:
  spec:
    # Simple transformation
    environment: "{{ toLower .spec.environment }}"

    # Conditional
    image: >
      {{ if contains .spec.image ":" }}{{ .spec.image }}{{ else }}{{ .spec.image }}:latest{{ end }}

    # Multi-format input
    schedule: "{{ cronFromAny .spec.schedule }}"

    # Default for a nested field — resourceCPU navigates spec.resources.requests.cpu safely.
    # Direct path (.spec.resources.requests.cpu) panics when spec.resources is absent.
    resources.requests.cpu: '{{ resourceCPU . | default "100m" }}'

    # Type coercion
    suspend: "{{ toBool .spec.suspend }}"

    # Composite field
    internalName: '{{ toLower (replace (printf "%s-%s" .spec.tenant .spec.env) " " "-") }}'
```

---

## Notes commonly used in normalize

| Note | Use |
|---|---|
| `default FALLBACK VALUE` | Inline fallback when field is absent — replaces webhook-based defaults |
| `cronFromAny` | Cron string or map → canonical cron string |
| `toLower` / `toUpper` | Case normalization |
| `trimSpace` / `trim` / `trimPrefix` / `trimSuffix` | String cleanup |
| `contains` / `hasPrefix` / `hasSuffix` | Conditional format detection |
| `replace` | Character substitution |
| `toBool` / `toInt` / `toString` | Type coercion |
| `typeString` / `typeMap` / `typeList` | Type branching |
| `printf` | Field composition |

---

## Limitations

**Normalize templates cannot read each other.** Every template sees the raw CR — the object before any normalization has run. If `spec.image` and `spec.tag` are each normalized separately, a third field cannot reference the already-normalized `spec.image`. Combine them in a single expression instead:

```yaml
normalize:
  spec:
    # Wrong — spec.image here is still raw, not yet normalized
    fullImage: '{{ .spec.image }}:{{ .spec.tag }}'   # may have "nginx:latest:v2"

    # Right — assemble in one expression
    fullImage: >
      {{ $img := trimSuffix .spec.image ":latest" }}
      {{ printf "%s:%s" $img .spec.tag }}
```

**Normalize does not write to etcd.** `kubectl get -o yaml` returns what the user wrote, not the normalized form. This is intentional — normalize is an operator concern, not a storage concern. If external tools need to read the canonical value, use `mutation:` rules with the Gateway, or set schema defaults in the CRD itself.

---

<!-- **Back →** [Normalize](index.md) -->
