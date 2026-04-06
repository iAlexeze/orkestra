---
title: "The Katalog Manifest"
weight: 1
description: "Understanding the core Katalog YAML specification."
---

The `Katalog` manifest is the central artifact in Orkestra. It describes your CRD, the child resources the operator should manage, status conditions, and observability settings — all in a single YAML file.

## Structure overview

```yaml
apiVersion: orkestra.sh/v1
kind: Katalog
metadata:
  name: <operator-name>
  namespace: <namespace>
spec:
  crd:          # CRD definition
  resources:    # Child resources to manage
  status:       # Status conditions
  hooks:        # Lifecycle hooks (optional)
  observe:      # Observability settings
  validation:   # Admission validation (optional)
  stateMachine: # State machine (optional)
```

## The `crd` section

Defines the Custom Resource Definition that Orkestra will register.

```yaml
spec:
  crd:
    group: apps.example.com      # API group
    kind: Website                # Resource kind (PascalCase)
    version: v1alpha1            # API version
    scope: Namespaced            # Namespaced or Cluster
    description: "Manages web apps"
    schema:                      # OpenAPI v3 schema (optional)
      spec:
        image:
          type: string
          description: "Container image"
        replicas:
          type: integer
          minimum: 0
          maximum: 100
          default: 1
```

{{< callout type="info" title="CRD generation" >}}
Orkestra auto-generates the full CRD manifest from this section and registers it with the API server. You never need to write a raw `CustomResourceDefinition` YAML.
{{< /callout >}}

## The `resources` section

Lists the Kubernetes resources the operator should create and manage for each CR instance.

```yaml
spec:
  resources:
    - kind: Deployment
      name: "{{ .Name }}"
      namespace: "{{ .Namespace }}"
      template: ./templates/deployment.yaml
    - kind: Service
      name: "{{ .Name }}-svc"
      namespace: "{{ .Namespace }}"
      template: ./templates/service.yaml
    - kind: ConfigMap
      name: "{{ .Name }}-config"
      template: ./templates/configmap.yaml
      optional: true   # Don't fail reconciliation if this errors
```

### Template variables

Inside resource templates, you have access to:

| Variable | Description |
|---|---|
| `{{ .Name }}` | Name of the CR instance |
| `{{ .Namespace }}` | Namespace of the CR instance |
| `{{ .Spec }}` | Full spec of the CR instance |
| `{{ .Labels }}` | Labels on the CR instance |
| `{{ .Annotations }}` | Annotations on the CR instance |
| `{{ .UID }}` | UID of the CR instance |

## The `status` section

Declares the status conditions your operator will manage.

```yaml
spec:
  status:
    conditions:
      - type: Ready
        description: "All child resources are healthy"
      - type: Degraded
        description: "One or more child resources are failing"
      - type: Progressing
        description: "Reconciliation is in progress"
```

Orkestra automatically updates these conditions during reconciliation.

## The `observe` section

Controls the built-in observability features.

```yaml
spec:
  observe:
    events: true          # Emit Kubernetes events on state changes
    controlCenter: true   # Show in Control Center UI
    metrics: true         # Expose Prometheus metrics
```

## Next steps

- [Resource Templates](/docs/basics/resource-templates/) — advanced templating patterns
- [Status Management](/docs/basics/status/) — custom status fields and conditions
- [Lifecycle Hooks](/docs/basics/hooks/) — pre/post reconciliation hooks
