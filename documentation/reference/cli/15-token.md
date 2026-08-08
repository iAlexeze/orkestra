# ork token

Inspect and verify tokens configured in `gateway.api.auth.tokens` — no cluster required.

```bash
ork token list
ork token verify -t token.jwt
ork token probe -n vault-ci
```

`--file` / `-f` defaults to `katalog.yaml` in all three subcommands.

---

## ork token list

Print all token entries in the katalog as a table.

```bash
ork token list
ork token list -f my-katalog.yaml
```

**Output**

```text
token list  6 entries

NAME           TYPE     PROVIDER    ALLOW
gh-ci          oidc     github      repository=myorg/payments ref=refs/heads/main
gl-ci          oidc     gitlab      namespacePath=mygroup/infra refProtected=true
vault-ci       oidc     vault       url=https://vault.myorg.io entityName=ci-agent
internal-ci    oidc     generic     issuer=https://auth.myorg.io sub=system:serviceaccount:ci:runner
static-token   static   token       (env var)
ci-pipeline    static   secretRef   rotateAfter=90d
```

---

## ork token verify

Verify a JWT against the configured token entries.

```bash
ork token verify
ork token verify -t token.jwt
ork token verify -f katalog.yaml -t my-token.jwt
```

**Flags**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | `katalog.yaml` | Katalog file |
| `--token` | `-t` | `token.jwt` | File containing the JWT |
| `--api` | | | Gateway base URL — enables live mode |
| `--audience` | | | Override audience check |

### Local mode (default)

Loads the katalog, extracts the `iss` claim from the JWT without verifying it, finds all entries whose issuer matches, then for each candidate:

1. Fetches JWKS from the real provider
2. Verifies the token signature and expiry
3. Checks the `allow` block against the verified claims

Reports which entry matched and prints the full claims table.

```text
  token verify  local

  token file  token.jwt
  issuer      https://vault.myorg.org/v1/identity/oidc
  candidates  1

  ────────────────────────────────────────────────────
  vault-ci  vault

  ✓ signature valid
  ✓ not expired
  ✓ issuer matched
  ✓ claims matched

    entity_id    6725bbdf-5e69-8d38-7ad8-c3df002de1da
    entity_name  ci-agent
    iss          https://vault.myorg.org/v1/identity/oidc
    namespace    root
    sub          6725bbdf-5e69-8d38-7ad8-c3df002de1da

  ────────────────────────────────────────────────────
  ✓ matched: vault-ci
```

### Live mode (`--api`)

Sends the JWT as `Authorization: Bearer <token>` to `GET /api/v1/schema` on a running gateway and reports the response. Use `ork proxy` to expose the gateway locally first.

```bash
ork proxy &
ork token verify --api http://localhost:8443 -t token.jwt
```

```text
  token verify  live

  gateway  http://localhost:8443
  issuer   https://vault.myorg.org/v1/identity/oidc

  ✓ token accepted
```

---

## ork token probe

Probe the OIDC discovery endpoint and JWKS for a named token entry.

```bash
ork token probe -n vault-ci
ork token probe -f katalog.yaml -n gh-ci
```

**Flags**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | `katalog.yaml` | Katalog file |
| `--name` | `-n` | | Token entry name (required) |

Fetches the discovery document and JWKS, then reports reachability, `jwks_uri`, key count, and signing algorithms.

```text
  token probe  vault-ci

  kind       vault
  issuer     https://vault.myorg.io/v1/identity/oidc
  discovery  https://vault.myorg.io/v1/identity/oidc/.well-known/openid-configuration

  ✓ discovery reachable
  jwks_uri   https://vault.myorg.io/v1/identity/oidc/.well-known/keys

  ✓ JWKS reachable
  keys       3  (RS256, RS256, RS256)
```

Useful for confirming that a provider endpoint is reachable before deploying — especially Vault, which uses a non-standard discovery path.

---

## Related

- [`ork serve tokens`](13-serve.md) — show CRD-level token map and effective alias permissions
- [`ork serve can-i`](13-serve.md) — live permission check for a token against a target
- [Token scoping concept](../../concepts/idp/03-token-scoping.md)
- [Serve token permissions](../../security/08-serve-permissions.md)
