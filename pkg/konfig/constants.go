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

	// LabelManaged is patched on every CR Orkestra manages.
	// Used by ork reconcile, ork get, and ork events to scope
	// their operations to exactly what this operator instance manages.
	LabelManaged       = "orkestra.konductor.io/managed"
	LabelManagedValue  = "true"
	LabelOrkestraOwner = "orkestra-owner"

	// Annotations

	// AnnotationManagedBy identifies which Orkestra operator instance
	// is managing this CR. Useful when multiple Orkestra operators
	// run in the same cluster managing different CRD sets.
	AnnotationManagedBy = "orkestra.konductor.io/managed-by"

	// AnnotationManagedSince records when Orkestra first took ownership.
	AnnotationManagedSince = "orkestra.konductor.io/managed-since"

	// Finalizers
	FinalizerOrkestra = "orkestra.konductor.io/finalizer"
)

var (
	apiVersions = []string{
		"orkestra.konductor.io/v1Alpha",
	}
)
