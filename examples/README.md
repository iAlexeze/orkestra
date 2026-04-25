# Orkestra Examples

A progressive set of examples — from a single Deployment to a multi-source Komposer with admission policy and Go hooks. Each example is complete and runnable. Each builds on the concepts introduced by the previous one.

---

## Learning path

### Beginner — `ork run`

Start here. No cluster setup required beyond a CRD and the `ork` CLI.

| Example | What you learn |
|---|---|
| [01 — Hello Website](./beginner/01-hello-website/README.md) | Your first operator. One CRD, one Deployment. |
| [02 — Website with Service](./beginner/02-website-with-service/README.md) | Multiple resources. Drift correction. |
| [03 — Copy Secret](./beginner/03-secret-copy/README.md) | Built-in Kubernetes kinds. Cross-namespace secret distribution. |
| [BONUS — 04 Copy ConfigMap](./beginner/03b-bonus-configmap-copy/README.md) | Built-in Kubernetes kinds. Cross-namespace configmap distribution. |

### Intermediate — `ork run`

You know the basics. Now use more of Orkestra's surface.

| Example | What you learn |
|---|---|
| [04 — Multi-Resource with Status](./intermediate/04-multi-resource/README.md) | ConfigMap, status fields, child resource status propagation. |
| [05 — When Conditions](./intermediate/05-when-conditions/README.md) | Conditional resource creation. Topology that changes with CR state. |
| [06 — Basic Komposer](./intermediate/06-komposer-basic/README.md) | Composing two Katalogs. Environment-specific overrides. |

### Advanced — `kubectl apply`

Production deployment. Admission policy, registry composition, Go hooks, custom reconcilers.

| Example | What you learn |
|---|---|
| [07 — Validation and Mutation](./advanced/07-validation-mutation/README.md) | Admission-time deny/warn. Defaults. Full status. |
| [08 — Komposer with Registry](./advanced/08-komposer-registry/README.md) | OCI registry source. Helm source. Multi-environment Komposer. |
| [09 — Go Hooks](./advanced/09-hooks/README.md) | Typed hooks. OrkestraRegistry from Go. External API calls. |
| [10 — Custom Constructor](./advanced/10-constructor/README.md) | Full reconciler control. Migration from existing operators. |

---

## Prerequisites

All examples require:
- A running Kubernetes cluster (kind, minikube, or remote)
- `kubectl` configured
- `ork` installed: `brew install orkspace/tap/ork`

Advanced examples additionally require:
- Orkestra deployed in-cluster: see [Deployment Guide](../docs/guides/deployment.md)
- `ENABLE_ADMISSION_WEBHOOK=true` for validation/mutation examples
- Go 1.22+ for hook and constructor examples

---

## How to use these examples

Each example directory contains:

```
example-name/
  README.md         step-by-step walkthrough
  crd.yaml          the CRD to install
  katalog.yaml      the operator definition
  cr.yaml           the custom resource to apply
  cleanup.sh        removes everything the example created
```

Advanced examples may also include:
```
  komposer.yaml         Komposer configuration
  hooks/                Go hook implementation
  install-xx-.yaml      in-cluster Orkestra deployment
```

Run `chmod +x cleanup.sh` before running cleanup.
