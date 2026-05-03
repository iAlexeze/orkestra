//go:build !runtime

package cli

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
	"rollback": {
		Name:        "rollback",
		Description: "Zero-config and configurable failure recovery",
		Path:        "use-cases/rollback",
	},
}

func GetPack(name string) (Pack, bool) {
	p, ok := Packs[name]
	return p, ok
}

func ListPacks() []Pack {
	out := make([]Pack, 0, len(Packs))
	for _, p := range Packs {
		out = append(out, p)
	}
	return out
}
