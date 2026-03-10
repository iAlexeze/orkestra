package konfig

import "time"

type Konfig struct {
	app          appKonfig
	cluster      clusterKonfig
	leader       leaderElection
	healthServer healthServer
	crdRegistry  crdRegistryKonfig
}

type appKonfig struct {
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

type clusterKonfig struct {
	KubekonfigPath string
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

type crdRegistryKonfig struct {
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
func (c *Konfig) IsDev() bool {
	return c.App().Environment == "devlopment"
}

// IsDev returns true for staging environment
func (c *Konfig) IsStaging() bool {
	return c.App().Environment == "staging"
}

// IsDev returns true for production environment
func (c *Konfig) IsProduction() bool {
	return c.App().Environment == "production"
}

// Health returns health konfigurations
func (c *Konfig) Health() *healthServer {
	return &c.healthServer
}

// App returns app Konfigurations
func (c *Konfig) App() *appKonfig {
	return &c.app
}

// Cluster returns app Konfigurations
func (c *Konfig) Cluster() *clusterKonfig {
	return &c.cluster
}

// Leader returns app Konfigurations
func (c *Konfig) Leader() *leaderElection {
	return &c.leader
}

// CRDRegistry returns app Konfigurations
func (c *Konfig) CRDRegistry() *crdRegistryKonfig {
	return &c.crdRegistry
}

// GoMode returns true if crdRegistry is using Go mode
func (c *Konfig) GoMode() bool {
	return c.CRDRegistry().Mode == "go"
}

// YamlMode returns true if crdRegistry is using yaml mode
func (c *Konfig) YamlMode() bool {
	return c.CRDRegistry().Mode == "yaml"
}
