# Security

Orkestra is designed for production from the start. Security is not layered on later — it is part of the execution model.

This document covers Orkestra's full security posture: how it is built, how it controls permissions, and how it protects your cluster at runtime.

---

## Table of contents

- [Minimal production binary](#minimal-production-binary)
- [Explicit, derived RBAC](#explicit-derived-rbac)
- [Admission webhooks — validation and mutation](#admission-webhooks--validation-and-mutation)
- [Deletion protection](#deletion-protection)
- [Namespace protection](#namespace-protection)
- [Two enforcement points](#two-enforcement-points)
- [Webhook self-healing](#webhook-self-healing)
- [TLS](#tls)
- [Credentials in sources](#credentials-in-sources)
- [Supply chain security](#supply-chain-security)
- [Template safety](#template-safety)
- [Logging and audit](#logging-and-audit)
- [Reporting vulnerabilities](#reporting-vulnerabilities)

---

## Minimal production binary

The Orkestra CLI ships in two distinct forms: a **full developer CLI** and a **runtime binary**.

The runtime binary is compiled with the `runtime` build tag, which strips every command that is not required for operator execution:

| Command | Developer CLI | Runtime binary |
|---------|:---:|:---:|
| `ork run` | ✓ | ✓ |
| `ork version` | ✓ | ✓ |
| `ork generate` | ✓ | — |
| `ork validate` | ✓ | — |
| `ork init` | ✓ | — |
| `ork template` | ✓ | — |
| `ork diff` | ✓ | — |

**Why this matters:** a compromised container cannot use the binary to generate RBAC bundles, enumerate registered CRDs, scaffold new operators, extract Katalog definitions, or exfiltrate configuration. The production binary knows only how to run. There is no code generation surface to exploit.

This is a structural guarantee — the commands are removed at compile time, not hidden behind flags or access controls.

---

## Explicit, derived RBAC

Orkestra never auto-creates `ServiceAccount`, `ClusterRole`, or `ClusterRoleBinding` resources. You generate them from your Katalog, review the output, commit it, and apply it explicitly:

```bash
ork generate bundle --file my-katalog.yaml -o bundle.yaml
kubectl apply -f bundle.yaml
```

Nothing is hidden inside the Helm chart. Every permission your operator has is visible in source control before it reaches the cluster.

### Derived permissions, not wildcards

Traditional operators often ship with:

```yaml
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
```

This is convenient but unsafe. Orkestra replaces it with permissions derived directly from your Katalog:

```yaml
- apiGroups: ["platform.orkestra.io"]
  resources: ["websites", "databases"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

Only the CRDs you declare, only the resources they produce, only the verbs they require. If you do not declare validation rules, `admissionregistration.k8s.io` permissions are not included.

### Rerun after Katalog changes

```bash
ork generate bundle --file my-katalog.yaml -o bundle.yaml
```

Run this whenever you add a new CRD or resource type. The output diffs cleanly in GitOps workflows.

---

## Admission webhooks — validation and mutation

When `security.webhooks.admission.enabled: true` is set in your Katalog, Orkestra registers webhook configurations that intercept every `CREATE` and `UPDATE` request for the CRs it manages.

### Webhook creation is conditional

Orkestra creates a webhook configuration only when there are rules that require it:

- Validation rules declared → `ValidatingWebhookConfiguration` registered
- Mutation rules declared → `MutatingWebhookConfiguration` registered
- No rules → no webhook configuration registered, no API server overhead

### Validation rules

Validation rules run at admission time and at reconcile time:

```yaml
crds:
  platform:
    validation:
      rules:
        - field: spec.image
          prefix: "registry.internal/"
          action: deny       # hard block — rejected before stored

        - field: spec.rateLimit
          operator: exists
          action: warn       # advisory — logged, CR is accepted
```

A `deny` rule rejects the request before the object is stored in etcd. A `warn` rule records the violation but allows the request through.

### Mutation rules

Mutation rules run before validation, filling in defaults for fields the user did not declare:

```yaml
crds:
  platform:
    mutation:
      mutateFirst: true
      rules:
        - field: spec.replicas    default: "2"
        - field: spec.environment default: "production"
```

`mutateFirst: true` ensures defaults are applied before any `deny` rule can fire. A CR that omits `spec.replicas` receives the default and then passes the `greaterThan: 0` rule cleanly.

### failurePolicy

```yaml
security:
  webhooks:
    failurePolicy: Ignore   # allow through if Orkestra is temporarily unreachable
```

For environments where availability is the priority, `Ignore` means the CR is accepted even if Orkestra's webhook endpoint is unreachable (e.g., during a rolling restart). For high-security environments, `Fail` blocks the request if the webhook is unreachable — combined with deletion protection and PodDisruptionBudgets, the unreachable window should be near zero.

---

## Deletion protection

When `security.deletionProtection.enabled: true` is set, Orkestra registers a `ValidatingWebhookConfiguration` that intercepts every `DELETE` request for CRs managed by your Katalog. A blocked deletion is rejected by the API server before it reaches etcd.

```yaml
security:
  deletionProtection:
    enabled: true
    cleanupOnShutdown: true
    failurePolicy: Fail
```

With `failurePolicy: Fail`, a DELETE is also blocked if Orkestra is temporarily unreachable — you cannot accidentally delete a protected CR during a rolling restart.

### Orkestra's own infrastructure is protected too

Every resource the Helm chart creates carries the label `orkestra.io/deletion-protection: "true"`. The deletion protection webhook's second rule matches any resource bearing that label — regardless of kind. This means the operator's own `Deployment`, `Service`, `ServiceAccount`, `ClusterRoleBinding`, and TLS `Secret` cannot be deleted while deletion protection is active:

```bash
kubectl delete deployment orkestra -n orkestra-system
# Error from server: admission webhook "orkestra-delete-protection.orkestra.io" denied the request:
# deletion of Deployment "orkestra" is protected by Orkestra
```

Optional components (HPA, PDB, NetworkPolicy, Ingress) carry the same label and are protected automatically when deployed.

### cleanupOnShutdown

`cleanupOnShutdown: true` instructs Orkestra to remove all webhook configurations and TLS Secrets it created during graceful shutdown. This lets you decommission the operator cleanly without manual cleanup:

```bash
helm uninstall orkestra -n orkestra-system
# ValidatingWebhookConfiguration "orkestra-delete-protection" removed automatically
```

---

## Namespace protection

When `security.namespaceProtection.enabled: true` is set, Orkestra registers a `ValidatingWebhookConfiguration` that intercepts every `CREATE` and `UPDATE` request for managed CRs and rejects those targeting a forbidden namespace.

Two rule types are available:

```yaml
crds:
  app:
    allowedNamespaces:
      - production         # whitelist — only this namespace is accepted

  cache:
    restrictedNamespaces:
      - kube-system        # blacklist — these namespaces are rejected
      - kube-public
```

A CR applied to a namespace outside the allowlist — or inside the restricted list — is rejected before it is stored. The reconciler never sees it.

---

## Two enforcement points

Every security rule declared in the Katalog is enforced at two independent points:

**Point 1 — Admission time (webhook)**

The API server calls Orkestra's webhook before the object is written to etcd. A `deny` rule, namespace violation, or deletion attempt is blocked immediately. This is the fast gate.

**Point 2 — Reconcile time (reconciler)**

On every reconcile cycle, the reconciler re-applies mutation defaults, re-checks validation rules, and re-checks namespace rules before creating any child resource. If a CR somehow bypassed the webhook (e.g., the webhook was absent during a brief restart window), the reconciler will not act on it — no Deployment, no ConfigMap, no Service is created. The CR exists in the API server but Orkestra treats it as if it does not.

```
CR applied
    │
    ▼
[Admission webhook]  ←── fast gate: rejects bad CRs before etcd
    │ (passes)
    ▼
  etcd (stored)
    │
    ▼
[Reconciler]  ←── backstop: re-enforces all rules before any child resource is created
    │ (passes)
    ▼
  Child resources created
```

Both layers must independently fail before a rule is violated. In practice, the webhook handles all admission-time violations and the reconciler is the backstop for the narrow window where the webhook is absent.

Without `security.webhooks.admission.enabled`, `security.deletionProtection.enabled`, or `security.namespaceProtection.enabled`, only enforcement point 2 is active. Bad CRs can be stored in etcd but Orkestra will never act on them.

---

## Webhook self-healing

Orkestra's webhook controller watches its own `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` objects. If either is deleted — accidentally or by an attacker who gained API server access — Orkestra detects the change immediately via a Kubernetes Watch and recreates it.

### How it works

The controller runs two goroutines that establish persistent Watch streams on Orkestra-owned webhook configurations (filtered by `app.kubernetes.io/tag=orkestra-internal`). On any `DELETED` or `MODIFIED` event, a non-blocking signal is sent to a reconcile trigger channel:

```
Watch (fast path) ──┐
                    ├──▶ trigger channel (buffered, cap 1) ──▶ reconcileAll()
Safety poll (30 s) ─┘
```

The buffered channel coalesces bursts — any number of rapid events produce exactly one reconcile call. The reconcile functions are idempotent.

The safety ticker (configurable via `WEBHOOK_CONTROLLER_SYNC_INTERVAL`, default 30 seconds, minimum 1 second) runs in parallel as a backstop for drift that the Watch silently misses — partial mutations, stream drops on some managed clusters, and token expiry.

### In practice

```bash
kubectl delete validatingwebhookconfiguration orkestra-namespace-protection
```

Orkestra detects the deletion through the Watch stream and recreates the configuration immediately — typically within one API server round-trip, well under one second:

```
{"level":"warn","message":"webhook watch: configuration deleted — triggering reconcile"}
{"level":"info","message":"namespace protection webhook registered: orkestra-namespace-protection"}
```

---

## TLS

Orkestra exposes a single HTTPS server (`:8443`) for all webhook traffic — validation, mutation, deletion protection, and namespace protection. All endpoints share one certificate.

Orkestra generates and rotates its own TLS certificate automatically. No manual certificate management is required.

To supply your own certificate:

```bash
helm install orkestra orkestra/orkestra \
  --set tls.certFile=/path/to/tls.crt \
  --set tls.keyFile=/path/to/tls.key \
  --namespace orkestra-system
```

### Required SANs

```
DNS: orkestra.<namespace>.svc
DNS: orkestra.<namespace>.svc.cluster.local
```

---

## Credentials in sources

Authentication for external sources must use environment variables — credentials are never inlined in the Katalog:

```yaml
imports:
  - type: github
    auth:
      fromEnv: GITHUB_TOKEN
```

In Kubernetes, bind the environment variable to a Secret:

```yaml
env:
  - name: GITHUB_TOKEN
    valueFrom:
      secretKeyRef:
        name: orkestra-credentials
        key: github-token
```

---

## Supply chain security

### Binary verification

Every release binary is signed. Verify before use:

```bash
gpg --verify ork_linux_amd64.tar.gz.asc ork_linux_amd64.tar.gz
```

### Pattern pinning

Pin external patterns to a specific version, never `latest`:

```yaml
# Preferred — exact version
- url: ghcr.io/myorg/postgres-pattern@v1.2.0

# Preferred for Git sources — commit SHA
- url: https://github.com/myorg/registry@a3f9c1b

# Avoid — mutable tag
- url: ghcr.io/myorg/postgres-pattern@latest
```

---

## Template safety

Katalog templates use Go `text/template`. They:

- cannot execute shell commands
- cannot read from the filesystem
- cannot make network calls
- only interpolate values from the CR's fields

There is no eval surface exposed to template authors.

---

## Logging and audit

- Structured JSON logs via zerolog — every reconcile, webhook decision, and error is logged with context
- Kubernetes audit logs capture all API interactions independently of Orkestra's own logs
- The Control Center records every admission decision, namespace violation, and deletion attempt — accessible at `kubectl port-forward svc/orkestra-cc -n orkestra-system 8090:8090`

---

## Reporting vulnerabilities

Report security issues privately with:

- reproduction steps
- relevant logs
- impact assessment

Responsible disclosure is appreciated and protects all users.

---

## Summary

Orkestra's security model has five interlocking properties:

| Property | Mechanism |
|----------|-----------|
| Minimal attack surface | `!runtime` tag strips all non-operational CLI commands from the production binary |
| Least-privilege RBAC | Permissions derived from the Katalog — explicit, auditable, GitOps-compatible |
| Admission enforcement | Validation + mutation webhooks gate every CREATE/UPDATE before etcd |
| Runtime enforcement | Reconciler re-checks all rules before creating any child resource |
| Self-healing protection | Event-driven Watch recreates deleted webhook configurations within milliseconds |

The constraint is intentional: Orkestra only has the permissions required to do what your Katalog declares, enforces the rules you write at every point in the request lifecycle, and heals its own security surface automatically.
