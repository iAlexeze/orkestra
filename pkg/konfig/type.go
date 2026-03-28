package konfig

import "time"

type Konfig struct {
	ork          orkKonfig
	cluster      clusterKonfig
	konductor    konductorElection
	healthServer healthServer
	katalog      katalogKonfig
	webhook      webhookConfig
	registry     registryConfig
}

type orkKonfig struct {
	Name        string `validate:"required"`
	ShortName   string
	Environment string
	LogLevel    string
}

type healthServer struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type clusterKonfig struct {
	KubekonfigPath   string
	MasterURL        string
	Name             string
	DefaultNamespace string `validate:"required"`

	// Worload specific
	DefaultResync  time.Duration
	DefaultWorkers int
}

type registryConfig struct {
	RegistryURL string
}

type webhookConfig struct {
	// Admission webhooks
	EnableWebhooks bool

	// Conversion webhooks
	EnableConversion bool
	ConversionWindow int

	// Certificates
	TLSCert string
	TLSKey  string
}

type katalogKonfig struct {
	Paths                   []string // Comma separated Paths to CRD katalog YAML file
	DefaultMaxQueueDepth    int
	DefaultDegradeThreshold int `validate:"required"`
}

type konductorElection struct {
	ElectionNamespace string
	LeaseDuration     time.Duration
	RenewDeadline     time.Duration
	RetryPeriod       time.Duration
}

// Methods

// IsDev returns true for development environment
func (k *Konfig) IsDev() bool {
	return k.Ork().Environment == "devlopment"
}

// IsDev returns true for staging environment
func (k *Konfig) IsStaging() bool {
	return k.Ork().Environment == "staging"
}

// IsDev returns true for production environment
func (c *Konfig) IsProduction() bool {
	return c.Ork().Environment == "production"
}

// Health returns health konfigurations
func (k *Konfig) Health() *healthServer {
	return &k.healthServer
}

// Ork returns Ork Konfigurations
func (k *Konfig) Ork() *orkKonfig {
	return &k.ork
}

// Cluster returns cluster Konfigurations
func (k *Konfig) Cluster() *clusterKonfig {
	return &k.cluster
}

// Konductor returns konductor Konfigurations
func (c *Konfig) Konductor() *konductorElection {
	return &c.konductor
}

// Katalog returns katalog Konfigurations
func (k *Konfig) Katalog() *katalogKonfig {
	return &k.katalog
}

// Finalizers return a list of default finalizers
func (k *Konfig) Finalizers() []string {
	return []string{FinalizerOrkestra}
}

// WebhookConfig returns true is enabled
func (k *Konfig) WebhookConfig() *webhookConfig {
	return &k.webhook
}

// RegistryConfig returns true is enabled
func (k *Konfig) RegistryConfig() *registryConfig {
	return &k.registry
}
