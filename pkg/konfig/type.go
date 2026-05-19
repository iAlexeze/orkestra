package konfig

import (
	"time"

	"github.com/orkspace/orkestra/pkg/labels"
)

type Konfig struct {
	ork          orkKonfig
	cluster      clusterKonfig
	konductor    konductorElection
	healthServer healthServer
	katalog      katalogKonfig
	security     SecurityConfig
	notification NotificationConfig
	registry     registryConfig
}

type orkKonfig struct {
	Name        string `validate:"required"`
	ShortName   string
	Environment string
	LogLevel    string
	// GatewayEndpoint is advertised in the runtime /katalog response so the
	// control center can locate the companion gateway and merge stats.
	// Populated from ORK_GATEWAY_ENDPOINT; empty when no gateway is configured.
	GatewayEndpoint string
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
	// One service name per orkestra instance
	ServiceName string

	DeletionProtection struct {
		Enabled           bool
		CleanupOnShutdown bool
		ServiceName       string
		FailurePolicy     string
	}
	Webhooks struct {
		Admission struct {
			Enabled bool
		}
		CleanupOnShutdown bool
		FailurePolicy     string
		ServiceName       string
		// TLS paths — shared with deletion protection, admission, and conversion.
		// Set by ensureSecurity() after cert generation/loading.
		TLSCert string
		TLSKey  string

		Controller struct {
			Enabled      bool
			SyncInterval time.Duration
		}
	}
	// Conversion is separate from admission webhooks — conversion has its own
	// /convert endpoint, window stats, and CRD patch logic.
	Conversion struct {
		Enabled bool
		// ConversionWindow is the rolling window size for latency/throughput stats.
		ConversionWindow  int
		CleanupOnShutdown bool // TODO
	}

	// NamespaceProtection controls the optional validating webhook that prevents
	// Orkestra-managed CRs from being created or updated in forbidden namespaces.
	//
	// This is an admission-time safeguard only. If disabled, namespace rules are
	// not enforced at apply time. If enabled, the webhook blocks CRs whose target
	// namespace violates the CRD’s declared allowedNamespaces or restrictedNamespaces.
	//
	// The webhook is managed by the WebhookController and will be recreated if
	// deleted, ensuring continuous enforcement when enabled.
	//
	// Precedence: Katalog YAML > SecurityConfig (ENV) > hard default.
	NamespaceProtection struct {
		Enabled           bool
		FailurePolicy     string
		ServiceName       string
		CleanupOnShutdown bool
	}
}

// NotificationConfig is the unified notification configuration populated from
// ENV vars at Init() time. Katalog YAML values are merged on top via the Katalog
// loader, so this represents the ENV-level defaults.
//
// Precedence: Katalog YAML > NotificationConfig (ENV) > hard default.
//
// This struct defines *capability* — whether Orkestra is able to send email or
// Slack notifications at all. Teams and conditions define *intent*.
type NotificationConfig struct {
	Email struct {
		Enabled  bool // true when SMTP_* env vars are present
		SMTPHost string
		SMTPPort int
		SMTPUser string
		SMTPPass string
		From     string // optional override for From: header
	}

	Slack struct {
		Enabled bool   // true when SLACK_WEBHOOK_URL is present
		Webhook string // default webhook URL
	}

	// DefaultInterval is the fallback notification interval when neither the
	// team nor the Katalog YAML defines one.
	DefaultInterval time.Duration
}

type katalogKonfig struct {
	Paths                   []string // Comma separated Paths to CRD katalog YAML file
	DefaultMaxQueueDepth    int
	DefaultDegradeThreshold int `validate:"required"`
	DefaultResync           time.Duration
	DefaultWorkers          int
	ShutdownTimeout         time.Duration
	ShutdownGracePeriod     time.Duration
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

// Orkestra service name
func (k *Konfig) OrkestraServiceName() string {
	return k.security.ServiceName
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
	return []string{labels.FinalizerOrkestra}
}

// Security returns the unified security configuration.
// This is the primary accessor for all security-related settings.
func (k *Konfig) Security() *SecurityConfig {
	return &k.security
}

// Notification returns the unified notification configuration.
// This is the primary accessor for all notification-related settings.
func (k *Konfig) Notification() *NotificationConfig {
	return &k.notification
}

// RegistryConfig returns registry configuration.
func (k *Konfig) RegistryConfig() *registryConfig {
	return &k.registry
}

// ConversionEnabled reports whether the conversion webhook is enabled.
// Reads from SecurityConfig (populated from ENV at Init).
func (k *Konfig) ConversionEnabled() bool {
	return k.security.Conversion.Enabled
}

// AdmissionEnabled reports whether admission webhooks are enabled.
// Reads from SecurityConfig (populated from ENV at Init).
func (k *Konfig) AdmissionEnabled() bool {
	return k.security.Webhooks.Admission.Enabled
}

// GatewayEndpoint returns the companion gateway URL advertised to the control center.
// Empty string when no gateway is configured (e.g. runtime-only deployment).
func (k *Konfig) GatewayEndpoint() string {
	return k.ork.GatewayEndpoint
}

// HTTPSPort returns the HTTPS port string (e.g. ":8443") used by the webhook server.
func (k *Konfig) HTTPSPort() string {
	return httpsPort
}

// HTTPSPortInt32 returns the HTTPS port as int32 (8443) used in webhook client configs.
func (k *Konfig) HTTPSPortInt32() int32 {
	return httpsPortInt32
}

// DefaultInternalTLSName returns the default name for Orkestra's internal TLS secret.
func DefaultInternalTLSName() string {
	return defaultInternalTLSSecretName
}

// DefaultWorkloadSecretName returns the base name used for generated workload secrets.
// The caller appends the CR's name to form the final secret name.
func DefaultWorkloadSecretName() string {
	return defaultWorkloadSecretName
}
