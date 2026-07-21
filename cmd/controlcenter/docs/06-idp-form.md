# 06 — IDP Self-Service Form

The IDP form is the browser-native delivery path for the Apply API. A platform team declares `idp.enabled: true` on a CRD entry; from that moment, any developer with access to the Control Center can create instances of that CRD by filling a form — no YAML, no `kubectl`, no cluster credentials.

## Activation

Three things must all be true for the `[+ Create]` button to appear:

1. `gateway.applyAPI.enabled: true` on the Katalog
2. `idp.enabled: true` on the CRD entry
3. `ORK_CC_APPLY_TOKEN` set on the CC process

The runtime sets `CRDSummary.IDPEnabled` in its `/katalog` response (field: `"idpEnabled"`). The CC reads it during the background fetch and passes it to `CRListView.IDPEnabled`. The button is suppressed server-side when no Apply token is configured so the route is never advertised without auth.

## Request flow

```
Browser GET /controlcenter/katalog/{kat}/crd/{crd}/idp
  │
  ├─ handleIDPCreate reads Instance.GatewayEndpoint
  ├─ FetchIDPSchema → GET {gateway}/api/v1/schema/{kind}
  │    Authorization: Bearer {ORK_CC_APPLY_TOKEN}
  │    ← SchemaResponse { kind, apiVersion, properties, idpFields }
  │
  └─ renderTemplate("idp_form.html", IDPFormView)
       Fields assembled by buildIDPFormFields:
         OpenAPI property type  →  HTML input type
         string                 →  <input type="text">
         string + enum          →  <select>
         integer / number       →  <input type="number">
         boolean                →  <input type="checkbox">
         IDP hint.label         →  <label>
         IDP hint.placeholder   →  placeholder=""
         IDP hint.hint          →  hint text below the field
         IDP hint.order         →  sort order (low values first)

Browser POST /controlcenter/katalog/{kat}/crd/{crd}/idp
  Body: { "name": "...", "namespace": "...", "spec": { ... } }
  │
  ├─ handleIDPCreate parses the JSON body
  ├─ FetchIDPSchema again → gets apiVersion and kind
  ├─ Wraps into full CR envelope:
  │    { apiVersion, kind, metadata: { name, namespace }, spec }
  ├─ PostIDPApply → POST {gateway}/api/v1/apply
  │    Authorization: Bearer {ORK_CC_APPLY_TOKEN}
  │    ← status code + gateway response body
  │
  └─ Returns gateway response JSON to browser
       JS redirects to CR list on 200/201
       JS shows inline error on any other status
```

The Apply token is never sent to the browser. The CC backend holds it in `Config.ApplyToken` and injects it into the gateway request headers.

## Field rendering

`buildIDPFormFields` in `cc/cr_types.go` merges the two sources:

| Source | Provides |
|--------|---------|
| `SchemaResponse.Properties` (OpenAPI) | field names, types, enum values |
| `SchemaResponse.IDPFields` (katalog hints) | labels, placeholders, hints, sort order |

Fields without an IDP hint are still rendered — they get the raw field name as the label and default sort order 999. Fields with an IDP hint but absent from the OpenAPI schema are silently dropped (schema is authoritative for existence).

## Configuration

| Env var | CC struct | Purpose |
|---------|-----------|---------|
| `ORK_CC_APPLY_TOKEN` | `Config.ApplyToken` | Bearer token for gateway Apply API calls; must match a token declared in `gateway.applyAPI.auth.tokens` |

Set it alongside the gateway's self-bootstrapped secret:

```yaml
# helm values
controlcenter:
  env:
    ORK_CC_APPLY_TOKEN:
      valueFrom:
        secretKeyRef:
          name: ork-apply-token
          key: token
```

## Files changed

| File | Change |
|------|--------|
| `pkg/kordinator/crd_health_handers.go` | `CRDSummaryResponse.IDPEnabled` — set from `crd.IDP != nil && crd.IDP.Enabled` |
| `cc/konfig.go` | `ControlCenterKonfig.ApplyToken` — from `ORK_CC_APPLY_TOKEN` |
| `cc/controlcenter.go` | `Config.ApplyToken`; `/idp` route case; `handleIDPCreate`; `handleCRList` now sets `IDPEnabled` |
| `cc/types.go` | `CRDSummary.IDPEnabled` |
| `cc/cr_types.go` | `CRListView.{IDPEnabled,GatewayEndpoint,CreateURL}`; `IDPField`, `IDPSchemaResponse`, `IDPFormField`, `IDPFormView`; `buildIDPFormFields` |
| `cc/client.go` | `FetchIDPSchema`, `PostIDPApply` |
| `cc/assets/templates/cr_list.html` | `[+ Create]` button when `IDPEnabled` |
| `cc/assets/templates/idp_form.html` | new — full self-service form page |

→ Next: see the `examples/use-cases/idp` example pack for an end-to-end walkthrough.
