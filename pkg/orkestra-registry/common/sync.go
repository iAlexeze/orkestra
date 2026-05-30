package common

import (
	"reflect"

	corev1 "k8s.io/api/core/v1"
)

// SyncContainerSpec applies all template-declared fields from desired to existing
// and returns true if anything changed. Only fields that are non-zero in desired
// are applied — zero/nil means the template did not declare the field and the
// existing value is left untouched.
//
// Call this from Update functions after building the desired object:
//
//	desired := buildDeployment(owner, spec, ns)
//	updated := existing.DeepCopy()
//	if SyncContainerSpec(&updated.Spec.Template.Spec.Containers[0], desired.Spec.Template.Spec.Containers[0]) {
//	    drifted = true
//	}
func SyncContainerSpec(existing *corev1.Container, desired corev1.Container) bool {
	drifted := false

	if existing.Image != desired.Image {
		existing.Image = desired.Image
		drifted = true
	}
	if !reflect.DeepEqual(existing.Ports, desired.Ports) {
		existing.Ports = desired.Ports
		drifted = true
	}
	if !reflect.DeepEqual(existing.Env, desired.Env) {
		existing.Env = desired.Env
		drifted = true
	}
	if !reflect.DeepEqual(existing.EnvFrom, desired.EnvFrom) {
		existing.EnvFrom = desired.EnvFrom
		drifted = true
	}
	if !reflect.DeepEqual(existing.VolumeMounts, desired.VolumeMounts) {
		existing.VolumeMounts = desired.VolumeMounts
		drifted = true
	}
	if !reflect.DeepEqual(existing.LivenessProbe, desired.LivenessProbe) {
		existing.LivenessProbe = desired.LivenessProbe
		drifted = true
	}
	if !reflect.DeepEqual(existing.ReadinessProbe, desired.ReadinessProbe) {
		existing.ReadinessProbe = desired.ReadinessProbe
		drifted = true
	}
	if !reflect.DeepEqual(existing.StartupProbe, desired.StartupProbe) {
		existing.StartupProbe = desired.StartupProbe
		drifted = true
	}
	if !reflect.DeepEqual(existing.SecurityContext, desired.SecurityContext) {
		existing.SecurityContext = desired.SecurityContext
		drifted = true
	}
	if existing.WorkingDir != desired.WorkingDir {
		existing.WorkingDir = desired.WorkingDir
		drifted = true
	}

	return drifted
}

// SyncPodSpec applies template-declared pod-level fields from desired to existing.
// Returns true if anything changed.
func SyncPodSpec(existing *corev1.PodSpec, desired corev1.PodSpec) bool {
	drifted := false

	if !reflect.DeepEqual(existing.Volumes, desired.Volumes) {
		existing.Volumes = desired.Volumes
		drifted = true
	}
	if !reflect.DeepEqual(existing.SecurityContext, desired.SecurityContext) {
		existing.SecurityContext = desired.SecurityContext
		drifted = true
	}
	if existing.ServiceAccountName != desired.ServiceAccountName {
		existing.ServiceAccountName = desired.ServiceAccountName
		drifted = true
	}
	if !reflect.DeepEqual(existing.NodeSelector, desired.NodeSelector) {
		existing.NodeSelector = desired.NodeSelector
		drifted = true
	}

	return drifted
}
