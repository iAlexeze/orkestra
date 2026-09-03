# 04 — TLS

The webhook HTTPS server requires a certificate and private key. The webhook package never generates certificates — that responsibility belongs to `cmd/internal.ensureSecurity`, which runs before `WebhookServer` is constructed.

---

## TLS modes

`ensureSecurity` resolves one of three modes based on konfig:

| Mode | Condition | Behaviour |
|------|-----------|-----------|
| **Disabled** | `NeedsCertificates()` is false | No TLS setup. Returns empty paths and nil certMgr. |
| **External** | `TLS_CERT` and `TLS_KEY` env vars are set | Paths passed through as-is. Orkestra does not manage the cert lifecycle. `certMgr` and `bundle` are nil. |
| **Self-signed** | Neither env var is set and `NeedsCertificates()` is true | Generates a self-signed bundle, stores it in the `orkestra-tls` Secret, writes cert/key to temp files. Returns `certMgr` and `bundle`. |

`NeedsCertificates()` is true when any of the following are declared in the Katalog: deletion protection, admission webhooks, conversion webhooks.

---

## Self-signed bundle

In self-signed mode, `certmanager.GenerateTLSBundle` produces a self-signed CA and a server certificate signed by it:

| Material | Secret key | Use |
|----------|------------|-----|
| Server certificate (PEM) | `tls.crt` | Served by the HTTPS server |
| Server private key (PEM) | `tls.key` | Served by the HTTPS server |
| CA certificate (PEM) | `ca.crt` | Injected as `caBundle` in webhook configurations |

The bundle is stored in a `kubernetes.io/tls` Secret named `orkestra-tls` in the Orkestra namespace. The Secret carries the deletion-protection label so accidental `kubectl delete` is blocked by the webhook itself.

`writeTLSToFiles` writes `tls.crt` and `tls.key` to OS temp files. The paths are injected into `kfg.Security().Webhooks.TLSCert` and `.TLSKey` before `NewWebhookServer` is called.

```
ensureSecurity()
  → certmanager.EnsureCertificate()    ← creates or reuses Secret in cluster
  → writeTLSToFiles(bundle)             ← writes cert+key to /tmp/orkestra-tls-*.pem
  → kfg.Security().Webhooks.TLSCert = certFile
  → kfg.Security().Webhooks.TLSKey  = keyFile
  → return certFile, keyFile, certMgr, bundle
```

`EnsureCertificate` is idempotent: if a valid Secret already exists it is reused without regeneration. If the Secret exists but is malformed, it is deleted and a new bundle is generated. This is the normal path during rolling restarts — the incoming pod reuses the same cert the outgoing pod was serving.

---

## How cert material flows to WebhookServer

```go
// gateway.go
tlsCert, tlsKey, certMgr, tlsBundle, err := ensureSecurity(ctx, kfg, kat, kube)

kfg.Security().Webhooks.TLSCert = tlsCert
kfg.Security().Webhooks.TLSKey  = tlsKey

ws := webhook.NewWebhookServer(kube.Clientset(), kat, kfg)

if certMgr != nil {
    ws.SetCertManager(certMgr)   // enables Secret cleanup on graceful shutdown
}
if tlsBundle != nil {
    ws.SetCertBundle(             // enables housekeeper Secret reconciliation
        tlsBundle.CertPEM, tlsBundle.KeyPEM, tlsBundle.CACertPEM,
        certmanager.DefaultTLSSecretName, kfg.Cluster().Namespace,
    )
}
```

Both setters are no-ops in external mode (`certMgr` and `tlsBundle` are nil). Orkestra never touches user-managed cert material.

---

## caBundle injection

Two places consume the CA certificate:

**Webhook configurations** — `readCABundle` reads `TLSCertFile` and passes the bytes as `caBundle` when creating or updating `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` objects. Called by every `register*Webhook` function in the housekeeper.

**CRD conversion** — `patchConversionCRDs` base64-encodes `bundle.CACertPEM` and patches `spec.conversion.webhook.clientConfig.caBundle` on every CRD that declares `conversion.updateCRD: true`. Required for the Kubernetes API server to verify the conversion endpoint's TLS certificate.

Both injections use the same cert material. If the cert is regenerated, both must be updated — the housekeeper handles all of it:

- Webhook configurations (`reconcileAdmissionWebhooks`, `reconcileDeletionProtectionWebhook`, etc.) — re-registered on every reconcile cycle using the current cert file.
- CRD conversion caBundle (`reconcileCRDConversionWebhooks`) — re-patched on every reconcile cycle via the `ConversionCRDPatchFn` registered by `WireWebhookHousekeeperInfra`. A dedicated watcher per conversion CRD (`watchSingleConversionCRD`) triggers an immediate reconcile on any MODIFIED event so a stripped caBundle is restored within one API round-trip.
- Namespace labels (`reconcileNamespaceLabels`) — re-applied on every reconcile cycle to ensure the deletion-protection webhook's ObjectSelector continues to match the operator namespace.

---

## Cleanup on shutdown

When `cleanupOnShutdown: true` is set and certs were auto-generated, `Shutdown()` deletes the Secret:

```go
if ws.certMgr != nil && shouldCleanupTLS {
    ws.certMgr.DeleteCertificateAndSecret(ctx, namespace, secretName)
}
```

This is safe for clean teardown. During rolling restarts it creates a hazard: the outgoing pod deletes a Secret the incoming pod depends on. The housekeeper's `reconcileCertSecret` exists specifically to recover from this. See [06-cert-lifecycle.md](06-cert-lifecycle.md).

---

→ Next: [05-conversion-notes.md](05-conversion-notes.md) | [06-cert-lifecycle.md](06-cert-lifecycle.md)
