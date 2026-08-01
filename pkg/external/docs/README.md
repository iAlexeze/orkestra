# External — Developer Documentation

`pkg/external` executes declarative external calls and injects their results into the resolver context under `.external.<name>`. It runs at two points in the request lifecycle: at reconcile time (from `runTemplateReconcile`) and at admission time (from the gateway webhook).

## Documents

| File | What it covers |
|------|----------------|
| [01-architecture.md](01-architecture.md) | The `Run()` pipeline — ordering, cache check, protocol dispatch, result injection |
| [02-result-maps.md](02-result-maps.md) | What each protocol puts in the result map and how to reference it in templates |
| [03-auth.md](03-auth.md) | Auth resolution — `secretRef`, `env`, credential threading |
| [04-cache.md](04-cache.md) | `cacheFor:` mechanics — key derivation, TTL, eviction |
| [05-adding-a-protocol.md](05-adding-a-protocol.md) | Step-by-step guide for adding a new protocol client |

Read them in order the first time. For adding a new protocol, jump straight to [05-adding-a-protocol.md](05-adding-a-protocol.md).
