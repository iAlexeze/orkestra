package domain

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
)

// Katalog is the subset of pkg/katalog.Katalog used by packages that cannot
// import pkg/katalog directly (pkg/runtime/informer, pkg/runtime/queue) without
// forming an import cycle. Implemented by *pkg/katalog.Katalog.
type Katalog interface {
	// EvaluateQueueBehaviourConditions completes queue behaviour evaluation for a CRD.
	// Called by the informer when the workqueue's NeedsBehaviourEval() is true — meaning
	// the workqueue detected a depth/threshold condition but deferred the when/or evaluation
	// because it requires the full preReconcile resolver context.
	// Returns true to enqueue, false to drop.
	EvaluateQueueBehaviourConditions(ctx context.Context, gvkString string, obj Object, sentinels map[string]string) bool

	// EvaluateEnqueueFilter evaluates preReconcile.enqueueGate conditions for the named CRD.
	// Returns true when the object should be enqueued, false when it should be dropped.
	EvaluateEnqueueFilter(ctx context.Context, gvkString string, obj Object, cs kubernetes.Interface, sentinels map[string]string) bool

	// EvaluatePreReconcile evaluates preReconcile.reconcileGate conditions for the named CRD.
	// Returns (true, "") when conditions pass and the reconciler should run.
	// Returns (false, reason) when gated — reconciler must not be called.
	EvaluatePreReconcile(ctx context.Context, gvk string, obj *unstructured.Unstructured, cs kubernetes.Interface, sentinels map[string]string) (allowed bool, reason string)

	// CRD name lookups — resolve a GVK/GVR/kind/target string to the katalog CRD entry name.
	GetNameByGVKString(gvkString string) string
	GetNameByGVRString(gvrString string) string
	GetNameByKind(kind string) string
	GetNameByTarget(target string) string
}
