// pkg/orkestra-registry/ingresses/ingress.go
package ingresses

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
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Create creates an Ingress owned by the CR if it does not already exist.
// Idempotent — if the Ingress exists, does nothing and returns nil.
func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedIngressSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("ingress.Create: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)

	_, err := kube.Clientset().NetworkingV1().Ingresses(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("ingress.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("ingress", spec.Name).
			Str("namespace", namespace).
			Msg("ingress already exists — skipping create")
		return nil
	}

	ing := buildIngress(owner, spec, namespace)

	_, err = kube.Clientset().NetworkingV1().Ingresses(namespace).Create(ctx, ing, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("ingress.Create: creating ingress %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("ingress", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("ingress created")

	return nil
}

// Update reconciles an existing Ingress to match the resolved spec.
// Patches host, backend service, port, and TLS when drift is detected.
// If the Ingress does not exist, creates it.
func Update(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedIngressSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("ingress.Update: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)

	existing, err := kube.Clientset().NetworkingV1().Ingresses(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info().
				Str("ingress", spec.Name).
				Str("namespace", namespace).
				Msg("ingress not found during reconcile — recreating")
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("ingress.Update: getting ingress %q: %w", spec.Name, err)
	}

	desired := buildIngress(owner, spec, namespace)
	drifted := false
	updated := existing.DeepCopy()

	// Reconcile rules
	if len(desired.Spec.Rules) > 0 {
		if len(existing.Spec.Rules) == 0 ||
			existing.Spec.Rules[0].Host != desired.Spec.Rules[0].Host ||
			(len(existing.Spec.Rules[0].HTTP.Paths) > 0 &&
				existing.Spec.Rules[0].HTTP.Paths[0].Backend.Service != nil &&
				existing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number != spec.ServicePort) {
			updated.Spec.Rules = desired.Spec.Rules
			drifted = true
		}
	}

	// Reconcile TLS
	tlsMatch := len(existing.Spec.TLS) == len(desired.Spec.TLS)
	if tlsMatch && len(desired.Spec.TLS) > 0 {
		tlsMatch = existing.Spec.TLS[0].SecretName == desired.Spec.TLS[0].SecretName
	}
	if !tlsMatch {
		updated.Spec.TLS = desired.Spec.TLS
		drifted = true
	}

	// Reconcile IngressClassName
	if desired.Spec.IngressClassName != nil {
		if existing.Spec.IngressClassName == nil || *existing.Spec.IngressClassName != *desired.Spec.IngressClassName {
			updated.Spec.IngressClassName = desired.Spec.IngressClassName
			drifted = true
		}
	}

	if !drifted {
		logger.Debug().Str("ingress", spec.Name).Msg("ingress in sync — no update needed")
		return nil
	}

	_, err = kube.Clientset().NetworkingV1().Ingresses(namespace).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("ingress.Update: updating ingress %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("ingress", spec.Name).
		Str("namespace", namespace).
		Msg("ingress updated")

	return nil
}

// Delete deletes the Ingress if it exists.
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedIngressSpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)

	err := kube.Clientset().NetworkingV1().Ingresses(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug().
				Str("ingress", spec.Name).
				Str("namespace", namespace).
				Msg("ingress already deleted — skipping")
			return nil
		}
		return fmt.Errorf("ingress.Delete: deleting ingress %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("ingress", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("ingress deleted")

	return nil
}

// DeleteIfOwned deletes the Ingress if it exists and is owned by the CR.
func DeleteIfOwned(ctx context.Context, kube *kubeclient.Kubeclient,
	owner domain.Object, name, namespace string) error {

	existing, err := kube.Clientset().NetworkingV1().Ingresses(namespace).
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
	return kube.Clientset().NetworkingV1().Ingresses(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedIngressSpec from an IngressTemplateSource.
// All template expressions must be evaluated before calling here.
func Resolve(src orktypes.IngressTemplateSource, ownerName string) ResolvedIngressSpec {
	spec := ResolvedIngressSpec{
		Name:         src.Name,
		Namespace:    src.Namespace,
		Host:         src.Host,
		ServiceName:  src.ServiceName,
		Path:         src.Path,
		PathType:     src.PathType,
		IngressClass: src.IngressClass,
		Labels:       make(map[string]string),
		Annotations:  make(map[string]string),
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-ingress"
	}
	if spec.Path == "" {
		spec.Path = "/"
	}
	if spec.PathType == "" {
		spec.PathType = "Prefix"
	}

	if src.ServicePort != "" {
		if p, err := strconv.ParseInt(src.ServicePort, 10, 32); err == nil {
			spec.ServicePort = int32(p)
		}
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}
	for _, a := range src.Annotations {
		spec.Annotations[a.Key] = a.Value
	}

	// System labels
	spec.Labels[labels.Managed] = labels.ManagedValue
	spec.Labels[labels.OrkestraOwner] = ownerName

	if src.TLS != nil && src.TLS.Enabled {
		secretName := src.TLS.SecretName
		if secretName == "" {
			secretName = ownerName + "-tls"
		}
		spec.TLS = &ResolvedIngressTLS{
			Enabled:    true,
			SecretName: secretName,
			Hosts:      src.TLS.Hosts,
			ValidFor:   src.TLS.ValidFor,
		}
	}

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildIngress(owner domain.Object, spec ResolvedIngressSpec, namespace string) *networkingv1.Ingress {
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

	pathType := networkingv1.PathTypePrefix
	switch spec.PathType {
	case "Exact":
		pathType = networkingv1.PathTypeExact
	case "ImplementationSpecific":
		pathType = networkingv1.PathTypeImplementationSpecific
	}

	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        spec.Name,
			Namespace:   namespace,
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
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: spec.Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     spec.Path,
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: spec.ServiceName,
											Port: networkingv1.ServiceBackendPort{
												Number: spec.ServicePort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if spec.IngressClass != "" {
		ing.Spec.IngressClassName = &spec.IngressClass
	}

	if spec.TLS != nil && spec.TLS.Enabled {
		hosts := spec.TLS.Hosts
		if len(hosts) == 0 && spec.Host != "" {
			hosts = []string{spec.Host}
		}
		ing.Spec.TLS = []networkingv1.IngressTLS{
			{
				Hosts:      hosts,
				SecretName: spec.TLS.SecretName,
			},
		}
	}

	return ing
}

func validateSpec(spec ResolvedIngressSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	return nil
}
