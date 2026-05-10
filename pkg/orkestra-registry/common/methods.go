package common

import (
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/logger"
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

// ResolveResources returns explicit resource requirements when set, or expands
// a named profile. Returns nil when neither is declared.
func ResolveResources(explicit *orktypes.ResourceRequirements, profile string) *orktypes.ResourceRequirements {
	if explicit != nil {
		return explicit
	}
	if profile == "" {
		return nil
	}
	r, err := ExpandResourceProfile(profile)
	if err != nil {
		logger.Warn().Str("profile", profile).Err(err).Msg("unknown resourceProfile — skipping")
		return nil
	}
	return r
}

// ToPullSecrets converts a slice of string to a []corev1.LocalObjectReference
// Acceptable as Pull secrets
func ToPullSecrets(names []string) []corev1.LocalObjectReference {
	out := make([]corev1.LocalObjectReference, len(names))
	for i, n := range names {
		out[i] = corev1.LocalObjectReference{Name: n}
	}
	return out
}
