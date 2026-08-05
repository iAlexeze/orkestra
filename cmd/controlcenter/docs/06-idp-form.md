# 06 — IDP Self-Service Form

The IDP form is the browser-native delivery path for the Apply API's target mode. A platform team declares `idp.enabled: true` and `idp.fields`/`idp.additionalFields` on a CRD entry; from that moment, any developer with access to the Control Center can create instances of that CRD by filling a form — no YAML, no `kubectl`, no cluster credentials, and no knowledge of the CRD's Kubernetes shape.

Control Center never constructs a Kubernetes CR itself. It submits a flat `{"target": "<target>", ...fields}` payload to the gateway's Apply API; the gateway (`pkg/gateway/applyapi`) resolves the target to a CRD and builds the CR via `BuildCRFromTarget`, using the CRD's `idp.fields`/`idp.additionalFields` declarations to route each field into `spec`, `metadata.labels`, or `metadata.annotations`, and `idp.name`/`idp.namespace` to resolve identity.

## Activation

Three things must all be true for the `[+ Create]` button to appear:

1. `gateway.applyAPI.enabled: true` on the Katalog
2. `idp.enabled: true` on the CRD entry
3. `GATEWAY_TOKEN` set on the CC process

The runtime sets `CRDSummaryResponse.IDPEnabled` and `CRDSummaryResponse.Target` in its `/katalog` response (`"idpEnabled"`, `"target"`). CC mirrors both onto `CRDSummary`. `Target` is the identifier CC submits to the gateway — it comes from `idp.target` if the platform team set one, otherwise the lowercased Kind — CC never derives it itself. `handleIDPCreateForm` redirects back if `IdpEnabled` is false or `Target` is empty (IDP not actually usable for this CRD).

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
  │      from the same idp.fields/idp.additionalFields declaration.
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
    RequireIDPName is true; the Identity section is omitted entirely from
    the form when idp.name is declared, since the gateway resolves the name
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

`fetchIDPFields` in `cc/controlcenter.go` calls `buildIDPField` once per entry in `SchemaResponse.Fields` — there is only one field source now, not two. Each entry decodes into `idpFieldHint` (mirrors `orktypes.IDPFieldConfig`), which carries its own `type`/`enum` — there's no CRD OpenAPI schema involved and no separate "required" list to merge (each field's `required` is already final).

One capability this dropped versus the old CRD-schema-driven approach: pre-populating a field's value from the CRD's OpenAPI `default:` — `IDPFieldConfig` has no `Default`, so the schema API doesn't expose one. If defaults become worth restoring, that's a schema-API change (`IDPFieldConfig.Default` + `SchemaResponse`), not something CC can add on its own.

## Configuration

| Env var | CC field | Purpose |
|---------|-----------|---------|
| `GATEWAY_TOKEN` | `ControlCenter.gatewayToken` | Bearer token for gateway Apply API calls; must match a token declared in `gateway.applyAPI.auth.tokens` |

## Files involved

| File | Role |
|------|------|
| `pkg/runtime/kordinator/crd_health_handers.go` | `CRDSummaryResponse.Target`/`IDPEnabled` — from `crd.IDPTargetOrEmpty()`/`crd.IDPEnabled()` |
| `pkg/gateway/applyapi/schema.go` | `SchemaResponse` — the flat field contract CC consumes |
| `pkg/gateway/applyapi/target.go` | `BuildCRFromTarget` — builds the CR the gateway applies; CC never sees this shape |
| `cc/types.go` | `CRDSummary.Target`; `IDPField`, `IDPSection`, `IDPFormData` |
| `cc/controlcenter.go` | `handleIDPCreateForm`, `fetchIDPFields`, `buildIDPField`, `handleIDPApplyForm`, `handleIDPSchema` |
| `cc/assets/templates/idp_form.html` | form page — `collectPayload()` builds the flat POST body |
| `cc/idp_fields_test.go` | `buildIDPField` coverage |

→ Next: the `examples/use-cases/idp` example pack is being rethought from scratch for target mode — not yet updated, don't treat it as current.
