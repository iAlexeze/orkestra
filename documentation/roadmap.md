# Roadmap

*Last updated: May 2026*

---

## Where we are

Orkestra is a complete declarative operator runtime for Kubernetes. The core is shipped and running in production. Here is what exists today:

**Runtime**

- Dynamic mode — zero-code operators, no generated types, no compilation step
- Typed mode — Go types, Go hooks, custom constructors when you need them
- GenericReconciler with three-path dispatch: templates, hooks, constructor
- Per-CRD isolation — dedicated informer, workqueue, and worker pool per CRD
- Dependency graph — topological startup order (`dependsOn`), cycle detection
- safeReconcile — panic recovery per CRD, other CRDs unaffected
- Konductor election — leader election with warm-cache follower failover
- Autoscale — dynamic worker and resync scaling based on metrics

**Declarations**

- Katalog — CRDs, reconcile templates, workers, resync, dependencies, conversion rules
- Komposer — compose Katalogs from files, Helm charts, and OCI/Git registries
- Motifs — reusable resource primitives shared across Katalogs via the motif registry
- Conditions (`when:`) — conditional resource creation based on CR field values
- Declarative version conversion — conversion rules in YAML, no Go code
- Declarative validation — deny/warn rules at reconcile and admission time
- Declarative mutation — defaults and overrides at reconcile and admission time

**Platform**

- Gateway (`ork gate`) — admission webhooks, TLS, conversion webhooks, notifications
- Control Center (`ork control`) — live operator dashboard, multi-runtime support
- Registry (`ork registry`) — publish and pull operator patterns as OCI artifacts
- E2E (`ork e2e`) — declarative end-to-end testing that gates registry publication

**Security**

- Namespace protection — admission and runtime enforcement, two independent layers
- Deletion protection — CR and CRD deletion guarded by labeled finalizers
- Admission control — deny/warn rules at admission time without a webhook server
- RBAC generation — `ork generate bundle --for runtime` produces scoped ClusterRoles

**CLI**

`ork init`, `ork run`, `ork gate`, `ork validate`, `ork template`, `ork simulate`, `ork plan`, `ork diff`, `ork generate`, `ork registry`, `ork control`, `ork notes`, `ork e2e`, `ork deploy`, `ork tunnel`, `ork version`

**Distribution**

- Homebrew tap — `brew install orkspace/tap/ork`
- curl installer — `curl -sSL get.orkestra.sh | bash` with GPG signing
- Docker image — GHCR, distroless, two-stage build
- Helm chart — production-ready deployment chart

---

## Where we are going

### Pod security profiles ✓ shipped

Declarative pod security per workload resource. Set a named profile in one line:

```yaml
securityContext:
  profile: hardened
podSecurity:
  profile: hardened
```

`hardened` sets `runAsNonRoot: true`, `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`, drops all capabilities. Profiles: `baseline`, `restricted`, `hardened`. Individual fields can be declared instead of a profile. → See [Pod security](./security/07-pod-security.md).

### Improved rollback — child resource tracking

Current rollback triggers on consecutive CR reconcile failures. The redesign watches the child resources the CR creates:

```yaml
rollback:
  trigger:
    consecutiveFailures: 3
  watchResources:
    deployments:
      - name: "{{ .metadata.name }}"
        severity: critical
```

A Deployment that never becomes `Available` within the timeout triggers rollback — not an abstract reconcile failure count. Snapshots are taken only after child resources confirm healthy; they are refreshed when the spec changes and resources are healthy. Rollback exits automatically when the CR generation changes (user fixed the spec).

### Operator as library

Orkestra is a Go library. Teams can import it (`go.mod` version pin) and write their own entrypoint — full control, no fork needed:

```go
func main() {
    kfg, err := konfig.Init()
    if err != nil {
        logger.Fatal().AnErr("failed to load configurations", err)
        utils.Exit(err)
    }
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    cli.Execute(kfg, ctx)
}
```

They get the full runtime, gateway, CLI, and webhook system. If they need a custom webhook, they know exactly where to plug it in. Two things needed: a version-pinned `go.mod` import and this entrypoint.

### Performance benchmarks

Published numbers for reconcile throughput, queue latency, and informer memory usage at 50+ and 100+ CRDs. Stress test results with quality gates.

### CNCF Sandbox

Target 2027. Prerequisite is production adoption at multiple organisations, with metrics. CNCF Sandbox gives Orkestra vendor neutrality, community governance, and the credibility that enterprise platform teams require before adopting an open-source runtime.

---

## The longer horizon

Katalog and Komposer as native Kubernetes kinds — registered by the cluster itself, understood by `kube-controller-manager`, auditable through the standard Kubernetes audit log.

```bash
kubectl get katalogs          # not yet, but this is where we are going
kubectl describe katalog website-operator
```

The path: production adoption → CNCF Sandbox → Kubernetes Enhancement Proposal → alpha behind a feature gate → beta → general availability. A realistic timeline is five years. The work is not primarily technical — the design is largely correct. The work is community trust.

See [Declarative Operators: A New Model for Kubernetes Extensibility](/publications/declarative-operators-whitepaper/)
for the full argument.

---

## What we are not building

**Multi-cluster federation.** Orkestra manages CRDs within one cluster. Cross-cluster operations belong to a different architectural layer.

**Replacing controller-runtime.** Orkestra is a higher-level abstraction. Custom constructors bridge to controller-runtime for use cases that need it. They are complementary, not competitive.

**A general-purpose policy engine.** Orkestra's validation and mutation are scoped to the CRDs it manages. Global cluster-wide policy belongs in OPA, Kyverno, or VAP.

---

## Contributing

The highest-value contributions right now:

| Area | What helps most |
|---|---|
| **Production deployments** | Run Orkestra on real workloads, report what breaks |
| **Registry patterns** | Five-file patterns for common CRDs — postgres, redis, cert-manager |
| **Testing at scale** | 50+ CRD deployments, stress test results |
| **Documentation** | Edge cases, gotchas, things that weren't obvious |
| **Hooks** | Real-world hook implementations for complex operators |

Open a [GitHub issue](https://github.com/orkspace/orkestra/issues) or
[Discussion](https://github.com/orkspace/orkestra/discussions) for anything not covered above.

---

[Start here →](./getting-started/index.md)
