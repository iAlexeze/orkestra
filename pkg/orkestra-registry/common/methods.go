package common

import (
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// BuildResourceRequirements converts an Orkestra ResourceRequirements spec into a
// Kubernetes corev1.ResourceRequirements object.
//
// The input uses plain string quantities (e.g. "100m", "256Mi") for both
// requests and limits. This helper parses those values into resource.Quantity
// and populates the corresponding ResourceList fields.
//
// Keys are preserved exactly as provided (e.g. "cpu", "memory",
// "ephemeral-storage", vendor-specific resources). Unknown or extended resource
// names are passed through without modification.
//
// This function is the canonical, shared implementation used by all resource
// builders (Deployments, StatefulSets, Jobs, HPAs, etc.) to ensure consistent,
// Kubernetes‑native resource handling across Orkestra.
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

// ResolveResources resolves resource requirements: if resources.profile is set
// it expands to explicit requests/limits; otherwise the block is returned as-is.
// Returns nil when neither profile nor explicit values are declared.
func ResolveResources(r *orktypes.ResourceRequirements) *orktypes.ResourceRequirements {
	if r == nil {
		return nil
	}
	if r.Profile != "" {
		expanded, err := ExpandResourceProfile(r.Profile)
		if err != nil {
			logger.Warn().Str("profile", r.Profile).Err(err).Msg("unknown resources.profile — skipping")
			return nil
		}
		return expanded
	}
	if len(r.Requests) == 0 && len(r.Limits) == 0 {
		return nil
	}
	return r
}

// ResourceRequirementsEqual reports whether two ResourceRequirements are equivalent.
func ResourceRequirementsEqual(a, b corev1.ResourceRequirements) bool {
	return resourceListEqual(a.Requests, b.Requests) && resourceListEqual(a.Limits, b.Limits)
}

func resourceListEqual(a, b corev1.ResourceList) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok || va.Cmp(vb) != 0 {
			return false
		}
	}
	return true
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
