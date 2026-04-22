package common

import (
	"github.com/orkspace/orkestra/domain"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func BuildResourceRequirements(r *orktypes.ResourceRequirements) corev1.ResourceRequirements {
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

// ResolveNamespace — priority: spec.Namespace → owner namespace → "default"
func ResolveNamespace(owner domain.Object, namespace string) string {
	if namespace != "" {
		return namespace
	}
	if owner.GetNamespace() != "" {
		return owner.GetNamespace()
	}
	return "default"
}
