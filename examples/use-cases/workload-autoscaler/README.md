# Workload Autoscaler

Orkestra collapses KEDA, VPA, and custom scaling controllers into one field on a Deployment declaration. The same condition engine that controls resource creation controls scaling — no new vocabulary, no new controllers, no plugins.

| Example | Signal | What you learn |
|---------|--------|----------------|
| [01-time-based](./01-time-based/README.md) | Time conditions | `autoscale:` with `target` jump scaling; business hours scale-up/down |
| [02-external-api](./02-external-api/README.md) | External HTTP endpoint | `external:` feeds live queue depth into `autoscale:` conditions; `increment`/`decrement` step scaling; flip load mid-test |
| [03-cross-operator](./03-cross-operator/README.md) | Sibling operator status | `cross:` reads a sibling CRD's queue depth in-memory; workers scale with the producer's load |

---

## Run all examples

```bash
ork simulate
```

```bash
ork run
```

```bash
ork e2e
```
