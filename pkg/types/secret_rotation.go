// pkg/types/secret_rotation.go
//
// Secret rotation — time-based credential renewal.
//
// Extends the existing once: true pattern with a rotateAfter duration.
// When rotateAfter is set, the secret is recreated when its age exceeds
// the declared duration. The creation time is tracked via an annotation
// on the Secret itself.
//
// ── How it works ──────────────────────────────────────────────────────────
//
//  First reconcile (Secret does not exist):
//    → generate credentials, create Secret
//    → annotate with orkestra.orkspace.io/generated-at: <RFC3339 timestamp>
//
//  Subsequent reconciles (Secret exists):
//    → read orkestra.orkspace.io/generated-at annotation
//    → if age < rotateAfter: no-op (preserve credentials)
//    → if age >= rotateAfter: delete Secret, recreate with new credentials
//       then re-annotate with new timestamp
//
//  This is idempotent: the check-then-act pattern means no rotation happens
//  unless the threshold is crossed. The annotation is the source of truth.
//
// ── YAML ──────────────────────────────────────────────────────────────────
//
//  secrets:
//    - name: "{{ .metadata.name }}-credentials"
//      once: true
//      rotateAfter: 90d      # rotate every 90 days
//      data:
//        password: "{{ randomAlphanumeric 32 }}"
//        apiKey:   "{{ randomHex 16 }}"
//
//  # TLS certificates — auto-generated, stored as standard tls Secret type
//  secrets:
//    - name: "{{ .metadata.name }}-tls"
//      once: true
//      rotateAfter: 1y       # rotate annually
//      tls:
//        commonName: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc"
//        dnsNames:
//          - "{{ .metadata.name }}"
//          - "{{ .metadata.name }}.{{ .metadata.namespace }}"
//          - "{{ .metadata.name }}.{{ .metadata.namespace }}.svc"
//          - "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
//        validFor: 1y
//      # Produces a Secret with:
//      #   tls.crt — PEM certificate
//      #   tls.key — PEM private key
//      #   ca.crt  — PEM CA certificate (self-signed CA)
//
// ── Duration format ───────────────────────────────────────────────────────
//
//  Supported: s (seconds), m (minutes), h (hours), d (days), y (years)
//  Examples: 30s, 5m, 12h, 90d, 1y, 365d
//  Note: d and y are extensions beyond Go's time.ParseDuration (which stops at h).
//  ParseTimeDuration handles d and y by converting to hours.
//
// ── Webhook certificate support ───────────────────────────────────────────
//
//  In the Katalog-level webhooks block:
//
//  webhooks:
//    createCerts: true         # false by default — Orkestra generates certs
//    certSecret: orkestra-tls  # default name
//    rotateAfter: 1y           # default: 1 year
//
//  When createCerts: true, Orkestra generates a self-signed CA, signs a
//  certificate for the webhook service, and stores both in the named Secret.
//  The webhook configuration's caBundle is patched automatically.
//  On each reconcile, Orkestra checks the rotation threshold and renews
//  if needed — the caBundle patch happens alongside renewal.
//
// ── Annotation ────────────────────────────────────────────────────────────
//
//  All Orkestra-generated secrets receive:
//    orkestra.orkspace.io/generated-at: "2026-04-06T08:00:00Z"
//    orkestra.orkspace.io/rotate-after: "90d"
//
//  These annotations are the source of truth for rotation decisions.
//  Do not remove them manually — doing so will cause regeneration on next reconcile.

package types

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// AnnotationGeneratedAt is the RFC3339 timestamp when the secret was last generated.
	AnnotationGeneratedAt = "orkestra.orkspace.io/generated-at"

	// AnnotationRotateAfter stores the declared rotation duration.
	AnnotationRotateAfter = "orkestra.orkspace.io/rotate-after"
)

// TLSSpec declares TLS certificate generation for a secret.
// When set, the secret is created as type kubernetes.io/tls with
// tls.crt, tls.key, and ca.crt fields.
type TLSSpec struct {
	// CommonName is the certificate's CN field.
	//   commonName: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc"
	CommonName string `yaml:"commonName" json:"commonName"`

	// DNSNames are the Subject Alternative Names.
	// Template expressions supported per entry.
	DNSNames []string `yaml:"dnsNames,omitempty" json:"dnsNames,omitempty"`

	// ValidFor is the certificate validity period.
	// Default: same as rotateAfter, or 1y if rotateAfter is not set.
	// Must be >= rotateAfter to avoid immediate expiry on rotation.
	ValidFor string `yaml:"validFor,omitempty" json:"validFor,omitempty"`

	// Organization is the cert's O field. Default: "orkestra"
	Organization string `yaml:"organization,omitempty" json:"organization,omitempty"`
}

// ParseTimeDuration parses a human‑friendly rotation duration string.
//
// It extends Go's time.ParseDuration by adding long‑term units:
//
//	d   = days (24h)
//	w   = weeks (7d)
//	mo  = months (30d)
//	y   = years (365d)
//
// Examples:
//
//	"30s"     → 30 seconds
//	"5m"      → 5 minutes (Go duration)
//	"12h"     → 12 hours
//	"10d"     → 10 days
//	"2w"      → 14 days
//	"3mo"     → 90 days
//	"1y"      → 365 days
//
// Notes:
//   - Only "mo" is accepted for months to avoid collision with Go's "m" (minutes).
//   - Fractional values are supported (e.g., "1.5mo").
//   - Falls back to time.ParseDuration for standard units.
func ParseTimeDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	if s == "never" {
		return 0, nil
	}

	s = strings.TrimSpace(s)

	// Years (365 days)
	if strings.HasSuffix(s, "y") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "y"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid year duration %q: %w", s, err)
		}
		return time.Duration(n * float64(365*24*time.Hour)), nil
	}

	// Months (30 days) — explicit "mo" only
	if strings.HasSuffix(s, "mo") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "mo"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid month duration %q: %w", s, err)
		}
		return time.Duration(n * float64(30*24*time.Hour)), nil
	}

	// Weeks (7 days)
	if strings.HasSuffix(s, "w") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "w"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid week duration %q: %w", s, err)
		}
		return time.Duration(n * float64(7*24*time.Hour)), nil
	}

	// Days (24 hours)
	if strings.HasSuffix(s, "d") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "d"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid day duration %q: %w", s, err)
		}
		return time.Duration(n * float64(24*time.Hour)), nil
	}

	// Fall back to Go's duration parser (s, m, h)
	return time.ParseDuration(s)
}

// NeedsRotation returns true when the secret has exceeded its rotation threshold.
// generatedAt is the value of the AnnotationGeneratedAt annotation.
// rotateAfter is the declared rotation duration string.
func NeedsRotation(generatedAt, rotateAfter string) bool {
	if rotateAfter == "" {
		return false // no rotation declared
	}

	t, err := time.Parse(time.RFC3339, generatedAt)
	if err != nil {
		// Cannot parse annotation — regenerate to be safe
		return true
	}

	threshold, err := ParseTimeDuration(rotateAfter)
	if err != nil {
		return false // invalid duration — do not rotate unexpectedly
	}

	return time.Since(t) >= threshold
}
