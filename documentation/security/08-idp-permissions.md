# IDP token permissions

`allowedNamespaces`/`restrictedNamespaces` answer one question: which namespaces does this CRD exist in at all. Every caller gets the same answer — it's a property of the CRD, not of who's asking. `idp.allowedTokens` answers a different question: which caller can do what, and where. Two tokens against the same CRD can have different answers — a `ci-pipeline` token allowed to create in `staging` but not touch `production`; a `security-audit` token that's read-only everywhere. This is authorization scoped to the caller's identity, layered on top of namespace protection, not a replacement for it.

---

## Two independent layers

### CRD-level — topology, same for everyone

`allowedNamespaces`/`restrictedNamespaces` on the CRD entry govern which namespaces the runtime's informer watches and the admission webhook accepts at all. See [Namespace protection](05-namespace-protection.md). Every caller — every token, `kubectl`, the reconciler itself — sees the same boundary.

### `idp.allowedTokens` — identity, per caller

Declared under the CRD's `idp` block, checked only by the Apply API — a request that never goes through `POST /api/v1/apply`, `GET /api/v1/resources/...`, or `GET /api/v1/schema` never hits this layer at all. It answers, for the token that authenticated this specific request: which operations, on which endpoint class, in which namespaces.

```yaml
idp:
  enabled: true
  target: app
  allowedTokens:
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

A CRD with no `idp.allowedTokens` block places no restriction here — any token valid at the gateway level (`gateway.applyAPI.auth.tokens`) can call any endpoint the Apply API exposes for that CRD, subject only to namespace protection.

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

The three denial reasons — unknown token (not in `idp.allowedTokens` at all), namespace not permitted for this token, operation not granted — each produce a distinct message, so a caller (or whoever's debugging their pipeline) can tell which rule fired without cross-referencing the Katalog by hand.

---

## Validation

`ork validate` checks, at load time, before anything touches a cluster:

- Every token name in `idp.allowedTokens` exists in `gateway.applyAPI.auth.tokens` — a typo'd token name is a hard error, not a silently-never-matching rule
- Every operation string is a valid `IDPOperation` (`get`, `list`, `create`, `update`, `delete`, `*`)
- No permission list repeats the same operation
- `schema` permissions contain only `get`/`list`
- `resources`/`schema` are subsets of `global` when `global` is also set
- Every namespace under a token's `namespaces` is inside the CRD's `allowedNamespaces` (when set) and outside `restrictedNamespaces` — a token can't be granted access to a namespace the CRD itself doesn't allow
- (Warning) A token entry with no permissions declared grants no access — probably a mistake, not rejected outright
- (Warning) Namespace restrictions on a cluster-scoped CRD are ignored — there's no namespace to restrict

---

## Where to go next

- **[Namespace protection](05-namespace-protection.md)** — the CRD-level layer this sits on top of
- **[Apply API reference](../reference/schema/02-katalog/17-katalog-applyapi.md#idpallowedtokens--fine-grained-permissions)** — full `idp.allowedTokens` field reference
- **[The IDP Concept](../concepts/idp/)** — what the Apply API is for, target mode
