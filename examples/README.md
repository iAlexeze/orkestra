# Orkestra Examples

A progressive set of examples — from a single Deployment to multi-source composition, autoscaling, cross-operator dependencies, and full platform stacks built from custom resources.

---

## The fastest way to start

```bash
ork init my-operator --pack beginner
cd my-operator/examples/beginner/01-hello-website
ork run --dev
```

The examples are **embedded in the CLI binary** — no internet connection needed. They are extracted instantly into your project folder.

---

## Choose your pack

```bash
ork init my-operator                          # defaults to beginner
ork init my-operator --pack beginner          # Simple CRDs, Deployments, Services
ork init my-operator --pack intermediate      # Multi-resource, conditions, Komposer
ork init my-operator --pack advanced          # Hooks, constructors, validation, registries, autoscale, custom resources
ork init my-operator --pack security          # Deletion protection, namespace isolation, admission webhooks
ork init my-operator --pack use-cases         # Full-stack, cross-CRD, external gates, multi-region
ork init my-operator --pack rollback          # Zero-config and configurable failure recovery
ork init my-operator --pack developer         # Deploy your app without writing YAML
```

List all packs:
```bash
ork init --list-packs
```

After init, your examples live at:
```
my-operator/
└── examples/
    └── <pack>/
        ├── e2e.yaml            full suite — runs all examples in one command
        └── <example>/
            ├── README.md       step-by-step walkthrough
            ├── katalog.yaml    operator definition
            ├── crd.yaml        the CRD to install
            ├── cr.yaml         sample custom resource
            ├── e2e.yaml        end-to-end test for this example
            └── cleanup.sh      removes everything the example created
```

---

## Learning Path

### Beginner — `--pack beginner`

Start here. No cluster setup beyond the `ork` CLI.

| Example | What you learn |
|---------|----------------|
| [01 — Hello Website](./beginner/01-hello-website/) | Your first operator. One CRD, one Deployment. |
| [02 — Website with Service](./beginner/02-website-with-service/) | Multiple resources. Drift correction. |
| [03 — Copy Secret](./beginner/03-secret-copy/) | Built-in Kubernetes kinds. Cross-namespace secret copy. |
| [BONUS 03b — Copy ConfigMap](./beginner/03b-bonus-configmap-copy/) | Same pattern applied to ConfigMaps. |

### Intermediate — `--pack intermediate`

You know the basics. Now use more of Orkestra's surface.

| Example | What you learn |
|---------|----------------|
| [04 — Multi-Resource with Status](./intermediate/04-multi-resource/) | ConfigMap, status fields, child resource status propagation. |
| [05 — When Conditions](./intermediate/05-when-conditions/) | Conditional resource creation. Topology that changes with CR state. |
| [06 — Basic Komposer](./intermediate/06-komposer-basic/) | Composing two Katalogs. Environment-specific overrides. |

### Advanced — `--pack advanced`

Production patterns. Admission policy, registry composition, Go hooks, autoscaling, custom resources.

| Example | What you learn |
|---------|----------------|
| [07 — Validation and Mutation](./advanced/07-validation-mutation/) | Admission-time deny/warn. Defaults. Full status. |
| [08 — Komposer with Registry](./advanced/08-komposer-registry/) | OCI registry source. Multi-environment Komposer. |
| [09 — Go Hooks](./advanced/09-hooks/) | Typed hooks. OrkestraRegistry from Go. External API calls. |
| [10 — Custom Constructor](./advanced/10-constructor/) | Full reconciler control. Migration from existing operators. |
| [11 — Mixed Operator Pattern](./advanced/11-mixed-operator-patterm/) | Dynamic + Hooks + Constructor together in one binary. |
| [12 — Autoscale](./advanced/12-autoscale/) | Queue-depth autoscaling, sibling metrics, external gates. |
| [13 — Dependencies](./advanced/13-dependencies/) | Ordered startup across CRDs in-binary, cross-binary, cross-cluster. |
| [14 — Cross-Operator Communication](./advanced/14-cross-operator/) | Share data between operators: in-binary, cross-binary, cross-cluster. |
| [15 — Any Language](./advanced/15-any-language/) | Generate Katalogs from Python, Go, or Node.js. |
| [16 — Custom Resources](./advanced/16-custom-resources/) | Compose third-party CRDs as children of your own — see below. |

#### 16 — Custom Resources (sub-examples)

This example set has its own progression, from a single child CR all the way to full platform composition:

| Sub-example | What you learn |
|-------------|----------------|
| [01 — Single Child CR](./advanced/16-custom-resources/01-single-child/) | `onCreate.custom`, templates, owner references, cascade deletion. |
| [02 — Status Propagation](./advanced/16-custom-resources/02-status-propagation/) | `hasStatus: true`, reading child status back into the parent. |
| [03 — Conditional Children](./advanced/16-custom-resources/03-conditional-children/) | `when:` — create, skip, and cleanup child CRs based on parent flags. |
| [04 — Drift Correction](./advanced/16-custom-resources/04-drift-correction/) | `reconcile: true` — keep child CRs in sync on every reconcile. |
| [05 — forEach Sharding](./advanced/16-custom-resources/05-forEach-sharding/) | `forEach:` fan-out — one child CR per list element. |
| [06 — Full Platform Composition](./advanced/16-custom-resources/06-full-platform-composition/) | Motif imports, three child CRs, status aggregation across the stack. |
| [07 — Multi-CRD Pipeline](./advanced/16-custom-resources/07-multi-crd-pipeline/) | Multiple child CRs + multiple controllers in one Katalog from a Motif. |

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
| [Full-Stack App](./use-cases/full-stack-app/) | Frontend + backend + database composed as one CR. |
| [CRD Conversion](./use-cases/crd-conversion/) | Multi-version CRDs with or without a conversion webhook. Two approaches, same result. |
| [Multi-Region Map](./use-cases/multi-region-map/) | Deploy the same workload across multiple regions using `forEach`. |
| [Rollback](./use-cases/rollback/) | Zero-config and configurable failure recovery. |

### Rollback — `--pack rollback`

Focus on failure recovery patterns.

| Example | What you learn |
|---------|----------------|
| [Rollback sub-examples](./use-cases/rollback/) | Automatic rollback, configurable policies, recovery hooks. |

### Developer — `--pack developer`

Deploy your app to Kubernetes without writing operator code or knowing CRDs.

| Example | What you learn |
|---------|----------------|
| [01 — One Project](./developer/01-one-project/) | Single-service deployment, zero YAML knowledge required. |
| [02 — Frontend + Backend](./developer/02-frontend-backend/) | Two services, automatic wiring. |
| [03 — Rollback + Ingress](./developer/03-rollback-ingress/) | Ingress, rollback, HTTPS. |
| [04 — Notifications](./developer/04-notify/) | Slack/webhook alerts on deployment events. |
| [05 — Deletion Protection](./developer/05-deletion-protection/) | Prevent accidental deletion of running services. |
| [06 — Docker Compose + Postgres](./developer/06-docker-compose-with-postgres/) | Lift a Docker Compose stack into Kubernetes as-is. |

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
```

Each suite creates a kind cluster, runs all examples in sequence, and tears everything down. No extra setup needed.

---

## Prerequisites

All examples:
- `ork` CLI — `curl get.orkestra.sh | bash`
- A running Kubernetes cluster (`kind create cluster` works for every example here)
- `kubectl` configured

Advanced typed examples require:
- Go 1.22+ for hook and constructor examples

---

## Running any example

```bash
# 1. Pick a pack and scaffold your project
ork init my-operator --pack beginner
cd my-operator/examples/beginner/01-hello-website

# 2. Start the operator
ork run --dev

# 3. Watch the resources appear
kubectl get websites -n default
kubectl get deployments -n default
kubectl get services -n default

# 4. Cleanup
chmod +x cleanup.sh
./cleanup.sh
```

Each example's `README.md` has the exact commands for that example.
