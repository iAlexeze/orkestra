# Learning to Orkestrate

Kubernetes operators are for everyone. That is Orkestra's promise — and the
best way to experience it is to run a real operator yourself, see it reconcile,
and work through progressively more capable patterns until the model clicks.

**Learning to Orkestrate** is the guided path for doing exactly that. It is a
series of example packs — collections of real, runnable operators — that ship
alongside every version of Orkestra. The examples are versioned with the runtime.
When Orkestra gains new capabilities, new examples follow. You always have access
to examples that work with the version you have installed.

---

## Getting started in two commands

No reading required to get your first operator running.

```bash
# Step 1: Install Orkestra
curl -sSL https://install.orkestra.dev | sh

# Step 2: Initialize your first operator project
ork init my-first-operator
```

Orkestra downloads the beginner example pack, scaffolds a project directory, and
prints exactly what to run next. The first operator — a Website CRD that creates
a Deployment and Service — takes about five minutes from installation to a
reconciling CR.

---

## Example packs

Each pack is a collection of complete, working operator examples. They are
intentionally ordered: start at the beginning even if you have Kubernetes
experience. The patterns build on each other.

### Beginner

The foundation. Every concept here is used in every more advanced example.

```bash
ork init my-operator --pack beginner
```

| Example | What it demonstrates |
|---|---|
| `01-hello-website` | Your first operator. Deployment, Service, owner references, status. |
| `02-website-with-service` | Multi-resource operators. LoadBalancer configuration. |
| `03-secret-copy` | Secret distribution across namespaces. `once:` and idempotency. |
| `03b-configmap-copy` | ConfigMap distribution. Statusless resource patterns. |
| `04-multi-resource` | Creating many resource types from one CR. |
| `05-when-conditions` | Async reconciliation. Gates that wait for readiness before proceeding. |
| `06-komposer-basic` | Composing multiple operators from one runtime. |

### Intermediate

Deeper patterns for operators that handle real production requirements.

```bash
ork init my-operator --pack intermediate
```

| Example | What it demonstrates |
|---|---|
| `07-validation-mutation` | Admission-time field validation. Default injection. |
| `08-komposer-registry` | Pulling operators from the Orkestra Registry. |
| `09-hooks` | When the declarative layer is not enough. Typed Go hooks. |
| `10-constructor` | Full ownership of the reconcile loop. Custom constructors. |

### Advanced

Patterns that solve the hard problems. These are the capabilities that
traditionally require expert-level operator engineering.

```bash
ork init my-operator --pack advanced
```

| Example | What it demonstrates |
|---|---|
| `11-cross-crd-ipc` | Operators that observe each other. Declarative dependency graphs. |
| `12-conversion` | Multi-version CRDs with lossless bidirectional conversion. No extra deployment. |
| `13-normalize` | Schema flexibility without conversion webhooks. The normalize block. |
| `14-foreach` | One declaration, N resources. List and map expansion. |
| `15-external-gate` | HTTP policy checks before resource creation. |
| `16-security` | Deletion protection, RBAC auto-apply, namespace guard. |
| `17-providers` | AWS and MongoDB integration without a single line of provider code. |

### Use Cases

Full-stack, end-to-end operator scenarios that mirror real platform engineering problems.

```bash
ork init my-operator --pack use-cases
```

| Example | What it demonstrates |
|---|---|
| `uc-01-platform-operator` | Complete platform team operator. Namespaces, RBAC, network policies, secrets. |
| `uc-02-multi-region-app` | forEach expansion across regions. Cross-CRD health gates. |
| `uc-03-configmap-operator` | ConfigMap as the input surface. Zero CRD, zero versioning. |
| `uc-04-cicd-pipeline` | Git + Docker build integration. Annotation-based state. |

---

## Running an example

Every example follows the same pattern:

```bash
# Initialize — downloads the pack for your current Orkestra version
ork init my-operator --pack beginner
cd my-operator

# Apply the CRD to your cluster
kubectl apply -f examples/beginner/01-hello-website/crd.yaml

# Start the operator locally
ork run --file examples/beginner/01-hello-website/katalog.yaml

# In another terminal — apply a CR
kubectl apply -f examples/beginner/01-hello-website/cr.yaml

# Watch it reconcile
kubectl get websites
kubectl get deployments

# Open the Control Center
ork control start
# → localhost:8081
```

The Control Center shows the operator's health, the CR's status, its child
resources, and the events emitted on every reconcile — all in real time.

---

## Keeping examples fresh

Examples are versioned with Orkestra. When you upgrade Orkestra, you get the
examples that ship with that version. Run `ork init` again after upgrading to
pull the updated pack.

```bash
# Check what you have
ork version

# Check if there's a newer version
ork upgrade --check

# Upgrade to the latest
ork upgrade

# Pull fresh examples for the new version
ork init my-operator-v2 --pack beginner
```

The upgrade command downloads the correct binary for your platform, replaces
the existing installation, and prints the new version. Your existing katalogs
and CRDs are unaffected — `ork upgrade` only updates the CLI.

> **Tip:** If you want to upgrade only the Orkestra runtime and skip the
> Control Center binary:
>
> ```bash
> ork upgrade --runtime-only
> ```

---

## Which pack to start with

**If you are new to Kubernetes operators:** Start with `beginner`. Work through
`01-hello-website` before anything else. The mental model it builds — declare
desired state, watch it reconcile — is the foundation for every other example.

**If you know Kubernetes but are new to Orkestra:** Start with `beginner/01`
to understand the Katalog, then skip to `intermediate/07` to see validation and
mutation. The declarative model will be familiar; the depth may surprise you.

**If you are migrating from Kubebuilder or Operator SDK:** Start with
`advanced/12-conversion` if you have multi-version CRDs, or `intermediate/09-hooks`
if you have existing Go reconcilers you want to preserve. Orkestra can wrap them.

**If you are building a platform:** Start with `use-cases`. These examples are
designed for the problems platform teams actually face — namespace provisioning,
secret distribution, multi-region deployments.

---

## Available packs

Run this at any time to see all available packs for your installed version:

```bash
ork init --list-packs
```

---

> **Tip — E2E tests and contribution**
>
> These same example packs power Orkestra's end-to-end test suite. Each example
> has a corresponding CI workflow that scaffolds it with `ork init`, applies the
> CRD and bundle, installs Orkestra via Helm, applies a CR, and verifies that
> reconciliation and cleanup work correctly — all in a real kind cluster.
>
> When you finish working through a pack, you are a contribution away from
> making Orkestra better for everyone. You can:
>
> - **Add a new example** to an existing pack — model it on the ones you just ran
> - **Add an e2e workflow** that validates it — copy the pattern from the existing workflows
> - **Do both** — one PR, one new operator scenario, one new test
>
> The rule is simple: `pack → e2e workflow`. New capability in Orkestra, new
> example demonstrating it, new test validating it. The examples stay fresh.
> The test suite grows. The community benefits.
>
> See [Contributing to the E2E Suite](../../.github/workflows/e2e/CONTRIBUTING.md) to get started.