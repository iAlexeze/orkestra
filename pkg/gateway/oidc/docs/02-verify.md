# 02 — JWT verification

## What a JWT is

A JWT (JSON Web Token) is three base64url-encoded segments separated by dots:

```
header.payload.signature
```

- **Header** — algorithm (`alg`) and key ID (`kid`)
- **Payload** — claims: `iss`, `sub`, `aud`, `exp`, plus provider-specific fields like `repository` or `environment`
- **Signature** — the header+payload signed with the provider's private key

Anyone can decode the header and payload — they are not encrypted. The signature is what makes the token trustworthy: only the holder of the private key can produce a valid one, and anyone with the matching public key can verify it. The JWKS contains those public keys.

## Verification steps

`cache.Verify(issuerURL, token, audience)` runs these steps in order. A failure at any step returns an error and the claims map is nil.

### 1. Fetch JWKS

```go
ks, err := c.keys(issuerURL)
```

Returns the cached key set for this issuer, fetching if expired. See [01-cache.md](01-cache.md).

### 2. Parse the JWT

```go
sig, err := jose.ParseSigned(token, supportedAlgs)
```

`ParseSigned` validates the JWT structure and checks that the `alg` header is in the allowed list (`RS256`, `RS384`, `RS512`, `ES256`, `ES384`, `ES512`). It does **not** verify the signature here — it only parses. If the token is malformed or uses an unsupported algorithm, this returns an error before touching the keys.

Restricting `supportedAlgs` matters: the classic `alg: none` attack sets the algorithm to `none` to bypass signature verification entirely. `ParseSigned` rejects any algorithm not in the list.

### 3. Look up the signing key

```go
kid := sig.Signatures[0].Header.KeyID
key, err := lookupKey(ks, kid)
```

The JWT header contains a `kid` that names which key in the JWKS was used to sign it. `lookupKey` finds that key. If the kid is absent (some providers omit it) and the JWKS contains exactly one key, that key is used. If the kid is not found — or there is no kid and more than one key — verification fails.

This is the right failure mode: a token signed with a key the gateway has never seen is not verifiable, regardless of what the payload claims.

### 4. Verify the signature

```go
payload, err := sig.Verify(key)
```

`Verify` checks the cryptographic signature. It uses the public key retrieved in step 3 to recompute what the signature should be for this header+payload combination. If the token was tampered with — even a single bit changed — or was signed with a different key, this fails.

The raw `payload` is the decoded JSON bytes of the claims.

### 5. Check standard claims

```go
err := checkStandardClaims(raw, issuerURL, audience)
```

Three checks, in order:

**`exp`** — the expiry timestamp. A token that has expired is not accepted regardless of a valid signature. The token could have been stolen before it expired; rejection here limits the window of abuse.

**`iss`** — the issuer. Must equal `issuerURL` exactly. This prevents a valid token from a different provider being accepted. If someone obtains a valid GitHub token for an unrelated repo, the `iss` check ensures it cannot be used against a gateway configured for an internal IdP.

**`aud`** — the audience. Only checked when `audience` is non-empty in the Katalog config. Must contain the configured audience string (the claim can be a single string or an array). Audience scoping prevents a valid token issued for one service from being replayed against another.

### 6. Return claims

```go
return flattenClaims(raw), nil
```

Every scalar claim (string, number, boolean) in the payload is returned as a `map[string]string`. Arrays and objects (like the `cnf` claim) are silently skipped — they are not used in claim matching.

Numbers are formatted without trailing zeros: `1234` not `1234.000000`. Booleans are `"true"` / `"false"`.

## Claim matching

The returned map is passed to `APIToken.MatchesOIDCClaims`:

```go
token.MatchesOIDCClaims(claims)
```

This checks every field declared in the token entry's `allow` block against the verified claims. All declared fields must match — unset fields in `allow` are not checked. A `github-payments-ci` entry that declares:

```yaml
allow:
  repository: myorg/payments
  ref: refs/heads/main
```

will only match a token where `claims["repository"] == "myorg/payments"` AND `claims["ref"] == "refs/heads/main"`. A token from `myorg/payments` on a feature branch (`ref: refs/heads/feature-x`) does not match and is rejected.

## Supported algorithms

```go
var supportedAlgs = []jose.SignatureAlgorithm{
    jose.RS256, jose.RS384, jose.RS512,
    jose.ES256, jose.ES384, jose.ES512,
}
```

RSA and ECDSA family only. GitHub uses RS256. `HS256` (HMAC) is intentionally excluded — HMAC-signed JWTs use a shared secret, not a public/private key pair, which is incompatible with the JWKS verification model.

→ Back: [01-cache.md](01-cache.md)  
→ How this wires into the gateway: [../../api/docs/05-auth.md](../../api/docs/05-auth.md)
