# redis

Verifies `protocol: redis` using a Redis instance queried from `spec.redisURL`.

Two calls are declared:

- `ping` — `PING` command; asserts connectivity. `result` is `"PONG"` on success.
- `dbSize` — `DBSIZE` command; returns key count. Has `cacheFor: 10s`.

## Local run (docker-compose)

```sh
docker compose -f pkg/external/fixtures/redis/docker-compose.yaml up -d
ork run -f pkg/external/fixtures/redis/katalog.yaml
```

## Check the Status

```sh
kubectl get webapp my-app -oyaml
```

## E2e

Deploys a Redis instance as a Deployment in `orkestra-system` before the CR is applied. The CR uses `redis://redis.orkestra-system.svc:6379`.

```sh
ork e2e pkg/external/fixtures/redis/e2e.yaml
```
