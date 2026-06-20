# Guides

End-to-end walkthroughs for specific goals. Each guide has a companion example pack you can run locally.

| Guide | Pack | What it covers |
|-------|------|----------------|
| [Registry](./registry/index.md) | `registry-guide` | Publish, version, gate, consume, and automate the full Katalog lifecycle |
| [Migration](./migration/index.md) | `from-controller-runtime` | Move an existing controller-runtime operator to Orkestra — five options |
| [Ecosystem Composition](./ecosystem/index.md) | `ecosystem-composition` | Wrap ArgoCD, cert-manager, Prometheus, and Crossplane with internal abstraction layers |

```bash
ork init --pack registry-guide
ork init --pack from-controller-runtime
ork init --pack ecosystem-composition
```
