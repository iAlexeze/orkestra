# 03 — Schema

Two schema endpoints, for two different callers. `GET /api/v1/schema` is the Serve contract — target mode's flat field list. `GET /api/v1/raw-schema` is the CRD's actual Kubernetes schema, for full-CR-mode callers — see the [section below](#get-apiv1raw-schemakindltkgt--the-escape-hatch-for-full-cr-mode).

## GET /api/v1/schema and GET /api/v1/schema?target=&lt;t&gt;

Two modes on one endpoint:

- No `target` — catalog mode: lists every target this caller can see, paginated.
- `?target=<t>` — the flat field contract for one target: what to submit, what's required.

Used by the Control Center to generate the Serve create form, and by any other caller (CI, a CLI, a Slack bot) that wants to discover what it can submit before submitting it.

## Why the gateway serves this

The gateway already reads the full Katalog — every CRD's `serve.fields`, `serve labels/annotations`, and `serve.tokens` are in memory. Serving the schema is reading that config back out, not resolving anything from Kubernetes. This endpoint is what makes `serve.fields` a caller-facing contract instead of an internal-only presentation hint.

## Catalog response — `GET /api/v1/schema`

```json
{
  "total": 2,
  "limit": 50,
  "offset": 0,
  "items": [
    { "target": "smartapp", "title": "Application", "category": "Workloads", "required": ["repository", "image"] },
    { "target": "database", "title": "Database", "required": ["dbName"] }
  ]
}
```

Only targets the caller's token can `list` under `schema` permissions appear — see [05-auth.md](05-auth.md#scoping--servetokens).

## Per-target response — `GET /api/v1/schema?target=smartapp`

```json
{
  "target": "smartapp",
  "title": "Application",
  "description": "Deploy an application",
  "fields": {
    "repository": {
      "label": "Repository",
      "placeholder": "myorg/payments-api",
      "required": true,
      "order": 1
    },
    "environment": {
      "label": "Environment",
      "type": "enum",
      "enum": ["staging", "production"],
      "hint": "Production deployments require platform-team review",
      "order": 2
    },
    "team": {
      "label": "Team",
      "required": true,
      "order": 0
    }
  },
  "required": ["repository", "environment", "team"]
}
```

`fields` is **flat and merged** — spec fields (`serve.fields`), label fields, and annotation fields (`serve.labels`/`.annotations`) all appear as ordinary entries in the same map, in the same shape. There is nothing in this response that says which Kubernetes location a field writes to. The caller submits a flat value for `team` exactly the way it submits one for `repository`; the gateway decides at apply time, from the Katalog, that `team` is a label and `repository` is a spec field. See [Target Mode](../../../../documentation/concepts/idp/02-target-mode.md).

Each field entry is `ServeFieldConfig` — `label`, `placeholder`, `hint`, `order`, `category`, `required`, `disabled`, `type` (`string` default, `integer`, `number`, `boolean`, `enum`), `enum`. `type`/`enum` come from what the platform team declared in the Katalog, never from introspecting the CRD — there is no CRD OpenAPI schema in this response at all.

## What the Control Center does with it

| `ServeFieldConfig` | Form element |
|-------------------|-------------|
| (no `type`, or `type: string`) | Text input |
| `type: integer` / `type: number` | Number input |
| `type: boolean` | Checkbox |
| `type: enum` + `enum: [...]` | Dropdown |
| `hint` | Text below the field |
| `required` | `*` marker + client-side validation, mirrored server-side |
| `disabled` | Greyed out, excluded from submission |
| `order` | Field sequence within its `category` section |

There is no `type: object`/`type: array` case — target mode has no nested-object concept on the caller-facing side. A spec field that lands in a nested location does so via `serve.fields.<name>.path` on the platform-team side; the caller still submits one flat scalar value for it. See [Nested fields with `path`](../../../../documentation/reference/schema/02-katalog/17-gateway-api.md#nested-fields-with-path).

## Serve field hints

The schema response *is* `serve.fields`/`serve labels/annotations` — there's no separate hint layer merged in on top of something else, unlike the old CRD-OpenAPI-driven schema this endpoint used to serve:

```yaml
spec:
  crds:
    application:
      serve:
        enabled: true
        target: smartapp
        fields:
          environment:
            label: "Environment"
            type: enum
            enum: ["staging", "production"]
            hint: "Production deployments require platform team approval"
            order: 1
```

Every property a caller sees at `?target=smartapp` is something the platform team wrote in the Katalog. There is no fallback to a schema-derived label or description — an undeclared field is not exposed at all, target mode has no equivalent of "renders from the raw property name."

## GET /api/v1/raw-schema?kind=&lt;k&gt; — the escape hatch for full CR mode

A different endpoint, for a different caller. `?target=` describes the Serve contract; `raw-schema` describes the CRD itself — the actual Kubernetes OpenAPI v3 schema, fetched live from the CRD object in the cluster, not from Katalog config. For a caller building a full CR by hand (`POST /api/v1/apply` with `apiVersion`/`kind`/`spec` instead of `target`), this is the schema that request body has to match.

```bash
curl "${GATEWAY}/api/v1/raw-schema?kind=AppRequest" -H "Authorization: Bearer $TOKEN"
```

```json
{
  "kind": "AppRequest",
  "apiVersion": "platform.myorg.io/v1",
  "spec": {
    "properties": {
      "repository": { "type": "string" },
      "environment": { "type": "string", "enum": ["staging", "production"] }
    },
    "required": ["repository"]
  },
  "labels": {
    "team": { "label": "Team", "required": true }
  },
  "annotations": {}
}
```

Differences from `?target=`, all deliberate:

- Keyed by `kind` (and optional `apiVersion` to disambiguate multi-version CRDs), never by `target` — this endpoint is for callers who think in Kubernetes terms.
- `spec.properties`/`spec.required` are the CRD's real OpenAPI schema, verbatim — not `ServeFieldConfig`. This is the one place in the Gateway API that still exposes Kubernetes schema shape to a caller.
- Works even when the CRD has no `serve.fields` declared at all — `raw-schema` only needs the CRD to exist and be serve-enabled, not to have a field contract defined.
- `labels`/`annotations`, when `serve labels/annotations` is declared, are still the curated `ServeFieldConfig` shape — those two are metadata Kubernetes has no schema for in the first place, so there's nothing "raw" to fall back to.

Fetches the CRD definition from the Kubernetes API on every call (unlike `?target=`, which only ever reads the in-memory Katalog) — expect this endpoint to be slower and to fail if the gateway's ServiceAccount can't read `customresourcedefinitions`.

## Access control

`GET /api/v1/schema`, `GET /api/v1/schema?target=<t>`, and `GET /api/v1/raw-schema` are only served for CRDs where `serve.enabled: true`. A `target` that doesn't resolve to an serve-enabled CRD returns 404 naming the target, along with the list of available targets — not the schema.

→ Next: [04-resources.md](04-resources.md)
