# Test Anything That Runs in Kubernetes

`ork e2e` is a Kubernetes behavioural verification tool. It spins up a real cluster, installs things, watches the API server, and asserts GVK state. Nothing in that pipeline requires the workload under test to be an Orkestra operator.

A Helm chart installs into the same API server. A `kubectl apply` produces the same objects. The assertion layer does not care about the installation mechanism — it watches GVKs. A `Deployment/my-app` is the same object whether it was created by an Orkestra operator, a Helm chart, a controller-runtime reconciler, or `kubectl apply`.

---

## The insight

Today there is no way to look at a container image, a deployment manifest, or a Helm chart and say it works. Signatures prove ownership. They say nothing about behaviour. A signed chart with a broken init container is still signed. A signed operator that silently drops events under load is still signed.

`ork e2e` built the verification layer in the hardest context — operators with long-lived state, drift correction, complex status, and partial failure modes. If you can prove an operator works correctly before distribution, you can prove a Helm chart works. The machinery is the same. The e2e file is the contract. The cluster is the witness.

---

## Contents

| Page | What it covers |
|---|---|
| [How it works](01-how-it-works.md) | `custom.target: kubernetes`, the e2e file structure, running it |
| [Use cases](02-use-cases.md) | Helm charts, third-party operators, platform stacks, kubectl DSL |
| [CI integration](03-ci.md) | GitHub Actions bridge — one step to add e2e to any workflow |

---

→ Start with [How it works](01-how-it-works.md)
