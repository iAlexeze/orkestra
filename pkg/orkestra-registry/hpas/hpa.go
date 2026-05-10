// pkg/orkestra-registry/hpas/hpa.go
package hpas

import (
	"context"
	"fmt"
	"strconv"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/orkestra-registry/common"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Create creates an HPA owned by the CR if it does not already exist.
// Idempotent — if the HPA exists, does nothing and returns nil.
func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedHPASpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("hpa.Create: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	_, err := kube.Clientset().AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("hpa.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("hpa", spec.Name).
			Str("namespace", namespace).
			Msg("hpa already exists — skipping create")
		return nil
	}

	hpa := buildHPA(owner, spec, namespace)

	_, err = kube.Clientset().AutoscalingV2().HorizontalPodAutoscalers(namespace).Create(ctx, hpa, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("hpa.Create: creating hpa %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("hpa", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("hpa created")

	return nil
}

// Update reconciles an existing HPA to match the resolved spec.
// Patches min/max replicas and CPU target when drift is detected.
// If the HPA does not exist, creates it.
func Update(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedHPASpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("hpa.Update: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	existing, err := kube.Clientset().AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info().
				Str("hpa", spec.Name).
				Str("namespace", namespace).
				Msg("hpa not found during reconcile — recreating")
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("hpa.Update: getting hpa %q: %w", spec.Name, err)
	}

	drifted := false
	updated := existing.DeepCopy()

	minR := spec.MinReplicas
	if existing.Spec.MinReplicas == nil || *existing.Spec.MinReplicas != minR {
		updated.Spec.MinReplicas = &minR
		drifted = true
		logger.Info().Str("hpa", spec.Name).Int32("desired", minR).Msg("hpa minReplicas drifted")
	}

	if existing.Spec.MaxReplicas != spec.MaxReplicas {
		updated.Spec.MaxReplicas = spec.MaxReplicas
		drifted = true
		logger.Info().Str("hpa", spec.Name).Int32("desired", spec.MaxReplicas).Msg("hpa maxReplicas drifted")
	}

	if !drifted {
		logger.Debug().Str("hpa", spec.Name).Msg("hpa in sync — no update needed")
		return nil
	}

	_, err = kube.Clientset().AutoscalingV2().HorizontalPodAutoscalers(namespace).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("hpa.Update: updating hpa %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("hpa", spec.Name).
		Str("namespace", namespace).
		Msg("hpa updated")

	return nil
}

// Delete deletes the HPA if it exists.
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedHPASpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	err := kube.Clientset().AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug().
				Str("hpa", spec.Name).
				Str("namespace", namespace).
				Msg("hpa already deleted — skipping")
			return nil
		}
		return fmt.Errorf("hpa.Delete: deleting hpa %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("hpa", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("hpa deleted")

	return nil
}

// DeleteIfOwned deletes the HPA if it exists and is owned by the CR.
func DeleteIfOwned(ctx context.Context, kube *kubeclient.Kubeclient,
	owner domain.Object, name, namespace string) error {

	existing, err := kube.Clientset().AutoscalingV2().HorizontalPodAutoscalers(namespace).
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
	return kube.Clientset().AutoscalingV2().HorizontalPodAutoscalers(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedHPASpec from an HPATemplateSource.
// All template expressions must be evaluated before calling here.
func Resolve(src orktypes.HPATemplateSource, ownerName string) ResolvedHPASpec {
	spec := ResolvedHPASpec{
		Name:           src.Name,
		Namespace:      src.Namespace,
		ScaleTargetRef: src.ScaleTargetRef,
		MinReplicas:    1,
		MaxReplicas:    1,
		Labels:         make(map[string]string),
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-hpa"
	}
	if spec.ScaleTargetRef.Name == "" {
		spec.ScaleTargetRef.Name = ownerName
	}

	if src.MinReplicas != "" {
		if v, err := strconv.ParseInt(src.MinReplicas, 10, 32); err == nil {
			spec.MinReplicas = int32(v)
		}
	}
	if src.MaxReplicas != "" {
		if v, err := strconv.ParseInt(src.MaxReplicas, 10, 32); err == nil {
			spec.MaxReplicas = int32(v)
		}
	}
	if src.TargetCPUUtilizationPercentage != "" {
		if v, err := strconv.ParseInt(src.TargetCPUUtilizationPercentage, 10, 32); err == nil {
			spec.TargetCPUUtilizationPercentage = int32(v)
		}
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}

	// System labels
	spec.Labels[labels.Managed] = labels.ManagedValue
	spec.Labels[labels.OrkestraOwner] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildHPA(owner domain.Object, spec ResolvedHPASpec, namespace string) *autoscalingv2.HorizontalPodAutoscaler {
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

	minR := spec.MinReplicas

	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: namespace,
			Labels:    spec.Labels,
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
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: spec.ScaleTargetRef.APIVersion,
				Kind:       spec.ScaleTargetRef.Kind,
				Name:       spec.ScaleTargetRef.Name,
			},
			MinReplicas: &minR,
			MaxReplicas: spec.MaxReplicas,
		},
	}

	if spec.TargetCPUUtilizationPercentage > 0 {
		cpuTarget := spec.TargetCPUUtilizationPercentage
		hpa.Spec.Metrics = []autoscalingv2.MetricSpec{
			{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: corev1.ResourceCPU,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: &cpuTarget,
					},
				},
			},
		}
	}

	return hpa
}

func validateSpec(spec ResolvedHPASpec) error {
	if spec.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	return nil
}
