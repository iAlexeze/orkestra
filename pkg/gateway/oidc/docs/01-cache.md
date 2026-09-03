# 01 — JWKS cache

## The problem

JWT signature verification requires the issuer's public key. That key lives at a URL controlled by the identity provider — GitHub, GitLab, or your own IdP. Fetching it on every request would:

- Add a network round trip to every authenticated call
- Fail loudly if the provider is temporarily unreachable
- Create an unnecessary dependency between request throughput and external network latency

The cache fetches keys once and holds them for a configurable TTL (default: 1 hour). The hot path for a verified request is a read from an in-process map — no network.

## JWKS and discovery

A **JWKS** (JSON Web Key Set) is a JSON document containing one or more public keys used to verify JWTs. Each key has a `kid` (key ID) field that JWT headers reference to select the right key for verification:

```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "abc123",
      "use": "sig",
      "alg": "RS256",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

Most OIDC providers publish their JWKS URL inside a **discovery document** at a well-known path:

```
GET {issuer}/.well-known/openid-configuration
→ { "jwks_uri": "https://auth.myorg.io/jwks", ... }

GET https://auth.myorg.io/jwks
→ { "keys": [...] }
```

This two-step flow is the OIDC standard. The discovery document also contains other provider metadata (supported algorithms, token endpoint, etc.) — the cache only needs `jwks_uri`.

## GitHub as a special case

GitHub's JWKS URL is stable and publicly documented:

```
https://token.actions.githubusercontent.com/.well-known/jwks
```

There is no value in hitting the discovery endpoint first — it would just tell us to go to the same URL. The cache short-circuits to fetch directly when the issuer is `https://token.actions.githubusercontent.com`.

All other issuers go through discovery regardless of whether they are "known" to the codebase. GitLab, GCP, Azure, and internal IdPs all follow the OIDC standard — discovery works for all of them without provider-specific code.

## Cache design

```
Cache
  entries  map[string]*cachedKeySet   // keyed by issuer URL
  ttl      time.Duration              // default: 1 hour
  mu       sync.RWMutex

cachedKeySet
  keys      jose.JSONWebKeySet
  fetchedAt time.Time
```

Read path (hot):

```
RLock
  entry = entries[issuer]
  if entry exists and age < TTL:
    RUnlock → return cached keys (no network)
RUnlock
```

Miss or expiry path:

```
resolveJWKSURL(issuer)           ← skip discovery for GitHub; OIDC discovery otherwise
fetchJWKS(jwksURL)               ← one HTTP GET
Lock
  entries[issuer] = { keys, fetchedAt: now }
Unlock
→ return keys
```

The fetch happens outside the write lock — if two goroutines both miss the cache simultaneously they will both fetch. The second write wins. This is intentional: a double-fetch on cold start is cheaper than holding a write lock during a network call. In practice the double-fetch only happens at startup or after TTL expiry.

## TTL and key rotation

`DefaultTTL = 1 hour`. Providers rotate their signing keys rarely — GitHub's have been stable for extended periods. One hour gives a comfortable margin between rotation and re-fetch while keeping the window short enough that a revoked key won't be trusted indefinitely.

If a provider rotates mid-TTL, the first request after the new key is used will fail verification — `lookupKey` returns "no key with kid X in JWKS". The cache then re-fetches (because the failed `lookupKey` causes `Verify` to return an error, not a cache hit), picks up the new key, and subsequent requests succeed. Callers that see the transient error should retry — the gateway's apply endpoint is safe to retry.

A shorter TTL (e.g. `5 * time.Minute`) can be set for testing or for providers with aggressive rotation policies:

```go
cache := oidc.NewCache(5 * time.Minute)
```

→ Next: [02-verify.md](02-verify.md)
