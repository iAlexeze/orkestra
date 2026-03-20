// pkg/konfig/katalog.go
package konfig

// KatalogKind returns the kind string for a Katalog document.
// Katalogs declare CRDs in spec.crds. No sources block.
func KatalogKind() string {
	return kindKatalog
}

// KomposerKind returns the kind string for a Komposer document.
// Komposers compose Katalogs from sources (files, helm).
func KomposerKind() string {
	return kindKomposer
}

// IsKatalogKind returns true if the given kind is a Katalog.
func IsKatalogKind(kind string) bool {
	return kind == kindKatalog
}

// IsKomposerKind returns true if the given kind is a Komposer.
func IsKomposerKind(kind string) bool {
	return kind == kindKomposer
}

// IsValidDocumentKind returns true if the kind is either Katalog or Komposer.
// Used by parseKatalogDoc to accept both kinds before dispatching.
func IsValidDocumentKind(kind string) bool {
	return kind == kindKatalog || kind == kindKomposer
}

// IsValidApiVersion returns true if the given apiVersion is a supported version.
func IsValidApiVersion(apiVersion string) bool {
	for _, v := range apiVersions {
		if v == apiVersion {
			return true
		}
	}
	return false
}

// ApiVersions returns the list of supported apiVersions.
func ApiVersions() []string {
	return apiVersions
}
