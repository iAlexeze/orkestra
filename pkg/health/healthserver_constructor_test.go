package health

import (
	"context"
	"testing"

	"github.com/orkspace/orkestra/pkg/konfig"
)

func TestNewHealthServer_ConstructorWiresAllFields(t *testing.T) {
	kfg := konfig.NewDefaultKonfig()
	kfg.Ork().SetName("test-runtime")
	kfg.Health().SetPort("8080")

	hs := NewHealthServer(kfg)

	if hs.mux == nil {
		t.Fatal("mux must not be nil")
	}
	if hs.client != "test-runtime" {
		t.Fatalf("client: expected %q, got %q", "test-runtime", hs.client)
	}
	if hs.httpPort != "8080" {
		t.Fatalf("httpPort: expected %q, got %q", "8080", hs.httpPort)
	}

	// Start() must not panic with default konfig
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Start() should not panic: %v", r)
		}
	}()

	_ = hs.Start(context.Background())
}
