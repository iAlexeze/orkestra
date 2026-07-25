# redis

Verifies `protocol: redis` using a Redis instance queried from `spec.redisURL`.

Two calls are declared:

- `ping` — `PING` command; asserts connectivity. `result` is `"PONG"` on success.
- `dbSize` — `DBSIZE` command; returns key count. Has `cacheFor: 10s`.

## Local run (docker-compose)

```sh
docker compose up -d
ork run -f katalog.yaml
```

The `cr.yaml` used for `ork run` should point to `redis://localhost:6379`. Edit `spec.redisURL` accordingly or apply a local override.

## E2e

Deploys a Redis instance as a Deployment in `orkestra-system` before the CR is applied. The CR uses `redis://redis.orkestra-system.svc:6379`.

```sh
ork e2e pkg/external/fixtures/redis/e2e.yaml
```
