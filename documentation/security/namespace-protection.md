# Namespace protection

Namespace protection lets you declare which namespaces a CR can — or cannot — be applied to. A CR applied to the wrong namespace is rejected at admission time, before it is stored in etcd.

```yaml
security:
  namespaceProtection:
    enabled: true
    cleanupOnShutdown: true
    failurePolicy: Fail
```

---

## Two rule types

### `allowedNamespaces` — whitelist

Only the namespaces you list are accepted. Any other namespace is rejected.

```yaml
crds:
  app:
    allowedNamespaces:
      - production
```

Applying an App CR to the `staging` namespace:

```
Error from server: admission webhook "namespace-protection.orkestra.orkspace.io" denied the request:
[App "my-app"] namespace "staging" is not in the allowed list: [production]
```

### `restrictedNamespaces` — blacklist

The listed namespaces are rejected. Any other namespace is accepted.

```yaml
crds:
  cache:
    restrictedNamespaces:
      - kube-system
      - kube-public
```

Applying a Cache CR to `kube-system`:

```
Error from server: admission webhook "namespace-protection.orkestra.orkspace.io" denied the request:
[Cache "my-cache"] namespace "kube-system" is restricted
```

Applying the same Cache CR to `default` or any other non-system namespace succeeds.

---

## Per-CRD rules

Each CRD in your Katalog can have its own namespace rules, independent of the others:

```yaml
spec:
  crds:
    app:
      allowedNamespaces:
        - production       # App: only production

    database:
      allowedNamespaces:
        - production       # Database: only production

    cache:
      restrictedNamespaces:
        - kube-system      # Cache: anywhere except system namespaces
        - kube-public
```

---

## What gets blocked

Namespace rules apply to both `CREATE` and `UPDATE` requests. Moving a CR from an allowed namespace to a restricted one (by updating `metadata.namespace`) is also blocked — though in practice Kubernetes does not allow changing a resource's namespace via update; that requires delete and recreate, which would hit the rule on the new CR.

---

## Enforcement at two points

Like all security rules in Orkestra, namespace protection is enforced at two independent points:

1. **Admission time** — the Gateway webhook rejects the CR before etcd
2. **Reconcile time** — the reconciler re-checks the namespace rule before creating any child resource

If the webhook was absent during a brief restart window, the reconciler is the backstop.

---

## Webhook self-healing

If the namespace-protection `ValidatingWebhookConfiguration` is deleted, Orkestra recreates it immediately via its Watch-based controller. Deletion of the webhook configuration is not a way to bypass the protection.

---

## cleanupOnShutdown

```yaml
security:
  namespaceProtection:
    cleanupOnShutdown: true
```

When the operator shuts down gracefully (e.g., `helm uninstall`), it removes the webhook configuration it created. The cluster is left clean with no orphaned webhook configs.

---

## Working example

For a complete scenario including allowed, restricted, and blocked CR examples, run:

```bash
ork init --pack security
cd namespace-protection
ork e2e
```

---

## Next

Return to [Security overview](index.md) or explore [Deletion protection](deletion-protection.md).
