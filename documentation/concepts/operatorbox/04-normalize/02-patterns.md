# Normalize Patterns

---

## Schedule format unification

Users write cron schedules as a string or a structured object. Normalize collapses both into a canonical cron string once — `onCreate`, `onReconcile`, and status fields all use `.spec.schedule` with no branching.

```yaml
normalize:
  spec:
    schedule: "{{ cronFromAny .spec.schedule }}"
```

```yaml
# Both are valid — downstream never sees the difference:
spec:
  schedule: "*/5 * * * *"

spec:
  schedule:
    minute: "*/5"
    hour: "*"
    dayOfMonth: "*"
    month: "*"
    dayOfWeek: "*"
```

**Try it:**
```bash
ork init --pack use-cases
cd crd-conversion/without-webhooks
ork run
```

---

## Image tag normalization

Make the tag explicit so child resource templates never receive an ambiguous image reference:

```yaml
normalize:
  spec:
    image: >
      {{ if contains .spec.image ":" }}
        {{ .spec.image }}
      {{ else }}
        {{ .spec.image }}:latest
      {{ end }}
```

`"nginx"` → `"nginx:latest"`. Every `onCreate` and status template uses the same resolved value.

---

## Registry prefix enforcement

Enforce a private registry prefix without requiring users to always write it:

```yaml
normalize:
  spec:
    image: >
      {{ if hasPrefix .spec.image "registry.internal/" }}
        {{ .spec.image }}
      {{ else }}
        registry.internal/{{ trimPrefix .spec.image "/" }}
      {{ end }}
```

---

## Environment name canonicalization

Accept any casing — produce one canonical value downstream:

```yaml
normalize:
  spec:
    environment: "{{ toLower (trimSpace .spec.environment) }}"
```

`" Production "`, `"PRODUCTION"`, `"production"` → `"production"`. Validation rules, label templates, and routing logic all see a consistent value.

---

## Domain cleanup

Strip protocol and trailing slash from user-provided URLs:

```yaml
normalize:
  spec:
    domain: >
      {{ trimSuffix (trimPrefix (trimPrefix (trimSpace .spec.domain) "https://") "http://") "/" }}
```

`"https://my-app.example.com/"` → `"my-app.example.com"`.

---

## Composite field assembly

Build a stable internal identifier from multiple spec fields:

```yaml
normalize:
  spec:
    internalName: '{{ toLower (replace (printf "%s-%s" .spec.tenant .spec.environment) " " "-") }}'
```

`tenant: "Acme Corp"`, `environment: "Production"` → `internalName: "acme-corp-production"`. Every child resource template references `.spec.internalName` — the assembly logic lives in one place.

---

## Nested path normalization

Dot-notation paths reach deep fields without touching siblings:

```yaml
normalize:
  spec:
    resources.requests.cpu:    '{{ default "100m" .spec.resources.requests.cpu }}'
    resources.requests.memory: '{{ default "128Mi" .spec.resources.requests.memory }}'
```

Only the declared paths are overwritten. Other fields under `spec.resources` are untouched.

---

## Boolean coercion

Normalize string booleans so `boolTernary` and typed comparisons always receive a real `bool`:

```yaml
normalize:
  spec:
    suspend:    "{{ toBool .spec.suspend }}"
    monitoring: "{{ toBool .spec.monitoring }}"
```

`"yes"`, `"1"`, `"true"`, `true` all produce `true`.

---

## Scalar or list unification

Accept either a single string or a list — produce a list for `forEach:` loops:

```yaml
normalize:
  spec:
    regions: >
      {{ if typeString .spec.regions }}
        {{ list .spec.regions | toJson }}
      {{ else }}
        {{ .spec.regions | toJson }}
      {{ end }}
```

After normalize, `.spec.regions` is always a list. `forEach:` in `onReconcile` requires no type check.

---

**Next →** [Reference](03-reference.md)
