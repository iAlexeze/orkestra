package e2e

import (
	"os/exec"
	"strconv"
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// exitErrorWithCode runs a trivial shell command that exits with code, and
// returns the resulting *exec.ExitError — a real one, not a hand-built
// stand-in, so assertKubectlApplyOutput's errors.As branch is exercised
// exactly as it would be against a real kubectl failure.
func exitErrorWithCode(t *testing.T, code int) error {
	t.Helper()
	cmd := exec.Command("sh", "-c", "exit "+strconv.Itoa(code))
	err := cmd.Run()
	if err == nil {
		t.Fatalf("command unexpectedly exited 0")
	}
	return err
}

func TestAssertKubectlApplyOutput_ExpectedSuccess(t *testing.T) {
	err := assertKubectlApplyOutput("kubectl apply -f x.yaml", []byte("website.testground.orkestra.io/site-a created"), nil, orktypes.E2EKubectlApply{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAssertKubectlApplyOutput_UnexpectedFailure(t *testing.T) {
	runErr := exitErrorWithCode(t, 1)
	err := assertKubectlApplyOutput("kubectl apply -f x.yaml", []byte("Error from server (Forbidden): admission webhook denied the request"), runErr, orktypes.E2EKubectlApply{})
	if err == nil {
		t.Fatal("expected error — apply failed but ExitCode: 0 (default) means success was expected")
	}
}

func TestAssertKubectlApplyOutput_ExpectedRejection(t *testing.T) {
	runErr := exitErrorWithCode(t, 1)
	e := orktypes.E2EKubectlApply{
		ExitCode:       1,
		OutputContains: "denied the request",
	}
	err := assertKubectlApplyOutput("kubectl apply -f x.yaml", []byte("Error from server (Forbidden): admission webhook denied the request"), runErr, e)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAssertKubectlApplyOutput_ExpectedRejectionButSucceeded(t *testing.T) {
	e := orktypes.E2EKubectlApply{ExitCode: 1}
	err := assertKubectlApplyOutput("kubectl apply -f x.yaml", []byte("website.testground.orkestra.io/site-b created"), nil, e)
	if err == nil {
		t.Fatal("expected error — apply succeeded but ExitCode: 1 means rejection was expected")
	}
}

func TestAssertKubectlApplyOutput_WrongExitCode(t *testing.T) {
	runErr := exitErrorWithCode(t, 2)
	e := orktypes.E2EKubectlApply{ExitCode: 1}
	err := assertKubectlApplyOutput("kubectl apply -f x.yaml", []byte("some other failure"), runErr, e)
	if err == nil {
		t.Fatal("expected error — got exit code 2, wanted 1")
	}
}

func TestAssertKubectlApplyOutput_RejectionMessageMismatch(t *testing.T) {
	runErr := exitErrorWithCode(t, 1)
	e := orktypes.E2EKubectlApply{
		ExitCode:       1,
		OutputContains: "spec.domain must be unique",
	}
	err := assertKubectlApplyOutput("kubectl apply -f x.yaml", []byte("Error from server (Forbidden): some unrelated rule denied the request"), runErr, e)
	if err == nil {
		t.Fatal("expected error — output doesn't contain the expected substring")
	}
}

func TestAssertKubectlApplyOutput_NonExitError(t *testing.T) {
	// A non-ExitError failure (e.g. "kubectl: command not found") should
	// always propagate, regardless of e.ExitCode — there's no exit code to
	// compare against.
	err := assertKubectlApplyOutput("kubectl apply -f x.yaml", nil, exec.ErrNotFound, orktypes.E2EKubectlApply{ExitCode: 1})
	if err == nil {
		t.Fatal("expected error to propagate for a non-ExitError failure")
	}
}
