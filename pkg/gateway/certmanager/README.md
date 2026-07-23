# pkg/gateway/certmanager

`certmanager` owns the TLS certificate lifecycle for Orkestra's webhook server. It generates self-signed certificates when the operator has not been given explicit TLS paths, stores them in a Kubernetes Secret, and cleans them up on graceful shutdown.

```go
mgr := certmanager.New(kube.Clientset())

bundle, err := mgr.EnsureCertificate(ctx, certmanager.CertificateSpec{
    ServiceName: kfg.Security().ServiceName,
    Namespace:   kfg.Cluster().Namespace,
    SecretName:  certmanager.DefaultTLSSecretName,
    ValidFor:    "1y",
    BaseLabels:  labels.OrkestraBaseLabels(),
})
```

## Secret shape

The generated Secret is of type `kubernetes.io/tls` and carries three keys:

| Key | Content |
|-----|---------|
| `tls.crt` | PEM-encoded signed server certificate |
| `tls.key` | PEM-encoded server private key |
| `ca.crt` | PEM-encoded CA certificate (used as `caBundle` in webhook configurations) |

The Secret is labelled with `orkestra.io/deletion-protection=true` so that Orkestra's own admission webhook will reject accidental delete requests against it.

## Idempotency

`EnsureCertificate` checks whether the Secret already exists before generating a new certificate. If the Secret is present and its certificate has not expired, the existing bundle is returned without any API mutations. This means the method is safe to call on every startup — it will not generate a new certificate on every restart.

## Shutdown cleanup

When `SecurityConfig.DeletionProtection.CleanupOnShutdown` is `true`, the health server calls `DeleteCertificateAndSecret` during `Shutdown()`. A `NotFound` error is silently ignored, so cleanup is idempotent across restarts.

## When Orkestra generates its own certificates

Without `TLS_CERT` / `TLS_KEY` ENV vars:
1. `konfig.Init()` leaves `Security().Webhooks.TLSCert` and `TLSKey` empty.
2. The gateway calls `EnsureCertificate()` at startup.
3. `EnsureCertificate()` generates the bundle and stores it in the Kubernetes Secret.
4. The Katalog loader writes the cert and key paths back into `kfg.Security()`.
5. The webhook server reads the populated paths from `kfg.Security()`.

No shared file system or init containers required.
