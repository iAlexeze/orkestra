// pkg/orkestra-registry/replicasets/replicaset.go
package replicasets

import (
	"context"
	"fmt"
	"strconv"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/orkestra-registry/common"
	"github.com/orkspace/orkestra/pkg/profiles"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create creates a ReplicaSet owned by the CR if it does not already exist.
// Idempotent — if the ReplicaSet exists, does nothing and returns nil.
func Create(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedReplicaSetSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("replicaset.Create: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	_, err := kube.Clientset().AppsV1().ReplicaSets(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("replicaset.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("replicaset", spec.Name).
			Str("namespace", namespace).
			Msg("replicaset already exists — skipping create")
		return nil
	}

	rs := buildReplicaSet(owner, spec, namespace)

	_, err = kube.Clientset().AppsV1().ReplicaSets(namespace).Create(ctx, rs, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("replicaset.Create: creating replicaset %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("replicaset", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("replicaset created")

	return nil
}

// Update reconciles an existing ReplicaSet to match the resolved spec.
// Handles drift — if replicas or image have changed, patches the ReplicaSet.
func Update(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedReplicaSetSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("replicaset.Update: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	existing, err := kube.Clientset().AppsV1().ReplicaSets(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info().
				Str("replicaset", spec.Name).
				Str("namespace", namespace).
				Msg("replicaset not found during reconcile — recreating")
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("replicaset.Update: getting replicaset %q: %w", spec.Name, err)
	}

	drifted := false
	updated := existing.DeepCopy()

	// Replicas drift
	if existing.Spec.Replicas == nil || *existing.Spec.Replicas != spec.Replicas {
		replicas := spec.Replicas
		updated.Spec.Replicas = &replicas
		drifted = true
		logger.Info().
			Str("replicaset", spec.Name).
			Int32("desired", spec.Replicas).
			Msg("replicaset replicas drifted")
	}

	// Image drift
	if len(existing.Spec.Template.Spec.Containers) > 0 &&
		existing.Spec.Template.Spec.Containers[0].Image != spec.Image {
		updated.Spec.Template.Spec.Containers[0].Image = spec.Image
		drifted = true
		logger.Info().
			Str("replicaset", spec.Name).
			Str("desired", spec.Image).
			Msg("replicaset image drifted")
	}

	// Resources drift
	if spec.Resources != nil {
		desiredRes := common.BuildResourceRequirements(spec.Resources)
		var existingRes corev1.ResourceRequirements
		if len(existing.Spec.Template.Spec.Containers) > 0 {
			existingRes = existing.Spec.Template.Spec.Containers[0].Resources
		}
		if !common.ResourceRequirementsEqual(existingRes, desiredRes) {
			updated.Spec.Template.Spec.Containers[0].Resources = desiredRes
			drifted = true
			logger.Info().Str("replicaset", spec.Name).Msg("replicaset resources drifted")
		}
	}

	if !drifted {
		logger.Debug().
			Str("replicaset", spec.Name).
			Msg("replicaset in sync — no update needed")
		return nil
	}

	_, err = kube.Clientset().AppsV1().ReplicaSets(namespace).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("replicaset.Update: updating replicaset %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("replicaset", spec.Name).
		Str("namespace", namespace).
		Msg("replicaset updated")

	return nil
}

// Delete deletes the ReplicaSet if it exists.
func Delete(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedReplicaSetSpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	err := kube.Clientset().AppsV1().ReplicaSets(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug().
				Str("replicaset", spec.Name).
				Str("namespace", namespace).
				Msg("replicaset already deleted — skipping")
			return nil
		}
		return fmt.Errorf("replicaset.Delete: deleting replicaset %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("replicaset", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("replicaset deleted")

	return nil
}

// DeleteIfOwned deletes the ReplicaSet only if it is owned by the CR.
func DeleteIfOwned(ctx context.Context, kube kubeclient.KubeClient,
	owner domain.Object, name, namespace string) error {

	existing, err := kube.Clientset().AppsV1().ReplicaSets(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if existing.Labels[labels.OrkestraOwner] != owner.GetName() {
		return nil
	}

	return kube.Clientset().AppsV1().ReplicaSets(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedReplicaSetSpec from a ReplicaSetTemplateSource.
func Resolve(src orktypes.ReplicaSetTemplateSource, ownerName string) ResolvedReplicaSetSpec {
	spec := ResolvedReplicaSetSpec{
		Name:            src.Name,
		Image:           src.Image,
		Namespace:       src.Namespace,
		Resources:       common.ResolveResources(src.Resources),
		Labels:          make(map[string]string),
		Annotations:     make(map[string]string),
		EnvFrom:         src.EnvFrom,
		Probes:          src.Probes,
		SecurityContext: common.ResolveContainerSecurityContext(src.SecurityContext),
		PodSecurity:     common.ResolvePodSecurityContext(src.PodSecurity),
		Volumes:         src.Volumes,
		VolumeMounts:    src.VolumeMounts,
		Sleep:           src.Sleep,
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-replicaset"
	}

	spec.Replicas = common.ParseReplicas(src.Replicas)

	if src.Port != "" {
		if p, err := strconv.ParseInt(src.Port, 10, 32); err == nil {
			spec.Port = int32(p)
		}
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}
	for _, a := range src.Annotations {
		spec.Annotations[a.Key] = a.Value
	}
	for _, a := range src.NodeSelector {
		spec.NodeSelector[a] = a
	}

	spec.Env = []orktypes.EnvVar(src.Env)

	if src.RollingUpdate != nil && src.RollingUpdate.Profile != "" {
		expansion, err := profiles.ApplyRollingUpdateProfile(src.RollingUpdate.Profile)
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

	spec.Labels[labels.ManagedKey] = labels.ManagedValue
	spec.Labels[labels.OrkestraOwner] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildReplicaSet(owner domain.Object, spec ResolvedReplicaSetSpec, namespace string) *appsv1.ReplicaSet {
	logger.Debug().
		Interface("env", spec.Env).
		Interface("envFrom", spec.EnvFrom).
		Msg("replicaset.buildReplicaSet")

	replicas := spec.Replicas
	var pullSecrets []corev1.LocalObjectReference
	for _, name := range spec.ImagePullSecrets {
		pullSecrets = append(pullSecrets, corev1.LocalObjectReference{
			Name: name,
		})
	}

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        spec.Name,
			Namespace:   namespace,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         owner.GetObjectKind().GroupVersionKind().GroupVersion().String(),
					Kind:               owner.GetObjectKind().GroupVersionKind().Kind,
					Name:               owner.GetName(),
					UID:                owner.GetUID(),
					Controller:         utils.BoolPtr(true),
					BlockOwnerDeletion: utils.BoolPtr(true),
				},
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"orkestra-owner": owner.GetName(),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: spec.Labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:   pullSecrets,
					ServiceAccountName: spec.ServiceAccountName,
					NodeSelector:       spec.NodeSelector,
					Containers: []corev1.Container{
						{
							Name:  spec.Name,
							Image: spec.Image,
						},
					},
				},
			},
		},
	}

	if spec.Port > 0 {
		rs.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
			{ContainerPort: spec.Port},
		}
	}

	if spec.Resources != nil {
		rs.Spec.Template.Spec.Containers[0].Resources = common.BuildResourceRequirements(spec.Resources)
	}

	common.ApplyProbes(&rs.Spec.Template.Spec.Containers[0], spec.Probes, spec.Port)

	// Security
	common.ApplySecurityContext(&rs.Spec.Template.Spec.Containers[0], &rs.Spec.Template.Spec, spec.SecurityContext, spec.PodSecurity)

	if len(spec.Env) > 0 {
		rs.Spec.Template.Spec.Containers[0].Env = make([]corev1.EnvVar, 0, len(spec.Env))
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
			rs.Spec.Template.Spec.Containers[0].Env = append(rs.Spec.Template.Spec.Containers[0].Env, kev)
		}
	}

	if spec.EnvFrom != nil {
		for _, name := range spec.EnvFrom.SecretRef {
			rs.Spec.Template.Spec.Containers[0].EnvFrom = append(
				rs.Spec.Template.Spec.Containers[0].EnvFrom,
				corev1.EnvFromSource{
					SecretRef: &corev1.SecretEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: name},
					},
				})
		}
		for _, name := range spec.EnvFrom.ConfigMapRef {
			rs.Spec.Template.Spec.Containers[0].EnvFrom = append(
				rs.Spec.Template.Spec.Containers[0].EnvFrom,
				corev1.EnvFromSource{
					ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: name},
					},
				})
		}
	}

	// Volumes / VolumeMounts
	if vols := common.BuildVolumes(spec.Volumes); len(vols) > 0 {
		rs.Spec.Template.Spec.Volumes = vols
	}
	if mounts := common.BuildVolumeMounts(spec.VolumeMounts); len(mounts) > 0 {
		rs.Spec.Template.Spec.Containers[0].VolumeMounts = mounts
	}

	return rs
}

func validateSpec(spec ResolvedReplicaSetSpec) error {
	var missing []string
	if spec.Name == "" {
		missing = append(missing, "name")
	}
	if spec.Image == "" {
		missing = append(missing, "image")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %v", missing)
	}
	return nil
}
