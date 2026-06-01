//go:build !runtime && !gateway

package cli

import (
	"path/filepath"

	"github.com/orkspace/orkestra/examples"
)

// packPaths maps CLI pack names to their paths inside the embedded FS.
// Most packs are top-level directories; rollback is nested under use-cases.
type Pack struct {
	Name        string
	Description string
	Path        string
}

var Packs = map[string]Pack{
	"beginner": {
		Name:        "beginner",
		Description: "Start here. Simple CRDs, Deployments, Services.",
		Path:        "beginner",
	},
	"intermediate": {
		Name:        "intermediate",
		Description: "Multi-resource patterns, when/anyOf, Komposer basics.",
		Path:        "intermediate",
	},
	"advanced": {
		Name:        "advanced",
		Description: "Hooks, constructors, validation/mutation, registries.",
		Path:        "advanced",
	},
	"security": {
		Name:        "security",
		Description: "Deletion protection, namespace protection, admission webhooks.",
		Path:        "security",
	},
	"use-cases": {
		Name:        "use-cases",
		Description: "Full-stack, cross-CRD, external gates, once-secrets.",
		Path:        "use-cases",
	},
}

func GetPack(name string) (Pack, bool) {
	if p, ok := Packs[name]; ok {
		return p, true
	}
	// Sub-path fallback: any valid directory in the embedded FS works as a pack.
	// ork init my-project --pack use-cases/multi-tenancy extracts into my-project/multi-tenancy.
	if f, err := examples.FS.Open(name); err == nil {
		f.Close()
		return Pack{Name: filepath.Base(name), Path: name}, true
	}
	return Pack{}, false
}

func ListPacks() []Pack {
	out := make([]Pack, 0, len(Packs))
	for _, p := range Packs {
		out = append(out, p)
	}
	return out
}

// Helpers
func (p Pack) isBeginnerPack() bool     { return p.Name == "beginner" }
func (p Pack) isCanonicalPack() bool    { return p.Name == "." }
func (p Pack) isIntermediatePack() bool { return p.Name == "intermediate" }
func (p Pack) isAdvancedPack() bool     { return p.Name == "advanced" }
func (p Pack) isSecurityPack() bool     { return p.Name == "security" }
func (p Pack) isUseCasesPack() bool     { return p.Name == "use-cases" }
func (p Pack) isRollbackPack() bool     { return p.Name == "rollback" }
func (p Pack) isDeveloperPack() bool    { return p.Name == "developer" }

func (p Pack) firstExample() string {
	switch {
	case p.isBeginnerPack(), p.isCanonicalPack():
		return "01-hello-website"
	case p.isIntermediatePack():
		return "04-multi-resource"
	case p.isAdvancedPack():
		return "07-validation-mutation"
	case p.isSecurityPack():
		return "admission"
	case p.isUseCasesPack():
		return "full-stack-app"
	case p.isRollbackPack():
		return "rollback"
	case p.isDeveloperPack():
		return "01-one-project"
	default:
		return ""
	}
}

func (p Pack) String() string {
	ex := p.firstExample()
	if ex == "" {
		return p.Name + " — " + p.Description
	}
	return p.Name + " — " + p.Description + " (start with: examples/" + p.Path + "/" + ex + ")"
}
