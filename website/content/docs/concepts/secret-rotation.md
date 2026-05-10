# Secret Rotation and TLS Certificate Generation

Orkestra manages the full credential lifecycle declaratively — generation,
storage, and rotation — without external tools like cert-manager or Vault
for the common cases.

---

## The problem

Credentials have two failure modes:

**Never rotating:** A compromised credential stays compromised indefinitely.
A certificate that expires brings down the application.

**Rotating incorrectly:** Generating a new value on every reconcile cycle
breaks every application currently using the old value. The reconcile loop
runs every 30 seconds. Random generation inside it is catastrophically wrong.

The correct model: generate once, store durably, rotate when a threshold
is crossed. Check before generating. Annotate when generated.

---

## once: true — stable generation

```yaml
secrets:
  - name: "{{ .metadata.name }}-credentials"
    once: true
    data:
      password: "{{ randomAlphanumeric 32 }}"
```

Behaviour:
- **First reconcile:** Secret does not exist → evaluate template → create
- **Every subsequent reconcile:** Secret exists → skip entirely (no template evaluation)
- **CR deleted:** Secret deleted via owner reference (garbage collection)

---

## rotateAfter — time-based renewal

```yaml
secrets:
  - name: "{{ .metadata.name }}-credentials"
    once: true
    rotateAfter: 90d      # rotate every 90 days
    data:
      password: "{{ randomAlphanumeric 32 }}"
      apiKey:   "{{ randomHex 16 }}"
```

On each reconcile, Orkestra reads the `orkestra.orkspace.io/generated-at`
annotation on the Secret and compares it to `now - rotateAfter`. If the
threshold is crossed, the Secret is deleted and recreated with new values.

The annotation is the source of truth:
```yaml
metadata:
  annotations:
    orkestra.orkspace.io/generated-at: "2026-04-06T08:00:00Z"
    orkestra.orkspace.io/rotate-after: "90d"
```

**Duration format:** `30s`, `5m`, `12h`, `90d`, `1y`
Days and years are extensions of Go's standard duration format.

---

## TLS certificate generation

For webhook certificates, internal service mTLS, and any other TLS use case:

```yaml
secrets:
  - name: "{{ .metadata.name }}-tls"
    once: true
    rotateAfter: 1y
    tls:
      commonName: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc"
      dnsNames:
        - "{{ .metadata.name }}"
        - "{{ .metadata.name }}.{{ .metadata.namespace }}"
        - "{{ .metadata.name }}.{{ .metadata.namespace }}.svc"
        - "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
      validFor: 1y
```

Produces a `kubernetes.io/tls` Secret with three fields:
```
tls.crt  — PEM-encoded signed certificate
tls.key  — PEM-encoded private key
ca.crt   — PEM-encoded self-signed CA certificate
```

The CA is generated per-Secret. For operators that need a shared CA across
multiple services, declare one TLS Secret as the CA and reference it from
others (coming in a future version).

---

## Webhook certificate automation

For operators that use Orkestra's admission webhooks, certificate management
can be fully automated:

```yaml
# In the Katalog metadata section
webhooks:
  createCerts: true     # false by default
  certSecret: orkestra-tls
  rotateAfter: 1y
```

When `createCerts: true`:
1. Orkestra generates a self-signed CA and certificate for the webhook service
2. Stores them in `orkestra-tls` (or the declared name)
3. Patches the webhook configuration's `caBundle` field automatically
4. On each reconcile, checks the rotation threshold
5. When threshold is crossed: generates new cert, updates Secret, patches `caBundle`

The webhook service never goes down during rotation — the new certificate is
installed before the old one is removed from the configuration.

This replaces cert-manager for the common case of self-signed operator webhooks.
For production clusters where a corporate CA must sign the certificate,
continue using cert-manager or an external PKI.

---

## Comparison to alternatives

| Approach | Rotation | TLS | External dependency |
|---|---|---|---|
| `once: true` + `rotateAfter:` | ✅ Declarative | ✅ Built-in | None |
| cert-manager | ✅ Automatic | ✅ Full PKI | cert-manager operator |
| Vault Agent | ✅ Automatic | ✅ Full PKI | Vault cluster |
| Manual Secrets | ❌ Manual | ❌ Manual | None |

For operators that only need self-signed TLS (the majority of internal
operators), `rotateAfter:` + `tls:` eliminates the cert-manager dependency
entirely while keeping the certificate lifecycle declarative and observable
in the Control Center.

---

## Implementation status

- `once: true` — ✅ Implemented and tested
- `rotateAfter:` + annotation check — 🔲 Designed, not yet implemented
- `tls:` block (cert generation) — 🔲 Designed, not yet implemented
- `webhooks.createCerts: true` — 🔲 Designed, not yet implemented

The type definitions (`TLSSpec`, `ParseTimeDuration`, `NeedsRotation`)
are in `pkg/types/secret_rotation.go`. Implementation in `run_secrets.go`
follows the same check-before-act pattern as `once: true`.