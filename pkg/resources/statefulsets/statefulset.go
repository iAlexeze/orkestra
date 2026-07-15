// pkg/resources/statefulsets/statefulset.go
package statefulsets

import (
	"context"
	"fmt"
	"strconv"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/profiles"
	"github.com/orkspace/orkestra/pkg/resources/common"
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
func Create(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedStatefulSetSpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	_, err := kube.Clientset().AppsV1().StatefulSets(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("statefulset.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("statefulset", spec.Name).
			Str("namespace", namespace).
			Msg("statefulset already exists — skipping create")
		return nil
	}

	sts := buildStatefulSet(owner, spec, namespace)
	_, err = kube.Clientset().AppsV1().StatefulSets(namespace).Create(ctx, sts, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("statefulset.Create: creating %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("statefulset", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("statefulset created")
	return nil
}

// Update reconciles an existing StatefulSet to match the resolved spec.
// Patches replicas and image when drift is detected.
func Update(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedStatefulSetSpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	existing, err := kube.Clientset().AppsV1().StatefulSets(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("statefulset.Update: getting %q: %w", spec.Name, err)
	}

	desired := buildStatefulSet(owner, spec, namespace)
	drifted := false
	updated := existing.DeepCopy()

	// Replicas — skip when autoscaler owns spec.replicas to avoid fighting it.
	if !spec.HasAutoscale {
		if existing.Spec.Replicas == nil || *existing.Spec.Replicas != *desired.Spec.Replicas {
			updated.Spec.Replicas = desired.Spec.Replicas
			drifted = true
			logger.Info().Str("statefulset", spec.Name).Msg("statefulset replicas drifted")
		}
	}

	// Labels
	if !common.LabelsEqual(existing.Labels, desired.Labels) {
		updated.Labels = desired.Labels
		drifted = true
	}

	// Resources
	if spec.Resources != nil {
		desiredRes := common.BuildResourceRequirements(spec.Resources)
		var existingRes corev1.ResourceRequirements
		if len(existing.Spec.Template.Spec.Containers) > 0 {
			existingRes = existing.Spec.Template.Spec.Containers[0].Resources
		}
		if !common.ResourceRequirementsEqual(existingRes, desiredRes) {
			updated.Spec.Template.Spec.Containers[0].Resources = desiredRes
			drifted = true
			logger.Info().Str("statefulset", spec.Name).Msg("statefulset resources drifted")
		}
	}

	if len(updated.Spec.Template.Spec.Containers) > 0 && len(desired.Spec.Template.Spec.Containers) > 0 {
		if common.SyncContainerSpec(&updated.Spec.Template.Spec.Containers[0], desired.Spec.Template.Spec.Containers[0]) {
			drifted = true
		}
	}
	if common.SyncPodSpec(&updated.Spec.Template.Spec, desired.Spec.Template.Spec) {
		drifted = true
	}

	if !drifted {
		logger.Debug().Str("statefulset", spec.Name).Msg("statefulset in sync — no update needed")
		return nil
	}

	_, err = kube.Clientset().AppsV1().StatefulSets(namespace).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("statefulset.Update: updating %q: %w", spec.Name, err)
	}

	logger.Info().Str("statefulset", spec.Name).Str("namespace", namespace).Msg("statefulset updated")
	return nil
}

// Delete deletes the StatefulSet if it exists.
func Delete(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedStatefulSetSpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	err := kube.Clientset().AppsV1().StatefulSets(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
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
func DeleteIfOwned(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, name, namespace string) error {
	existing, err := kube.Clientset().AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if existing.Labels[labels.OrkestraOwner] != owner.GetName() {
		return nil
	}
	return kube.Clientset().AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedStatefulSetSpec from a StatefulSetTemplateSource.
func Resolve(src orktypes.StatefulSetTemplateSource, ownerName string, reg orktypes.ProfileRegistry) ResolvedStatefulSetSpec {
	spec := ResolvedStatefulSetSpec{
		Name:            src.Name,
		Namespace:       src.Namespace,
		Image:           src.Image,
		ServiceName:     src.ServiceName,
		Replicas:        common.ParseReplicas(src.Replicas),
		HasAutoscale:    src.Autoscale != nil,
		Labels:          make(map[string]string),
		Annotations:     make(map[string]string),
		Env:             src.Env,
		EnvFrom:         src.EnvFrom,
		Resources:       common.ResolveResources(src.Resources, reg),
		Probes:          src.Probes,
		Profiles:        reg,
		SecurityContext: common.ResolveContainerSecurityContext(src.SecurityContext, reg),
		PodSecurity:     common.ResolvePodSecurityContext(src.PodSecurity, reg),
		Volumes:         src.Volumes,
		VolumeMounts:    src.VolumeMounts,
		Sleep:           src.Sleep,
	}

	for _, vct := range src.VolumeClaimTemplates {
		resolved := ResolvedVolumeClaimTemplate{
			Name:         vct.Name,
			StorageClass: vct.StorageClass,
			StorageSize:  vct.StorageSize,
			MountPath:    vct.MountPath,
			AccessModes:  vct.AccessModes,
		}
		if resolved.Name == "" {
			resolved.Name = "data"
		}
		if resolved.MountPath == "" {
			resolved.MountPath = "/data"
		}
		spec.VolumeClaimTemplates = append(spec.VolumeClaimTemplates, resolved)
	}

	if spec.Name == "" {
		spec.Name = ownerName
	}
	if spec.ServiceName == "" {
		spec.ServiceName = spec.Name
	}

	if src.Tag != "" {
		spec.Image = src.Image + ":" + src.Tag
	}
	if p, err := strconv.ParseInt(src.Port, 10, 32); err == nil {
		spec.Port = int32(p)
	}
	spec.Protocol = common.ParseProtocol(src.Protocol)

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}
	for _, a := range src.Annotations {
		spec.Annotations[a.Key] = a.Value
	}

	spec.Labels[labels.ManagedKey] = labels.ManagedValue
	spec.Labels[labels.OrkestraOwner] = ownerName

	if src.RollingUpdate != nil && src.RollingUpdate.Profile != "" {
		expansion, err := profiles.ApplyRollingUpdateProfile(src.RollingUpdate.Profile, reg)
		if err != nil {
			logger.Warn().Str("profile", src.RollingUpdate.Profile).Err(err).Msg("unknown rolling update profile — skipping")
		} else {
			spec.RollingUpdate = &orktypes.RollingUpdateBehavior{
				MaxSurge:       expansion.MaxSurge,
				MaxUnavailable: expansion.MaxUnavailable,
			}
		}
	} else if src.RollingUpdate != nil {
		r := *src.RollingUpdate
		spec.RollingUpdate = &r
	}

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func resolveAccessModes(modes []string) []corev1.PersistentVolumeAccessMode {
	if len(modes) == 0 {
		return []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	out := make([]corev1.PersistentVolumeAccessMode, 0, len(modes))
	for _, m := range modes {
		switch m {
		case "ReadWriteMany":
			out = append(out, corev1.ReadWriteMany)
		case "ReadOnlyMany":
			out = append(out, corev1.ReadOnlyMany)
		case "ReadWriteOncePod":
			out = append(out, corev1.ReadWriteOncePod)
		default:
			out = append(out, corev1.ReadWriteOnce)
		}
	}
	return out
}

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
		container.Ports = []corev1.ContainerPort{{ContainerPort: spec.Port, Protocol: spec.Protocol}}
	}

	if spec.Resources != nil {
		container.Resources = common.BuildResourceRequirements(spec.Resources)
	}

	common.ApplyProbes(&container, spec.Probes, spec.Port, spec.Profiles)

	for _, ev := range spec.Env {
		kev := corev1.EnvVar{Name: ev.Name}
		if ev.ValueFrom != nil {
			kev.ValueFrom = &corev1.EnvVarSource{}
			if ev.ValueFrom.SecretKeyRef != nil {
				kev.ValueFrom.SecretKeyRef = &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: ev.ValueFrom.SecretKeyRef.Name},
					Key:                  ev.ValueFrom.SecretKeyRef.Key,
				}
			}
			if ev.ValueFrom.ConfigMapKeyRef != nil {
				kev.ValueFrom.ConfigMapKeyRef = &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: ev.ValueFrom.ConfigMapKeyRef.Name},
					Key:                  ev.ValueFrom.ConfigMapKeyRef.Key,
				}
			}
		} else {
			kev.Value = ev.Value
		}
		container.Env = append(container.Env, kev)
	}

	if spec.EnvFrom != nil {
		for _, name := range spec.EnvFrom.SecretRef {
			container.EnvFrom = append(container.EnvFrom, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: name},
				},
			})
		}
		for _, name := range spec.EnvFrom.ConfigMapRef {
			container.EnvFrom = append(container.EnvFrom, corev1.EnvFromSource{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: name},
				},
			})
		}
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
					labels.OrkestraOwner: owner.GetName(),
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
			UpdateStrategy: func() appsv1.StatefulSetUpdateStrategy {
				if spec.RollingUpdate != nil {
					return common.BuildStatefulSetUpdateStrategy(spec.RollingUpdate)
				}
				return appsv1.StatefulSetUpdateStrategy{Type: appsv1.OnDeleteStatefulSetStrategyType}
			}(),
			PodManagementPolicy: appsv1.ParallelPodManagement,
		},
	}

	// Security
	common.ApplySecurityContext(&sts.Spec.Template.Spec.Containers[0], &sts.Spec.Template.Spec, spec.SecurityContext, spec.PodSecurity)

	for _, vct := range spec.VolumeClaimTemplates {
		storageQty := resource.MustParse(vct.StorageSize)
		name := vct.Name
		if name == "" {
			name = "data"
		}
		mountPath := vct.MountPath
		if mountPath == "" {
			mountPath = "/data"
		}
		accessModes := resolveAccessModes(vct.AccessModes)
		sts.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			sts.Spec.Template.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{Name: name, MountPath: mountPath},
		)
		sts.Spec.VolumeClaimTemplates = append(sts.Spec.VolumeClaimTemplates, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes:      accessModes,
				StorageClassName: &vct.StorageClass,
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: storageQty,
					},
				},
			},
		})
	}

	// Volumes / VolumeMounts (generic, in addition to VolumeClaimTemplates)
	if vols := common.BuildVolumes(spec.Volumes); len(vols) > 0 {
		sts.Spec.Template.Spec.Volumes = vols
	}
	if mounts := common.BuildVolumeMounts(spec.VolumeMounts); len(mounts) > 0 {
		sts.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			sts.Spec.Template.Spec.Containers[0].VolumeMounts, mounts...,
		)
	}

	return sts
}
