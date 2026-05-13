// Package katalog — Probe Profiles
//
// Named timing presets for Kubernetes probes (startup, liveness, readiness).
// Probe profiles control initialDelaySeconds, periodSeconds, and failureThreshold.
// They are validated at katalog load time — unknown profile names fail fast.
//
// Profiles DO NOT expand at load time (unlike resource profiles); timing values
// are resolved at reconcile time when the probe is built. Profiles validated here
// must match the map in pkg/orkestra-registry/common/probes.go.

package katalog

// ProbeProfile is a named set of probe timing parameters.
type ProbeProfile string

const (
	// ProbeFast — quick detection, low tolerance. Good for HTTP APIs that
	// start quickly and should fail loud on the first real problem.
	// initialDelay: 5s, period: 10s, failureThreshold: 2
	ProbeFast ProbeProfile = "fast"

	// ProbeStandard — balanced defaults suitable for most web services.
	// initialDelay: 15s, period: 20s, failureThreshold: 3
	ProbeStandard ProbeProfile = "standard"

	// ProbePatient — tolerant of slow operations. Good for batch workers or
	// services with non-trivial startup that still finish within ~2 minutes.
	// initialDelay: 30s, period: 30s, failureThreshold: 5
	ProbePatient ProbeProfile = "patient"

	// ProbeSlowStart — 5-minute startup window, designed for startup probes on
	// JVM applications, databases, or other slow-initializing services.
	// initialDelay: 0s, period: 10s, failureThreshold: 30 (= 300s window)
	ProbeSlowStart ProbeProfile = "slow-start"
)

// isValidProbeProfile reports whether p is a recognized probe profile name.
func isValidProbeProfile(p string) bool {
	switch ProbeProfile(p) {
	case ProbeFast, ProbeStandard, ProbePatient, ProbeSlowStart:
		return true
	default:
		return false
	}
}
