package kontroller

import "time"

const (
	// Dependency checks
	DefaultDependencyTimeout  = 60 * time.Second
	DefaultDependencyRetries  = 3
	DefaultDependencyInterval = 10 * time.Second

	// PostStart Retry loop
	PostStartRetryInterval = 30 * time.Second
	PostStartBackoff       = 5 * time.Second
)
