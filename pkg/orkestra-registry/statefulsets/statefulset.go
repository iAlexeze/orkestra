// pkg/orkestra-registry/statefulsets/statefulset.go
package statefulsets

import (
	"context"
	"fmt"
	"strconv"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/orkestra-registry/common"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Create creates a StatefulSet owned by the CR if it does not already exist.
func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedStatefulSetSpec) error {
	ns := common.ResolveNamespace(owner, spec.Namespace)

	_, err := kube.Clientset().AppsV1().StatefulSets(ns).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("statefulset.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("statefulset", spec.Name).
			Str("namespace", ns).
			Msg("statefulset already exists — skipping create")
		return nil
	}

	sts := buildStatefulSet(owner, spec, ns)
	_, err = kube.Clientset().AppsV1().StatefulSets(ns).Create(ctx, sts, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("statefulset.Create: creating %q in %q: %w", spec.Name, ns, err)
	}

	logger.Info().
		Str("statefulset", spec.Name).
		Str("namespace", ns).
		Str("owner", owner.GetName()).
		Msg("statefulset created")
	return nil
}

// Update reconciles an existing StatefulSet to match the resolved spec.
// Patches replicas and image when drift is detected.
func Update(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedStatefulSetSpec) error {
	ns := common.ResolveNamespace(owner, spec.Namespace)

	existing, err := kube.Clientset().AppsV1().StatefulSets(ns).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("statefulset.Update: getting %q: %w", spec.Name, err)
	}

	desired := buildStatefulSet(owner, spec, ns)
	drifted := false
	updated := existing.DeepCopy()

	if *existing.Spec.Replicas != *desired.Spec.Replicas {
		updated.Spec.Replicas = desired.Spec.Replicas
		drifted = true
	}
	if len(existing.Spec.Template.Spec.Containers) > 0 &&
		existing.Spec.Template.Spec.Containers[0].Image != desired.Spec.Template.Spec.Containers[0].Image {
		updated.Spec.Template.Spec.Containers[0].Image = desired.Spec.Template.Spec.Containers[0].Image
		drifted = true
	}

	if !drifted {
		logger.Debug().Str("statefulset", spec.Name).Msg("statefulset in sync — no update needed")
		return nil
	}

	_, err = kube.Clientset().AppsV1().StatefulSets(ns).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("statefulset.Update: updating %q: %w", spec.Name, err)
	}

	logger.Info().Str("statefulset", spec.Name).Str("namespace", ns).Msg("statefulset updated")
	return nil
}

// Delete deletes the StatefulSet if it exists.
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedStatefulSetSpec) error {
	ns := common.ResolveNamespace(owner, spec.Namespace)

	err := kube.Clientset().AppsV1().StatefulSets(ns).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("statefulset.Delete: %w", err)
	}
	logger.Info().Str("statefulset", spec.Name).Str("owner", owner.GetName()).Msg("statefulset deleted")
	return nil
}

// DeleteIfOwned deletes the StatefulSet only if it is owned by the given CR.
func DeleteIfOwned(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, name, ns string) error {
	existing, err := kube.Clientset().AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if existing.Labels[konfig.LabelOrkestraOwner] != owner.GetName() {
		return nil
	}
	return kube.Clientset().AppsV1().StatefulSets(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedStatefulSetSpec from a StatefulSetTemplateSource.
func Resolve(src orktypes.StatefulSetTemplateSource, ownerName string) ResolvedStatefulSetSpec {
	spec := ResolvedStatefulSetSpec{
		Name:         src.Name,
		Namespace:    src.Namespace,
		Image:        src.Image,
		ServiceName:  src.ServiceName,
		StorageClass: src.StorageClass,
		StorageSize:  src.StorageSize,
		MountPath:    src.MountPath,
		Replicas:     1,
		Labels:       make(map[string]string),
		Annotations:  make(map[string]string),
		Env:          src.Env,
		EnvFrom:      src.EnvFrom,
		Resources:    src.Resources,
	}

	if spec.Name == "" {
		spec.Name = ownerName
	}
	if spec.ServiceName == "" {
		spec.ServiceName = spec.Name
	}
	if spec.MountPath == "" {
		spec.MountPath = "/data"
	}

	if src.Tag != "" {
		spec.Image = src.Image + ":" + src.Tag
	}
	if r, err := strconv.ParseInt(src.Replicas, 10, 32); err == nil && r > 0 {
		spec.Replicas = int32(r)
	}
	if p, err := strconv.ParseInt(src.Port, 10, 32); err == nil {
		spec.Port = int32(p)
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}
	for _, a := range src.Annotations {
		spec.Annotations[a.Key] = a.Value
	}

	spec.Labels[konfig.LabelManaged] = konfig.LabelManagedValue
	spec.Labels[konfig.LabelOrkestraOwner] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildStatefulSet(owner domain.Object, spec ResolvedStatefulSetSpec, ns string) *appsv1.StatefulSet {
	apiVersion := ""
	kind := ""
	if u, ok := owner.(*unstructured.Unstructured); ok {
		apiVersion = u.GetAPIVersion()
		kind = u.GetKind()
	} else {
		gvk := owner.GetObjectKind().GroupVersionKind()
		apiVersion = gvk.GroupVersion().String()
		kind = gvk.Kind
	}

	replicas := spec.Replicas
	container := corev1.Container{
		Name:  spec.Name,
		Image: spec.Image,
	}

	if spec.Port > 0 {
		container.Ports = []corev1.ContainerPort{{ContainerPort: spec.Port}}
	}

	if spec.Resources != nil {
		container.Resources = common.BuildResourceRequirements(spec.Resources)
	}

	for k, v := range spec.Env {
		ev := corev1.EnvVar{Name: k}
		if v.Value != "" {
			ev.Value = v.Value
		} else if v.SecretKeyRef != nil {
			ev.ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: v.SecretKeyRef.Name},
					Key:                  v.SecretKeyRef.Key,
				},
			}
		} else if v.ConfigMapKeyRef != nil {
			ev.ValueFrom = &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: v.ConfigMapKeyRef.Name},
					Key:                  v.ConfigMapKeyRef.Key,
				},
			}
		}
		container.Env = append(container.Env, ev)
	}

	for _, ef := range spec.EnvFrom {
		var src corev1.EnvFromSource
		if ef.SecretRef != "" {
			src.SecretRef = &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: ef.SecretRef},
			}
		}
		if ef.ConfigMapRef != "" {
			src.ConfigMapRef = &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: ef.ConfigMapRef},
			}
		}
		container.EnvFrom = append(container.EnvFrom, src)
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        spec.Name,
			Namespace:   ns,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         apiVersion,
					Kind:               kind,
					Name:               owner.GetName(),
					UID:                owner.GetUID(),
					Controller:         utils.BoolPtr(true),
					BlockOwnerDeletion: utils.BoolPtr(true),
				},
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: spec.ServiceName,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					konfig.LabelOrkestraOwner: owner.GetName(),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: spec.Labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:   common.ToPullSecrets(spec.ImagePullSecrets),
					ServiceAccountName: spec.ServiceAccountName,
					NodeSelector:       spec.NodeSelector,
					Containers:         []corev1.Container{container},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{},
			PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
				WhenDeleted: appsv1.PersistentVolumeClaimRetentionPolicyType(spec.VolumeClaimRetentionPolicy.WhenDeleted),
				WhenScaled:  appsv1.PersistentVolumeClaimRetentionPolicyType(spec.VolumeClaimRetentionPolicy.WhenScaled),
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.OnDeleteStatefulSetStrategyType,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
		},
	}

	// Add VolumeClaimTemplate when storage is declared.
	if spec.StorageClass != "" && spec.StorageSize != "" {
		storageQty := resource.MustParse(spec.StorageSize)
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      "data",
			MountPath: spec.MountPath,
		})
		sts.Spec.Template.Spec.Containers[0].VolumeMounts = container.VolumeMounts
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "data"},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					StorageClassName: &spec.StorageClass,
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: storageQty,
						},
					},
				},
			},
		}
	}

	return sts
}
