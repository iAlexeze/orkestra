// pkg/reconciler/finalizer_test.go
package reconciler

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func makeObj(finalizers ...string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetFinalizers(finalizers)
	return obj
}

// ── ContainsFinalizer ─────────────────────────────────────────────────────────

func TestContainsFinalizer_Present(t *testing.T) {
	obj := makeObj("orkestra.io/cleanup", "other.io/finalizer")
	if !ContainsFinalizer(obj, "orkestra.io/cleanup") {
		t.Error("expected finalizer to be found")
	}
}

func TestContainsFinalizer_Absent(t *testing.T) {
	obj := makeObj("other.io/finalizer")
	if ContainsFinalizer(obj, "orkestra.io/cleanup") {
		t.Error("expected finalizer to be absent")
	}
}

func TestContainsFinalizer_Empty(t *testing.T) {
	obj := makeObj()
	if ContainsFinalizer(obj, "orkestra.io/cleanup") {
		t.Error("no finalizers — must return false")
	}
}

// ── AddFinalizer ──────────────────────────────────────────────────────────────

func TestAddFinalizer_AddsWhenAbsent(t *testing.T) {
	obj := makeObj()
	updated := AddFinalizer(obj, "orkestra.io/cleanup")
	if !updated {
		t.Error("expected updated=true when adding a new finalizer")
	}
	if !ContainsFinalizer(obj, "orkestra.io/cleanup") {
		t.Error("finalizer must be present after Add")
	}
}

func TestAddFinalizer_NoopWhenPresent(t *testing.T) {
	obj := makeObj("orkestra.io/cleanup")
	updated := AddFinalizer(obj, "orkestra.io/cleanup")
	if updated {
		t.Error("expected updated=false when finalizer already present")
	}
	if len(obj.GetFinalizers()) != 1 {
		t.Error("finalizer must not be duplicated")
	}
}

func TestAddFinalizer_MultipleFinalizers(t *testing.T) {
	obj := makeObj("first.io/f")
	AddFinalizer(obj, "second.io/f")
	AddFinalizer(obj, "third.io/f")
	f := obj.GetFinalizers()
	if len(f) != 3 {
		t.Errorf("expected 3 finalizers, got %d: %v", len(f), f)
	}
}

// ── RemoveFinalizer ───────────────────────────────────────────────────────────

func TestRemoveFinalizer_RemovesWhenPresent(t *testing.T) {
	obj := makeObj("orkestra.io/cleanup", "other.io/f")
	updated := RemoveFinalizer(obj, "orkestra.io/cleanup")
	if !updated {
		t.Error("expected updated=true when removing a present finalizer")
	}
	if ContainsFinalizer(obj, "orkestra.io/cleanup") {
		t.Error("finalizer must be gone after Remove")
	}
	if !ContainsFinalizer(obj, "other.io/f") {
		t.Error("other finalizer must remain")
	}
}

func TestRemoveFinalizer_NoopWhenAbsent(t *testing.T) {
	obj := makeObj("other.io/f")
	updated := RemoveFinalizer(obj, "orkestra.io/cleanup")
	if updated {
		t.Error("expected updated=false when finalizer not present")
	}
	if len(obj.GetFinalizers()) != 1 {
		t.Error("existing finalizer must remain unchanged")
	}
}

func TestRemoveFinalizer_EmptyList(t *testing.T) {
	obj := makeObj()
	updated := RemoveFinalizer(obj, "orkestra.io/cleanup")
	if updated {
		t.Error("expected updated=false on empty finalizer list")
	}
}

func TestRemoveFinalizer_AllRemoved(t *testing.T) {
	obj := makeObj("orkestra.io/cleanup")
	RemoveFinalizer(obj, "orkestra.io/cleanup")
	if len(obj.GetFinalizers()) != 0 {
		t.Error("all finalizers should be removed")
	}
}

func TestRemoveFinalizer_OnlyTargetRemoved(t *testing.T) {
	obj := makeObj("a", "b", "c")
	RemoveFinalizer(obj, "b")
	f := obj.GetFinalizers()
	if len(f) != 2 {
		t.Errorf("expected 2 remaining, got %v", f)
	}
	for _, fin := range f {
		if fin == "b" {
			t.Error("b must be removed")
		}
	}
}
