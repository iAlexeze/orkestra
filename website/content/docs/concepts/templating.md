---
title: "Templating in Orkestra"
weight: 4
description: "This document explains how Orkestra uses templates to turn your CR spec into actual Kubernetes resources — without writi..."
---

This document explains how Orkestra uses templates to turn your CR spec into actual Kubernetes resources — without writing code.

---

## What Are Templates?

Templates are placeholders in your Katalog that get replaced with values from your CR. Think of them like fill‑in‑the‑blanks.

For example, instead of hardcoding an image name, you can write:

```yaml
image: "{{ .spec.image }}"
```

When Orkestra reconciles your CR, it replaces `{{ .spec.image }}` with whatever value is in the CR's `spec.image` field.

---

## Why Templates Matter

Without templates, every CR would need the exact same resource configuration. Templates let you:

- Use different images for different CRs
- Scale replicas per CR
- Put resources in different namespaces
- Generate names from the CR name
- Use any field from the CR as a value

**One Katalog, many CRs.**

---

## The Template Syntax

Templates use Go's `text/template` syntax. The most common pattern is:

```
{{ .path.to.field }}
```

The `.` refers to the CR object. Everything you can access in a Kubernetes object is available.

### Available Fields

| Path | What It Contains |
|------|------------------|
| `.metadata.name` | The CR's name |
| `.metadata.namespace` | The CR's namespace |
| `.metadata.labels` | Labels on the CR |
| `.metadata.annotations` | Annotations on the CR |
| `.spec` | The entire spec section |
| `.spec.field` | Any field inside spec |
| `.status` | The entire status section |

---

## Examples

### Basic Field Reference

```yaml
deployments:
  - name: "{{ .metadata.name }}-web"
    image: "{{ .spec.image }}"
    replicas: "{{ .spec.replicas }}"
```

If your CR is:

```yaml
metadata:
  name: my-blog
spec:
  image: nginx:1.25
  replicas: 3
```

Orkestra resolves:

| Template | Becomes |
|----------|---------|
| `{{ .metadata.name }}-web` | `my-blog-web` |
| `{{ .spec.image }}` | `nginx:1.25` |
| `{{ .spec.replicas }}` | `3` |

---

### Combining Static and Dynamic Values

You can mix static text with templates:

```yaml
name: "{{ .metadata.name }}-svc"
image: "myregistry.com/{{ .spec.image }}"
namespace: "{{ .metadata.namespace }}-staging"
```

---

### Nested Fields

```yaml
labels:
  - key: app
    value: "{{ .metadata.labels.app }}"
  - key: version
    value: "{{ .spec.version.tag }}"
```

---

### Optional Fields

If a template references a field that doesn't exist, Orkestra replaces it with an empty string. This is intentional — it allows optional fields without breaking reconciliation.

```yaml
# If .spec.port doesn't exist, this becomes empty
port: "{{ .spec.port }}"
```

---

## Template Context

The full CR is available in the template context. This includes everything Kubernetes stores:

```yaml
# All of this is available
{{ .metadata.name }}
{{ .metadata.namespace }}
{{ .metadata.labels.env }}
{{ .metadata.annotations["prometheus.io/scrape"] }}
{{ .spec.replicas }}
{{ .spec.image }}
{{ .status.phase }}
{{ .status.conditions[0].type }}
```

---

## Where Templates Can Be Used

Templates can be used in any field of any resource template:

### Deployments

```yaml
deployments:
  - name: "{{ .metadata.name }}"
    image: "{{ .spec.image }}"
    replicas: "{{ .spec.replicas }}"
    port: "{{ .spec.port }}"
    namespace: "{{ .metadata.namespace }}"
```

### Services

```yaml
services:
  - name: "{{ .metadata.name }}-svc"
    type: "{{ .spec.serviceType }}"
    port: "80"
    targetPort: "{{ .spec.port }}"
```

### Secrets

```yaml
secrets:
  - name: "{{ .metadata.name }}-creds"
    data:
      USERNAME: "{{ .spec.username }}"
      PASSWORD: "{{ .spec.password }}"
```

### ConfigMaps

```yaml
configMaps:
  - name: "{{ .metadata.name }}-config"
    data:
      LOG_LEVEL: "{{ .spec.logLevel }}"
      MAX_CONNECTIONS: "{{ .spec.maxConnections }}"
```

### Jobs (onDelete)

```yaml
onDelete:
  jobs:
    - name: "cleanup-{{ .metadata.name }}"
      image: "{{ .spec.cleanupImage }}"
      command: ["/bin/cleanup", "{{ .metadata.name }}"]
```

---

## Default Values

If a template resolves to an empty string, Orkestra applies defaults:

| Resource | Field | Default |
|----------|-------|---------|
| Deployment | `name` | `<cr-name>-deployment` |
| Deployment | `replicas` | `1` |
| Service | `name` | `<cr-name>-svc` |
| Service | `type` | `ClusterIP` |
| Secret | `name` | `<cr-name>-secret` |
| ConfigMap | `name` | `<cr-name>-config` |
| Job | `name` | `<cr-name>-job` |

You only need to specify values that change per CR.

---

## Conditions (when)

Templates work with conditions to create resources only when certain conditions are met.

```yaml
services:
  - name: "{{ .metadata.name }}-public"
    type: LoadBalancer
    port: "80"
    targetPort: "{{ .spec.port }}"
    when:
      - field: spec.exposePublicly
        equals: "true"
```

The template is still resolved (the name uses `{{ .metadata.name }}`), but the resource is only created if the condition is true.

---

## Template Resolution Flow

1. **Orkestra reads the Katalog** — finds all templates
2. **A CR is created or updated** — triggers reconciliation
3. **Resolver evaluates templates** — replaces `{{ .path }}` with actual values
4. **Resources are created or updated** — using the resolved values

All of this happens automatically on every reconcile.

---

## Debugging Templates

If a template isn't resolving as expected, use debug mode:

```bash
ork run --file my-katalog.yaml --debug
```

You'll see:

```
DEBUG resolved template: image="{{ .spec.image }}" → "nginx:1.25"
DEBUG resolved template: replicas="{{ .spec.replicas }}" → "3"
```

This shows you exactly what value each template became.

---

## Common Patterns

### Resource Name from CR Name

```yaml
name: "{{ .metadata.name }}-app"
```

### Namespace from CR Namespace

```yaml
namespace: "{{ .metadata.namespace }}"
```

### Dynamic Image from Spec

```yaml
image: "{{ .spec.image }}"
```

### Dynamic Replicas from Spec

```yaml
replicas: "{{ .spec.replicas }}"
```

### Conditional Resource with Label

```yaml
services:
  - name: "{{ .metadata.name }}-public"
    when:
      - field: spec.environment
        equals: "production"
```

---

## What Templates Cannot Do

Templates are for values, not logic. They cannot:

- Perform arithmetic (`{{ .spec.replicas + 1 }}` doesn't work)
- Call functions (`{{ upper .metadata.name }}` doesn't work)
- Iterate over lists (`{{ range .spec.containers }}` doesn't work)

For complex logic, write a **hook** (Go code). Templates are for simple value substitution.

---

## Summary

| Feature | How It Works |
|---------|--------------|
| **Basic substitution** | `{{ .spec.image }}` → value from CR |
| **Static text** | `"{{ .metadata.name }}-app"` → `my-app-app` |
| **Optional fields** | Missing field → empty string |
| **Defaults** | Empty value → resource default |
| **Conditions** | `when` block controls creation |
| **Debugging** | `--debug` shows resolved values |

**Templates turn one Katalog into many operators.** 🎼