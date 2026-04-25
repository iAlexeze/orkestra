package konfig

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	// LabelManaged is patched on every CR Orkestra manages.
	// Used by ork reconcile, ork get, and ork events to scope
	// their operations to exactly what this operator instance manages.
	LabelManaged       = "orkestra.orkspace.io/managed"
	LabelManagedValue  = "true"
	LabelOrkestraOwner = "orkestra-owner"

	// Annotations

	// AnnotationManagedBy identifies which Orkestra operator instance
	// is managing this CR. Useful when multiple Orkestra operators
	// run in the same cluster managing different CRD sets.
	AnnotationManagedBy = "orkestra.orkspace.io/managed-by"

	// AnnotationManagedSince records when Orkestra first took ownership.
	AnnotationManagedSince = "orkestra.orkspace.io/managed-since"

	// Finalizers
	FinalizerOrkestra = "orkestra.orkspace.io/finalizer"

	// HTTPS Port
	httpsPort      = ":8443"
	httpsPortInt32 = 8443
)

var (
	apiVersions = []string{
		"orkestra.orkspace.io/v1",
	}
)

// orkestraResourceLabels defines the labels used to identify Orkestra-managed
// resources for deletion protection.
var orkestraResourceLabels = map[string]string{
	"app.kubernetes.io/name": "orkestra",
	"app.kubernetes.io/tag":  "orkestra-internal",
}

// Label selector shared by all Orkestra-managed Kubernetes resources.
// Narrows the webhook to only the operator's own deployment, service, ingress,
// and admission webhook configurations (validation + mutation).
var orkestraResourceSelector = &metav1.LabelSelector{
	MatchLabels: orkestraResourceLabels,
}
