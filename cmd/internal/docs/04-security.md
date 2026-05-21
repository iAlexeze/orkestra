# 04 — Security

`ensureSecurity` in `internal/security.go` applies TLS and namespace labeling
before any komponent starts. It is called synchronously from `KonductGateway`.

## What ensureSecurity does

`ensureSecurity` performs three operations in order.

First, if deletion protection is enabled, it patches the Orkestra namespace with
the standard Orkestra resource labels. The deletion-protection webhook uses an
`ObjectSelector` to narrow which resources it intercepts. Kubernetes namespaces
are cluster-scoped and carry no labels by default, so the webhook would never
fire for the Orkestra namespace unless the label is applied explicitly.

Second, it checks whether TLS is required. TLS is required when deletion
protection, admission webhooks, or conversion webhooks are enabled
(`kat.NeedsCertificates()`). If TLS is not required, `ensureSecurity` returns
immediately with empty paths and no cert manager.

Third, if TLS is required and the user has provided explicit `TLS_CERT` and
`TLS_KEY` environment variables, those paths are used as-is. No certificate is
generated and no `Secret` is written.

If no explicit cert is configured, `ensureSecurity` calls
`certmanager.EnsureCertificate`. The cert manager generates a self-signed CA and
leaf certificate, stores the bundle in the `orkestra-tls` Kubernetes `Secret`,
and writes the cert and key to temporary files. The file paths are written into
`kfg.Security().Webhooks.TLSCert` and `kfg.Security().Webhooks.TLSKey` so the
webhook server picks them up when it binds its HTTPS listener.

After generating the certificate, `ensureSecurity` patches each enabled CRD that
declares `conversion.updateCRD: true`. It writes the CA bundle into
`spec.conversion.webhook.clientConfig.caBundle` so the Kubernetes API server can
verify the webhook's TLS certificate during conversion requests.

```
ensureSecurity
    ├── ensureNamespaceLabeled   (deletion protection only)
    ├── kat.NeedsCertificates()  → return early if false
    ├── kfg: TLS_CERT + TLS_KEY  → return early if provided
    ├── certmanager.EnsureCertificate → orkestra-tls Secret
    ├── writeTLSToFiles          → /tmp/orkestra-tls-cert-*.pem
    └── patchConversionCRDs      → caBundle into CRD spec
```

## Why gateway-only

`ensureSecurity` is called only from `KonductGateway`, not from `konstructRuntime`.

The runtime does not serve webhooks. It has no HTTPS listener and no need for
TLS certificates. Calling `ensureSecurity` from the runtime would generate
certificates and patch CRDs unnecessarily, and would require the runtime
service account to hold permissions it does not otherwise need.

The gateway is the single source of truth for TLS in the Orkestra deployment.
When running the gateway alongside the runtime, only the gateway pod modifies
the `orkestra-tls` Secret and the CRD conversion webhook config.

## Skipped outside a pod

`KonductGateway` calls `utils.IsRunningInCluster()` before calling
`ensureSecurity` and exits immediately if the check fails. Outside a Kubernetes
pod there is no service account token, no API server to patch, and no webhook
endpoint that can be reached by the Kubernetes control plane.

→ Back: [01-overview.md](01-overview.md)
