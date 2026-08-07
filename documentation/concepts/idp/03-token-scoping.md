# Token Scoping

Every example so far has assumed one caller. Real platforms have several: a Control Center that needs full access, a CI pipeline that should only touch staging, a production deploy path that can create but never delete, an audit tool that should only ever read. They authenticate with the same gateway, against the same CRD, through the same `POST /api/v1/apply` — but they should not all be able to do the same things. `serve.tokens` is what makes that true.

---

## Without it, every token is equal

`gateway.api.auth.tokens` declares which bearer tokens the gateway accepts at all — but by itself, a valid token can call any endpoint the Gateway API exposes for a CRD. That's fine for a single-caller setup. It stops being fine the moment a CI token and a human-facing Control Center token exist side by side: a leaked CI token, without `serve.tokens`, is exactly as powerful as the token behind your production create button.

`serve.tokens`, declared per CRD, closes that gap:

```yaml
serve:
  enabled: true
  target: smartapp
  tokens:
    control-center:
      namespaces: [default]
      permissions:
        global: ["*"]

    ci-pipeline:
      namespaces: [team-payments-staging, team-orders-staging]
      permissions:
        resources: [create, update, list, get]    # no delete — needs a human
        schema: [get]                             # can read field contracts, not modify them

    security-audit:
      namespaces: [default, team-payments-production, team-orders-production]
      permissions:
        global: [get, list]                       # read-only, everywhere it's listed

    prod-deploy:
      namespaces: [team-payments-production, team-orders-production]
      permissions:
        resources: [create, update, get, list]     # no delete, no schema access at all
```

Four tokens, four different answers, one CRD. `ci-pipeline` can ship to staging but not touch production, and can read the schema (to generate its own pipeline config from it) but never write anything through the `schema` endpoints — not that there's anything to write there anyway, `schema` only ever supports `get`/`list`. `prod-deploy` can ship to production but can't delete anything there, and has no `schema` permissions declared at all, so `GET /api/v1/schema?target=smartapp` is a `403` for it. `security-audit` can read everywhere it's listed and write nowhere. None of this is a namespace-watch rule — the runtime still watches whatever the CRD's own `allowedNamespaces` says, for every token equally. This is authorization scoped to *who's asking*, layered on top of that.

`serve.tokens` is the per-CRD scoping layer. `gateway.api.auth.tokens` is still where the tokens themselves are declared — `serve.tokens` just references names from that list and says what each one may do on this specific CRD.

---

## Three scopes, not one

Permissions aren't a single list — they're three, matching the three kinds of thing the Gateway API exposes:

| Scope | Governs | Valid operations |
|-------|---------|-------------------|
| `resources` | `POST /api/v1/apply`, `GET`/`DELETE /api/v1/resources/...` — the actual CRUD | `get`, `list`, `create`, `update`, `delete`, `*` |
| `schema` | `GET /api/v1/schema`, `GET /api/v1/raw-schema` — discovery | `get`, `list` — any other operation is rejected at `ork validate` |
| `global` | Both, when set and neither of the above is — a shorthand, not a third thing to reconcile | same as `resources` |

`security-audit` above uses `global: [get, list]` because it wants the same read-only answer everywhere. `prod-deploy` sets `resources` explicitly instead, specifically so it gets no `schema` access at all — a token can create CRs without being able to introspect the CRD's shape, if that's the boundary you want. Setting both `global` and a class list is legal, but the class list must be a narrower subset of `global` — `ork validate` rejects a `resources` permission `global` doesn't already grant, so a token can't accidentally end up more powerful in one scope than its own `global` setting implies.

---

## What a denied caller sees

Not a silent drop, not a generic `401`:

```json
{
  "error": "permission denied",
  "message": "token \"ci-pipeline\" lacks \"delete\" permission on \"App\""
}
```

`403`, with a message naming exactly which check failed — unknown token, wrong namespace, or (as above) missing operation. Whoever's debugging a failed CI job doesn't have to cross-reference the Katalog by hand to find out why.

---

## See also

→ [Serve token permissions](../../security/08-serve-permissions.md) — the security-model write-up: denial responses, `ork validate` checks, how this layers with `allowedNamespaces`

→ [Gateway API reference — `serve.tokens`](../../reference/schema/02-katalog/17-gateway-api.md#servetokens--fine-grained-permissions) — fine-grained permissions

→ [Namespace protection](../../security/05-namespace-protection.md) — the CRD-level layer every token's `namespaces` list is still bounded by

**Inspect from the CLI:**
`ork serve tokens --target <t>` — show CRD-level token map · `ork serve tokens --alias <a>` — effective tokens for an alias · `ork serve can-i --token <t> --target <t> --operation <op>` — live permission check

→ [CLI reference — ork serve](../../reference/cli/13-serve.md)
