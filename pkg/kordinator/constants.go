package kordinator

import "time"

const (
	// Dependency checks
	DefaultDependencyTimeout  = 60 * time.Second
	DefaultDependencyRetries  = 3
	DefaultDependencyInterval = 10 * time.Second

	// PostStart Retry loop
	PostStartRetryInterval        = 90 * time.Second
	PostStartBackoff              = 5 * time.Second
	PostStartBackoffMax           = 5 * time.Minute
	DependencyHealthCheckInterval = 10 * time.Second

	// Draining
	drainSentinel = "__drain_sentinel__"
	drainTimeout  = 10 * time.Second

	// Healthcheck
	RuntimeHealthCheckInterval = 5 * time.Second
)
