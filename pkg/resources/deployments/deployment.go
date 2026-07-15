// pkg/resources/deployments/deployment.go
package deployments

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create creates a Deployment owned by the CR if it does not already exist.
// Idempotent — if the Deployment exists, does nothing and returns nil.
// Sets owner reference so the Deployment is garbage collected when the CR is deleted.
func Create(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedDeploymentSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("deployment.Create: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	_, err := kube.Clientset().AppsV1().Deployments(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("deployment.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("deployment", spec.Name).
			Str("namespace", namespace).
			Msg("deployment already exists — skipping create")
		return nil
	}

	deployment := buildDeployment(owner, spec, namespace)

	_, err = kube.Clientset().AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("deployment.Create: creating deployment %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("deployment", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("deployment created")

	return nil
}

// Update reconciles an existing Deployment to match the resolved spec.
// Handles drift — if replicas or image have changed, patches the Deployment.
// If the Deployment does not exist, creates it.
func Update(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedDeploymentSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("deployment.Update: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	existing, err := kube.Clientset().AppsV1().Deployments(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info().
				Str("deployment", spec.Name).
				Str("namespace", namespace).
				Msg("deployment not found during reconcile — recreating")
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("deployment.Update: getting deployment %q: %w", spec.Name, err)
	}

	desired := buildDeployment(owner, spec, namespace)
	drifted := false
	updated := existing.DeepCopy()

	// Replicas — skip when autoscaler owns spec.replicas to avoid fighting it.
	if !spec.HasAutoscale {
		if existing.Spec.Replicas == nil || *existing.Spec.Replicas != spec.Replicas {
			updated.Spec.Replicas = desired.Spec.Replicas
			drifted = true
			logger.Info().Str("deployment", spec.Name).Int32("desired", spec.Replicas).Msg("deployment replicas drifted")
		}
	}

	// Labels
	if !common.LabelsEqual(existing.Labels, desired.Labels) {
		updated.Labels = desired.Labels
		drifted = true
	}

	// Resources (uses custom equality to handle Kubernetes-defaulted values)
	if spec.Resources != nil {
		desiredRes := common.BuildResourceRequirements(spec.Resources)
		var existingRes corev1.ResourceRequirements
		if len(existing.Spec.Template.Spec.Containers) > 0 {
			existingRes = existing.Spec.Template.Spec.Containers[0].Resources
		}
		if !common.ResourceRequirementsEqual(existingRes, desiredRes) {
			updated.Spec.Template.Spec.Containers[0].Resources = desiredRes
			drifted = true
			logger.Info().Str("deployment", spec.Name).Msg("deployment resources drifted")
		}
	}

	// All other container and pod-spec fields from the template
	if len(updated.Spec.Template.Spec.Containers) > 0 && len(desired.Spec.Template.Spec.Containers) > 0 {
		if common.SyncContainerSpec(&updated.Spec.Template.Spec.Containers[0], desired.Spec.Template.Spec.Containers[0]) {
			drifted = true
		}
	}
	if common.SyncPodSpec(&updated.Spec.Template.Spec, desired.Spec.Template.Spec) {
		drifted = true
	}

	if !drifted {
		logger.Debug().Str("deployment", spec.Name).Msg("deployment in sync — no update needed")
		return nil
	}

	_, err = kube.Clientset().AppsV1().Deployments(namespace).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("deployment.Update: updating deployment %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("deployment", spec.Name).
		Str("namespace", namespace).
		Msg("deployment updated")

	return nil
}

// Delete deletes the Deployment if it exists.
// For most cases owner references handle cascade deletion — use this only
// for explicit cleanup declared in onDelete templates.
func Delete(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedDeploymentSpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	err := kube.Clientset().AppsV1().Deployments(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug().
				Str("deployment", spec.Name).
				Str("namespace", namespace).
				Msg("deployment already deleted — skipping")
			return nil
		}
		return fmt.Errorf("deployment.Delete: deleting deployment %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("deployment", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("deployment deleted")

	return nil
}

// DeleteIfOwned deletes the Deployment if it exists and is owned by the CR.
func DeleteIfOwned(ctx context.Context, kube kubeclient.KubeClient,
	owner domain.Object, name, namespace string) error {

	existing, err := kube.Clientset().AppsV1().Deployments(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	// Only delete if we own it
	if existing.Labels[labels.OrkestraOwner] != owner.GetName() {
		return nil
	}
	return kube.Clientset().AppsV1().Deployments(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedDeploymentSpec from a DeploymentTemplateSource.
// Fields with template expressions must already be evaluated before calling Resolve.
// Use pkg/orkestra-registry/template.Resolver to evaluate expressions first.
//
// The resolver already evaluated template expressions — here we just merge.
func Resolve(src orktypes.DeploymentTemplateSource, ownerName string, reg orktypes.ProfileRegistry) ResolvedDeploymentSpec {
	spec := ResolvedDeploymentSpec{
		Name:            src.Name,
		Image:           src.Image,
		Namespace:       src.Namespace,
		Resources:       common.ResolveResources(src.Resources, reg),
		Labels:          make(map[string]string),
		Annotations:     make(map[string]string),
		EnvFrom:         src.EnvFrom,
		Probes:          src.Probes,
		Profiles:        reg,
		SecurityContext: common.ResolveContainerSecurityContext(src.SecurityContext, reg),
		PodSecurity:     common.ResolvePodSecurityContext(src.PodSecurity, reg),
		Volumes:         src.Volumes,
		VolumeMounts:    src.VolumeMounts,
		Sleep:           src.Sleep,
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-deployment"
	}

	spec.Replicas = common.ParseReplicas(src.Replicas)
	spec.HasAutoscale = src.Autoscale != nil

	// Port — prefer dynamic resolved string, fall back to static int
	if src.Port != "" {
		if p, err := strconv.ParseInt(src.Port, 10, 32); err == nil {
			spec.Port = int32(p)
		}
	}

	spec.Protocol = common.ParseProtocol(src.Protocol)

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}
	for _, a := range src.Annotations {
		spec.Annotations[a.Key] = a.Value
	}

	spec.Env = []orktypes.EnvVar(src.Env)

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

	// Orkestra system labels — always added
	spec.Labels[labels.ManagedKey] = labels.ManagedValue
	spec.Labels[labels.OrkestraOwner] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildDeployment(owner domain.Object, spec ResolvedDeploymentSpec, namespace string) *appsv1.Deployment {
	// Debug line
	logger.Debug().
		Interface("env", spec.Env).
		Interface("envFrom", spec.EnvFrom).
		Msg("deployment.buildDeployment")

	replicas := spec.Replicas

	d := &appsv1.Deployment{
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
		Spec: appsv1.DeploymentSpec{
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
					ImagePullSecrets:   common.ToPullSecrets(spec.ImagePullSecrets),
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

	// Port — Protocol is resolved in Resolve() and defaults to TCP when not declared.
	if spec.Port > 0 {
		d.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
			{ContainerPort: spec.Port, Protocol: spec.Protocol},
		}
	}

	// Resources
	if spec.Resources != nil {
		d.Spec.Template.Spec.Containers[0].Resources = common.BuildResourceRequirements(spec.Resources)
	}

	// Probes
	common.ApplyProbes(&d.Spec.Template.Spec.Containers[0], spec.Probes, spec.Port, spec.Profiles)

	// Rolling update strategy
	if spec.RollingUpdate != nil {
		d.Spec.Strategy = common.BuildDeploymentRollingUpdateStrategy(spec.RollingUpdate)
	}

	// Security
	common.ApplySecurityContext(&d.Spec.Template.Spec.Containers[0], &d.Spec.Template.Spec, spec.SecurityContext, spec.PodSecurity)

	// Env
	if len(spec.Env) > 0 {
		d.Spec.Template.Spec.Containers[0].Env = make([]corev1.EnvVar, 0, len(spec.Env))
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
			d.Spec.Template.Spec.Containers[0].Env = append(d.Spec.Template.Spec.Containers[0].Env, kev)
		}
	}

	// EnvFrom
	if spec.EnvFrom != nil {
		for _, name := range spec.EnvFrom.SecretRef {
			d.Spec.Template.Spec.Containers[0].EnvFrom = append(
				d.Spec.Template.Spec.Containers[0].EnvFrom,
				corev1.EnvFromSource{
					SecretRef: &corev1.SecretEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: name},
					},
				})
		}
		for _, name := range spec.EnvFrom.ConfigMapRef {
			d.Spec.Template.Spec.Containers[0].EnvFrom = append(
				d.Spec.Template.Spec.Containers[0].EnvFrom,
				corev1.EnvFromSource{
					ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: name},
					},
				})
		}
	}

	// Volumes / VolumeMounts
	if vols := common.BuildVolumes(spec.Volumes); len(vols) > 0 {
		d.Spec.Template.Spec.Volumes = vols
	}
	if mounts := common.BuildVolumeMounts(spec.VolumeMounts); len(mounts) > 0 {
		d.Spec.Template.Spec.Containers[0].VolumeMounts = mounts
	}

	return d
}

func validateSpec(spec ResolvedDeploymentSpec) error {
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
