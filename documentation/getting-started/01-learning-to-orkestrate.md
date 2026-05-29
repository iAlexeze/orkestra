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

## Beginner pack

The foundation. Every concept here appears in every more advanced example. Work through these in order.

```bash
ork init --pack beginner
```

| Example | What it teaches |
|---|---|
| `01-hello-website` | Your first operator. CRD declaration, Katalog template expressions, `reconcile: true`, owner references, status fields. The mental model everything else builds on. |
| `02-with-serviceaccount` | Three resources from one CR. ServiceAccount wiring, pod identity, reading live cluster state into status via Notes. |
| `03-secret-copy` | Built-in Kubernetes resource management. A Secret distribution operator: copies a Secret from a platform namespace into every team namespace. `fromSecret`, `toNamespaces`, `reconcile: true` keeping copies in sync. |
| `03b-configmap-copy` | Same pattern as 03 applied to ConfigMaps. Statusless resource distribution. Good companion to 03. |

---

## Advanced pack

Every production operator pattern. Some examples have sub-examples showing the same concept across different deployment topologies (in-binary, cross-binary, cross-cluster).

```bash
ork init --pack advanced
cd advanced/09-hooks
ork run
# follow the README inside each example — they each show what to observe
```

### Rules and lifecycle

| Example | What it teaches |
|---|---|
| `07-validation-mutation` | Admission-time policy without a webhook server. `deny` and `warn` rules. `default` and `override` mutation. `mutateFirst`. Two enforcement boundaries: admission and reconcile, from one declaration. |
| `16-custom-resources` | The full resource lifecycle in one place. Seven sub-examples: single child, status propagation, conditional children with `when:`, drift correction with `reconcile: true`, `forEach` sharding across a list, full platform composition, multi-CRD pipeline. |

### Composition

| Example | What it teaches |
|---|---|
| `08-komposer-registry` | Production Komposer pulling from OCI registry, local Katalog, and Helm chart sources simultaneously. Version pinning, multi-source composition, environment-specific overrides. |
| `13-dependencies` | Startup ordering with `dependsOn`. Three sub-examples: in-binary (same Orkestra process), cross-binary (different processes), cross-cluster. The `healthy` vs `started` condition difference. |
| `14-cross-operator` | CRDs reading each other's live state. Two operators sharing data via the informer cache — zero API server calls. Three sub-examples: in-binary, cross-binary, cross-cluster. |
| `17-apitype-override` | White-label operator pattern. Import a vendor Katalog and override only `apiTypes` — your API group, your CRD identity, identical reconcile logic. |
| `18-crd-file-komposer` | Combine `apiTypes` override and `crdFile` in one Komposer. Fully self-contained: hand it off and `ork run` handles the rest. |

### Escape hatches

| Example | What it teaches |
|---|---|
| `09-hooks` | When the declarative layer is not enough. Typed Go hooks called at `OnReconcile` and `OnDelete`. You write the function; Orkestra calls it at the right point. The runtime still manages informers, workqueue, health, metrics, and events. |
| `10-constructor` | Full ownership of the reconcile loop. Replace the GenericReconciler entirely. For migrating existing Go controllers or building custom state machines. Orkestra still manages informers, workqueue, and workers. |
| `11-mixed-operator-patterm` | All three patterns in one Komposer: pure declarative, hooks, and constructor operators running side-by-side in the same runtime. |

### Operations

| Example | What it teaches |
|---|---|
| `12-autoscale` | Dynamic worker and resync scaling based on metrics. Five sub-examples: no-autoscaler baseline, own metrics, sibling operator metrics (in-binary), sibling operator metrics (cross-cluster), external metrics source. |

### Tooling

| Example | What it teaches |
|---|---|
| `15-any-language` | Orkestra only cares about the final Katalog YAML. Generate it from Python, Go, or Node.js — three language examples producing identical output. |

---

## Running any example

Every example follows the same pattern:

```bash
# Pull the pack
ork init --pack beginner

# Enter an example
cd beginner/01-hello-website

# Run — Orkestra applies the CRD, applies cr.yaml, starts the operator
ork run

# Open the Control Center in another terminal
ork control
# → localhost:8081 · username:password → orkestra
```

---

## Which example to start with

**New to Kubernetes operators** — start with `beginner/01-hello-website`. The mental model it builds is the foundation for everything else.

**Know Kubernetes, new to Orkestra** — start with `beginner/01`, then jump to `advanced/07-validation-mutation`. The declarative model will be familiar; the depth may surprise you.

**Migrating from Kubebuilder or Operator SDK** — start with `advanced/09-hooks` to wrap existing Go logic, or `advanced/10-constructor` to bring a full reconciler across intact.

**Building a platform** — start with `advanced/08-komposer-registry` and `advanced/13-dependencies`. These are the composition patterns platform teams actually use.

**Want status propagation and drift correction** — `advanced/16-custom-resources` sub-examples 02 and 04.

**Cross-operator data sharing** — `advanced/14-cross-operator/01-in-binary`.

**Autoscaling workers** — `advanced/12-autoscale/02-based-on-own-metrics`.

---

## Available packs

```bash
ork init --list
```

---

## Next

- **[Orkestra Core](../orkestra-core/index.md)** — runtime, gateway, and control center architecture
- **[Security](../security/index.md)** — admission control, RBAC, namespace protection
- **[Reference](../reference/index.md)** — full schema and CLI reference
