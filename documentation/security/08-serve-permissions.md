# Serve token permissions

`allowedNamespaces`/`restrictedNamespaces` answer one question: which namespaces does this CRD exist in at all. Every caller gets the same answer — it's a property of the CRD, not of who's asking. `serve.tokens` answers a different question: which caller can do what, and where. Two tokens against the same CRD can have different answers — a `ci-pipeline` token allowed to create in `staging` but not touch `production`; a `security-audit` token that's read-only everywhere. This is authorization scoped to the caller's identity, layered on top of namespace protection, not a replacement for it.

Authentication (proving who you are) and authorization (what you're allowed to do) are distinct steps. The gateway supports three authentication modes:

- **Static tokens** — a pre-shared bearer value from a Kubernetes Secret or environment variable.
- **OIDC tokens** — a short-lived JWT issued by GitHub Actions, GitLab CI, or any OIDC provider. No stored secret; the token is verified against the provider's public JWKS. The verified `sub` claim is stamped on the CR as `orkestra.orkspace.io/serve-source`.
- **Webhook entries** (`gateway.webhooks`) — GitHub/GitLab push, Slack, or a generic JSON caller. Not a bearer token at all: each entry verifies the request itself (an HMAC signature or a static shared secret, matching how that source actually signs its own deliveries), and the entry's own `name` — not a claim, not a header — becomes the identity `serve.tokens` checks and the value stamped as `serve-source`.

The first two authenticate to `gateway.api.auth.tokens`; the third authenticates to `gateway.webhooks`. All three are subject to the same `serve.tokens` authorization rules below — authorization doesn't care which of the three proved the caller's identity. See [Webhook credential verification](09-webhook-verification.md) for how each webhook source verifies itself.

---

## Two independent layers

### CRD-level — topology, same for everyone

`allowedNamespaces`/`restrictedNamespaces` on the CRD entry govern which namespaces the runtime's informer watches and the admission webhook accepts at all. See [Namespace protection](05-namespace-protection.md). Every caller — every token, `kubectl`, the reconciler itself — sees the same boundary.

### `serve.tokens` — identity, per caller

Declared under the CRD's `serve` block, checked only by the Gateway API — a request that never goes through `POST /api/v1/apply`, `GET /api/v1/resources/...`, or `GET /api/v1/schema` never hits this layer at all. It answers, for the token that authenticated this specific request: which operations, on which endpoint class, in which namespaces.

```yaml
serve:
  enabled: true
  target: app
  tokens:
    ci-pipeline:
      namespaces: ["staging"]
      permissions:
        resources: ["create", "update", "get", "list"]
        schema: ["get"]
    security-audit:
      namespaces: ["staging", "production"]
      permissions:
        global: ["get", "list"]
```

`ci-pipeline` can create and update `App` CRs in `staging` — and nowhere else, not even if it later gets used against a different CRD that also lists it, since permissions are declared per CRD, not globally. `security-audit` can read (not write) in both namespaces. Neither is a namespace-watch rule — the runtime still watches whatever `allowedNamespaces` says regardless of which tokens exist.

A CRD with no `serve.tokens` block places no restriction here — any caller valid at the gateway level (`gateway.api.auth.tokens`, or a `gateway.webhooks` entry authenticating with its own credential) can call any endpoint the Gateway API exposes for that CRD, subject only to namespace protection.

---

## Per-entry token scoping (aliases)

When a CRD uses the `serve.target` map form, each entry — primary or alias — can declare its own `tokens:` block. Entry-level tokens override the CRD-level `serve.tokens` for callers reaching that surface. The resolution chain for every request is:

```text
entry tokens  →  serve.tokens (CRD default)  →  allow all
```

A token absent from the entry's map is denied at that surface, even if it is valid at the CRD level. Aliases can only *narrow* access, never widen it.

```yaml
serve:
  enabled: true
  # CRD-level fallback — applies to any entry that declares no tokens:
  tokens:
    control-center:
      permissions:
        global: ["*"]
    ci-pipeline:
      permissions:
        resources: [get, list]
        schema: [get]

  target:
    app:
      primary: true
      # No tokens: declared — inherits CRD-level serve.tokens above.
      # Both control-center and ci-pipeline can reach the primary surface.

    preview:
      tokens:
        # Only control-center — ci-pipeline is denied even though it has CRD access.
        control-center:
          permissions:
            global: [get, list]

    internal:
      tokens:
        # Separate token for internal tooling — not listed at the CRD level at all.
        # The internal entry declares it here; serve.tokens fallback is not used.
        platform-team:
          namespaces: ["production"]
          permissions:
            global: ["*"]
```

`ork validate` checks every token name in every entry's `tokens:` block — a name not in `gateway.api.auth.tokens` is a hard error regardless of which entry declares it.

---

## Permission scopes

| Scope | Endpoints it governs | Valid operations |
|-------|----------------------|-------------------|
| `global` | All of the below — a fallback when `schema`/`resources` aren't set | `get`, `list`, `create`, `update`, `delete`, `*` |
| `schema` | `GET /api/v1/schema`, `GET /api/v1/raw-schema` | `get`, `list` only — `create`/`update`/`delete` are meaningless for a read-only discovery endpoint and rejected at `ork validate` |
| `resources` | `POST /api/v1/apply`, `GET`/`DELETE /api/v1/resources/...` | `get`, `list`, `create`, `update`, `delete`, `*` |

When `global` is set alongside `schema`/`resources`, the more specific list must be a subset of `global` — `ork validate` rejects a `resources` list that grants an operation `global` doesn't. Setting only `global` applies it to every scope uniformly, as in `security-audit` above.

---

## Denial response

A denied request gets `403`, not a silent drop or a generic `401`:

```json
{
  "error": "permission denied",
  "message": "token \"ci-pipeline\" lacks \"delete\" permission on \"App\""
}
```

The three denial reasons — unknown token (not in `serve.tokens` at all), namespace not permitted for this token, operation not granted — each produce a distinct message, so a caller (or whoever's debugging their pipeline) can tell which rule fired without cross-referencing the Katalog by hand.

---

## Validation

`ork validate` checks, at load time, before anything touches a cluster:

- Every token name in `serve.tokens` exists in `gateway.api.auth.tokens` **or** as a `gateway.webhooks` entry's own `name` — a typo'd token name is a hard error, not a silently-never-matching rule
- Every operation string is a valid `ServeOperation` (`get`, `list`, `create`, `update`, `delete`, `*`)
- No permission list repeats the same operation
- `schema` permissions contain only `get`/`list`
- `resources`/`schema` are subsets of `global` when `global` is also set
- Every namespace under a token's `namespaces` is inside the CRD's `allowedNamespaces` (when set) and outside `restrictedNamespaces` — a token can't be granted access to a namespace the CRD itself doesn't allow
- (Warning) A token entry with no permissions declared grants no access — probably a mistake, not rejected outright
- (Warning) Namespace restrictions on a cluster-scoped CRD are ignored — there's no namespace to restrict

---

## Testing tokens locally

`ork token` lets you work with `gateway.api.auth.tokens` entries without a running cluster.

```bash
# see all configured entries
ork token list

# verify a JWT locally — fetches real JWKS, checks claims
ork token verify -t token.jwt

# probe the OIDC discovery endpoint for a named entry
ork token probe -n vault-ci

# test against a live gateway (via ork proxy)
ork token verify --api http://localhost:8443 -t token.jwt
```

`ork token verify` local mode calls the same signature verification and claim-matching logic the gateway uses at request time. A token that passes locally will pass at the gateway.

→ [ork token reference](../reference/cli/15-token.md)

---

## Where to go next

- **[Namespace protection](05-namespace-protection.md)** — the CRD-level layer this sits on top of
- **[Webhook credential verification](09-webhook-verification.md)** — how GitHub/GitLab/Slack/generic entries prove they're genuine
- **[Gateway API reference](../reference/schema/02-katalog/17-gateway-api.md#servetokens--fine-grained-permissions)** — full `serve.tokens` field reference
- **[The Serve Concept](../concepts/self-service/)** — what the Gateway API is for, target mode
- **[Aliases and Intent Provenance](../concepts/self-service/04-aliases-and-provenance.md)** — per-alias token scoping, admission gating, reconcile routing
- **[Webhook Intake](../concepts/self-service/09-webhook-intake.md)** — GitOps push delivery through the same serve.tokens model
