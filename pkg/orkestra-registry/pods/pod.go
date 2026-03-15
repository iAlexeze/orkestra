// pkg/orkestra-registry/pods/pod.go
package pods

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create creates a Pod owned by the CR if it does not already exist.
// Idempotent — if the Pod exists, does nothing and returns nil.
// Owner reference is set so the Pod is garbage collected when the CR is deleted.
func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedPodSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("pod.Create: invalid spec: %w", err)
	}

	namespace := resolveNamespace(owner, spec)

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
func Update(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedPodSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("pod.Update: invalid spec: %w", err)
	}

	namespace := resolveNamespace(owner, spec)

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

	if len(existing.Spec.Containers) > 0 && existing.Spec.Containers[0].Image != spec.Image {
		logger.Info().
			Str("pod", spec.Name).
			Str("current", existing.Spec.Containers[0].Image).
			Str("desired", spec.Image).
			Msg("pod image drifted — deleting and recreating")

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
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedPodSpec) error {
	namespace := resolveNamespace(owner, spec)

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

// Resolve builds a ResolvedPodSpec from a PodTemplateSource.
//
// Option B — fields are flat on the source struct. Template expressions
// must already be evaluated by template.Resolver before calling Resolve.
// This function only reads already-resolved string values and assembles the spec.
//
// Orkestra system labels (managed-by, orkestra-owner) are always added
// and cannot be overridden by the user.
func Resolve(src orktypes.PodTemplateSource, ownerName string) ResolvedPodSpec {
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

	if src.Port != "" {
		spec.Port = parsePort(src.Port)
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}
	for _, a := range src.Annotations {
		spec.Annotations[a.Key] = a.Value
	}

	// System labels — always present
	spec.Labels["managed-by"] = "orkestra"
	spec.Labels["orkestra-owner"] = ownerName

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
					Controller:         boolPtr(true),
					BlockOwnerDeletion: boolPtr(true),
				},
			},
		},
		Spec: corev1.PodSpec{
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
			{ContainerPort: int32(spec.Port)},
		}
	}

	if spec.Resources != nil {
		pod.Spec.Containers[0].Resources = buildResourceRequirements(spec.Resources)
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

// resolveNamespace — priority: spec.Namespace → owner namespace → "default"
func resolveNamespace(owner domain.Object, spec ResolvedPodSpec) string {
	if spec.Namespace != "" {
		return spec.Namespace
	}
	if owner.GetNamespace() != "" {
		return owner.GetNamespace()
	}
	return "default"
}

func buildResourceRequirements(r *orktypes.ResourceRequirements) corev1.ResourceRequirements {
	req := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}
	for k, v := range r.Requests {
		req.Requests[corev1.ResourceName(k)] = resource.MustParse(v)
	}
	for k, v := range r.Limits {
		req.Limits[corev1.ResourceName(k)] = resource.MustParse(v)
	}
	return req
}

func parsePort(s string) int {
	var p int
	fmt.Sscanf(s, "%d", &p)
	return p
}

func boolPtr(b bool) *bool { return &b }
