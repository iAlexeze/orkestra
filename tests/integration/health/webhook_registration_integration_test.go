//go:build integration

package health_test

import (
	"context"
	"os"
	"testing"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestWebhookRegistration_Integration(t *testing.T) {
	ctx := context.Background()

	// Start envtest API server
	testEnv := &envtest.Environment{}
	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("envtest start failed: %v", err)
	}
	defer testEnv.Stop()

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("client init failed: %v", err)
	}

	// Fake CA bundle
	tmpCert := "testdata/tls.crt"
	if err := os.WriteFile(tmpCert, []byte("FAKECERT"), 0600); err != nil {
		t.Fatalf("failed to write cert: %v", err)
	}
	defer os.Remove(tmpCert)

	// Fake admission registry
	reg := katalog.NewInMemoryAdmissionRegistry()
	reg.RegisterValidationRules(gvr.Key, &orktypes.ValidationConfig{})
	reg.RegisterMutationRules(gvr.Key, &orktypes.MutationConfig{})

	opts := WebhookRegistrationOptions{
		ServiceName:      "orkestra",
		ServiceNamespace: "default",
		Port:             8443,
		FailurePolicy:    admissionv1.Ignore,
		TLSCertFile:      tmpCert,
	}

	// Register
	if err := ExportRegisterAdmissionWebhooks(ctx, client, reg, opts); err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Validate: ValidatingWebhookConfiguration exists
	vwc, err := client.AdmissionregistrationV1().
		ValidatingWebhookConfigurations().
		Get(ctx, validatingWebhookConfigName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("validating webhook missing: %v", err)
	}

	if len(vwc.Webhooks) != 1 {
		t.Fatalf("expected 1 validating webhook, got %d", len(vwc.Webhooks))
	}

	// Validate: MutatingWebhookConfiguration exists
	mwc, err := client.AdmissionregistrationV1().
		MutatingWebhookConfigurations().
		Get(ctx, mutatingWebhookConfigName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("mutating webhook missing: %v", err)
	}

	if len(mwc.Webhooks) != 1 {
		t.Fatalf("expected 1 mutating webhook, got %d", len(mwc.Webhooks))
	}

	// Idempotency: calling again should not error
	if err := ExportRegisterAdmissionWebhooks(ctx, client, reg, opts); err != nil {
		t.Fatalf("idempotent registration failed: %v", err)
	}
}
