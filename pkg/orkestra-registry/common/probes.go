// pkg/orkestra-registry/common/probes.go
package common

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// probeTimings holds timing parameters for a probe profile.
type probeTimings struct {
	InitialDelaySeconds int32
	PeriodSeconds       int32
	FailureThreshold    int32
	SuccessThreshold    int32
	TimeoutSeconds      int32
}

// probeProfiles maps profile names to their timing configuration.
//
//   - fast:       quick detection, low tolerance — good for responsive HTTP APIs
//   - standard:   balanced defaults — suitable for most web services
//   - patient:    tolerant of slow operations — good for batch workers
//   - slow-start: 5-minute startup window — designed for startup probes on heavy apps
var probeProfiles = map[string]probeTimings{
	"fast":       {InitialDelaySeconds: 5, PeriodSeconds: 10, FailureThreshold: 2, SuccessThreshold: 1, TimeoutSeconds: 5},
	"standard":   {InitialDelaySeconds: 15, PeriodSeconds: 20, FailureThreshold: 3, SuccessThreshold: 1, TimeoutSeconds: 10},
	"patient":    {InitialDelaySeconds: 30, PeriodSeconds: 30, FailureThreshold: 5, SuccessThreshold: 1, TimeoutSeconds: 10},
	"slow-start": {InitialDelaySeconds: 0, PeriodSeconds: 10, FailureThreshold: 30, SuccessThreshold: 1, TimeoutSeconds: 10},
}

var defaultTimings = probeTimings{InitialDelaySeconds: 15, PeriodSeconds: 20, FailureThreshold: 3, SuccessThreshold: 1, TimeoutSeconds: 10}

// BuildProbe constructs a Kubernetes Probe from a ProbeConfig.
// containerPort is used as the probe target when cfg.Port is 0.
// Returns nil if cfg is nil or no probe type can be inferred.
func BuildProbe(cfg *orktypes.ProbeConfig, containerPort int32) *corev1.Probe {
	if cfg == nil {
		return nil
	}

	isHTTP := cfg.Type == "http" || (cfg.Type == "" && cfg.Path != "")
	isTCP := cfg.Type == "tcp"

	if !isHTTP && !isTCP {
		return nil
	}

	t := defaultTimings
	if pt, ok := probeProfiles[cfg.Profile]; ok {
		t = pt
	}

	if cfg.InitialDelaySeconds != nil {
		t.InitialDelaySeconds = *cfg.InitialDelaySeconds
	}
	if cfg.PeriodSeconds != nil {
		t.PeriodSeconds = *cfg.PeriodSeconds
	}
	if cfg.FailureThreshold != nil {
		t.FailureThreshold = *cfg.FailureThreshold
	}
	if cfg.SuccessThreshold != nil {
		t.SuccessThreshold = *cfg.SuccessThreshold
	}
	if cfg.TimeoutSeconds != nil {
		t.TimeoutSeconds = *cfg.TimeoutSeconds
	}

	port := containerPort
	if cfg.Port > 0 {
		port = cfg.Port
	}

	probe := &corev1.Probe{
		InitialDelaySeconds: t.InitialDelaySeconds,
		PeriodSeconds:       t.PeriodSeconds,
		FailureThreshold:    t.FailureThreshold,
		SuccessThreshold:    t.SuccessThreshold,
		TimeoutSeconds:      t.TimeoutSeconds,
	}

	if isHTTP {
		probe.ProbeHandler = corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: cfg.Path,
				Port: intstr.FromInt32(port),
			},
		}
	} else {
		probe.ProbeHandler = corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt32(port),
			},
		}
	}

	return probe
}

// ApplyProbes sets startup, liveness, and readiness probes on c.
// containerPort is used for probes that do not declare their own port.
// No-op when probes is nil.
func ApplyProbes(c *corev1.Container, probes *orktypes.ProbesConfig, containerPort int32) {
	if probes == nil {
		return
	}
	c.StartupProbe = BuildProbe(probes.Startup, containerPort)
	c.LivenessProbe = BuildProbe(probes.Liveness, containerPort)
	c.ReadinessProbe = BuildProbe(probes.Readiness, containerPort)
}
