package kontroller

import "time"

const (
	// Dependency checks
	DefaultDependencyTimeout  = 60 * time.Second
	DefaultDependencyRetries  = 3
	DefaultDependencyInterval = 10 * time.Second

	// PostStart Retry loop
	// PostStartRetryInterval = 30 * time.Second
	PostStartRetryInterval        = 3 * time.Second // DEBUG
	PostStartBackoff              = 5 * time.Second
	DependencyHealthCheckInterval = 10 * time.Second

	// Draining
	drainSentinel = "__drain_sentinel__"
	drainTimeout  = 10 * time.Second

	// Healthcheck
	RuntimeHealthCheckInterval = 5 * time.Second
)
