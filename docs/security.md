# Security

Orkestra is designed to run safely in production environments. This page covers
the security model, what permissions Orkestra needs and why, how to scope them,
and how to report vulnerabilities.

!!! note
    No security vulnerabilities have been reported for Orkestra at this time.
    This page exists to help you adopt secure practices from day one.

---

## The permission model

Orkestra needs broad permissions because it is a general-purpose operator runtime.
It watches and manages whatever CRDs are declared in the Katalog — which means
it needs read and write access to those resource types.

The default ClusterRole grants:

```yaml
rules:
  # Manage any resource declared in the Katalog
  - apiGroups: ["*"]
    resources: ["*"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

  # Leader election lease
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    verbs: ["get", "create", "update"]

  # Emit Kubernetes events on managed CRs
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["create", "patch"]

  # Register webhook configurations (when ENABLE_WEBHOOKS=true)
  - apiGroups: ["admissionregistration.k8s.io"]
    resources:
      - validatingwebhookconfigurations
      - mutatingwebhookconfigurations
    verbs: ["get", "create", "update", "patch"]
```

### Scoping permissions

If you know exactly which resource types Orkestra will manage, replace the
`["*"]` rule with explicit rules:

```yaml
rules:
  - apiGroups: ["demo.orkestra.io"]
    resources: ["websites", "applications"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: [""]
    resources: ["services", "secrets", "configmaps"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

This is more work to maintain but reduces blast radius if Orkestra is compromised.

### Namespace restriction

Use `restrictedNamespaces` in the Katalog to prevent Orkestra from creating child
resources in sensitive namespaces:

```yaml
- name: website
  restrictedNamespaces:
    - kube-system
    - kube-*
    - "*-system"
```

Restrictions declared in a Komposer apply to all CRDs in that Komposer and cannot
be removed by a CRD-level declaration. Restrictions are additive — they only add
constraints, never remove them.

!!! tip "Restrict by default"
    Add `kube-system`, `kube-public`, and your monitoring namespace to
    `restrictedNamespaces` for every CRD. These namespaces rarely need
    operator-managed resources, and restricting them reduces the attack surface
    if a CR is maliciously crafted.

---

## TLS certificates

Orkestra's HTTPS server (`:8443`) serves the conversion webhook, validation webhook,
and mutation webhook. All three share one certificate. The API server uses the
certificate's CA as the `caBundle` in the webhook configurations.

Three options:

| Approach | Suitable for |
|---|---|
| Self-signed (via [generate-certs.sh](./guides/user-guide/generate-certs.sh) or [Follow along here](./guides/user-guide/self-signed-certificate-with-openssl.md)) | Development and local testing only |
| [cert-manager](./guides/user-guide/self-signed-certificate-with-cert-manager.md) `Certificate` resource | Production — automated rotation |
| External PKI / corporate CA | Enterprise environments with existing certificate infrastructure |

!!! warning "Self-signed certificates in production"
    Self-signed certificates cannot be revoked. They require manual CA distribution
    to every node that the API server runs on. Do not use them in production.
    Use cert-manager or your organisation's PKI.

### Certificate SANs

The certificate must include SANs for:

```
DNS: orkestra.<namespace>.svc
DNS: orkestra.<namespace>.svc.cluster.local
```

Where `<namespace>` is the namespace where Orkestra runs. The Helm chart generates
the correct cert-manager `Certificate` resource automatically.

### Certificate rotation

When using cert-manager, the certificate rotates automatically before expiry.
Orkestra reads the certificate from the mounted Secret — a rolling restart picks
up the new certificate. No manual intervention needed.

If you manage certificates manually, rotate before expiry and restart Orkestra
to load the new certificate.

---

## Credentials in sources

Komposer sources that require authentication — private Git repositories, private
OCI registries — must never have credentials in YAML. Use environment variable
references:

```yaml
sources:
  registry:
    - url: https://github.com/myorg/private-registry@main
      auth:
        type: github
        fromEnv: GITHUB_TOKEN   # resolved at startup, never stored

  files:
    - url: https://private.myorg.io/katalog.yaml
      auth:
        type: bearer
        fromEnv: PLATFORM_TOKEN
```

In cluster deployments, inject credentials from Kubernetes Secrets:

```yaml
env:
  - name: GITHUB_TOKEN
    valueFrom:
      secretKeyRef:
        name: orkestra-credentials
        key: github-token
```

!!! warning "Never commit credentials"
    `fromEnv` is not optional — it is the only way the auth block works.
    There is no field for a literal token value. This is intentional.

---

## Admission webhook security

When `ENABLE_WEBHOOKS=true`, Orkestra registers a `ValidatingWebhookConfiguration`
and optionally a `MutatingWebhookConfiguration`. These tell the API server to call
Orkestra during `kubectl apply`.

**FailurePolicy:** Defaults to `Ignore`. If Orkestra is unreachable, the API server
allows the operation through and reconcile-time validation catches violations later.
Set `Fail` only with multiple Orkestra replicas and a PodDisruptionBudget.

**Scope:** Orkestra only registers webhook rules for CRDs that have validation or
mutation rules declared. CRDs without rules are not included — no unnecessary
API server calls.

**TLS:** The same certificate used for conversion serves the admission endpoints.
One certificate, one server, one trust relationship with the API server.

---

## Supply chain security

### Orkestra binary

Every Orkestra release is GPG-signed. Verify before installing:

```bash
# Import the public key (once)
curl -sSL https://github.com/iAlexeze/orkestra/releases/download/v1.0.0/orkestra-public-key.asc \
  | gpg --import

# Verify the binary
gpg --verify ork_linux_amd64.tar.gz.asc ork_linux_amd64.tar.gz
# gpg: Good signature from "Orkestra Releases <releases@orkestra.io>"
```

### Registry patterns

Pin patterns to specific versions, not `latest`:

```yaml
sources:
  registry:
    # Good — pinned, immutable
    - url: ghcr.io/konduktor-io/orkestra-registry/postgres@v14.2.0
      oci: true

    # Avoid — tracks whatever the author published last
    - url: ghcr.io/konduktor-io/orkestra-registry/postgres@latest
      oci: true
```

OCI version tags in the Orkestra registry are immutable — `postgres:v14.2.0`
cannot be overwritten once published.

For Git-based registries, pin to a commit SHA rather than a branch for the
strongest immutability guarantee:

```yaml
- url: https://github.com/myorg/registry@abc123def456
```

### Template expressions

Katalog templates are evaluated by Go's `text/template` against the live CR
object. Template expressions cannot execute system commands or access the
filesystem — Go `text/template` does not provide those capabilities. The template
context is limited to the CR's fields.

Validate third-party Katalogs with `ork validate` before running them. Review
any Go hooks or constructors they reference — these are arbitrary code.

!!! warning "Review external hooks"
    A `hooks.location` in a Katalog points to arbitrary Go code that runs in
    your cluster with Orkestra's permissions. Treat external hook modules with
    the same scrutiny as any third-party dependency. Prefer patterns from the
    official OrkestraRegistry for CRDs managed by the community.

---

## Logging and audit

Orkestra emits structured logs via zerolog. Every reconcile cycle is logged at
debug level. Errors are logged at error level with the CR name, CRD, and error
details.

Forward logs to your centralised system. Enable structured log parsing to extract
the `crd`, `cr`, and `error` fields for alerting.

Kubernetes audit logging captures every API call Orkestra makes — resource
creates, updates, deletes, and event emissions. Enable audit logging in your
cluster for a full audit trail of Orkestra's actions.

---

## Reporting vulnerabilities

!!! warning "Do not open public GitHub issues for security vulnerabilities"

If you discover a potential security issue in Orkestra:

- Contact the maintainers privately before disclosure
- Include steps to reproduce the issue
- Include logs or details that help assess the severity and scope
- Allow reasonable time for a fix before public disclosure

Responsible disclosure protects the community. Maintainer contact details are
in the repository.