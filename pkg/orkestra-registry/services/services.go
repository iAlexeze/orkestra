// pkg/orkestra-registry/services/service.go
package services

import (
	"context"
	"fmt"
	"strconv"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Create creates a Service owned by the CR if it does not already exist.
// Idempotent — if the Service exists, does nothing and returns nil.
// Owner reference is set so the Service is garbage collected when the CR is deleted.
func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedServiceSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("service.Create: invalid spec: %w", err)
	}

	namespace := resolveNamespace(owner, spec)

	_, err := kube.Clientset().CoreV1().Services(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("service.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("service", spec.Name).
			Str("namespace", namespace).
			Msg("service already exists — skipping create")
		return nil
	}

	svc := buildService(owner, spec, namespace)

	_, err = kube.Clientset().CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("service.Create: creating service %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("service", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("service created")

	return nil
}

// Update reconciles an existing Service to match the resolved spec.
// Services are largely immutable on type/selector — only port changes are patched.
// If the Service does not exist, creates it.
func Update(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedServiceSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("service.Update: invalid spec: %w", err)
	}

	namespace := resolveNamespace(owner, spec)

	existing, err := kube.Clientset().CoreV1().Services(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info().
				Str("service", spec.Name).
				Str("namespace", namespace).
				Msg("service not found during reconcile — recreating")
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("service.Update: getting service %q: %w", spec.Name, err)
	}

	// Check port drift
	drifted := false
	updated := existing.DeepCopy()

	if len(existing.Spec.Ports) > 0 && existing.Spec.Ports[0].Port != spec.Port {
		updated.Spec.Ports[0].Port = spec.Port
		updated.Spec.Ports[0].TargetPort = intstr.FromInt(int(spec.TargetPort))
		drifted = true
		logger.Info().
			Str("service", spec.Name).
			Int32("desired", spec.Port).
			Msg("service port drifted")
	}

	if !drifted {
		logger.Debug().
			Str("service", spec.Name).
			Msg("service in sync — no update needed")
		return nil
	}

	_, err = kube.Clientset().CoreV1().Services(namespace).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("service.Update: updating service %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("service", spec.Name).
		Str("namespace", namespace).
		Msg("service updated")

	return nil
}

// Delete deletes the Service if it exists.
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedServiceSpec) error {
	namespace := resolveNamespace(owner, spec)

	err := kube.Clientset().CoreV1().Services(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug().
				Str("service", spec.Name).
				Str("namespace", namespace).
				Msg("service already deleted — skipping")
			return nil
		}
		return fmt.Errorf("service.Delete: deleting service %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("service", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("service deleted")

	return nil
}

// DeleteIfOwned deletes the Service if it exists and is owned by the CR.
func DeleteIfOwned(ctx context.Context, kube *kubeclient.Kubeclient,
	owner domain.Object, name, namespace string) error {

	existing, err := kube.Clientset().CoreV1().Services(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	// Only delete if we own it
	if existing.Labels[konfig.LabelOrkestraOwner] != owner.GetName() {
		return nil
	}
	return kube.Clientset().CoreV1().Services(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedServiceSpec from a ServiceTemplateSource.
// All fields already resolved by template.Resolver before calling here.
// This function assembles the spec and applies defaults.
func Resolve(src orktypes.ServiceTemplateSource, ownerName string) ResolvedServiceSpec {
	spec := ResolvedServiceSpec{
		Name:   src.Name,
		Labels: make(map[string]string),
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-svc"
	}

	spec.Namespace = src.Namespace

	spec.Type = src.Type
	if spec.Type == "" {
		spec.Type = "ClusterIP"
	}

	if src.Port != "" {
		if p, err := strconv.ParseInt(src.Port, 10, 32); err == nil {
			spec.Port = int32(p)
		}
	}

	if src.TargetPort != "" {
		if p, err := strconv.ParseInt(src.TargetPort, 10, 32); err == nil {
			spec.TargetPort = int32(p)
		}
	}

	// If TargetPort not set, default to Port
	if spec.TargetPort == 0 {
		spec.TargetPort = spec.Port
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}

	// System labels
	spec.Labels[konfig.LabelManaged] = konfig.LabelManagedValue
	spec.Labels[konfig.LabelOrkestraOwner] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildService(owner domain.Object, spec ResolvedServiceSpec, namespace string) *corev1.Service {
	svcType := corev1.ServiceTypeClusterIP
	switch spec.Type {
	case "NodePort":
		svcType = corev1.ServiceTypeNodePort
	case "LoadBalancer":
		svcType = corev1.ServiceTypeLoadBalancer
	}

	// Selector matches pods created by deployments with the same owner
	selector := map[string]string{
		"orkestra-owner": owner.GetName(),
	}

	// For unstructured owners the GVK may not be set on the object itself —
	// use the unstructured helper to get it
	// TODO: If there is a better way
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

	return &corev1.Service{
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
		Spec: corev1.ServiceSpec{
			Type:     svcType,
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Port:       spec.Port,
					TargetPort: intstr.FromInt(int(spec.TargetPort)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

func validateSpec(spec ResolvedServiceSpec) error {
	var missing []string
	if spec.Name == "" {
		missing = append(missing, "name")
	}
	if spec.Port == 0 {
		missing = append(missing, "port")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %v", missing)
	}
	return nil
}

func resolveNamespace(owner domain.Object, spec ResolvedServiceSpec) string {
	if spec.Namespace != "" {
		return spec.Namespace
	}
	if owner.GetNamespace() != "" {
		return owner.GetNamespace()
	}
	return "default"
}
