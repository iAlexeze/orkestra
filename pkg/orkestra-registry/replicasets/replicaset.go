// pkg/orkestra-registry/replicasets/replicaset.go
package replicasets

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create creates a ReplicaSet owned by the CR if it does not already exist.
// Idempotent — if the ReplicaSet exists, does nothing and returns nil.
func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedReplicaSetSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("replicaset.Create: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)

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
func Update(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedReplicaSetSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("replicaset.Update: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)

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
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedReplicaSetSpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)

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
func DeleteIfOwned(ctx context.Context, kube *kubeclient.Kubeclient,
	owner domain.Object, name, namespace string) error {

	existing, err := kube.Clientset().AppsV1().ReplicaSets(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if existing.Labels[konfig.LabelOrkestraOwner] != owner.GetName() {
		return nil
	}

	return kube.Clientset().AppsV1().ReplicaSets(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedReplicaSetSpec from a ReplicaSetTemplateSource.
func Resolve(src orktypes.ReplicaSetTemplateSource, staticReplicas int, ownerName string) ResolvedReplicaSetSpec {
	spec := ResolvedReplicaSetSpec{
		Name:        src.Name,
		Image:       src.Image,
		Namespace:   src.Namespace,
		Resources:   src.Resources,
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
		Env:         make(map[string]orktypes.EnvVarSource),
		EnvFrom:     src.EnvFrom,
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-replicaset"
	}

	if src.Replicas != "" {
		if r, err := strconv.ParseInt(src.Replicas, 10, 32); err == nil {
			spec.Replicas = int32(r)
		}
	}
	if spec.Replicas == 0 && staticReplicas > 0 {
		spec.Replicas = int32(staticReplicas)
	}
	if spec.Replicas == 0 {
		spec.Replicas = 1
	}

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

	for k, v := range src.Env {
		spec.Env[k] = v
	}

	spec.Labels[konfig.LabelManaged] = konfig.LabelManagedValue
	spec.Labels[konfig.LabelOrkestraOwner] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildReplicaSet(owner domain.Object, spec ResolvedReplicaSetSpec, namespace string) *appsv1.ReplicaSet {
	logger.Debug().
		Interface("env", spec.Env).
		Interface("envFrom", spec.EnvFrom).
		Msg("replicaset.buildReplicaSet")

	replicas := spec.Replicas

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

	if len(spec.Env) > 0 {
		rs.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{}
		for k, src := range spec.Env {
			ev := corev1.EnvVar{Name: k}

			switch {
			case src.SecretKeyRef != nil:
				ev.ValueFrom = &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: src.SecretKeyRef.Name,
						},
						Key: src.SecretKeyRef.Key,
					},
				}

			case src.ConfigMapKeyRef != nil:
				ev.ValueFrom = &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: src.ConfigMapKeyRef.Name,
						},
						Key: src.ConfigMapKeyRef.Key,
					},
				}

			default:
				ev.Value = src.Value
			}

			rs.Spec.Template.Spec.Containers[0].Env =
				append(rs.Spec.Template.Spec.Containers[0].Env, ev)
		}
	}

	if len(spec.EnvFrom) > 0 {
		rs.Spec.Template.Spec.Containers[0].EnvFrom = []corev1.EnvFromSource{}
		for _, src := range spec.EnvFrom {
			if src.ConfigMapRef != "" {
				rs.Spec.Template.Spec.Containers[0].EnvFrom = append(
					rs.Spec.Template.Spec.Containers[0].EnvFrom,
					corev1.EnvFromSource{
						ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: src.ConfigMapRef,
							},
						},
					},
				)
			}
			if src.SecretRef != "" {
				rs.Spec.Template.Spec.Containers[0].EnvFrom = append(
					rs.Spec.Template.Spec.Containers[0].EnvFrom,
					corev1.EnvFromSource{
						SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: src.SecretRef,
							},
						},
					},
				)
			}
		}
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
	if spec.Env == nil {
		spec.Env = map[string]orktypes.EnvVarSource{}
	}
	if spec.EnvFrom == nil {
		spec.EnvFrom = []orktypes.EnvFromSource{}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %v", missing)
	}
	return nil
}
