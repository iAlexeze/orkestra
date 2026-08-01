package proxy

import (
	"errors"
	"fmt"
	"testing"
)

func TestResolveRemotePort(t *testing.T) {
	tests := []struct {
		komponent string
		want      int32
		localPort int
	}{
		{KomponentRuntime, 8080, 9000},
		{KomponentGateway, 8080, 9000},
		{KomponentCC, 8081, 9000},
		{"unknown", 9000, 9000},
	}
	for _, tt := range tests {
		got := resolveRemotePort(ForwardTarget{Komponent: tt.komponent, LocalPort: tt.localPort})
		if got != tt.want {
			t.Errorf("resolveRemotePort(%q) = %d, want %d", tt.komponent, got, tt.want)
		}
	}
}

func TestIsNotDeployed(t *testing.T) {
	if !isNotDeployed(errNotDeployed) {
		t.Error("expected isNotDeployed(errNotDeployed) to be true")
	}
	if !isNotDeployed(fmt.Errorf("not deployed in orkestra-system: %w", errNotDeployed)) {
		t.Error("expected isNotDeployed to unwrap a wrapped errNotDeployed")
	}
	if isNotDeployed(errors.New("some other error")) {
		t.Error("expected isNotDeployed to be false for an unrelated error")
	}
}
