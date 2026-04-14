package konfig

import (
	"time"
)

type Konfig struct {
	ork          orkKonfig
	cluster      clusterKonfig
	konductor    konductorElection
	healthServer healthServer
	katalog      katalogKonfig
	security     SecurityConfig
	registry     registryConfig
}

func (k *Konfig) WebhookConfig() {
	panic("unimplemented")
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
	KubekonfigPath string
	MasterURL      string
	Name           string
	Namespace      string `validate:"required"`

	// Worload specific
	DefaultResync       time.Duration
	DefaultWorkers      int
	ShutdownTimeout     time.Duration
	ShutdownGracePeriod time.Duration
}

type registryConfig struct {
	RegistryURL string
}

// SecurityConfig is the unified security configuration populated from ENV vars
// at Init() time. Katalog YAML values are merged on top via the Katalog
// loader, so this represents the ENV-level defaults.
//
// Precedence: Katalog YAML > SecurityConfig (ENV) > hard default.
type SecurityConfig struct {
	DeletionProtection struct {
		Enabled       bool
		ServiceName   string
		FailurePolicy string
	}
	Webhooks struct {
		Admission struct {
			Enabled bool
		}
		Conversion struct {
			Enabled bool
			// ConversionWindow is the rolling window size for latency/throughput stats.
			ConversionWindow int
		}
		FailurePolicy string
		ServiceName   string
		// TLS paths — shared with deletion protection, admission, and conversion.
		// Set by ensureSecurity() after cert generation/loading.
		TLSCert string
		TLSKey  string
	}
	RBAC struct {
		Enabled           bool
		CleanupOnShutdown bool
	}
}

type katalogKonfig struct {
	Paths                   []string // Comma separated Paths to CRD katalog YAML file
	DefaultMaxQueueDepth    int
	DefaultDegradeThreshold int `validate:"required"`
}

type konductorElection struct {
	Namespace     string
	LeaseDuration time.Duration
	RenewDeadline time.Duration
	RetryPeriod   time.Duration
}

// Methods

func NewDefaultKonfig() *Konfig {
	return &Konfig{
		ork: orkKonfig{
			Name:        "orkestra",
			ShortName:   "ork",
			Environment: "development",
			LogLevel:    "info",
		},
		cluster: clusterKonfig{
			KubekonfigPath: "",
			MasterURL:      "",
			Name:           "orkestra",
			Namespace:      "orkestra",
		},
		konductor: konductorElection{
			Namespace:     "orkestra",
			LeaseDuration: 15 * time.Second,
			RenewDeadline: 10 * time.Second,
			RetryPeriod:   2 * time.Second,
		},
		healthServer: healthServer{
			Port:         "8080",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		katalog: katalogKonfig{
			Paths: []string{"katalog.yaml"},
		},
	}
}

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

// Security returns the unified security configuration.
// This is the primary accessor for all security-related settings.
func (k *Konfig) Security() *SecurityConfig {
	return &k.security
}

// RegistryConfig returns registry configuration.
func (k *Konfig) RegistryConfig() *registryConfig {
	return &k.registry
}

// ConversionEnabled reports whether the conversion webhook is enabled.
// Reads from SecurityConfig (populated from ENV at Init).
func (k *Konfig) ConversionEnabled() bool {
	return k.security.Webhooks.Conversion.Enabled
}

// AdmissionEnabled reports whether admission webhooks are enabled.
// Reads from SecurityConfig (populated from ENV at Init).
func (k *Konfig) AdmissionEnabled() bool {
	return k.security.Webhooks.Admission.Enabled
}

// HTTPSPort returns the HTTPS port string (e.g. ":8443") used by the webhook server.
func (k *Konfig) HTTPSPort() string {
	return httpsPort
}

// HTTPSPortInt32 returns the HTTPS port as int32 (8443) used in webhook client configs.
func (k *Konfig) HTTPSPortInt32() int32 {
	return httpsPortInt32
}
