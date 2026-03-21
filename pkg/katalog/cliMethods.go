package katalog

import (
	"fmt"
	"strings"

	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

func (k *Katalog) List() []orktypes.CRDEntry {
	return k.Spec.CRDs
}

// All returns every CRD in the katalog, including disabled ones.
// Useful for CLI commands like `ork katalog list --all`.
func (k *Katalog) All() []orktypes.CRDEntry {
	return k.Spec.CRDs
}

// Useful Metadata
func (k *Katalog) Meta() orktypes.KatalogMeta {
	return k.metadata
}

// Exists returns true if a CRD with the given name exists in the katalog.
func (k *Katalog) Exists(name string) bool {
	for _, crd := range k.Spec.CRDs {
		if crd.Name == name {
			return true
		}
	}
	return false
}

// Describe returns a human‑readable summary of a CRD.
// The CLI can print this directly.
func (k *Katalog) Describe(name string) (string, error) {
	crd, err := k.Get(name)
	if err != nil {
		return "", err
	}

	b := &strings.Builder{}
	fmt.Fprintf(b, "Name:        %s\n", crd.Name)
	fmt.Fprintf(b, "Group:       %s\n", crd.APITypes.Group)
	fmt.Fprintf(b, "Version:     %s\n", crd.APITypes.Version)
	fmt.Fprintf(b, "Kind:        %s\n", crd.APITypes.Kind)
	fmt.Fprintf(b, "Plural:      %s\n", crd.APITypes.Plural)
	fmt.Fprintf(b, "Namespaced:  %v\n", crd.Namespaced)
	if crd.Namespaced {
		fmt.Fprintf(b, "Namespace:   %s\n", crd.Namespace)
	}
	fmt.Fprintf(b, "Workers:     %d\n", crd.Workers)
	fmt.Fprintf(b, "Resync:      %s\n", crd.Resync)
	fmt.Fprintf(b, "Enabled:     %v\n", crd.Enabled)

	if len(crd.DependsOn) > 0 {
		fmt.Fprintf(b, "Dependencies: %v\n", strings.Join(crd.DependsOn, " "))
	} else {
		fmt.Fprint(b, "Dependencies: None")
	}

	fmt.Fprintf(b, "Description:\n%s\n", crd.Description)

	return b.String(), nil
}

// Explain returns a technical explanation of how Orkestra handles this CRD.
// Useful for `ork explain <crd>`.
func (k *Katalog) Explain(name string) (string, error) {
	crd, err := k.Get(name)
	if err != nil {
		return "", err
	}

	b := &strings.Builder{}
	fmt.Fprintf(b, "%s (%s/%s)\n", crd.APITypes.Kind, crd.APITypes.Group, crd.APITypes.Version)
	fmt.Fprintf(b, "----------------------------------------\n")
	fmt.Fprintf(b, "API Path:     %s\n", crd.APITypes.APIPath)
	fmt.Fprintf(b, "GVK:          %s\n", crd.GroupVersionKind.String())
	fmt.Fprint(b, "List Type:    runtime.Object")
	fmt.Fprint(b, "Object Type:  runtime.Object")
	if crd.ReconcilerConfig.Default {
		fmt.Fprint(b, "Reconciler:   Default\n")
	} else {
		fmt.Fprintf(b, "Reconciler:   %T\n", crd.ReconcilerConfig.Constructor)
	}
	fmt.Fprintf(b, "Informer:     LIST, WATCH\n") // later: dynamic

	if len(crd.DependsOn) > 0 {
		fmt.Fprintf(b, "Dependencies: %v\n", strings.Join(crd.DependsOn, " "))
	} else {
		fmt.Fprint(b, "Dependencies: None")
	}

	return b.String(), nil
}

// Graph returns a map of CRD -> dependencies.
// Useful for CLI graph visualization.
func (k *Katalog) Graph() map[string][]string {
	graph := make(map[string][]string)
	for _, crd := range k.enabledCRDs {
		graph[crd.Name] = crd.DependsOn
	}
	return graph
}

// Order returns CRDs in dependency‑safe order (topological sort).
func (k *Katalog) Order() []string {
	depGraph := NewDependencyGraph(k)
	return depGraph.ShutdownOrder()
}

// Controllers returns a list of CRDs that have reconcilers.
func (k *Katalog) Controllers() []string {
	var out []string
	for _, crd := range k.enabledCRDs {
		if crd.ReconcilerConfig.Constructor != nil && crd.ReconcilerConfig.Default {
			out = append(out, crd.Name)
		}
	}
	return out
}

// CRDNames returns the names of all enabled CRDs.
func (k *Katalog) CRDNames() []string {
	names := make([]string, 0, len(k.enabledCRDs))
	for _, crd := range k.enabledCRDs {
		names = append(names, crd.Name)
	}
	return names
}

// Depends returns true if crdName depends on target.
func (k *Katalog) Depends(crdName, target string) bool {
	crd, err := k.Get(crdName)
	if err != nil {
		return false
	}
	for _, dep := range crd.DependsOn {
		if dep == target {
			return true
		}
	}
	return false
}

// Dependents returns all CRDs that depend on the given CRD.
func (k *Katalog) Dependents(name string) []string {
	var out []string
	for _, crd := range k.enabledCRDs {
		for _, dep := range crd.DependsOn {
			if dep == name {
				out = append(out, crd.Name)
			}
		}
	}
	return out
}

// Enabled returns only the enabled CRDs in the katalog.
func (k *Katalog) Enabled() []orktypes.CRDEntry {
	return k.enabledCRDs
}

// Get tries to get an enabled crd
func (k *Katalog) Get(name string) (*orktypes.CRDEntry, error) {
	for _, crd := range k.enabledCRDs {
		if crd.Name == name {
			return &crd, nil
		}
	}
	return nil, fmt.Errorf("crd not found in katalog")
}
