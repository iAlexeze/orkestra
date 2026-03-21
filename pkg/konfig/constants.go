package konfig

const (
	// Ork
	Orkestra = "OrKestra"
	Ork      = "ork"

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

	// Labels
	ManagedByLabel         = "managed-by"
	ManagedByOrkestraLabel = "managed-by=orkestra"
	OrkestraOwnerLabel     = "orkestra-owner"
)

var (
	apiVersions = []string{
		"orkestra.konductor.io/v1Alpha",
	}
)
