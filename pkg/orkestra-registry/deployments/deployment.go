// pkg/orkestra-registry/deployments/deployment.go
package deployments

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create creates a Deployment owned by the CR if it does not already exist.
// Idempotent — if the Deployment exists, does nothing and returns nil.
// Sets owner reference so the Deployment is garbage collected when the CR is deleted.
func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedDeploymentSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("deployment.Create: invalid spec: %w", err)
	}

	namespace := resolveNamespace(owner, spec)

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
func Update(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedDeploymentSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("deployment.Update: invalid spec: %w", err)
	}

	namespace := resolveNamespace(owner, spec)

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

	// Check for drift — replicas and image are the reconcilable fields
	drifted := false
	updated := existing.DeepCopy()

	if existing.Spec.Replicas == nil || *existing.Spec.Replicas != spec.Replicas {
		replicas := spec.Replicas
		updated.Spec.Replicas = &replicas
		drifted = true
		logger.Info().
			Str("deployment", spec.Name).
			Int32("desired", spec.Replicas).
			Msg("deployment replicas drifted")
	}

	if len(existing.Spec.Template.Spec.Containers) > 0 &&
		existing.Spec.Template.Spec.Containers[0].Image != spec.Image {
		updated.Spec.Template.Spec.Containers[0].Image = spec.Image
		drifted = true
		logger.Info().
			Str("deployment", spec.Name).
			Str("desired", spec.Image).
			Msg("deployment image drifted")
	}

	if !drifted {
		logger.Debug().
			Str("deployment", spec.Name).
			Msg("deployment in sync — no update needed")
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
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedDeploymentSpec) error {
	namespace := resolveNamespace(owner, spec)

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

// Resolve builds a ResolvedDeploymentSpec from a DeploymentTemplateSource.
// Fields with template expressions must already be evaluated before calling Resolve.
// Use pkg/orkestra-registry/template.Resolver to evaluate expressions first.
//
// The resolver already evaluated template expressions — here we just merge.
// deployments/deployment.go
func Resolve(src orktypes.DeploymentTemplateSource, staticReplicas int, ownerName string) ResolvedDeploymentSpec {
	spec := ResolvedDeploymentSpec{
		Name:        src.Name,
		Image:       src.Image,
		Namespace:   src.Namespace,
		Resources:   src.Resources,
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
	}

	// Default name
	if spec.Name == "" {
		spec.Name = ownerName + "-deployment"
	}

	// Replicas — prefer dynamic resolved string, fall back to static int
	if src.Replicas != "" {
		if r, err := strconv.ParseInt(src.Replicas, 10, 32); err == nil {
			spec.Replicas = int32(r)
		}
	}
	if spec.Replicas == 0 && staticReplicas > 0 {
		spec.Replicas = int32(staticReplicas)
	}
	if spec.Replicas == 0 {
		spec.Replicas = 1 // default
	}

	// Port — prefer dynamic resolved string, fall back to static int
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

	// Orkestra system labels — always added
	spec.Labels[konfig.LabelManaged] = konfig.LabelManagedValue
	spec.Labels[konfig.LabelOrkestraOwner] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildDeployment(owner domain.Object, spec ResolvedDeploymentSpec, namespace string) *appsv1.Deployment {
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

	// Port
	if spec.Port > 0 {
		d.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
			{ContainerPort: spec.Port},
		}
	}

	// Resources
	if spec.Resources != nil {
		d.Spec.Template.Spec.Containers[0].Resources = buildResourceRequirements(spec.Resources)
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

func resolveNamespace(owner domain.Object, spec ResolvedDeploymentSpec) string {
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
