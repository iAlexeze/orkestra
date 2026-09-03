# Learning to Orkestrate

Every Orkestra capability is demonstrated in a runnable example. This page is the map. Whether you are writing your first operator or converting an existing Go controller, start here.

---

## Two commands to your first operator

```bash
ork init
ork run
```

No cluster? Add `--dev` to create a temporary kind cluster. Requires Docker.

---

## Packs

Examples are grouped into packs. Pull any pack with `ork init --pack <name>`.

| Pack | Focus |
|---|---|
| [Beginner](./01-beginner.md) | Foundation — CRDs, templates, status, owner references |
| [Intermediate](./02-intermediate.md) | `when:`, Komposer basics, state machines, CRD files |
| [Advanced](./03-advanced.md) | Admission, composition, escape hatches, autoscaling, cross-operator |
| [Use-cases](./04-use-cases.md) | Normalize, enrich, profiles, full-stack patterns, external calls, motif composition |
| [Security](./05-security.md) | Admission, deletion protection, namespace isolation |
| [Resilience](./09-resilience.md) | Panic recovery, runtime admission protection, degraded state, operator isolation |
| [Registry Guide](./06-registry.md) | Distribution — publish, version, gate, consume, and automate the full pattern lifecycle |
| [Migration Guide](./07-migration.md) | Migrating an existing controller-runtime operator to Orkestra — five options, zero lock-in |
| [Ecosystem Guide](./08-ecosystem.md) | Wrapping ArgoCD, cert-manager, Prometheus, and Crossplane with Orkestra abstraction layers |

---

## Running any example

Every example follows the same pattern:

```bash
ork init --pack <pack>
cd <pack>/<example>
ork run           # defaults to katalog.yaml, then komposer.yaml
ork control       # localhost:8081 · orkestra / orkestra
```

Pass `-f <file>` when the Katalog or Komposer uses a non-default filename. `ork e2e` defaults to `e2e.yaml`.

---

## E2E test suites

Every example ships with a runnable `e2e.yaml`. Every pack ships with a root `e2e.yaml` that imports all sub-examples.

```bash
# Run a single example end-to-end
cd beginner/01-hello-website && ork e2e

# Run an entire pack in one command
ork e2e -f beginner/e2e.yaml
ork e2e -f intermediate/e2e.yaml
ork e2e -f security/e2e.yaml
```

Each suite creates a kind cluster, runs all examples in sequence, and tears everything down. No setup required beyond the `ork` CLI and Docker.

Each example has its own `README.md`. Follow it — it tells you exactly what to apply, what to observe, and what the expected output is. The README is the guide; the YAML files are the demonstration.

---

## Which example to start with

**New to Kubernetes operators**
- Read [Kubernetes, operators, and why Orkestra](../00-kubernetes-basics.md) first — CRDs, the reconcile loop, and where Orkestra fits.
- Then [`beginner/01-hello-website`](./01-beginner.md) — the mental model it builds is the foundation for everything else.

**Know Kubernetes, new to Orkestra**
- [`beginner/01-hello-website`](./01-beginner.md), then [`intermediate/05-when-conditions`](./02-intermediate.md) and [`advanced/07-validation-mutation`](./03-advanced.md)

**Migrating from controller-runtime, Kubebuilder, or Operator SDK**
- [Migration Guide](./07-migration.md) — the full migration pack; see all five options before choosing one
- `ork migrate <file>` to automate the constructor path for an existing reconciler
- [`advanced/09-hooks`](./03-advanced.md) to wrap existing Go logic
- [`advanced/10-constructor`](./03-advanced.md) to bring a full reconciler across intact

**Building a platform**
- [`advanced/08-komposer-registry`](./03-advanced.md) and [`advanced/13-dependencies`](./03-advanced.md)
- Then [`use-cases/full-stack-app/06-full-stack`](./04-use-cases.md) for how the patterns compose
- [`use-cases/namespace-provisioner`](./04-use-cases.md) — tenant namespaces, RBAC, quotas, and network policies from a single CRD
- [`ecosystem-composition`](./08-ecosystem.md) — add an Orkestra abstraction layer over ArgoCD, cert-manager, Prometheus, and Crossplane

**Supply chain and policy enforcement**
- [`use-cases/external/03-image-signing`](./04-use-cases.md) → [`07-vault-secret-gate`](./04-use-cases.md) → [`08-opa-policy`](./04-use-cases.md)
- Then [`10-motif-composition`](./04-use-cases.md) to see them composed as reusable motifs

**Enforcing org policy without Go**
- [`use-cases/external/08-opa-policy`](./04-use-cases.md) for runtime decisions
- [`advanced/07-validation-mutation`](./03-advanced.md) for admission-time enforcement

**Status propagation and drift correction**
- [`advanced/16-custom-resources`](./03-advanced.md) sub-examples 02 and 04

**Cross-operator data sharing**
- [`advanced/14-cross-operator/01-in-binary`](./03-advanced.md)

**Autoscaling workers**
- [`advanced/12-autoscale/02-based-on-own-metrics`](./03-advanced.md)

**Publishing and distributing operators**
- Start with [`registry-guide/00-consume`](./06-registry.md) — pull a proven pattern and deploy it before building your own
- Then [`registry-guide/02-katalog-api`](./06-registry.md) through [`04-katalog-platform`](./06-registry.md) to build the publish → gate → distribute pipeline
- [`registry-guide/05-komposer`](./06-registry.md) for production deployment with admission and deletion protection
- [`registry-guide/12-ork-action`](./06-registry.md) to automate everything in CI

**Supply chain verification**
- [`registry-guide/08-bad-actor`](./06-registry.md) — simulate passed, `ork plan` caught what the assertions missed

---

## Available packs

```bash
ork init --list
```

---

## Further reading

- **[Orkestra Core](../../orkestra-core/index.md)** — runtime, gateway, and control center architecture
- **[Security](../../security/index.md)** — admission control, RBAC, namespace protection
- **[Reference](../../reference/index.md)** — full schema and CLI reference
