# Orkestra Examples

A progressive set of examples — from a single Deployment to multi-source composition, autoscaling, cross-operator dependencies, and full platform stacks built from custom resources.

---

## The fastest way to start

```bash
ork init my-operator --pack beginner
cd my-operator/beginner/01-hello-website
ork run --dev
```

The examples are **embedded in the CLI binary** — no internet connection needed. They are extracted instantly into your project folder.

---

## Choose your pack

```bash
ork init my-operator                          # defaults to beginner
ork init my-operator --pack beginner          # Simple CRDs, Deployments, Services
ork init my-operator --pack intermediate      # Multi-resource, conditions, state machines, Komposer
ork init my-operator --pack advanced          # Hooks, constructors, validation, registries, autoscale, custom resources
ork init my-operator --pack security          # Deletion protection, namespace isolation, admission webhooks
ork init my-operator --pack use-cases         # Full-stack, cross-CRD, external gates, multi-region, and more
```

List all packs:
```bash
ork init --list-packs
```

After init, your examples live at:
```
my-operator/
└── <pack>/
    ├── e2e.yaml            full suite — runs all examples in one command
    └── <example>/
        ├── README.md       step-by-step walkthrough
        ├── katalog.yaml    operator definition
        ├── crd.yaml        the CRD to install
        ├── cr.yaml         sample custom resource
        └── e2e.yaml        end-to-end test for this example
```

---

## Learning Path

### Beginner — `--pack beginner`

Start here. No cluster setup beyond the `ork` CLI.

| Example | What you learn |
|---------|----------------|
| [01 — Hello Website](./beginner/01-hello-website/) | Your first operator. One CRD, one Deployment. |
| [02 — With ServiceAccount](./beginner/02-with-serviceaccount/) | RBAC, ServiceAccount wiring, multiple resources. |
| [03 — Copy Secret](./beginner/03-secret-copy/) | Built-in Kubernetes kinds. Cross-namespace secret copy. |
| [03b — Copy ConfigMap](./beginner/03b-configmap-copy/) | Same pattern applied to ConfigMaps. |

### Intermediate — `--pack intermediate`

You know the basics. Now use more of Orkestra's surface.

| Example | What you learn |
|---------|----------------|
| [04 — Multi-Resource](./intermediate/04-multi-resource/) | ConfigMap, status fields, child resource status propagation. |
| [05 — When Conditions](./intermediate/05-when-conditions/) | Conditional resource creation. Topology that changes with CR state. |
| [06 — Basic Komposer](./intermediate/06-komposer-basic/) | Composing two Katalogs. Environment-specific overrides. |
| [07 — CRD File](./intermediate/07-crd-file/) | `crdFile:` — derive API types from the CRD YAML, no `apiTypes:` needed. |
| [08 — State Machine](./intermediate/08-state-machine/) | Phase-driven reconciliation. Ordered transitions with status guards. |

### Advanced — `--pack advanced`

Production patterns. Admission policy, typed Go operators, autoscaling, custom resources, and more.

| Example | What you learn |
|---------|----------------|
| [07 — Validation and Mutation](./advanced/07-validation-mutation/) | Admission-time deny/warn. Defaults. Full status. |
| [08 — Komposer with Registry](./advanced/08-komposer-registry/) | OCI registry source. Multi-environment Komposer. |
| [09 — Go Hooks](./advanced/09-hooks/) | Typed hooks. OrkestraRegistry from Go. |
| [10 — Custom Constructor](./advanced/10-constructor/) | Full reconciler control. Migration from existing operators. |
| [11 — Mixed Operator Pattern](./advanced/11-mixed-operator-pattern/) | Dynamic + Hooks + Constructor in one binary. |
| [12 — Autoscale](./advanced/12-autoscale/) | Queue-depth autoscaling, sibling metrics, external gates. |
| [13 — Dependencies](./advanced/13-dependencies/) | Ordered startup across CRDs in-binary, cross-binary, cross-cluster. |
| [14 — Cross-Operator](./advanced/14-cross-operator/) | Share data between operators. |
| [15 — Any Language](./advanced/15-any-language/) | Generate Katalogs from Python, Go, or Node.js. |
| [16 — Custom Resources](./advanced/16-custom-resources/) | Compose third-party CRDs as children — 7 sub-examples from single child to full platform. |
| [17 — API Type Override](./advanced/17-apitype-override/) | Override API types per Komposer import without editing source Katalogs. |
| [18 — CRD File Komposer](./advanced/18-crd-file-komposer/) | `crdFile:` across Komposer imports. |

### Security — `--pack security`

Protect your cluster from accidental deletions, rogue workloads, and bad input.

| Example | What you learn |
|---------|----------------|
| [Admission Webhooks](./security/admission/) | Validate and mutate CRs at admission time. |
| [Deletion Protection](./security/deletion-protection/) | Block deletion of critical CRs via admission. |
| [Namespace Protection](./security/namespace-protection/) | Restrict what namespaces operators can act on. |

### Use Cases — `--pack use-cases`

Real-world patterns combining multiple Orkestra features.

| Example | What you learn |
|---------|----------------|
| [Full-Stack App](./use-cases/full-stack-app/) | forEach + external + cross + once + anyOf in one CR. |
| [Multi-Region Map](./use-cases/multi-region-map/) | Deploy across regions using `forEach` over a map. |
| [CRD Conversion](./use-cases/crd-conversion/) | Multi-version CRDs with or without a conversion webhook. |
| [Custom Operator](./use-cases/custom-operator/) | `spec.customOperator: true` — use `ork e2e` as a test harness for any operator. |
| [External](./use-cases/external/) | Gate resource creation on upstream health checks. |
| [Multi-Tenancy](./use-cases/multi-tenancy/) | Namespace isolation, per-tenant configuration. |
| [Enrich](./use-cases/enrich/) | Inject data from external sources into CR status. |
| [Normalize](./use-cases/normalize/) | Validate and normalise CR fields at reconcile time. |
| [Profiles](./use-cases/profiles/) | Apply different resource configurations based on environment profiles. |

---

## E2E test suites

Every example ships with `e2e.yaml`. Every pack ships with a root `e2e.yaml` that runs the full suite.

```bash
# Run a single example
cd beginner/01-hello-website && ork e2e

# Run an entire pack
ork e2e -f beginner/e2e.yaml
ork e2e -f intermediate/e2e.yaml
ork e2e -f security/e2e.yaml

# Simulate the full pack (no cluster needed)
ork simulate ./...
```

---

## Prerequisites

All examples:
- `ork` CLI — `curl get.orkestra.sh | bash`
- A running Kubernetes cluster (`kind create cluster` works for every example here)
- `kubectl` configured

Advanced typed examples (09, 10, 11) also require:
- Go 1.22+
- `make registry && make build` to compile your operator binary before running e2e

---

## Running any example

```bash
# 1. Pick a pack and scaffold your project
ork init my-operator --pack beginner
cd my-operator/beginner/01-hello-website

# 2. Start the runtime
ork run --dev

# 3. Watch the resources appear
kubectl get websites -n default

# 4. Cleanup
./cleanup.sh
```

Each example's `README.md` has the exact commands for that example.
