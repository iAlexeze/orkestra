package konfig

const (
	// Ork
	Orkestra    = "OrKestra"
	Ork         = "ork"
	OrkOperator = "orkestra-operator"

	// Environment
	DevShort     = "dev"
	StagingShort = "uat"
	Live         = "live"
	ProdShort    = "prod"
	Development  = "development"
	Staging      = "staging"
	Production   = "production"

	// Modes
	DynamicMode = "dynamic"
	TypedMode   = "typed"

	// Kind
	kindKatalog   = "Katalog"
	kindKonductor = "Konductor"
	kindKomposer  = "Komposer"
	kindMotif     = "Motif"
	kindE2E       = "E2E"
	kindSimulate  = "Simulate"

	// HTTPS Port
	httpsPort      = ":8443"
	httpsPortInt32 = 8443

	// Secrets
	defaultInternalTLSSecretName = "orkestra-internal-tls"
	defaultWorkloadSecretName    = "orkestra-tls"
)

// Instance identifiers used by Orkestra to distinguish between the internal
// runtime service and gateway service.
type Instance string

const (
	InstanceRuntime Instance = "runtime"
	InstanceGateway Instance = "gateway"
)

var (
	apiVersions = []string{
		"orkestra.orkspace.io/v1",
	}
)
