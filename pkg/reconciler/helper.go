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
// ensureManagedLabel adds the Orkestra managed label to the given object
// if it is not already present with the correct value.
//
// The managed label (`orkestra.orkspace.io/managed: "true"`) identifies
// resources that Orkestra owns and controls. It is applied unconditionally
// to every resource created or updated by Orkestra, including the main
// custom resource and all child resources (Deployment, Service, ConfigMap, etc.).
//
// This label is used by the Orkestra ecosystem for:
//   - Resource ownership tracking (e.g., garbage collection decisions)
//   - Webhook selectors (e.g., mutation/validation scoping)
//   - CLI and UI filtering of Orkestra-managed resources
//
// The function patches the object's metadata.labels only when a change is
// required. It returns an error if the API patch fails.
//
// Example:
//
//	After calling ensureManagedLabel on a Resource:
//	  metadata:
//	    labels:
//	      orkestra.orkspace.io/managed: "true"
func (r *GenericReconciler[PTR]) ensureManagedLabel(ctx context.Context, obj PTR) error {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}

	// Already present?
	if v, ok := labels[orklabels.ManagedKey]; ok && v == orklabels.ManagedValue {
		return nil
	}

	// Add/overwrite the managed label
	labels[orklabels.ManagedKey] = orklabels.ManagedValue

	return r.kube.PatchLabels(ctx, obj, r.crd.GVR(), labels)
}

// ensureDeletionProtectionLabel adds the deletion‑protection label to the
// given object when the Orkestra security feature is globally enabled.
//
// The label (`orkestra.io/deletion-protection: "true"`) marks a resource
// as protected from accidental deletion. The deletion‑protection webhook
// intercepts DELETE requests on resources with this label and denies them
// unless the protection is explicitly disabled by the administrator.
//
// This function is called for every resource that Orkestra creates or
// updates (including the main CR and all child resources) when:
//   - security.deletionProtection.enabled is true in the Katalog
//   - the resource does not already have the label set to "true"
//
// If deletion protection is disabled globally, this function does nothing.
//
// The label is only applied to resources that Orkestra directly manages.
// Users may manually add the same label to any resource (even those not
// created by Orkestra) to extend protection; the webhook will honour it.
//
// Example:
//
//	After calling ensureDeletionProtectionLabel on a Resource:
//	  metadata:
//	    labels:
//	      orkestra.io/deletion-protection: "true"
func (r *GenericReconciler[PTR]) ensureDeletionProtectionLabel(ctx context.Context, obj PTR) error {
	if !r.kat.IsDeletionProtectionEnabled() {
		return nil
	}
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}

	// Already present?
	if v, ok := labels[orklabels.DeletionProtectionLabel]; ok && v == "true" {
		return nil
	}

	// Add/overwrite the deletio protection label
	labels[orklabels.DeletionProtectionLabel] = "true"
	return r.kube.PatchLabels(ctx, obj, r.crd.GVR(), labels)
}

// ── Annotation management ──────────────────────────────────────────────────────
// ensureManagedAnnotations adds Orkestra's management tracking annotations to the
// given object if they are missing.
//
// The managed-by annotation (`orkestra.orkspace.io/managed-by`) records the operator
// name (e.g., the Katalog name or "orkestra-gateway") that controls this resource.
// This is useful for debugging, auditing, and multi-operator environments where
// different controllers may manage different resources.
//
// The managed-since annotation (`orkestra.orkspace.io/managed-since`) stores the
// UTC timestamp when Orkestra first took ownership of the resource. It does not
// change on subsequent updates, providing a reliable creation‑handover time.
//
// Both annotations are only set if they are absent or empty. Existing values are
// never overwritten. The function patches the object's metadata.annotations only
// when a change is necessary.
//
// Example:
//
//	After calling ensureManagedAnnotations:
//	  metadata:
//	    annotations:
//	      orkestra.orkspace.io/managed-by: "platform-security"
//	      orkestra.orkspace.io/managed-since: "2026-05-23T12:34:56Z"
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

// ensureFinalizers adds the finalizers declared in the CRD's OperatorBox to the
// given object if they are not already present.
//
// Finalizers block deletion of a resource until the controller has performed
// necessary cleanup (e.g., deleting child resources, releasing external resources).
// The list of finalizers is defined per CRD in the Katalog under spec.crds[].operatorBox.finalizers.
//
// This function is idempotent: it checks which finalizers from the configured
// list are missing and appends them. It never removes finalizers; removal is
// handled separately in the finalizer reconciliation logic (typically after
// successful cleanup).
//
// An event is emitted when finalizers are added, providing an audit trail.
//
// Edge cases:
//   - If the CRD has no finalizers configured, this function does nothing.
//   - If `crd.RemoveFinalizers` is true (a development flag), the function
//     currently only checks for existence but still adds missing finalizers.
//     This behaviour is marked for future review.
//
// Example:
//
//	Configured finalizers: ["protection.orkestra.io/finalizer"]
//	After calling ensureFinalizers, the resource's metadata.finalizers will include it.
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
