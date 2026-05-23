// pkg/orkestra-registry/pvs/pv.go
package pvs

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/orkestra-registry/common"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create creates a PersistentVolume if it does not already exist.
// PVs are cluster-scoped — owner references are set as labels only.
func Create(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedPVSpec) error {
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}
	_, err := kube.Clientset().CoreV1().PersistentVolumes().Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("pv.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().Str("pv", spec.Name).Msg("pv already exists — skipping create")
		return nil
	}

	pv := buildPV(owner, spec)
	_, err = kube.Clientset().CoreV1().PersistentVolumes().Create(ctx, pv, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("pv.Create: creating %q: %w", spec.Name, err)
	}

	logger.Info().Str("pv", spec.Name).Str("owner", owner.GetName()).Msg("pv created")
	return nil
}

// Update reconciles an existing PV. Capacity and reclaim policy are patched on drift.
func Update(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedPVSpec) error {
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	existing, err := kube.Clientset().CoreV1().PersistentVolumes().Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("pv.Update: getting %q: %w", spec.Name, err)
	}

	updated := existing.DeepCopy()
	drifted := false

	desired := buildPV(owner, spec)

	if desired.Spec.PersistentVolumeReclaimPolicy != existing.Spec.PersistentVolumeReclaimPolicy {
		updated.Spec.PersistentVolumeReclaimPolicy = desired.Spec.PersistentVolumeReclaimPolicy
		drifted = true
	}

	for k, v := range spec.Labels {
		if updated.Labels[k] != v {
			updated.Labels[k] = v
			drifted = true
		}
	}

	if !drifted {
		logger.Debug().Str("pv", spec.Name).Msg("pv in sync — no update needed")
		return nil
	}

	_, err = kube.Clientset().CoreV1().PersistentVolumes().Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("pv.Update: updating %q: %w", spec.Name, err)
	}

	logger.Info().Str("pv", spec.Name).Msg("pv updated")
	return nil
}

// Delete deletes the PV if it exists.
func Delete(ctx context.Context, kube kubeclient.KubeClient, name string) error {
	err := kube.Clientset().CoreV1().PersistentVolumes().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("pv.Delete: %w", err)
	}
	logger.Info().Str("pv", name).Msg("pv deleted")
	return nil
}

// DeleteIfOwned deletes the PV only if the owner label matches.
func DeleteIfOwned(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, name string) error {
	existing, err := kube.Clientset().CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if existing.Labels[labels.OrkestraOwner] != owner.GetName() {
		return nil
	}
	return kube.Clientset().CoreV1().PersistentVolumes().Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedPVSpec from a PVTemplateSource.
func Resolve(src orktypes.PVTemplateSource, ownerName string) ResolvedPVSpec {
	spec := ResolvedPVSpec{
		Name:             src.Name,
		StorageClassName: src.StorageClassName,
		Capacity:         src.Capacity,
		AccessModes:      src.AccessModes,
		ReclaimPolicy:    src.ReclaimPolicy,
		HostPath:         src.HostPath,
		CSIDriver:        src.CSIDriver,
		CSIVolumeHandle:  src.CSIVolumeHandle,
		Labels:           make(map[string]string),
		Sleep:            src.Sleep,
	}

	if len(spec.AccessModes) == 0 {
		spec.AccessModes = []string{"ReadWriteOnce"}
	}
	if spec.ReclaimPolicy == "" {
		spec.ReclaimPolicy = "Retain"
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}
	spec.Labels[labels.ManagedKey] = labels.ManagedValue
	spec.Labels[labels.OrkestraOwner] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildPV(owner domain.Object, spec ResolvedPVSpec) *corev1.PersistentVolume {
	capacityQty := resource.MustParse(spec.Capacity)

	var accessModes []corev1.PersistentVolumeAccessMode
	for _, m := range spec.AccessModes {
		accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(m))
	}

	reclaimPolicy := corev1.PersistentVolumeReclaimRetain
	switch spec.ReclaimPolicy {
	case "Delete":
		reclaimPolicy = corev1.PersistentVolumeReclaimDelete
	case "Recycle":
		reclaimPolicy = corev1.PersistentVolumeReclaimRecycle
	}

	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   spec.Name,
			Labels: spec.Labels,
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: capacityQty,
			},
			AccessModes:                   accessModes,
			PersistentVolumeReclaimPolicy: reclaimPolicy,
			StorageClassName:              spec.StorageClassName,
		},
	}

	if spec.HostPath != "" {
		pv.Spec.PersistentVolumeSource = corev1.PersistentVolumeSource{
			HostPath: &corev1.HostPathVolumeSource{Path: spec.HostPath},
		}
	} else if spec.CSIDriver != "" {
		pv.Spec.PersistentVolumeSource = corev1.PersistentVolumeSource{
			CSI: &corev1.CSIPersistentVolumeSource{
				Driver:       spec.CSIDriver,
				VolumeHandle: spec.CSIVolumeHandle,
			},
		}
	}

	// PVs are cluster-scoped; label with owner for DeleteIfOwned.
	_ = owner

	return pv
}
