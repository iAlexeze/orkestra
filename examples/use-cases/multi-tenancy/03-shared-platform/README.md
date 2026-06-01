# Multi-tenancy 03 — Shared platform

Platform team publishes shared infrastructure CRDs (cache, queue). Application teams read their status via `cross:` and wire their deployments to the shared services.

- `platform/katalog.yaml` — `namespace: platform`, manages `cache` and `queue` CRDs
- `team-a/katalog.yaml` — `namespace: team-a`, reads cache and queue endpoints to configure its API deployment

The `team-a/api` Deployment is only created when both `sharedCache.found` and `sharedQueue.found` are `"true"`.

## Try it

```bash
ork init my-project --pack use-cases/multi-tenancy/03-shared-platform
cd my-project/03-shared-platform
ork run -f komposer.yaml
ork control
```
