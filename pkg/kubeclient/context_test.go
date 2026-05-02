// pkg/kubeclient/context_test.go
package kubeclient

import (
	"context"
	"testing"
)

func TestWithKubeclient_StoresInContext(t *testing.T) {
	kube := &Kubeclient{}
	ctx := WithKubeclient(context.Background(), kube)
	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("expected Kubeclient in context")
	}
	if got != kube {
		t.Error("expected same Kubeclient pointer")
	}
}

func TestFromContext_NotPresent_ReturnsFalse(t *testing.T) {
	_, ok := FromContext(context.Background())
	if ok {
		t.Error("empty context must return ok=false")
	}
}

func TestFromContext_NilKubeclient_ReturnsFalse(t *testing.T) {
	ctx := context.WithValue(context.Background(), ContextKey, (*Kubeclient)(nil))
	_, ok := FromContext(ctx)
	if ok {
		t.Error("nil Kubeclient in context must return ok=false")
	}
}

func TestWithKubeclient_ReplacesExisting(t *testing.T) {
	first := &Kubeclient{}
	second := &Kubeclient{}
	ctx := WithKubeclient(context.Background(), first)
	ctx = WithKubeclient(ctx, second)
	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("expected Kubeclient in context")
	}
	if got != second {
		t.Error("second WithKubeclient must replace the first")
	}
}
