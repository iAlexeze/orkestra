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
}

// PatternEntry is one row in the pattern index.
type PatternEntry struct {
	Name          string   `json:"name"`
	LatestVersion string   `json:"latestVersion"`
	Description   string   `json:"description"`
	Tags          []string `json:"tags"`
	Author        string   `json:"author,omitempty"`
	Kind          string   `json:"kind,omitempty"` // "Katalog" or "Motif"
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
