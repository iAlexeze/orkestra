# Multi-tenancy 02 — Cross-read access control

Three CRDs, two teams, one runtime. The internal team closes its Katalog to cross reads. One CRD overrides back to open. The analytics team's `cross:` reference to the private CRD returns `found: "false"` — no error, graceful degradation.

- `internal-team/katalog.yaml` — `crossAccess: false`, two CRDs: `payment` (private) and `ledger` (`crossAccess: true` override)
- `analytics-team/katalog.yaml` — reads `ledger` successfully, gates on `payment` being unavailable

## Access matrix

| Reader | payment | ledger |
|---|---|---|
| analytics/report | blocked | allowed |

## Try it

```bash
ork init my-project --pack use-cases/multi-tenancy/02-cross-access-control
cd my-project/02-cross-access-control
ork run -f komposer.yaml
ork control
```
