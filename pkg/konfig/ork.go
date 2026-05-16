// pkg/konfig/ork.go
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

// MotifKind returns the kind string for a Motif document.
func MotifKind() string {
	return kindMotif
}

// KonduktorKind returns the kind string for a Konduktor document.
func KonduktorKind() string {
	return kindKonductor
}

// IsKatalogKind returns true if the given kind is a Katalog.
func IsKatalogKind(kind string) bool {
	return kind == kindKatalog
}

// IsKonduktorKind returns true if the given kind is a Konduktor.
func IsKonduktorKind(kind string) bool {
	return kind == kindKonductor
}

// IsMotifKind returns true if the given kind is a Motif.
func IsMotifKind(kind string) bool {
	return kind == kindMotif
}

// IsKomposerKind returns true if the given kind is a Komposer.
func IsKomposerKind(kind string) bool {
	return kind == kindKomposer
}

// IsE2EKind returns true if the given kind is an E2E.
func IsE2EKind(kind string) bool {
	return kind == kindE2E
}

// IsValidDocumentKind reports whether the given kind is one of the supported
// Orkestra document kinds.
func IsValidDocumentKind(kind string) bool {
	return kind == kindKatalog ||
		kind == kindKomposer ||
		kind == kindMotif ||
		kind == kindE2E
}

// ValidKindsString returns a comma‑separated list of all supported document kinds.
// Useful for error messages and CLI diagnostics.
func ValidKindsString() string {
	return "Katalog, Komposer, Motif, E2E"
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
