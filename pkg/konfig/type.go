package konfig

import "time"

type Konfig struct {
	app          appKonfig
	cluster      clusterKonfig
	konductor    konductorElection
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
	DefaultNamespace      string `validate:"required"`

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

type konductorElection struct {
	LeaseDuration time.Duration
	RenewDeadline time.Duration
	RetryPeriod   time.Duration
}

// Methods

// IsDev returns true for development environment
func (k *Konfig) IsDev() bool {
	return k.App().Environment == "devlopment"
}

// IsDev returns true for staging environment
func (k *Konfig) IsStaging() bool {
	return k.App().Environment == "staging"
}

// IsDev returns true for production environment
func (c *Konfig) IsProduction() bool {
	return c.App().Environment == "production"
}

// Health returns health konfigurations
func (k *Konfig) Health() *healthServer {
	return &k.healthServer
}

// App returns app Konfigurations
func (k *Konfig) App() *appKonfig {
	return &k.app
}

// Cluster returns app Konfigurations
func (k *Konfig) Cluster() *clusterKonfig {
	return &k.cluster
}

// Konductor returns konducri Konfigurations
func (c *Konfig) Konductor() *konductorElection {
	return &c.konductor
}

// Katalog returns katalog Konfigurations
func (k *Konfig) Katalog() *katalogKonfig {
	return &k.katalog
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
