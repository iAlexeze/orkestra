package konfig

const (
	// Ork
	orkestra = "OrKestra"
	ork      = "ork"

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
)

var (
	apiVersions = []string{
		"orkestra.konductor.io/v1Alpha",
	}
)
