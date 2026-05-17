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

	// HTTPS Port
	httpsPort      = ":8443"
	httpsPortInt32 = 8443

	// Secrets
	defaultInternalTLSSecretName = "orkestra-internal-tls"
	defaultWorkloadSecretName    = "orkestra-tls"
)

var (
	apiVersions = []string{
		"orkestra.orkspace.io/v1",
	}
)
