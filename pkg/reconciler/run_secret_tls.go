// pkg/reconciler/run_secrets_tls.go
//
// TLS certificate generation and secret rotation for Orkestra secrets.
//
// Extends the once: true pattern with:
//   - rotateAfter: <duration> — time-based rotation using a generated-at annotation
//   - tls: {...}              — self-signed CA + signed certificate generation
//
// The generated Secret is annotated with:
//
//	orkestra.orkspace.io/generated-at: "2026-04-06T08:00:00Z"
//	orkestra.orkspace.io/rotate-after: "90d"
//
// On each reconcile, secretNeedsRotation reads these annotations and returns
// true when the threshold is crossed. The caller deletes and recreates.
//
// TLS generation produces a Secret of type kubernetes.io/tls with:
//
//	tls.crt — PEM-encoded signed certificate
//	tls.key — PEM-encoded private key (for the server)
//	ca.crt  — PEM-encoded self-signed CA certificate
//
// The CA is unique per Secret — appropriate for self-signed operator webhooks.
// For shared CAs across multiple services, store the CA in a separate Secret
// and reference it (planned: ca.secretRef field).
package reconciler

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// secretNeedsRotation returns true when the Secret exists but has exceeded
// its rotation threshold. Reads the generated-at annotation and compares
// to the declared rotateAfter duration.
//
// Returns false (no rotation) when:
//   - rotateAfter is not set
//   - Secret does not exist
//   - Generated-at annotation is missing or unparseable → regenerate to be safe
func secretNeedsRotation(ctx context.Context, kube *kubeclient.Kubeclient, namespace, name, rotateAfter string) (bool, error) {
	if rotateAfter == "" {
		return false, nil
	}

	secret, err := kube.Clientset().CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{
		ResourceVersion: "0", // watch cache
	})
	if err != nil {
		if isNotFoundErr(err) {
			return false, nil // does not exist — needs creation, not rotation
		}
		return false, fmt.Errorf("checking secret %s/%s for rotation: %w", namespace, name, err)
	}

	generatedAt := secret.Annotations[orktypes.AnnotationGeneratedAt]
	if generatedAt == "" {
		// Secret exists but has no annotation — was it created outside Orkestra?
		// Annotate it now and start the rotation clock from this reconcile.
		logger.FromContext(ctx).Warn().
			Str("secret", name).
			Msg("secret missing generated-at annotation — annotating now, rotation clock starts")
		return false, annotateSecret(ctx, kube, namespace, name, rotateAfter)
	}

	return orktypes.NeedsRotation(generatedAt, rotateAfter), nil
}

// annotateSecret writes the generated-at and rotate-after annotations onto
// an existing Secret. Used when a Secret exists but was created without annotations.
func annotateSecret(ctx context.Context, kube *kubeclient.Kubeclient, namespace, name, rotateAfter string) error {
	secret, err := kube.Clientset().CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{
		ResourceVersion: "0",
	})
	if err != nil {
		return err
	}
	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}
	secret.Annotations[orktypes.AnnotationGeneratedAt] = time.Now().UTC().Format(time.RFC3339)
	secret.Annotations[orktypes.AnnotationRotateAfter] = rotateAfter
	_, err = kube.Clientset().CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	return err
}

// deleteSecretForRotation deletes a Secret so it can be recreated with fresh values.
// Called when secretNeedsRotation returns true. The next create call in run_secrets.go
// then generates new credentials and annotates with the current time.
func deleteSecretForRotation(ctx context.Context, kube *kubeclient.Kubeclient, namespace, name string) error {
	err := kube.Clientset().CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !isNotFoundErr(err) {
		return fmt.Errorf("deleting secret %s/%s for rotation: %w", namespace, name, err)
	}
	return nil
}

// generationAnnotations returns the annotations to add to a freshly generated Secret.
func generationAnnotations(rotateAfter string) map[string]string {
	annotations := map[string]string{
		orktypes.AnnotationGeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if rotateAfter != "" {
		annotations[orktypes.AnnotationRotateAfter] = rotateAfter
	}
	return annotations
}

// ─────────────────────────────────────────────────────────────────────────────
// TLS certificate generation
// ─────────────────────────────────────────────────────────────────────────────

// TLSBundle holds the generated certificate material.
type TLSBundle struct {
	CertPEM   []byte // tls.crt — signed server certificate, PEM
	KeyPEM    []byte // tls.key — server private key, PEM
	CACertPEM []byte // ca.crt  — CA certificate, PEM (for caBundle in webhooks)
}

// GenerateTLSBundle generates a self-signed CA and a server certificate signed by it.
// The server certificate has the given common name and DNS SANs.
// validFor is the certificate validity duration ("1y", "90d", etc.).
//
// Returns a TLSBundle containing PEM-encoded cert, key, and CA cert.
// All three are stored in the Secret so consumers have what they need:
//   - tls.crt + tls.key for the server
//   - ca.crt for clients that need to verify the server cert
func GenerateTLSBundle(commonName string, dnsNames []string, validFor string) (*TLSBundle, error) {
	validity := 365 * 24 * time.Hour // default: 1 year
	if validFor != "" {
		if d, err := orktypes.ParseRotationDuration(validFor); err == nil {
			validity = d
		}
	}

	// ── Step 1: Generate CA ───────────────────────────────────────────────────
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generating CA key: %w", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: randomSerial(),
		Subject: pkix.Name{
			CommonName:   "orkestra-ca",
			Organization: []string{"Orkestra"},
		},
		NotBefore:             time.Now().Add(-5 * time.Minute), // clock skew tolerance
		NotAfter:              time.Now().Add(validity + 24*time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("creating CA certificate: %w", err)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return nil, fmt.Errorf("parsing CA certificate: %w", err)
	}

	// ── Step 2: Generate server key + cert signed by the CA ───────────────────
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generating server key: %w", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: randomSerial(),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Orkestra"},
		},
		DNSNames:    dnsNames,
		NotBefore:   time.Now().Add(-5 * time.Minute),
		NotAfter:    time.Now().Add(validity),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("creating server certificate: %w", err)
	}

	// ── Step 3: PEM encode everything ─────────────────────────────────────────
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCertDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	return &TLSBundle{
		CertPEM:   certPEM,
		KeyPEM:    keyPEM,
		CACertPEM: caCertPEM,
	}, nil
}

// createTLSSecret creates a kubernetes.io/tls Secret from a TLSBundle.
// The Secret name defaults to "orkestra-tls" when src.Name resolves to "".
func createTLSSecret(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	owner domain.Object,
	name, namespace, rotateAfter string,
	bundle *TLSBundle,
) error {
	if name == "" {
		name = "orkestra-tls"
	}

	annotations := generationAnnotations(rotateAfter)

	controller := true
	blockOwner := true
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
			Labels: map[string]string{
				"orkestra-owner":               owner.GetName(),
				"app.kubernetes.io/managed-by": "orkestra",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         owner.GetObjectKind().GroupVersionKind().GroupVersion().String(),
					Kind:               owner.GetObjectKind().GroupVersionKind().Kind,
					Name:               owner.GetName(),
					UID:                owner.GetUID(),
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwner,
				},
			},
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": bundle.CertPEM,
			"tls.key": bundle.KeyPEM,
			"ca.crt":  bundle.CACertPEM,
		},
	}

	_, err := kube.Clientset().CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("creating TLS secret %s/%s: %w", namespace, name, err)
	}

	logger.FromContext(ctx).Info().
		Str("secret", name).
		Str("namespace", namespace).
		Str("commonName", "").
		Msg("TLS secret created")

	return nil
}

func randomSerial() *big.Int {
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	return serial
}
