# 06 — IDP Form

The IDP form is the Control Center's browser-native delivery path for the Gateway API's target mode. A platform team declares `serve.enabled: true` and `serve.fields`/`serve.labels`/`serve.annotations` on a CRD entry; from that moment, any developer with access to the Control Center can create instances of that CRD by filling a form — no YAML, no `kubectl`, no cluster credentials, and no knowledge of the CRD's Kubernetes shape.

The Control Center calls itself an IDP form because that is its framing of the surface — a developer self-service portal. The underlying Orkestra schema uses `serve.*` keys; the CC is just one client of those, and it chooses to present them as an IDP.

Control Center never constructs a Kubernetes CR itself. It submits a flat `{"target": "<target>", ...fields}` payload to the gateway's Gateway API; the gateway (`pkg/gateway/api`) resolves the target to a CRD and builds the CR via `BuildCRFromTarget`, using the CRD's `serve.fields`/`serve.labels`/`serve.annotations` declarations to route each field into `spec`, `metadata.labels`, or `metadata.annotations`, and `serve.name`/`serve.namespace` to resolve identity.

## Activation

Three things must all be true for the `[+ Create]` button to appear:

1. `gateway.api.enabled: true` on the Katalog
2. `serve.enabled: true` on the CRD entry
3. `GATEWAY_TOKEN` set on the CC process

The runtime sets `CRDSummaryResponse.IDPEnabled` and `CRDSummaryResponse.Target` in its `/katalog` response (`"serveEnabled"`, `"target"`). CC mirrors both onto `CRDSummary`. `Target` is the identifier CC submits to the gateway — it comes from `serve.target` if the platform team set one, otherwise the lowercased Kind — CC never derives it itself. `handleIDPCreateForm` redirects back if `IdpEnabled` is false or `Target` is empty (serve not actually usable for this CRD).

## Request flow

```
Browser GET /controlcenter/katalog/{kat}/crd/{crd}/cr/create
  │
  ├─ handleIDPCreateForm looks up CRDSummary.Target for this CRD
  ├─ fetchIDPFields → GET {gateway}/api/v1/schema?target={target}
  │    Authorization: Bearer {GATEWAY_TOKEN}
  │    ← SchemaResponse { target, title, description, fields, required }
  │      fields is a FLAT map[string]IDPFieldConfig — no spec/label/
  │      annotation distinction. The gateway resolves that at apply time
  │      from the same serve.fields/serve.labels/serve.annotations declaration.
  │
  └─ renderTemplate("idp_form.html", IDPFormData)
       Fields assembled by buildIDPField, one per schemaResp.Fields entry:
         field.type   →  HTML input type (string→text, integer/number→number,
                          boolean→checkbox, enum(+values)→select)
         field.label       →  <label> (falls back to the capitalized field name)
         field.placeholder →  placeholder=""
         field.hint        →  hint text below the field
         field.order       →  sort order (0/unset sorts last)
         field.category    →  section heading (fields with no category group
                               under a single default "Fields" section)
         field.when/anyOf  →  data-when/data-anyof — evaluated client-side to
                               show/hide the field as the form is filled

Browser POST /controlcenter/katalog/{kat}/crd/{crd}/cr/create
  Body: { "target": "<target>", "name": "...", ...flatFields }
    (collectPayload() in idp_form.html builds this directly — one flat
    object, field name → value, no bucketing. "name" is only present when
    RequireServeName is true; the Identity section is omitted entirely from
    the form when serve.name is declared, since the gateway resolves the name
    server-side in that case.)
  │
  ├─ handleIDPApplyForm decodes the body, overwrites "target" with the
  │    server-resolved value (never trusts a client-supplied one), and
  │    forwards the flat payload as-is
  ├─ proxyIDPRequest → POST {gateway}/api/v1/apply
  │    Authorization: Bearer {GATEWAY_TOKEN}
  │    ← status code + gateway response body (the gateway builds the CR)
  │
  └─ Returns gateway response JSON to browser
       JS redirects to CR list on 200/201 with no warnings
       JS shows inline violations (field-level) or a rejection banner otherwise
```

The gateway token is never sent to the browser. The CC backend holds it in `cc.gatewayToken` and injects it into the gateway request headers via `proxyIDPRequest`.

## Field rendering

`fetchIDPFields` in `cc/controlcenter.go` calls `buildIDPField` once per entry in `SchemaResponse.Fields` — there is only one field source now, not two. Each entry decodes into `idpFieldHint` (mirrors `orktypes.ServeFieldConfig`), which carries its own `type`/`enum` — there's no CRD OpenAPI schema involved and no separate "required" list to merge (each field's `required` is already final).

One capability this dropped versus the old CRD-schema-driven approach: pre-populating a field's value from the CRD's OpenAPI `default:` — `ServeFieldConfig` has no `Default`, so the schema API doesn't expose one. If defaults become worth restoring, that's a schema-API change (`ServeFieldConfig.Default` + `SchemaResponse`), not something CC can add on its own.

## Configuration

| Env var | CC field | Purpose |
|---------|-----------|---------|
| `GATEWAY_TOKEN` | `ControlCenter.gatewayToken` | Bearer token for gateway API calls; must match a token declared in `gateway.api.auth.tokens` |

## Files involved

| File | Role |
|------|------|
| `pkg/runtime/kordinator/crd_health_handers.go` | `CRDSummaryResponse.Target`/`IDPEnabled` — from `crd.ServeTargetOrEmpty()`/`crd.IsServeEnabled()` |
| `pkg/gateway/api/schema.go` | `SchemaResponse` — the flat field contract CC consumes |
| `pkg/gateway/api/target.go` | `BuildCRFromTarget` — builds the CR the gateway applies; CC never sees this shape |
| `cc/types.go` | `CRDSummary.Target`; `IDPField`, `IDPSection`, `IDPFormData` |
| `cc/controlcenter.go` | `handleIDPCreateForm`, `fetchIDPFields`, `buildIDPField`, `handleIDPApplyForm`, `handleIDPSchema` |
| `cc/assets/templates/idp_form.html` | form page — `collectPayload()` builds the flat POST body |
| `cc/idp_fields_test.go` | `buildIDPField` coverage |

→ Next: the `examples/use-cases/idp` example pack is being rethought from scratch for target mode — not yet updated, don't treat it as current.
