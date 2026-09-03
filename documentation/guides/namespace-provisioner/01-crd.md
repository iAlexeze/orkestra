# CRD Design

The first decision is scope. The provisioner creates namespaces — and Kubernetes does not allow a namespaced resource to own a cluster-scoped one. If `NamespaceClaim` were `scope: Namespaced`, its CR couldn't hold OwnerReferences to the `Namespace`, `ClusterRole`, or `ClusterRoleBinding` it creates.

!!! note "RBAC in the generated bundle"
    Because the provisioner creates `ClusterRoles` and `ClusterRoleBindings`, Orkestra automatically adds `escalate` and `bind` to the runtime ServiceAccount's ClusterRole in the generated bundle. `escalate` allows creating a ClusterRole that grants permissions the Orkestra SA does not hold. `bind` allows creating a ClusterRoleBinding that references such a role. Both verbs are absent from operators that do not manage RBAC resources. See [RBAC security](../../security/02-rbac.md#clusterrole-and-role-management-the-escalate-verb) for details.

Set `scope: Cluster`:

```yaml
spec:
  group: provisioner.orkestra.io
  scope: Cluster
  names:
    plural: namespaceclaims
    kind: NamespaceClaim
    shortNames: [nsc]
```

**This is the right default for any CRD that manages cross-namespace or cluster-scoped resources.** A cluster-scoped CRD means cluster-scoped CRs — no namespace in `metadata`, applied with `kubectl apply -f cr.yaml` directly.

---

## Fields

Keep the spec minimal. The CR author should express intent, not Kubernetes internals:

```yaml
spec:
  targetNamespace: team-alpha   # namespace to create
  team: alpha                   # used for labels and resource naming
  owner: alpha-operator         # ServiceAccount that receives admin access
  ownerNamespace: default
```

`01-explicit` adds explicit quota fields. `02-profiles` replaces them with a single tier name. Both map onto the same eight output resources.

---

## Try it

```bash
ork init --pack use-cases/namespace-provisioner
cd namespace-provisioner
cat crd.yaml
```

→ Next: [Explicit limits](02-explicit.md)
