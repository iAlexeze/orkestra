# 03 — Schema

## GET /api/v1/schema/{kind}

Returns the OpenAPI schema for the `spec` of a CRD registered in this Katalog. Used by the Control Center to generate IDP forms.

## Why the gateway serves this

The gateway already resolves CRD OpenAPI schemas in the webhook conversion path. The REST mapper discovers all registered CRDs and their versions; the dynamic client reads the CRD object which includes the `spec.versions[*].schema.openAPIV3Schema` block.

This endpoint makes that resolution available to callers — so the Control Center (or any tool) can discover what fields a CRD has without reading Kubernetes directly.

## Response

```json
{
  "kind": "Application",
  "group": "platform.myorg.io",
  "version": "v1",
  "spec": {
    "properties": {
      "environment": {
        "type": "string",
        "enum": ["staging", "production"],
        "description": "Target environment for this application"
      },
      "image": {
        "type": "string",
        "description": "Container image reference"
      },
      "replicas": {
        "type": "integer",
        "minimum": 1,
        "maximum": 20,
        "default": 2
      }
    },
    "required": ["environment", "image"]
  }
}
```

Only the `spec` properties are returned — not `status`, not `metadata`. Callers building forms do not need the full CRD schema.

## What the Control Center does with it

| OpenAPI annotation | Form element |
|--------------------|-------------|
| `type: string` | Text input |
| `type: integer` / `number` | Number input |
| `type: boolean` | Toggle |
| `enum: [...]` | Dropdown |
| `type: object` | JSON textarea (v1) |
| `type: array` | JSON textarea (v1) |
| `description` | Label + tooltip |
| `default` | Pre-populated value |
| `required` | `*` marker + client-side validation |
| `minimum` / `maximum` | Range hint + client validation |

`type: object` and `type: array` are rendered as JSON textarea inputs in v1. Richer nested rendering (expandable field groups) is documented in [contributing-controlcenter.md](../../../documentation/contributing/contributing-controlcenter.md) as a follow-on contribution.

## IDP field hints

The schema provides the machine-readable contract. The Katalog `idp.fields` block adds presentation hints on top:

```yaml
spec:
  crds:
    application:
      idp:
        enabled: true
        fields:
          environment:
            label: "Environment"
            hint: "Production deployments require platform team approval"
            order: 1
          image:
            label: "Container Image"
            placeholder: "ghcr.io/myorg/myapp:v1.0.0"
            order: 2
```

Field hints are merged with the schema properties when the CC renders the form. A field without a hint renders from its schema `description` and property name. A field with a hint overrides the label, adds a tooltip, and sets tab order.

## Access control

`GET /api/v1/schema/{kind}` is only served for CRDs where `idp.enabled: true`. A request for a kind that exists in the Katalog but does not have IDP enabled returns 404, not the schema. This prevents schema discovery for CRDs that are not intentionally exposed via the Apply API.

→ Next: [04-resources.md](04-resources.md)
