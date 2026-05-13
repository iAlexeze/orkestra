package reconciler

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/orkspace/orkestra/domain"
	orklabels "github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	corev1 "k8s.io/api/core/v1"
)

// ── Label management ──────────────────────────────────────────────────────
func (r *GenericReconciler[PTR]) ensureManagedLabel(ctx context.Context, obj PTR) error {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}

	// Already present?
	if v, ok := labels[orklabels.Managed]; ok && v == orklabels.ManagedValue {
		return nil
	}

	// Add/overwrite the managed label
	labels[orklabels.Managed] = orklabels.ManagedValue

	return r.kube.PatchLabels(ctx, obj, r.crd.GVR(), labels)
}

// ── Annotation management ──────────────────────────────────────────────────────
func (r *GenericReconciler[PTR]) ensureManagedAnnotations(ctx context.Context, obj PTR, operator string) error {
	ann := obj.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}

	changed := false

	// Ensure managed-by annotation
	if v, ok := ann[orklabels.AnnotationManagedBy]; !ok || v == "" {
		ann[orklabels.AnnotationManagedBy] = operator
		changed = true
	}

	// Ensure managed-since annotation
	if v, ok := ann[orklabels.AnnotationManagedSince]; !ok || v == "" {
		ann[orklabels.AnnotationManagedSince] = time.Now().UTC().Format(time.RFC3339)
		changed = true
	}

	// Nothing to patch
	if !changed {
		return nil
	}

	return r.kube.PatchAnnotations(ctx, obj, r.crd.GVR(), ann)
}

// ── Finalizer management ──────────────────────────────────────────────────────

func (r *GenericReconciler[PTR]) ensureFinalizers(ctx context.Context, obj PTR) error {
	if len(r.crd.OperatorBox.Finalizers) == 0 {
		return nil
	}

	logger.Debug().
		Str("name", obj.GetName()).
		Any("crd finalizers", r.crd.OperatorBox.Finalizers).
		Msgf("checking finalizers: %v", obj.GetFinalizers())

	needsUpdate := false
	for _, f := range r.crd.OperatorBox.Finalizers {
		if !ContainsFinalizer(obj, f) && r.crd.RemoveFinalizers { // Added for testing -> could be useful in future
			needsUpdate = true
			break
		}
	}
	if !needsUpdate {
		return nil
	}

	newFinalizers := obj.GetFinalizers()
	for _, f := range r.crd.OperatorBox.Finalizers {
		if !ContainsFinalizer(obj, f) {
			newFinalizers = append(newFinalizers, f)
		}
	}

	logger.Debug().
		Str("name", obj.GetName()).
		Msgf("adding finalizers: %v → %v", obj.GetFinalizers(), newFinalizers)

	r.event.Eventf(obj, corev1.EventTypeNormal, r.crd.APITypes.Kind+"FinalizerAdded",
		fmt.Sprintf("Added finalizers to %s/%s", obj.GetNamespace(), obj.GetName()))

	return r.kube.PatchFinalizers(ctx, obj, r.crd.GVR(), newFinalizers)
}

func (r *GenericReconciler[PTR]) removeFinalizers(ctx context.Context, obj PTR) error {
	if len(obj.GetFinalizers()) == 0 {
		return nil
	}

	newFinalizers := make([]string, 0, len(obj.GetFinalizers()))
	for _, f := range obj.GetFinalizers() {
		if !slices.Contains(r.crd.OperatorBox.Finalizers, f) {
			newFinalizers = append(newFinalizers, f)
		}
	}

	if len(newFinalizers) == len(obj.GetFinalizers()) {
		return nil
	}

	logger.Debug().
		Str("name", obj.GetName()).
		Msgf("removing finalizers: %v → %v", obj.GetFinalizers(), newFinalizers)

	return r.kube.PatchFinalizers(ctx, obj, r.crd.GVR(), newFinalizers)
}

// ── Finalizer helpers — exported for custom reconcilers ───────────────────────

func AddFinalizer(o domain.Object, finalizer string) (updated bool) {
	if ContainsFinalizer(o, finalizer) {
		return false
	}
	o.SetFinalizers(append(o.GetFinalizers(), finalizer))
	return true
}

func RemoveFinalizer(o domain.Object, finalizer string) (updated bool) {
	f := o.GetFinalizers()
	length := len(f)
	index := 0
	for i := range length {
		if f[i] == finalizer {
			continue
		}
		f[index] = f[i]
		index++
	}
	o.SetFinalizers(f[:index])
	return length != index
}

func ContainsFinalizer(o domain.Object, finalizer string) bool {
	return slices.Contains(o.GetFinalizers(), finalizer)
}
