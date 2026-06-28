// pkg/resources/pods/pod.go
package pods

import (
	"context"
	"fmt"
	"reflect"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/resources/common"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create creates a Pod owned by the CR if it does not already exist.
// Idempotent — if the Pod exists, does nothing and returns nil.
// Owner reference is set so the Pod is garbage collected when the CR is deleted.
func Create(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedPodSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("pod.Create: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	_, err := kube.Clientset().CoreV1().Pods(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("pod.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("pod", spec.Name).
			Str("namespace", namespace).
			Msg("pod already exists — skipping create")
		return nil
	}

	pod := buildPod(owner, spec, namespace)

	_, err = kube.Clientset().CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("pod.Create: creating pod %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("pod", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("pod created")

	return nil
}

// Update reconciles an existing Pod to match the resolved spec.
// Pods are largely immutable — image drift triggers delete + recreate.
// If the Pod does not exist, creates it.
func Update(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedPodSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("pod.Update: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	existing, err := kube.Clientset().CoreV1().Pods(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info().
				Str("pod", spec.Name).
				Str("namespace", namespace).
				Msg("pod not found during reconcile — recreating")
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("pod.Update: getting pod %q: %w", spec.Name, err)
	}

	// Pods are largely immutable — drift is handled by delete + recreate.
	// Build the desired pod and compare all template-declared fields.
	desired := buildPod(owner, spec, namespace)
	needsRecreate := false

	if len(existing.Spec.Containers) > 0 && len(desired.Spec.Containers) > 0 {
		ec := existing.Spec.Containers[0]
		dc := desired.Spec.Containers[0]
		if ec.Image != dc.Image ||
			!common.ResourceRequirementsEqual(ec.Resources, dc.Resources) ||
			!reflect.DeepEqual(ec.Env, dc.Env) ||
			!reflect.DeepEqual(ec.EnvFrom, dc.EnvFrom) ||
			!reflect.DeepEqual(ec.VolumeMounts, dc.VolumeMounts) ||
			!reflect.DeepEqual(ec.LivenessProbe, dc.LivenessProbe) ||
			!reflect.DeepEqual(ec.ReadinessProbe, dc.ReadinessProbe) ||
			!reflect.DeepEqual(ec.StartupProbe, dc.StartupProbe) ||
			!reflect.DeepEqual(ec.SecurityContext, dc.SecurityContext) {
			logger.Info().Str("pod", spec.Name).Msg("pod container spec drifted — deleting and recreating")
			needsRecreate = true
		}
	}
	if !needsRecreate {
		if !reflect.DeepEqual(existing.Spec.Volumes, desired.Spec.Volumes) ||
			!reflect.DeepEqual(existing.Spec.SecurityContext, desired.Spec.SecurityContext) ||
			existing.Spec.ServiceAccountName != desired.Spec.ServiceAccountName {
			logger.Info().Str("pod", spec.Name).Msg("pod spec drifted — deleting and recreating")
			needsRecreate = true
		}
	}
	if needsRecreate {
		if err := Delete(ctx, kube, owner, spec); err != nil {
			return err
		}
		return Create(ctx, kube, owner, spec)
	}

	logger.Debug().
		Str("pod", spec.Name).
		Str("namespace", namespace).
		Msg("pod in sync — no update needed")

	return nil
}

// Delete deletes the Pod if it exists.
// For most cases owner references handle cascade deletion automatically —
// only use this when you need explicit cleanup control in onDelete.
func Delete(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedPodSpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	err := kube.Clientset().CoreV1().Pods(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug().
				Str("pod", spec.Name).
				Str("namespace", namespace).
				Msg("pod already deleted — skipping")
			return nil
		}
		return fmt.Errorf("pod.Delete: deleting pod %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("pod", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("pod deleted")

	return nil
}

// DeleteIfOwned deletes the Pod if it exists and is owned by the CR.
func DeleteIfOwned(ctx context.Context, kube kubeclient.KubeClient,
	owner domain.Object, name, namespace string) error {

	existing, err := kube.Clientset().CoreV1().Pods(namespace).
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
	return kube.Clientset().CoreV1().Pods(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedPodSpec from a PodTemplateSource.
//
// Fields are flat on the source struct. Template expressions
// must already be evaluated by template.Resolver before calling Resolve.
// This function only reads already-resolved string values and assembles the spec.
//
// Orkestra system labels (managed-by, orkestra-owner) are always added
// and cannot be overridden by the user.
func Resolve(src orktypes.PodTemplateSource, ownerName string, reg orktypes.ProfileRegistry) ResolvedPodSpec {
	spec := ResolvedPodSpec{
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
	}

	spec.Name = src.Name
	if spec.Name == "" {
		spec.Name = ownerName + "-pod"
	}

	spec.Image = src.Image
	spec.Namespace = src.Namespace
	spec.Resources = src.Resources
	spec.Probes = src.Probes
	spec.Profiles = reg
	spec.SecurityContext = common.ResolveContainerSecurityContext(src.SecurityContext, reg)
	spec.PodSecurity = common.ResolvePodSecurityContext(src.PodSecurity, reg)
	spec.Volumes = src.Volumes
	spec.VolumeMounts = src.VolumeMounts
	spec.Sleep = src.Sleep

	if src.Port != "" {
		spec.Port = common.ParsePort(src.Port)
	}
	spec.Protocol = common.ParseProtocol(src.Protocol)

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}
	for _, a := range src.Annotations {
		spec.Annotations[a.Key] = a.Value
	}

	// System labels — always present
	spec.Labels[labels.ManagedKey] = labels.ManagedValue
	spec.Labels[labels.OrkestraOwner] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildPod(owner domain.Object, spec ResolvedPodSpec, namespace string) *corev1.Pod {
	pod := &corev1.Pod{
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
	}

	if spec.Port > 0 {
		pod.Spec.Containers[0].Ports = []corev1.ContainerPort{
			{ContainerPort: int32(spec.Port), Protocol: spec.Protocol},
		}
	}

	if spec.Resources != nil {
		pod.Spec.Containers[0].Resources = common.BuildResourceRequirements(spec.Resources)
	}

	common.ApplyProbes(&pod.Spec.Containers[0], spec.Probes, int32(spec.Port), spec.Profiles)

	// Security
	common.ApplySecurityContext(&pod.Spec.Containers[0], &pod.Spec, spec.SecurityContext, spec.PodSecurity)

	// Volumes / VolumeMounts
	if vols := common.BuildVolumes(spec.Volumes); len(vols) > 0 {
		pod.Spec.Volumes = vols
	}
	if mounts := common.BuildVolumeMounts(spec.VolumeMounts); len(mounts) > 0 {
		pod.Spec.Containers[0].VolumeMounts = mounts
	}

	return pod
}

func validateSpec(spec ResolvedPodSpec) error {
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
