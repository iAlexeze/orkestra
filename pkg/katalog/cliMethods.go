package katalog

import (
	"fmt"
	"strings"

	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

func (k *Katalog) List() map[string]orktypes.CRDEntry {
	return k.Spec.CRDs
}

// All returns every CRD in the katalog, including disabled ones.
func (k *Katalog) All() map[string]orktypes.CRDEntry {
	return k.Spec.CRDs
}

// Useful Metadata
func (k *Katalog) Meta() orktypes.KatalogMeta {
	return k.metadata
}

// Exists returns true if a CRD with the given name exists in the katalog.
func (k *Katalog) Exists(name string) bool {
	_, ok := k.Spec.CRDs[name]
	return ok
}

// Describe returns a human‑readable summary of a CRD.
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
	if crd.IsNamespaced() {
		fmt.Fprintf(b, "Namespace:   %s\n", crd.Namespace)
	}
	fmt.Fprintf(b, "Workers:     %d\n", crd.Workers)
	fmt.Fprintf(b, "Resync:      %s\n", crd.Resync)
	fmt.Fprintf(b, "Enabled:     %v\n", crd.Enabled)

	deps := crd.DependsOn.Names()
	if len(deps) > 0 {
		fmt.Fprintf(b, "Dependencies: %v\n", strings.Join(deps, " "))
	} else {
		fmt.Fprint(b, "Dependencies: None")
	}

	fmt.Fprintf(b, "Description:\n%s\n", crd.Description)

	return b.String(), nil
}

// Explain returns a technical explanation of how Orkestra handles this CRD.
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
	if crd.DefaultReconcile() {
		fmt.Fprint(b, "Reconciler:   Default\n")
	} else {
		fmt.Fprintf(b, "Reconciler:   %T\n", crd.ReconcilerConfig.Constructor)
	}
	fmt.Fprintf(b, "Informer:     LIST, WATCH\n")

	deps := crd.DependsOn.Names()
	if len(deps) > 0 {
		fmt.Fprintf(b, "Dependencies: %v\n", strings.Join(deps, " "))
	} else {
		fmt.Fprint(b, "Dependencies: None")
	}

	return b.String(), nil
}

// Graph returns a map of CRD name → dependency names.
func (k *Katalog) Graph() map[string][]string {
	graph := make(map[string][]string, len(k.enabledCRDs))
	for name, crd := range k.enabledCRDs {
		graph[name] = crd.DependsOn.Names()
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
		if crd.ReconcilerConfig.Constructor != nil && crd.DefaultReconcile() {
			out = append(out, crd.Name)
		}
	}
	return out
}

// CRDNames returns the names of all enabled CRDs.
func (k *Katalog) CRDNames() []string {
	names := make([]string, 0, len(k.enabledCRDs))
	for name := range k.enabledCRDs {
		names = append(names, name)
	}
	return names
}

// Depends returns true if crdName depends on target.
func (k *Katalog) Depends(crdName, target string) bool {
	crd, err := k.Get(crdName)
	if err != nil {
		return false
	}
	_, ok := crd.DependsOn[target]
	return ok
}

// Dependents returns all CRDs that depend on the given CRD.
func (k *Katalog) Dependents(name string) []string {
	var out []string
	for _, crd := range k.enabledCRDs {
		if _, ok := crd.DependsOn[name]; ok {
			out = append(out, crd.Name)
		}
	}
	return out
}

// Enabled returns only the enabled CRDs in the katalog.
func (k *Katalog) Enabled() map[string]orktypes.CRDEntry {
	return k.enabledCRDs
}

// Get returns an enabled CRD by name.
func (k *Katalog) Get(name string) (*orktypes.CRDEntry, error) {
	crd, ok := k.enabledCRDs[name]
	if !ok {
		return nil, fmt.Errorf("crd not found in katalog")
	}
	return &crd, nil
}
