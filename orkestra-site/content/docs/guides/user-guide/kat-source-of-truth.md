---
title: "Kat Source Of Truth"
weight: 41
---

# The Katalog as Single Source of Truth

Write one file. Generate everything else.

---

## The observation

A Katalog already contains everything needed to run an operator:

- `apiTypes` — the CRD's group, version, kind, plural, and scope
- `validation` — which fields are required (deny + exists rules)
- `mutation` — which fields have defaults, and what their types are
- template expressions — every `{{ .spec.* }}` reference names a spec field
- `status.fields` — the complete status schema and printer columns
- `conversion` — the webhook configuration when version paths are declared
- `reconciler` — what Kubernetes and external resources to manage
- `providers` — which external systems are touched

Every artifact the operator needs is derivable from this. Not approximated —
fully derivable.

---

## What can be generated

```
katalog.yaml
    │
    ├── ork generate crd     → crd.yaml
    │     openAPIV3Schema from validation + mutation + templates
    │     required[] from deny+exists rules
    │     additionalPrinterColumns from status.fields
    │     conversion webhook from conversion paths
    │
    ├── ork generate cr      → cr.yaml
    │     required fields with typed placeholders
    │     optional fields with their declared defaults
    │
    ├── ork generate bundle  → bundle.yaml
    │     ConfigMap with Komposer or Katalog YAML
    │     ClusterRole + ClusterRoleBinding
    │
    ├── ork generate bundle  → bundle.yaml
    │     ConfigMap with Komposer or Katalog YAML
    │     Ready to apply 
    │
    ├── ork generate rbac    → rbac.yaml
    │     least-privilege ClusterRole
    │     exactly the verbs the reconciler needs
    │
    └── ork validate         → errors/warnings
          schema validation of the Katalog itself
          provider registration checks
          conversion path consistency
```

---

## The workflow

Before this, building a Kubernetes operator required:

```
1. Write Go types (cronjob_types.go)         ~80 lines
2. Run controller-gen to generate CRD        generated
3. Write the CRD schema by hand or fix gen   ~100 lines
4. Write example CRs                         ~30 lines per CR
5. Write the reconciler                      ~200 lines
6. Write RBAC by hand                        ~40 lines
7. Write the Deployment/ConfigMap            ~60 lines
8. Build, push image, deploy
```

After this:

```
1. Write the Katalog                         ~80 lines YAML
2. ork generate crd --katalog katalog.yaml
3. ork generate cr  --katalog katalog.yaml
4. ork generate bundle --katalog katalog.yaml
5. kubectl apply -f crd.yaml
6. kubectl apply -f bundle.yaml
7. kubectl apply -f cr.yaml
```

No Go. No controller-gen. No handwritten RBAC. No manually maintained CRD
schema that drifts from the actual field usage.

---

## How CRD generation works

The generator reads the Katalog and derives the schema from three sources,
in priority order:

**Source 1 — Validation deny+exists rules**

```yaml
validation:
  rules:
    - field: spec.image
      operator: exists
      action: deny
      message: "spec.image is required"
```

Produces:
```yaml
spec:
  type: object
  required: [image]
  properties:
    image:
      type: string
      description: "spec.image is required"
```

**Source 2 — Mutation defaults**

```yaml
mutation:
  rules:
    - field: spec.replicas
      default: 2
```

Produces:
```yaml
properties:
  replicas:
    type: integer
    default: 2
```

Type is inferred from the default value: `2` → `integer`, `"true"` → `string`,
`false` → `boolean`, `[]` → `array`.

**Source 3 — Template expressions**

```yaml
onCreate:
  deployments:
    - image: "{{ .spec.image }}"
      name: "{{ .metadata.name }}"
  services:
    - port: "{{ .spec.port }}"
```

The generator scans all template values for `{{ .spec.fieldName }}` patterns.
Fields not already covered by validation or mutation rules are added as
`type: string` fields. This catches any field the reconciler uses that was
not explicitly declared.

**Printer columns** — derived from `status.fields`. The `phase` field is
always the first column if declared. Up to three additional status fields
become columns.

**Conversion webhook** — when `conversion.paths` are declared, the generator
adds the conversion webhook configuration pointing at Orkestra's `/convert`
endpoint. The `caBundle` field is left empty with a comment instructing
where to paste the cert.

---

## How CR generation works

The example CR is built from the same sources in reverse:

- Required fields (from deny+exists validation) → typed placeholders:
  - `image` → `"my-image:latest"`
  - `steps` → `[]`
  - `replicas` → `1`
  - `region` → `"us-east-1"`
  - `port` → `8080`

- Optional fields (from mutation defaults) → their declared default values

The placeholder logic uses field name heuristics: a field named `image`
gets `"my-image:latest"`, a field named `port` gets `8080`, a field named
`region` gets `"us-east-1"`. This produces a CR that can be applied
immediately without modification for simple cases.

---

## The schema drift problem — solved

Without generation from the Katalog, CRD schemas drift. Someone adds a field
reference in a template expression but forgets to update the CRD. The field
works (Kubernetes stores unknown fields with `x-kubernetes-preserve-unknown-fields`)
but validation is missing, documentation is incomplete, and `kubectl explain`
shows nothing.

With Katalog-driven generation, the schema is always in sync with actual
usage. Every field the reconciler references is in the schema. Every required
field has validation. Every optional field has its default documented.

Run `ork generate crd --katalog katalog.yaml` in CI alongside `ork validate`
and the schema is always accurate.

---

## Limitations

**Type inference is approximate.** The generator infers `string` for fields
that appear only in template expressions. If `spec.replicas` is used as
`{{ .spec.replicas }}` in a deployment template but has no mutation default,
it is typed as `string`. Add a mutation default to get the correct type:

```yaml
mutation:
  rules:
    - field: spec.replicas
      default: 1   # ← integer default → integer type in schema
```

**Nested spec fields are not deeply schematised.** `spec.database.host` and
`spec.database.port` both appear in templates as `.spec.database.host` but
the generator only extracts the top-level `database` segment and types it as
`object`. Deep schema generation requires explicit type declarations — a future
`spec.schema` block in the Katalog could provide this.

**Arrays require element type declarations.** `spec.steps` is typed as `array`
when inferred from a default of `[]`, but the item schema is not generated.
The CRD will have `{type: array}` without `items`. This is valid but less
precise than a handwritten schema. A future `spec.schema` block covers this.

---

## The future: `spec.schema`

A planned addition to the Katalog allows explicit schema declarations for
cases where inference is insufficient:

```yaml
spec:
  crds:
    pipeline:
      schema:
        spec:
          steps:
            type: array
            items:
              type: object
              required: [name, command]
              properties:
                name:
                  type: string
                command:
                  type: string
```

When declared, the explicit schema takes priority over inference. The generator
merges declared and inferred schemas — declared fields override, inferred
fields fill gaps. This gives full control when needed while keeping the
common case automatic.