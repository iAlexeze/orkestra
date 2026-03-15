package generate

import (
	"time"

	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

type generateKatalog struct {
	Spec struct {
		CRDs []orktypes.CRDEntry `yaml:"crds"`
	} `yaml:"spec"`
}

type registryTemplateData struct {
	Timestamp string
	Imports   []importEntry
	Entries   []registryEntry
	//	SchemeEntries    []registryEntry
	HookEntries      []hookEntry
	RecEntries       []reconcilerEntry
	NeedsRecImports  bool // true when RecEntries is non-empty
	NeedsHookImports bool // true when HookEntries is non-empty
}

type importEntry struct {
	Alias    string
	Location string
}

type registryEntry struct {
	Alias   string
	Object  string
	List    string
	Group   string
	Version string
	Kind    string
}

type hookEntry struct {
	Alias    string
	Function string
	Group    string
	Version  string
	Kind     string
}

type reconcilerEntry struct {
	Alias    string
	Function string
	Group    string
	Version  string
	Kind     string
}

// Docs and dashboards
type CRDMeta struct {
	Name        string
	Description string

	Group      string
	Version    string
	Kind       string
	Plural     string
	Namespaced bool
	Namespace  string

	Workers int
	Resync  string

	DependsOn []string

	Reconciler struct {
		Default  bool
		Function string // for custom reconcilers
	}

	Queue struct {
		MaxQueueDepth int
		Default       bool
	}

	API struct {
		Object string
		List   string
		Alias  string
		Import string
	}
}

type DocsTemplateData struct {
	Timestamp time.Time
	CRDs      []CRDMeta
	CRD       CRDMeta // for per‑CRD templates
}

type DashboardTemplateData struct {
	Timestamp time.Time
	CRD       CRDMeta
}
