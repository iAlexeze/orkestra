# Profiles Examples

Nine examples showing Orkestra's named presets and user-defined profiles. Each profile expands at Katalog load time into fully-formed configuration — the runtime never sees a profile name.

| Example | What it teaches |
|---|---|
| [01 — Resource](01-resource/README.md) | `resources.profile` — CPU and memory presets from `tiny` to `memory-heavy` |
| [02 — Security](02-security/README.md) | `securityContext.profile` + `podSecurity.profile` — `baseline`, `restricted`, `hardened` |
| [03 — Probes](03-probes/README.md) | `probes.liveness.profile` — `fast`, `standard`, `patient`, `slow-start` timing presets |
| [04 — Rolling Update](04-rolling-update/README.md) | `rollingUpdate.profile` — `safe`, `fast`, `blue-green` rollout strategies |
| [05 — PDB](05-pdb/README.md) | `pdb.behavior.profile` — `zero-downtime`, `rolling`, `relaxed` disruption budgets |
| [06 — NetworkPolicy](06-networkpolicy/README.md) | `networkPolicies.profile` — `deny-all`, `deny-all-ingress`, `allow-dns-egress` and more |
| [07 — ResourceQuota](07-resourcequota/README.md) | `resourceQuotas.profile` — `small`, `medium`, `large`, `xlarge` tier presets |
| [08 — LimitRange](08-limitrange/README.md) | `limitRanges.profile` — user-defined presets; LimitRange has no built-ins |
| [09 — User-Defined](09-user-defined/README.md) | `profiles:` block — declare your own names for any class; `ork validate` enforces every reference |

All nine share one CRD (`crd.yaml`) and one CR (`cr.yaml`) at this directory level.

For autoscale profiles (`autoscale.profile`):

```bash
ork init --pack advanced
cd 12-autoscale
```

---

**Further reading:** [Profiles concept doc](https://orkestra.sh/docs/concepts/profiles/) · [Full profile reference](https://github.com/orkspace/orkestra/blob/main/pkg/profiles/docs/01-profiles.md)

---

## Simulate (no cluster needed)

```bash
ork simulate
```

This runs [simulate.yaml](./simulate.yaml), which chains all nine sub-examples.

To run a single example:

```bash
cd 06-networkpolicy && ork simulate
```

---

## E2E

Run the full suite — all nine profile examples in one command:

```bash
ork e2e -f e2e.yaml
```

This runs [e2e.yaml](./e2e.yaml), which imports each sub-example's `e2e.yaml` and runs them sequentially in the same cluster. To run a single example:

```bash
cd 01-resource && ork e2e
```
