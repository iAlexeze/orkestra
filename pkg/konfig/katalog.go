package konfig

import "time"

// katalog.go provides accessor methods for the unexported katalogKonfig
// fields. Callers should use these methods to read and mutate katalog
// configuration without exposing the underlying struct fields.
//
// Example usage:
//   kfg.Katalog().DefaultWorkers()
//   kfg.Katalog().ShutdownTimeout()
//
// The underlying katalogKonfig struct is expected to use unexported field
// names (paths, defaultQueueDepth, defaultFailureThreshold, defaultResync,
// defaultWorkers, shutdownTimeout, shutdownGracePeriod).
//
// This file intentionally contains only simple, side-effect-free accessors
// and a small set of helpers for common operations.

// Paths returns the configured katalog file paths.
func (k *katalogKonfig) Paths() []string {
	return k.paths
}

// HasPaths reports whether any katalog paths have been configured.
func (k *katalogKonfig) HasPaths() bool {
	return len(k.paths) > 0
}

// AddPath appends a katalog path to the configuration.
func (k *katalogKonfig) AddPath(path string) {
	if path == "" {
		return
	}
	k.paths = append(k.paths, path)
}

// DefaultQueueDepth returns the defaultmaximum  queue depth for CRD workers.
func (k *katalogKonfig) DefaultQueueDepth() int {
	return k.defaultQueueDepth
}

// DefaultFailureThreshold returns the default degrade threshold.
func (k *katalogKonfig) DefaultFailureThreshold() int {
	return k.defaultFailureThreshold
}

// DefaultResync returns the default informer resync period.
func (k *katalogKonfig) DefaultResync() time.Duration {
	return k.defaultResync
}

// DefaultWorkers returns the default number of reconcile workers per CRD.
func (k *katalogKonfig) DefaultWorkers() int {
	return k.defaultWorkers
}

// ShutdownTimeout returns the hard timeout used during shutdown.
func (k *katalogKonfig) ShutdownTimeout() time.Duration {
	return k.shutdownTimeout
}

// ShutdownGracePeriod returns the grace period before forced shutdown.
func (k *katalogKonfig) ShutdownGracePeriod() time.Duration {
	return k.shutdownGracePeriod
}

// GatewayEndpoint returns the advertised gateway endpoint; it may be empty
// when no gateway is configured.
func (k *katalogKonfig) GatewayEndpoint() string {
	return k.gatewayEndpoint
}
