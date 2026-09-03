# pkg/external

Executes declarative external calls and injects their results into the resolver context under `.external.<name>`. Runs at reconcile time and at admission time (gateway webhook).

## Developer docs

Full internals in [`docs/`](docs/README.md):

| | |
|---|---|
| [01 — Architecture](docs/01-architecture.md) | The `Run()` pipeline, where it sits in reconcile and admission, protocol dispatch |
| [02 — Result Maps](docs/02-result-maps.md) | What each protocol returns and how to reference it in templates |
| [03 — Auth](docs/03-auth.md) | `secretRef`, `env`, credential threading per protocol |
| [04 — Cache](docs/04-cache.md) | `cacheFor:` mechanics, key derivation, TTL, eviction |
| [05 — Adding a Protocol](docs/05-adding-a-protocol.md) | Step-by-step guide with checklist |

## Fixtures

Integration fixtures in [`fixtures/`](fixtures/):

| Fixture | What it tests |
|---------|---------------|
| [`prometheus/`](fixtures/prometheus/) | `protocol: prometheus` — goroutine count, CPU total, threshold notes, cacheFor |
| [`redis/`](fixtures/redis/) | `protocol: redis` — GET, LLEN, password auth |
| [`postgres/`](fixtures/postgres/) | `protocol: postgres` — COUNT query, secretRef auth |
| [`mongo/`](fixtures/mongo/) | `protocol: mongo` — CountDocuments, URI auth |
| [`kafka/`](fixtures/kafka/) | `protocol: kafka` — consumer group lag, topic metadata, SASL/PLAIN |
| [`http-auth/`](fixtures/http-auth/) | `auth:` — secretRef + env, HTTP header injection |
