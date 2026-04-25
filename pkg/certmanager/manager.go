// Package certmanager centralises the TLS certificate lifecycle for Orkestra.
//
// Orkestra generates self-signed TLS certificates when security features
// (deletion protection, admission webhooks, conversion webhooks) are enabled and
// the operator has not been given explicit TLS_CERT/TLS_KEY paths. This package
// owns that lifecycle: generation, Secret storage, and optional deletion on
// graceful shutdown.
//
// # Architecture
//
// Manager is the public interface; k8sManager is its only production
// implementation. The separation lets tests inject a fake without importing
// client-go fakes into every caller.
//
// # Secret shape
//
// The generated Secret is of type kubernetes.io/tls and carries three keys:
//
//	tls.crt — PEM-encoded signed server certificate
//	tls.key — PEM-encoded server private key
//	ca.crt  — PEM-encoded CA certificate (used as caBundle in webhook configs)
//
// The Secret is labelled with the deletion-protection label so that Orkestra's
// own admission webhook will reject accidental delete requests against it.
//
// # Shutdown cleanup
//
// When DeletionProtection.CleanupOnShutdown is true, the HealthServer calls
// DeleteCertificateAndSecret during Shutdown(). A NotFound error is silently
// ignored — the operator may have been restarted without the Secret present.
//
// # Usage
//
//	mgr := certmanager.New(kube.Clientset())
//	bundle, err := mgr.EnsureCertificate(ctx, certmanager.CertificateSpec{
//	    ServiceName: "orkestra",
//	    Namespace:   "orkestra-system",
//	    SecretName:  certmanager.DefaultTLSSecretName,
//	    ValidFor:    "1y",
//	    BaseLabels:  kfg.OrkestraResourceLabels(),
//	})
package certmanager

import (
	"context"
	"fmt"
	"time"

	orklabels "github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/reconciler"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DefaultTLSSecretName is the Secret name used for Orkestra's auto-generated TLS bundle.
const DefaultTLSSecretName = "orkestra-tls"

// CertificateSpec describes the TLS certificate Orkestra should generate and store.
type CertificateSpec struct {
	// ServiceName is the Kubernetes Service that will serve the certificate (e.g. "orkestra").
	ServiceName string
	// Namespace is the namespace where the Service and Secret live.
	Namespace string
	// SecretName is the name of the Secret to create or update.
	SecretName string
	// ValidFor is the certificate validity duration string ("1y", "90d", etc.).
	ValidFor string
	// BaseLabels are the labels to apply to the Secret in addition to the deletion-protection label.
	BaseLabels map[string]string
}

// Manager handles TLS certificate generation and Secret lifecycle.
type Manager interface {
	// EnsureCertificate generates a TLS bundle and stores it in a Kubernetes Secret.
	// If the Secret already exists it is updated in-place.
	EnsureCertificate(ctx context.Context, spec CertificateSpec) (*reconciler.TLSBundle, error)
	// DeleteCertificateAndSecret removes the TLS Secret from the cluster.
	// A NotFound error is silently ignored.
	DeleteCertificateAndSecret(ctx context.Context, namespace, secretName string) error
}

type k8sManager struct {
	client kubernetes.Interface
}

// New returns a Manager backed by the given Kubernetes client.
func New(client kubernetes.Interface) Manager {
	return &k8sManager{client: client}
}

// EnsureCertificate generates a self-signed TLS bundle and stores it in a Secret.
func (m *k8sManager) EnsureCertificate(ctx context.Context, spec CertificateSpec) (*reconciler.TLSBundle, error) {
	bundle, err := reconciler.GenerateTLSBundle(
		spec.ServiceName+"."+spec.Namespace+".svc",
		[]string{
			spec.ServiceName,
			spec.ServiceName + "." + spec.Namespace,
			spec.ServiceName + "." + spec.Namespace + ".svc",
			spec.ServiceName + "." + spec.Namespace + ".svc.cluster.local",
		},
		spec.ValidFor,
	)
	if err != nil {
		return nil, fmt.Errorf("generating tls bundle: %w", err)
	}

	secretLabels := orklabels.WithDeletionProtection(spec.BaseLabels)
	secretLabels["app.kubernetes.io/component"] = "tls"

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.SecretName,
			Namespace: spec.Namespace,
			Labels:    secretLabels,
			Annotations: map[string]string{
				"orkestra.orkspace.io/generated-at": time.Now().UTC().Format(time.RFC3339),
			},
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": bundle.CertPEM,
			"tls.key": bundle.KeyPEM,
			"ca.crt":  bundle.CACertPEM,
		},
	}

	client := m.client.CoreV1().Secrets(spec.Namespace)
	_, err = client.Create(ctx, secret, metav1.CreateOptions{})
	if err == nil {
		return bundle, nil
	}
	if !k8serrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("creating tls secret %s/%s: %w", spec.Namespace, spec.SecretName, err)
	}

	existing, err := client.Get(ctx, spec.SecretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting existing tls secret %s/%s: %w", spec.Namespace, spec.SecretName, err)
	}

	secret.ResourceVersion = existing.ResourceVersion
	if _, err = client.Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		return nil, fmt.Errorf("updating tls secret %s/%s: %w", spec.Namespace, spec.SecretName, err)
	}

	return bundle, nil
}

// DeleteCertificateAndSecret removes the TLS Secret. Silently succeeds if already gone.
func (m *k8sManager) DeleteCertificateAndSecret(ctx context.Context, namespace, secretName string) error {
	err := m.client.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("deleting tls secret %s/%s: %w", namespace, secretName, err)
	}
	return nil
}
