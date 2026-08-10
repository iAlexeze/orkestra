# 01 — Overview

## Push vs pull

`ork serve apply` and a direct `POST /api/v1/apply` call are pull-based: something with a token — a person, a CI job — decides to send one intent, at a moment of its choosing. `intake` is push-based: the gateway sits listening, and an external system decides when to trigger the apply. A commit lands on a watched branch. Someone types a slash command. PagerDuty fires an incident.

```
kubectl apply           ↓
POST /api/v1/apply      ↓
ork serve apply         ↓  →  CR in Kubernetes  →  Runtime reconciles
GitHub/GitLab push      ↓
Slack slash command     ↓
Generic JSON webhook    ↓
```

From the runtime's perspective, nothing changes — a CR arrived, it reconciles. From the gateway's perspective, `intake` is one more way a flat field map reaches `BuildCRFromTarget`, no different in kind from a browser form submission or a curl command.

## Why a separate package

`intake` imports `pkg/gateway/api` directly — `api.ApplyTargetFields` is the entire point of every handler here, and `api.ResolveSecretRef` gives every webhook credential the same self-bootstrap/rotation behavior `gateway.api.auth.tokens` gets, for free.

That import direction is fixed. `api` cannot import `intake` back — Go would reject the cycle, and even without the language rule, `api` shouldn't need to know intake exists. So `intake.Server` (`server.go`) is not a field on `api.APIServer`; it's a second, independent server object, resolved and registered by the same caller that builds `api.APIServer`:

```go
// cmd/internal/gateway.go
api, _      := apigateway.NewAPIServer(ctx, kat, kube, ns)
intakeSrv, _ := intake.NewIntakeServer(ctx, kat, kube, ns)

api.Register(hs)
if intakeSrv != nil {
    intakeSrv.Register(hs, kat.Notes)
}

ws.SetTokenReloader(func(ctx) error {
    if err := api.ReloadTokens(ctx); err != nil {
        return err
    }
    if intakeSrv != nil {
        return intakeSrv.Reload(ctx)
    }
    return nil
})
```

Composition over nesting. Two servers, one reload cycle, no cycle in the import graph.

## What `intake.Server` owns

`NewIntakeServer` resolves every enabled entry's credentials up front — self-bootstrapping missing Secrets, same as `api.NewAPIServer` does for `gateway.api.auth.tokens`. `Server.Register` wires each entry's `Path` onto the shared mux **without** the Bearer-token `auth()` middleware `api.Register` wraps its own routes in — a webhook sender doesn't send `Authorization: Bearer <token>`, it signs its own payload on its own terms. Verification happens inside each handler, not in shared middleware, because the four schemes genuinely differ (see [02-verification.md](02-verification.md)).

`Server.Reload` re-resolves every entry's credentials — the same rotation check `pkg/runners` uses for operator-managed secrets — and swaps the in-memory `Set` atomically under a mutex, so a reload never serves a half-updated credential set to a concurrent request.

## The tail every handler shares

Regardless of source, every handler ends the same way:

```go
resp, status := api.ApplyTargetFields(ctx, kube, kat, notes, src.Config.Name, fields, false)
```

`src.Config.Name` — the entry's own declared name — is the `tokenName` argument. It does double duty inside `ApplyTargetFields`: it's checked against `serve.tokens` for permission, and it's stamped as the `serve-source` provenance annotation on the applied CR, the same slot an OIDC caller's verified `sub` claim fills for a bearer-token request. No separate `gateway.api.auth.tokens` entry is needed for a webhook source to be authorized — its own name is a first-class identity, enforced the same way at `ork validate` time (`validateServeTokenRestrictions` in `pkg/katalog`, extended to check `GatewayWebhookEntryNames()` alongside `GatewayTokenNames()`).

→ Next: [02-verification.md](02-verification.md)
