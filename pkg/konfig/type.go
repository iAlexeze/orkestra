package konfig

import "time"

type Konfig struct {
	app          appKonfig
	cluster      clusterKonfig
	leader       leaderElection
	healthServer healthServer
	katalog      katalogKonfig
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

type katalogKonfig struct {
	Path                    string // Path to CRD registry YAML file
	Mode                    string `validate:"required"` // Mode of CRD registry
	DefaultMaxQueueDepth    int
	DefaultDegradeThreshold int `validate:"required"`
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

// Katalog returns app Konfigurations
func (c *Konfig) Katalog() *katalogKonfig {
	return &c.katalog
}

const (
	ModeGo   = "go"
	ModeYaml = "yaml"
)

// Mode returns the mode in use
func (c *Konfig) Mode() string {
	if c.Katalog().Mode == "yaml" {
		return "yaml"
	} else if c.Katalog().Mode == "go" {
		return "go"
	}
	return ""
}
