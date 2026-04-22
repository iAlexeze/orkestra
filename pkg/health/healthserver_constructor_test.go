package health

import (
	"context"
	"testing"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/kubeclient"
)

func TestNewHealthServer_ConstructorWiresAllFields(t *testing.T) {
	kfg := konfig.NewDefaultKonfig()

	// Fake kube client
	kube := kubeclient.NewFakeClientset()

	// Minimal konfig
	kfg.Ork().Name = "test-runtime"
	kfg.Health().Port = "8080"

	// Empty katalog (no rules)
	kat := katalog.NewEmptyKatalog()

	hs := NewHealthServer(kube, kat, kfg)

	// --- Required fields must not be nil ---
	if hs.kubeClient == nil {
		t.Fatal("kubeClient must not be nil")
	}
	if hs.katalog == nil {
		t.Fatal("katalog must not be nil")
	}
	if hs.mux == nil {
		t.Fatal("mux must not be nil")
	}
	if hs.hookMux == nil {
		t.Fatal("hookMux must not be nil")
	}
	if hs.conversionRegistry == nil {
		t.Fatal("conversionRegistry must not be nil")
	}
	if hs.admissionRegistry == nil {
		t.Fatal("admissionRegistry must not be nil")
	}

	// --- Webhook config must be populated from konfig ---
	if hs.hookKfg.TLSCert != kfg.Security().Webhooks.TLSCert {
		t.Fatal("hookKfg.TLSCert not wired correctly")
	}
	if hs.hookReg.ServiceName != kat.WebhooksServiceName() {
		t.Fatal("hookReg.ServiceName not wired correctly")
	}

	// --- Start() must not panic with empty katalog ---
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Start() should not panic with empty katalog: %v", r)
		}
	}()

	// We don't care about actual serving — just that Start() doesn't crash
	_ = hs.Start(context.Background())
}
