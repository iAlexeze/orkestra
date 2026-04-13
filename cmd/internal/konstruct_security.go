// internal/construct_security.go
//
// Security wiring — called from konstructOrkestra after the HealthServer
// is constructed and before Orkestra starts.
//
// Handles:
//
//  1. TLS certificate generation — when deletionProtection is enabled and
//     no explicit cert is configured. Uses the same GenerateTLSBundle from
//     run_secrets_tls.go. Stores in the orkestra-tls Secret.
//
//  2. RBAC auto-apply — when security.rbac.enabled is true.
//     Applies ClusterRole, ClusterRoleBinding, ServiceAccount at startup.
//     Registers cleanup for shutdown when cleanupOnShutdown: true.
//
// Both operations are best-effort with fatal logging on failure — if RBAC
// or TLS cannot be applied, the operator cannot function correctly.
package internal

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/reconciler"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// orkestraTLSSecretName is the default Secret name for Orkestra's TLS cert.
	// Matches the default name in run_secrets_tls.go.
	orkestraTLSSecretName = "orkestra-tls"

	// Default cert validity — 1 year, rotated when deletionProtection is active.
	defaultCertValidFor = "1y"
)

// ensureSecurity applies RBAC and TLS certificates when security features
// are enabled in the Katalog. Called synchronously before Start().
//
// Order:
//  1. RBAC — must succeed before the operator can interact with the cluster
//  2. TLS  — must succeed before the HTTPS server starts (webhook endpoint)
func ensureSecurity(
	ctx context.Context,
	kfg *konfig.Konfig,
	kat *katalog.Katalog,
	kube *kubeclient.Kubeclient,
) (tlsCertFile, tlsKeyFile string, err error) {
	namespace := kfg.Cluster().Namespace // "ORKESTRA_NAMESPACE"

	// ── 1. RBAC auto-apply ────────────────────────────────────────────────────
	if kat.IsRBACEnabled() {
		saName := kat.Metadata().Name + "-operator"
		if saName == "-operator" {
			saName = "orkestra-operator"
		}

		bundle := kat.BuildRBACBundle(namespace, saName)
		if err := katalog.ApplyRBAC(ctx, kube.Clientset(), bundle); err != nil {
			logger.Fatal().Err(err).Msg("security: failed to apply RBAC — cannot start")
		}
		logger.Info().
			Str("clusterRole", bundle.ClusterRole.Name).
			Str("serviceAccount", saName).
			Msg("security: RBAC applied")
	}

	// ── 2. TLS certificate management ────────────────────────────────────────
	// When deletion protection is enabled, the HTTPS server needs TLS.
	// If the user has not configured explicit cert paths (via env vars),
	// Orkestra generates self-signed certs and stores them in orkestra-tls.
	//
	// If the user HAS configured TLS_CERT and TLS_KEY, those are used instead.
	// This preserves compatibility with existing cert-manager integrations.

	if kat.IsDeletionProtectionEnabled() {
		configuredCert := kfg.WebhookConfig().TLSCert // "TLS_CERT"
		configuredKey := kfg.WebhookConfig().TLSKey   // "TLS_KEY"

		if configuredCert != "" && configuredKey != "" {
			// User provided certs — use them as-is
			logger.Info().
				Str("cert", configuredCert).
				Msg("security: using provided TLS certificates for deletion protection webhook")
			return configuredCert, configuredKey, nil
		}

		// No explicit cert configured — generate self-signed
		logger.Info().Msg("security: generating TLS certificates for deletion protection webhook")

		serviceName := kfg.WebhookRegistration().ServiceName // "ORKESTRA_SERVICE_NAME"
		svcNamespace := namespace                            // "ORKESTRA_NAMESPACE"

		bundle, err := reconciler.GenerateTLSBundle(
			serviceName+"."+svcNamespace+".svc",
			[]string{
				serviceName,
				serviceName + "." + svcNamespace,
				serviceName + "." + svcNamespace + ".svc",
				serviceName + "." + svcNamespace + ".svc.cluster.local",
			},
			defaultCertValidFor,
		)
		if err != nil {
			logger.Fatal().Err(err).Msg("security: failed to generate TLS certificates")
		}

		// Write to temp files so the HTTPS server can use them
		// In a production deployment, these would be written to a Secret instead.
		// The Secret path is the preferred approach — avoids ephemeral temp files.
		certFile, keyFile, writeErr := writeTLSToFiles(bundle)
		if writeErr != nil {
			logger.Fatal().Err(writeErr).Msg("security: failed to write TLS certificates to files")
		}

		// Also store in the orkestra-tls Secret for the webhook caBundle
		if err := storeTLSSecret(ctx, kube, namespace, orkestraTLSSecretName, bundle); err != nil {
			logger.Warn().Err(err).
				Str("secret", orkestraTLSSecretName).
				Msg("security: failed to store TLS secret — webhook caBundle may be unavailable")
		}

		logger.Info().
			Str("cert", certFile).
			Str("secret", orkestraTLSSecretName).
			Msg("security: TLS certificates generated and stored")

		return certFile, keyFile, nil
	}

	return "", "", nil
}

// writeTLSToFiles writes the TLS bundle to temporary files.
// Returns the cert file path and key file path.
func writeTLSToFiles(bundle *reconciler.TLSBundle) (certFile, keyFile string, err error) {
	cert, err := os.CreateTemp("", "orkestra-tls-cert-*.pem")
	if err != nil {
		return "", "", err
	}
	if _, err := cert.Write(bundle.CertPEM); err != nil {
		return "", "", err
	}
	cert.Close()

	key, err := os.CreateTemp("", "orkestra-tls-key-*.pem")
	if err != nil {
		return "", "", err
	}
	if _, err := key.Write(bundle.KeyPEM); err != nil {
		return "", "", err
	}
	key.Close()

	return cert.Name(), key.Name(), nil
}

// storeTLSSecret creates or updates the orkestra-tls Secret with the generated
// TLS bundle. The Secret holds three keys:
//
//	tls.crt — PEM-encoded signed certificate (server cert)
//	tls.key — PEM-encoded private key
//	ca.crt  — PEM-encoded CA certificate (used as caBundle in webhook config)
//
// Strategy: always overwrite on startup when certs were auto-generated.
// Orkestra regenerates certs each time it starts without explicit TLS_CERT/TLS_KEY.
// The HTTPS server, the webhook caBundle, and the Secret all receive the same
// freshly-generated cert — they stay consistent.
//
// If the Secret cannot be created or updated, we log a warning and continue.
// The HTTPS server will still work (it uses the temp files); the webhook
// caBundle update may fail, but the handler will still run correctly until
// the webhook config is reconciled on next startup.
func storeTLSSecret(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	namespace, secretName string,
	bundle *reconciler.TLSBundle,
) error {
	client := kube.Clientset().CoreV1().Secrets(namespace)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "orkestra",
				"app.kubernetes.io/component":  "tls",
			},
			Annotations: map[string]string{
				// Record when this cert was generated so future rotation
				// logic (rotateAfter) can compare against it.
				"orkestra.konductor.io/generated-at": time.Now().UTC().Format(time.RFC3339),
			},
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": bundle.CertPEM,
			"tls.key": bundle.KeyPEM,
			"ca.crt":  bundle.CACertPEM,
		},
	}

	// Try create first
	_, err := client.Create(ctx, secret, metav1.CreateOptions{})
	if err == nil {
		return nil
	}

	// Already exists — update it
	if !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating tls secret %s/%s: %w", namespace, secretName, err)
	}

	existing, err := client.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting existing tls secret %s/%s: %w", namespace, secretName, err)
	}

	// Preserve resource version for the update
	secret.ResourceVersion = existing.ResourceVersion

	_, err = client.Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("updating tls secret %s/%s: %w", namespace, secretName, err)
	}

	return nil
}
