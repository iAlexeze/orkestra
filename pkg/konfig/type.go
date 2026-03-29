package konfig

import (
	"strings"
	"time"

	admissionv1 "k8s.io/api/admissionregistration/v1"
)

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
	KubekonfigPath string
	MasterURL      string
	Name           string
	Namespace      string `validate:"required"`

	// Worload specific
	DefaultResync   time.Duration
	DefaultWorkers  int
	ShutdownTimeout time.Duration
	ShutdownGrace   time.Duration
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

	// Port
	Port    string
	PortInt int32

	// Registration
	WebhookRegistration webhookRegistration
}

type webhookRegistration struct {
	ServiceName      string
	ServiceNamespace string
	FailurePolicy    string

	TLSCert string // Same as the one above

	// FailurePolicy admissionv1.FailurePolicyType
	// Used to return the appropriate admission policy type
	FailurePolicyType admissionv1.FailurePolicyType
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

// ConversionEnabled returns true if mutation rules
func (k *Konfig) ConversionEnabled() bool {
	return k.webhook.EnableConversion
}

// AdmissionEnabled returns true if admission rules
func (k *Konfig) AdmissionEnabled() bool {
	return k.webhook.EnableWebhooks
}

// WebhookRegistration
func (k *Konfig) WebhookRegistration() *webhookRegistration {
	// Convert failurePolicy input to failurePolicyType
	switch strings.ToLower(k.webhook.WebhookRegistration.FailurePolicy) {
	case "ignore":
		k.webhook.WebhookRegistration.FailurePolicyType = admissionv1.Ignore
	case "fail":
		k.webhook.WebhookRegistration.FailurePolicyType = admissionv1.Fail
	default:
		k.webhook.WebhookRegistration.FailurePolicyType = admissionv1.Ignore
	}

	// Assign ports and return
	k.webhook.Port = httpsPort
	k.webhook.PortInt = httpsPortInt32
	return &k.webhook.WebhookRegistration
}
