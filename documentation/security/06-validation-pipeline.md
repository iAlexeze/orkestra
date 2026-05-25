# Validation pipeline

Every Orkestra document goes through a strict validation pipeline before anything runs. Most of this pipeline runs entirely offline — no cluster required.

---

## Strict YAML parsing

Orkestra uses strict YAML unmarshaling. Unknown fields produce an immediate error — there are no silent defaults, no ignored typos. A field you didn't mean to write and one you did mean to write are both caught.

```text
error: unknown field "reconcileMode" in operatorBox — did you mean "reconcile"?
```

This applies to every document type: Katalog, Komposer, Motif, E2E spec.

---

## `ork validate`

Validates a Katalog or Komposer fully without touching a cluster.

```bash
ork validate
ork validate -f path/to/katalog.yaml
```

What it checks:

- Schema: every field against the declared types and constraints
- Template expressions: `{{ }}` syntax is parsed and validated
- Dependency graph: cycles in `dependsOn` chains are a hard error
- Namespace rules: `allowedNamespaces`/`restrictedNamespaces` consistency
- CRD references: `crdFile` paths resolve, CRD group/version/kind/plural are valid

`ork validate` runs entirely offline. No `~/.kube/config` is read. No cluster is contacted.

---

## Commands that run offline

The following commands require no cluster connection:

| Command | What it does |
|---------|-------------|
| `ork init` | Scaffold a new operator project |
| `ork validate` | Full document validation |
| `ork template` | Render the merged, resolved Katalog |
| `ork simulate` | Reconcile against a fake in-memory cluster |
| `ork notes` | Browse and search template expression library |
| `ork generate` | Generate bundle, CRD, or RBAC manifests |
| `ork plan --bundle` | Diff local Katalog against a local bundle file |

---

## Commands that require a cluster

Only two commands actively connect to and manage a real Kubernetes cluster:

| Command | Why it needs a cluster |
|---------|----------------------|
| `ork run` | Applies CRDs, starts reconcilers, watches resources |
| `ork gate` | Registers webhooks, processes admission requests |

`ork e2e` creates or reuses a kind cluster. `ork plan` (without `--bundle`) reads the cluster's ConfigMap. All other commands are fully local.

---

## Defense in depth

Validation runs at multiple points independently:

1. **Parse time** — strict YAML unmarshaling rejects unknown fields immediately
2. **`ork validate`** — full offline validation before any cluster interaction
3. **`ork simulate`** — exercises the reconcile loop in-memory before running against a real cluster
4. **Admission webhooks** — the Gateway rejects invalid CRs at CREATE/UPDATE time, before etcd storage
5. **Reconcile time** — the runtime re-checks rules on every reconcile as a backstop

A document that passes `ork validate` and `ork simulate` is very unlikely to fail at runtime — the same validation logic runs in all five stages.

---

## Next

- **[Admission Control](./admission.md)** — deny and warn rules enforced by the webhook
- **[Namespace Protection](./namespace-protection.md)** — two-point namespace enforcement
- **[ork plan](../reference/cli/plan.md)** — diff a Katalog before applying it
