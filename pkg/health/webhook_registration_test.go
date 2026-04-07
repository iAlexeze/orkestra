package health

import (
	"context"
	"os"
	"testing"

	"github.com/ialexeze/orkestra/pkg/katalog"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRegisterWebhooks_CreatesValidatingAndMutating(t *testing.T) {
	ctx := context.Background()

	// Fake CA bundle
	tmpCert := "testdata/tls.crt"
	if err := os.WriteFile(tmpCert, []byte("FAKECERT"), 0600); err != nil {
		t.Fatalf("failed to write cert: %v", err)
	}
	defer os.Remove(tmpCert)

	// Fake admission registry
	gvr := katalog.GVREntry{
		Group:      "demo.orkestra.io",
		Version:    "v1alpha1",
		Resource:   "websites",
		Operations: []string{"CREATE", "UPDATE"},
	}

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

	client := fake.NewClientset()

	// Register
	if err := RegisterWebhooks(ctx, client, reg, opts); err != nil {
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

	// Validate service reference
	vsvc := vwc.Webhooks[0].ClientConfig.Service
	if vsvc.Name != "orkestra" || vsvc.Namespace != "default" {
		t.Fatalf("unexpected service reference: %+v", vsvc)
	}

	// Validate CA bundle
	if string(vwc.Webhooks[0].ClientConfig.CABundle) != "FAKECERT" {
		t.Fatalf("CA bundle mismatch")
	}

	// Validate rules
	if len(vwc.Webhooks[0].Rules) != 1 {
		t.Fatalf("expected 1 rule in validating webhook")
	}

	rule := vwc.Webhooks[0].Rules[0]
	if rule.APIGroups[0] != "demo.orkestra.io" ||
		rule.APIVersions[0] != "v1alpha1" ||
		rule.Resources[0] != "websites" {
		t.Fatalf("unexpected rule: %+v", rule)
	}

	// Idempotency: calling again should not error
	if err := RegisterWebhooks(ctx, client, reg, opts); err != nil {
		t.Fatalf("idempotent registration failed: %v", err)
	}
}
