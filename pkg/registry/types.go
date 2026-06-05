package registry

import "time"

// PatternKind is the Orkestra pattern type, read from the kind: field.
type PatternKind string

// PatternSpec describes the conventions for one pattern kind.
type PatternSpec struct {
	Kind          PatternKind
	MediaType     string
	PrimaryFile   string
	RequiredFiles []string
	OptionalFiles []string
}

// PatternMeta holds metadata for an pattern — read from its primary YAML on
// push, and reconstructed from OCI annotations on info/list.
type PatternMeta struct {
	Kind        PatternKind
	Name        string
	Version     string
	Description string
	Author      string
	License     string
	Tags        []string
	E2E         *PatternE2E      // populated at push time from running ork e2e; nil when absent
	Simulate    *PatternSimulate // populated at push time from running simulate gate; nil when absent
}

// PatternE2E holds E2E verification metadata embedded in OCI annotations at push time.
type PatternE2E struct {
	Status   string // "passed", "skipped"
	Duration string // e.g. "45s"
	TestedAt string // RFC3339
	Runner   string // "local" or "github-actions"
}

// PatternSimulate holds simulate gate metadata embedded in OCI annotations at push time.
// Status values:
//   - "passed"       — simulate.yaml had expect: blocks and all assertions passed
//   - "no-assertion" — simulate.yaml present but no expect: block; ran clean, nothing asserted
//   - "skipped"      — --no-simulate or --force used
type PatternSimulate struct {
	Status   string // "passed", "no-assertion", "skipped"
	Duration string // e.g. "4ms"; empty for skipped/no-assertion
	TestedAt string // RFC3339
}

// PatternEntry is one row in the pattern index.
type PatternEntry struct {
	Name           string   `json:"name"`
	LatestVersion  string   `json:"latestVersion"`
	Description    string   `json:"description"`
	Tags           []string `json:"tags"`
	Author         string   `json:"author,omitempty"`
	Kind           string   `json:"kind,omitempty"`           // "Katalog" or "Motif"
	E2EStatus      string   `json:"e2eStatus,omitempty"`      // "passed", "skipped", or ""
	SimulateStatus string   `json:"simulateStatus,omitempty"` // "passed", "skipped", "no-assertion", or ""
}

// PatternIndex is the top-level index stored at registry/index:latest.
type PatternIndex struct {
	UpdatedAt string         `json:"updatedAt"`
	Entries   []PatternEntry `json:"entries"`
}

// PatternInfo holds the metadata returned by Info.
type PatternInfo struct {
	Ref      *Ref
	Digest   string
	Size     int64
	PushedAt time.Time
	Meta     *PatternMeta
}
