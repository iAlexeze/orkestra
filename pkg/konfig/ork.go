// pkg/konfig/ork.go
package konfig

import "time"

// ── Runtime ───────────────────────────────────────────────────────────

// Runtime returns the instance identifier
// Used by pkg/orkestra and other packages to determine what is running.
func Runtime() Instance {
	return InstanceRuntime
}

// Gateway returns the instance identifier for the gateway service,
// Used by pkg/orkestra and other packages to determine what is running.
func Gateway() Instance {
	return InstanceGateway
}

// String returns the string representation of the Instance, suitable for
// logging, printing, and serialization.
func (i Instance) String() string {
	return string(i)
}

// ── Orkestra Config ─────────────────────────────────────────────────────────

// SetName sets the configured name for this Orkestra instance.
func (k *orkKonfig) SetName(v string) {
	k.name = v
}

// Name returns the configured name for this Orkestra instance.
func (k *orkKonfig) Name() string {
	return k.name
}

// Instance returns the configured Instance (runtime or gateway) for this config.
func (k *orkKonfig) Instance() Instance {
	return k.instance
}

// ShortName returns the short name used for display or compact identifiers.
func (k *orkKonfig) ShortName() string {
	return k.shortName
}

// Environment returns the environment label (dev, staging, production, etc.).
func (k *orkKonfig) Environment() string {
	return k.environment
}

// LogLevel returns the configured logging level for the process.
func (k *orkKonfig) LogLevel() string {
	return k.logLevel
}

// ── Konductor Election ───────────────────────────────────────────────────────────

// Namespace returns the election namespace.
func (e *konductorElection) Namespace() string {
	return e.namespace
}

// SetNamespace sets the election namespace.
func (e *konductorElection) SetNamespace(v string) {
	e.namespace = v
}

// LeaseDuration returns the configured lease duration for leader election.
func (e *konductorElection) LeaseDuration() time.Duration {
	return e.leaseDuration
}

// SetLeaseDuration sets the lease duration for leader election.
func (e *konductorElection) SetLeaseDuration(v time.Duration) {
	e.leaseDuration = v
}

// RenewDeadline returns the renew deadline used in leader election.
func (e *konductorElection) RenewDeadline() time.Duration {
	return e.renewDeadline
}

// SetRenewDeadline sets the renew deadline used in leader election.
func (e *konductorElection) SetRenewDeadline(v time.Duration) {
	e.renewDeadline = v
}

// RetryPeriod returns the retry period used in leader election.
func (e *konductorElection) RetryPeriod() time.Duration {
	return e.retryPeriod
}

// SetRetryPeriod sets the retry period used in leader election.
func (e *konductorElection) SetRetryPeriod(v time.Duration) {
	e.retryPeriod = v
}

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

// E2EKind returns the kind string for an E2E document.
func E2EKind() string {
	return kindE2E
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
