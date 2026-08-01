package e2e

import (
	"context"
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func TestAssertPortForwardOutput(t *testing.T) {
	e := orktypes.E2EKubectlPortForward{Path: "/healthz", OutputContains: "ok"}
	if err := assertPortForwardOutput(context.Background(), "", `{"status":"ok"}`, e, "pod/orkestra-runtime-0"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := assertPortForwardOutput(context.Background(), "", `{"status":"degraded"}`, e, "pod/orkestra-runtime-0"); err == nil {
		t.Fatal("expected error — output does not contain \"ok\"")
	}
}

func TestAssertPortForwardOutput_StatusCode(t *testing.T) {
	e := orktypes.E2EKubectlPortForward{Path: "/healthz", StatusCode: 200, Equals: "200"}
	if err := assertPortForwardOutput(context.Background(), "", "200", e, "pod/orkestra-runtime-0"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := assertPortForwardOutput(context.Background(), "", "503", e, "pod/orkestra-runtime-0"); err == nil {
		t.Fatal("expected error — status code 503 does not equal \"200\"")
	}
}
