# 03 — Deletion Protection

Deletion protection blocks `kubectl delete crd` and operator deployment deletions for CRDs managed by this Katalog. It is implemented as a Kubernetes `ValidatingWebhookConfiguration`.

## How it works

The `ValidatingWebhookConfiguration` contains two webhook entries with different filtering strategies:

**Entry 1 — CRD protection**
```
Rule:    DELETE on customresourcedefinitions (apiextensions.k8s.io/v1)
Filter:  None at webhook level — handler calls isProtectedCRD(name)
Result:  "websites.other-operator.io" → allowed (not in ProtectedCRDNames)
         "cronjobs.demo.orkestra.io"  → denied  (in ProtectedCRDNames)
```

**Entry 2 — Orkestra resource protection**
```
Rule:           DELETE on deployments, services, ingresses
ObjectSelector: app.kubernetes.io/name: orkestra
                app.kubernetes.io/tag: orkestra-internal
Result:         Only resources with both labels reach the handler.
                If the handler is called, it always blocks — the ObjectSelector
                already confirmed the resource belongs to this operator.
```

This means resources from other operators or without the Orkestra labels are never intercepted by Entry 2, even though the GVR matches.

## Protected CRD name format

`ProtectedCRDNames()` returns a set keyed by `plural + "." + group`:

```go
names := k.ProtectedCRDNames()
// map["cronjobs.demo.orkestra.io":{}]
```

Built-in types (ConfigMap, Deployment) are excluded — they are not CRDs and cannot be deleted via the CRD API.

## Protected GVRs

`DeletionProtectionGVRs()` returns the four rules that back the two webhook entries:

| Resource | Group | Entry |
|----------|-------|-------|
| `customresourcedefinitions` | `apiextensions.k8s.io/v1` | Entry 1 (CRD, name-filtered) |
| `deployments` | `apps/v1` | Entry 2 (ObjectSelector) |
| `services` | `v1` (core) | Entry 2 (ObjectSelector) |
| `ingresses` | `networking.k8s.io/v1` | Entry 2 (ObjectSelector) |

## In-cluster vs local

The webhook is only registered when the operator runs inside a Kubernetes cluster. With `ork run` (local development), the webhook endpoint is unreachable. Registering it with `failurePolicy: Fail` would block ALL matched resource deletions cluster-wide when unreachable.

```go
func (k *Katalog) DeletionProtectionGVRs() []GVREntry {
    if !k.IsDeletionProtectionEnabled() { return nil }
    if !isRunningInCluster() { return nil }   // ← skips local mode
    ...
}
```

Detection: presence of `/var/run/secrets/kubernetes.io/serviceaccount/token` — always present inside a pod, never present in `ork run`.

## Enabling

In the Katalog YAML:

```yaml
security:
  deletionProtection:
    enabled: true
```

When the `security` block is present but `deletionProtection` is not declared, protection is **enabled by default**. If you have a security block, protection is on unless you explicitly opt out.

→ Next: [04-security.md](04-security.md)
