package config

import "time"

type Config struct {
	app          appConfig
	cluster      clusterConfig
	leader       leaderElection
	healthServer healthServer
	crdRegistry  crdRegistryConfig
}

type appConfig struct {
	Name        string `validate:"required"`
	Version     string
	Environment string
	LogLevel    string
}

type healthServer struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type clusterConfig struct {
	KubeconfigPath string
	MasterURL      string
	InCluster      bool
	Name           string
	Namespace      string `validate:"required"`

	// Worload specific
	DefaultResync  time.Duration
	DefaultWorkers int
	LabelSelector  string
	Finalizer      string
}

type crdRegistryConfig struct {
	Path          string // Path to CRD registry YAML file
	Mode          string `validate:"required"` // Mode of CRD registry
	MaxQueueDepth int
}

type leaderElection struct {
	LeaseDuration time.Duration
	RenewDeadline time.Duration
	RetryPeriod   time.Duration
}

// Methods

// IsDev returns true for development environment
func (c *Config) IsDev() bool {
	return c.App().Environment == "devlopment"
}

// IsDev returns true for staging environment
func (c *Config) IsStaging() bool {
	return c.App().Environment == "staging"
}

// IsDev returns true for production environment
func (c *Config) IsProduction() bool {
	return c.App().Environment == "production"
}

// Health returns health configurations
func (c *Config) Health() *healthServer {
	return &c.healthServer
}

// App returns app configurations
func (c *Config) App() *appConfig {
	return &c.app
}

// Cluster returns app configurations
func (c *Config) Cluster() *clusterConfig {
	return &c.cluster
}

// Leader returns app configurations
func (c *Config) Leader() *leaderElection {
	return &c.leader
}

// CRDRegistry returns app configurations
func (c *Config) CRDRegistry() *crdRegistryConfig {
	return &c.crdRegistry
}

// GoMode returns true if crdRegistry is using Go mode
func (c *Config) GoMode() bool {
	return c.CRDRegistry().Mode == "go"
}

// YamlMode returns true if crdRegistry is using yaml mode
func (c *Config) YamlMode() bool {
	return c.CRDRegistry().Mode == "yaml"
}
