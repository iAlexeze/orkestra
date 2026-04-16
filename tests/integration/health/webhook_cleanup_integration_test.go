//go:build integration

package health_test

import (
	"context"
	"testing"

	"github.com/orkspace/orkestra/pkg/health"
	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestWebhookLifecycle_Integration(t *testing.T) {
	ctx := context.Background()

	testEnv := &envtest.Environment{}
	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("failed to start envtest: %v", err)
	}
	defer testEnv.Stop()

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Fake registry with one validating + one mutating rule
	gvr := katalog.GVREntry{
		Group:      "demo.orkestra.io",
		Version:    "v1alpha1",
		Resource:   "websites",
		Operations: []string{"CREATE", "UPDATE"},
	}

	reg := katalog.NewInMemoryAdmissionRegistry()
	reg.RegisterValidationRules(gvr.Key, &orktypes.ValidationConfig{})
	reg.RegisterMutationRules(gvr.Key, &orktypes.MutationConfig{})

	opts := health.WebhookRegistrationOptions{
		ServiceName:      "orkestra",
		ServiceNamespace: "default",
		Port:             8443,
		FailurePolicy:    admissionv1.Ignore,
		TLSCertFile:      "testdata/tls.crt",
	}

	// Register
	err = health.RegisterWebhooks(ctx, client, reg, opts)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Ensure they exist
	if _, err := client.AdmissionregistrationV1().
		ValidatingWebhookConfigurations().
		Get(ctx, "orkestra-validation", metav1.GetOptions{}); err != nil {
		t.Fatalf("validating webhook missing: %v", err)
	}

	if _, err := client.AdmissionregistrationV1().
		MutatingWebhookConfigurations().
		Get(ctx, "orkestra-mutation", metav1.GetOptions{}); err != nil {
		t.Fatalf("mutating webhook missing: %v", err)
	}

	// Cleanup
	err = health.UnregisterWebhooks(ctx, client)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// Ensure deletion
	if _, err := client.AdmissionregistrationV1().
		ValidatingWebhookConfigurations().
		Get(ctx, "orkestra-validation", metav1.GetOptions{}); err == nil {
		t.Fatalf("validating webhook still exists")
	}

	if _, err := client.AdmissionregistrationV1().
		MutatingWebhookConfigurations().
		Get(ctx, "orkestra-mutation", metav1.GetOptions{}); err == nil {
		t.Fatalf("mutating webhook still exists")
	}
}
