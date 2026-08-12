package types

// PublishConfig declares the publishing and consumer policy for a pattern.
// Distinct from security: (which covers admission, RBAC, and namespace protection
// at runtime) — publish: is about supply chain: who may publish, what tests must
// pass, and what consumers must verify before accepting.
//
//	publish:
//	  signing:
//	    verify: true
//	    expectedIdentities:
//	      - github.com/myorg/postgres/.github/workflows/release.yaml@refs/heads/main
//	  tests:
//	    e2e: true
//	    simulate: true
//	    intent: false
type PublishConfig struct {
	// Signing declares cryptographic proof requirements.
	// When absent, no signing is required or enforced.
	Signing *SigningConfig `yaml:"signing,omitempty" json:"signing,omitempty"`

	// Tests controls which quality gates run at push time and are required by
	// consumers at pull time. Absent fields default to their zero values (see SigningConfig).
	Tests *PublishTestsConfig `yaml:"tests,omitempty" json:"tests,omitempty"`
}

// SigningConfig declares the Cosign keyless signing policy for a pattern.
type SigningConfig struct {
	// Verify, when true, causes ork pull to refuse unsigned artifacts.
	// Default: false.
	Verify bool `yaml:"verify,omitempty" json:"verify,omitempty"`

	// ExpectedIdentities is the list of OIDC subject claims trusted to publish
	// this pattern. Any one match passes. Empty means any valid signature is
	// accepted — only Verify: true matters in that case.
	//
	// Format: <host>/<org>/<repo>/<path>@<ref>
	// Example: github.com/myorg/postgres/.github/workflows/release.yaml@refs/heads/main
	ExpectedIdentities []string `yaml:"expectedIdentities,omitempty" json:"expectedIdentities,omitempty"`
}

// PublishTestsConfig controls which quality gates run at push time and are
// required by consumers at pull time.
type PublishTestsConfig struct {
	// E2E controls whether the e2e gate runs at push time.
	// Default: true — matches current behaviour.
	// Setting false is equivalent to passing --no-e2e at push.
	E2E *bool `yaml:"e2e,omitempty" json:"e2e,omitempty"`

	// Simulate controls whether the simulate gate runs at push time.
	// Default: true — matches current behaviour.
	// Setting false is equivalent to passing --no-simulate at push.
	Simulate *bool `yaml:"simulate,omitempty" json:"simulate,omitempty"`

	// Intent controls whether ork serve play runs at push time.
	// Default: false — opt-in. Equivalent to passing --add-intent at push.
	// When true, intent.yaml or intent.json must be present in the pattern directory.
	Intent *bool `yaml:"intent,omitempty" json:"intent,omitempty"`
}

// E2EEnabled reports whether the e2e gate is enabled.
// Defaults to true when the field is absent.
func (t *PublishTestsConfig) E2EEnabled() bool {
	if t == nil || t.E2E == nil {
		return true
	}
	return *t.E2E
}

// SimulateEnabled reports whether the simulate gate is enabled.
// Defaults to true when the field is absent.
func (t *PublishTestsConfig) SimulateEnabled() bool {
	if t == nil || t.Simulate == nil {
		return true
	}
	return *t.Simulate
}

// IntentEnabled reports whether the intent play gate is enabled.
// Defaults to false when the field is absent.
func (t *PublishTestsConfig) IntentEnabled() bool {
	if t == nil || t.Intent == nil {
		return false
	}
	return *t.Intent
}

// HasSigningConfig reports whether a signing: block is explicitly declared.
func (p *PublishConfig) HasSigningConfig() bool {
	return p != nil && p.Signing != nil
}

// VerifyRequired reports whether consumers must verify the signature at pull time.
func (p *PublishConfig) VerifyRequired() bool {
	if p == nil || p.Signing == nil {
		return false
	}
	return p.Signing.Verify
}

// ExpectedIdentities returns the trusted OIDC subject claims, or nil when none
// are declared (meaning any valid signature is accepted).
func (p *PublishConfig) ExpectedIdentities() []string {
	if p == nil || p.Signing == nil {
		return nil
	}
	return p.Signing.ExpectedIdentities
}

// HasExpectedIdentities reports whether at least one expected identity is declared.
// When false and Verify is true, any valid keyless signature is accepted.
func (p *PublishConfig) HasExpectedIdentities() bool {
	return len(p.ExpectedIdentities()) > 0
}

// TestsConfig returns the test gate config, never nil.
func (p *PublishConfig) TestsConfig() *PublishTestsConfig {
	if p == nil || p.Tests == nil {
		return &PublishTestsConfig{}
	}
	return p.Tests
}

// HasTestsConfig reports whether a tests: block is explicitly declared.
func (p *PublishConfig) HasTestsConfig() bool {
	return p != nil && p.Tests != nil
}
