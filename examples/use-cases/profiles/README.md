# Profiles Examples

Five examples showing Orkestra's named presets. Each profile expands at Katalog load time into fully-formed resource, security, probe, or rollout configuration — the runtime never sees a profile name.

| Example | What it teaches |
|---|---|
| [01 — Resource](01-resource/README.md) | `resources.profile` — CPU and memory presets from `tiny` to `memory-heavy` |
| [02 — Security](02-security/README.md) | `securityContext.profile` + `podSecurity.profile` — `baseline`, `restricted`, `hardened` |
| [03 — Probes](03-probes/README.md) | `probes.liveness.profile` — `fast`, `standard`, `patient`, `slow-start` timing presets |
| [04 — Rolling Update](04-rolling-update/README.md) | `rollingUpdate.profile` — `safe`, `fast`, `blue-green` rollout strategies |
| [05 — PDB](05-pdb/README.md) | `pdb.behavior.profile` — `zero-downtime`, `rolling`, `relaxed` disruption budgets |

All five share one CRD (`crd.yaml`) and one CR (`cr.yaml`) at this directory level.

For autoscale profiles (`autoscale.profile`):

```bash
ork init --pack advanced
cd 12-autoscale
```

---

**Further reading:** [Profiles concept doc](../../../documentation/concepts/operatorbox/06-profiles/index.md) · [Full profile reference](../../../pkg/profiles/docs/01-profiles.md)
