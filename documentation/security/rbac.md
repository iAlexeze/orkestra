# RBAC

Orkestra never auto-creates permissions. Every right your operator has is generated from your Katalog, reviewed by you, committed to source control, and applied explicitly. Nothing is hidden inside the Helm chart.

---

## How it works

Orkestra reads your Katalog and derives the minimal set of permissions required to do exactly what you declared. Running `ork generate bundle` produces a ready-to-apply Kubernetes YAML with all RBAC resources:

```bash
ork generate bundle
```

If your file is named `katalog.yaml` or `komposer.yaml`, `--file` is optional — Orkestra finds it automatically. For any other name:

```bash
ork generate bundle -f my-katalog.yaml -o bundle.yaml
```

The bundle is a multi-document YAML stream. Review it, commit it, apply it:

```bash
kubectl apply -f bundle.yaml
```

---

## What the bundle contains

```
Namespace
ServiceAccount     orkestra              (runtime)
ServiceAccount     orkestra-gateway      (gateway)
ServiceAccount     orkestra-cc           (control center)
ClusterRole        orkestra              (runtime rules only)
ClusterRoleBinding orkestra
ClusterRole        orkestra-gateway      (gateway rules only)
ClusterRoleBinding orkestra-gateway
ConfigMap          orkestra-katalog      (embedded katalog YAML)
```

Three separate `ServiceAccounts`, each with its own `ClusterRole`. The control center gets only a `ServiceAccount` — it has no direct cluster access.

---

## Derived permissions, not wildcards

Traditional operators often ship with:

```yaml
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
```

This grants every API access to the operator process. Orkestra replaces it with rules derived directly from what you declared in the Katalog:

```yaml
- apiGroups: ["platform.orkestra.io"]
  resources: ["websites", "databases"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["platform.orkestra.io"]
  resources: ["websites/status", "databases/status"]
  verbs: ["get", "update", "patch"]
```

Only the API groups you declared. Only the resource kinds those groups produce. Only the verbs those resources need. Status subresources get a narrower verb set — `delete` is never granted on them.

Built-in Kubernetes resources (Deployments, Services, ConfigMaps, etc.) only appear in the rules if your Katalog actually uses them. If your operator creates no Deployments, the runtime has no `apps/deployments` permission.

---

## Runtime vs gateway — permissions are separate

The runtime and gateway run as separate processes with separate credentials. Their permissions do not overlap.

**Runtime** (`orkestra` ClusterRole) is scoped to what the reconciler needs:

| What | Permissions |
|------|-------------|
| Your CRDs | Full CRUD + watch on main and `/status` subresource |
| Built-in resources used in your Katalog | Full CRUD + watch (only those declared) |
| Kubernetes leases | `get`, `create`, `update` — for leader election |
| Events | `create`, `patch` — for status reporting |

**Gateway** (`orkestra-gateway` ClusterRole) is scoped to what admission and certificate management needs:

| What | Permissions |
|------|-------------|
| ValidatingWebhookConfigurations | `get`, `list`, `watch`, `create`, `update`, `patch`, `delete` |
| MutatingWebhookConfigurations | same (only if mutation rules are enabled) |
| Secrets | `get`, `create`, `update` — for TLS certificate storage |
| Namespaces | `get`, `patch` — for deletion-protection label management |
| CustomResourceDefinitions | `patch` — for CA bundle injection (only if conversion is enabled) |

A compromise of the gateway cannot read or modify your CRs. A compromise of the runtime cannot touch webhook configurations.

---

## Permissions scale with your Katalog

Gateway permissions are only generated if the features that need them are declared. Runtime permissions only include built-in resources your Katalog actually uses.

| Katalog feature | Added to |
|----------------|----------|
| Each CRD you declare | Runtime — CRUD on that group/resource |
| Deployments, Services, etc. in `onCreate` | Runtime — only the kinds you use |
| Validation rules | Gateway — `validatingwebhookconfigurations` |
| Mutation rules | Gateway — `mutatingwebhookconfigurations` |
| Deletion protection | Gateway — namespaces `get`/`patch` |
| Conversion webhooks | Gateway — `customresourcedefinitions` patch (CA bundle) |
| TLS / certificates | Gateway — secrets |

If you declare no validation rules, the gateway ClusterRole has no `admissionregistration.k8s.io` entries at all.

---

## Generating for specific components

By default `ork generate bundle` includes all three components. Use `--for` to generate only what you need:

```bash
# Runtime only
ork generate bundle --for runtime

# Gateway only
ork generate bundle --for gateway

# Runtime and control center, no gateway
ork generate bundle --for runtime,cc
```

Valid values: `runtime` (alias `run`), `gateway` (alias `gw`), `cc` (aliases `controlcenter`, `control-center`).

This is useful when deploying components to different clusters or namespaces, or when rotating credentials for one component without touching the others.

---

## Rerun after Katalog changes

Run `ork generate bundle` whenever you add a new CRD or resource type. The output diffs cleanly in GitOps workflows — you see exactly which permissions are being added or removed before applying.

---

## GitOps compatibility

The generated bundle is a plain Kubernetes YAML file. It works with any GitOps tool (Flux, ArgoCD, Helm, plain `kubectl apply`). Lint it, diff it in PRs, require approval before it reaches the cluster. No cluster-admin access required at any point.

---

## Next

- [Admission webhooks](admission.md) — validation, mutation, and conversion
- [Deletion protection](deletion-protection.md) — protecting your CRs from accidental deletion
