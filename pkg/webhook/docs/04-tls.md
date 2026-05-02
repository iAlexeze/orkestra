# 04 — TLS

The HTTPS server requires TLS certificates. The webhook package never generates its own certificates — that is `cmd/internal.ensureSecurity`'s responsibility.

## How certs reach the webhook server

`ensureSecurity` runs in `konstructor.go` before `WebhookServer` is constructed. It writes the certificate paths back into `kfg.Security().Webhooks.TLSCert` and `.TLSKey`. `NewWebhookServer` reads them from konfig.

```go
// In konstructor.go:
if kat.NeedsCertificates() {
    tlsCert, tlsKey, certMgr, err = ensureSecurity(ctx, kfg, kat, kube)
    kfg.Security().Webhooks.TLSCert = tlsCert
    kfg.Security().Webhooks.TLSKey = tlsKey
}

ws := webhook.NewWebhookServer(kube.Clientset(), kat, kfg)
if certMgr != nil {
    ws.SetCertManager(certMgr)  // for cleanup on shutdown
}
```

## TLS modes

`ensureSecurity` implements three modes:

| Mode | Condition | Behaviour |
|------|-----------|-----------|
| External | `TLS_CERT` and `TLS_KEY` are set | Paths used as-is. Orkestra does not manage the cert lifecycle. |
| Self-signed | Neither env var is set, `NeedsCertificates()` is true | Generates a self-signed bundle, stores it in the `orkestra-tls` Secret, writes cert/key to temp files. CRD caBundle is patched automatically. |

## CRD caBundle patching

When a CRD declares `conversion.updateCRD: true`, `ensureSecurity` patches the CRD's `spec.conversion.webhook.clientConfig.caBundle` with the generated CA certificate. This is required for the Kubernetes API server to trust the conversion webhook TLS endpoint.

## Cert cleanup on shutdown

When `cleanupOnShutdown: true` is set in the Katalog and certs were auto-generated (not user-provided), `Shutdown()` deletes the `orkestra-internal-tls` Secret:

```go
if ws.certMgr != nil && kat.DeletionProtectionCleanupOnShutdown() {
    ws.certMgr.DeleteCertificateAndSecret(ctx, namespace, "orkestra-internal-tls")
}
```

`certMgr` is `nil` when the user provided their own certs — Orkestra never deletes user-managed cert material.
