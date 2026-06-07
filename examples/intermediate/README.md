# Intermediate Pack

You know the basics. Now use more of Orkestra's surface — conditional topologies, Komposer composition, automatic CRD management, and declarative state machines.

```bash
ork init --pack intermediate
```

---

## Examples

| Example | What it teaches |
|---------|-----------------|
| [04 — Multi-Resource with Status](04-multi-resource/README.md) | Three resources from one CR — Deployment, Service, and ConfigMap. Status Layer 3: readyReplicas propagated back from the live Deployment. |
| [05 — When Conditions](05-when-conditions/README.md) | `when:` conditions — same CRD, different resource topology per tier. Free tier deploys only core resources; enterprise tier adds LoadBalancer and monitoring. |
| [06 — Basic Komposer](06-komposer-basic/README.md) | Two Katalogs composed into one runtime. `spec.crds` overrides for per-environment tuning without modifying the base Katalog. |
| [07 — CRD File](07-crd-file/README.md) | `crdFile` in the Katalog — two CRDs declared inline, applied automatically before reconcile loops start. No manual `kubectl apply -f crd.yaml` needed. |
| [08 — Declarative State Machine](08-state-machine/README.md) | A multi-step pipeline operator with `when:` on status fields. Phase transitions, Job creation, and terminal states — no Go, no binary build. |

---

## Running an example

```bash
ork init --pack intermediate
cd intermediate/04-multi-resource
ork run
```

No cluster? Add `--dev` to create a temporary kind cluster.

---

## Simulate

Some examples ship with a `simulate.yaml`. Run simulate for a single example:

```bash
cd 06-komposer-basic && ork simulate
```

Or run the full intermediate suite — all examples that have a simulate.yaml:

```bash
ork simulate -f simulate.yaml
```

No cluster needed. Each run completes in under a second.

---

## E2E

Every example ships with a runnable `e2e.yaml`. Run a single example end-to-end:

```bash
cd 04-multi-resource && ork e2e
```

Or run the full intermediate suite — all five examples in one kind cluster:

```bash
ork e2e -f e2e.yaml
```

The suite spins up a cluster, runs each example in sequence, and tears everything down. Total runtime: ~8 minutes.
