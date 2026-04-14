package cli

import (
	"fmt"

	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

type CRDEntryDTO struct {
	Name       string           `json:"name" yaml:"name"`
	Enabled    bool             `json:"enabled" yaml:"enabled"`
	Group      string           `json:"group" yaml:"group"`
	Version    string           `json:"version" yaml:"version"`
	Kind       string           `json:"kind" yaml:"kind"`
	Plural     string           `json:"plural" yaml:"plural"`
	Namespaced bool             `json:"namespaced" yaml:"namespaced"`
	Namespace  string           `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Workers    int              `json:"workers" yaml:"workers"`
	Resync     string           `json:"resync" yaml:"resync"`
	DependsOn  []string         `json:"dependsOn,omitempty" yaml:"dependsOn,omitempty"`
	Finalizers []string         `json:"finalizers,omitempty" yaml:"finalizers,omitempty"`
	Mode       orktypes.CRDMode `json:"mode" yaml:"mode"`
}

func toDTO(crd orktypes.CRDEntry) CRDEntryDTO {
	return CRDEntryDTO{
		Name:       crd.Name,
		Enabled:    crd.IsEnabled(),
		Group:      crd.APITypes.Group,
		Version:    crd.APITypes.Version,
		Kind:       crd.APITypes.Kind,
		Plural:     crd.APITypes.Plural,
		Namespaced: crd.IsNamespaced(),
		Namespace:  crd.Namespace,
		Workers:    crd.Workers,
		Resync:     crd.Resync.String(),
		DependsOn:  crd.DependsOn.Names(),
		Finalizers: crd.OperatorBox.Finalizers,
		Mode:       crd.Mode,
	}
}

func printPrettyCRD(crd orktypes.CRDEntry) {
	fmt.Printf("CRD: %s\n", crd.Name)
	fmt.Printf("  Group: %s\n", crd.APITypes.Group)
	fmt.Printf("  Version: %s\n", crd.APITypes.Version)
	fmt.Printf("  Kind: %s\n", crd.APITypes.Kind)
	fmt.Printf("  Plural: %s\n", crd.APITypes.Plural)
	fmt.Printf("  Enabled: %v\n", crd.Enabled)
	fmt.Printf("  Mode: %s\n", crd.Mode)
	fmt.Printf("  Namespaced: %v\n", crd.Namespaced)
	if crd.Namespace != "" {
		fmt.Printf("  Namespace: %s\n", crd.Namespace)
	}
	fmt.Printf("  Workers: %d\n", crd.Workers)
	fmt.Printf("  Resync: %s\n", crd.Resync)

	if len(crd.DependsOn) > 0 {
		fmt.Println("  DependsOn:")
		for _, dep := range crd.DependsOn.Names() {
			fmt.Printf("    - %s\n", dep)
		}
	}

	fmt.Println("  Reconciler:")
	fmt.Printf("    Default: %v\n", crd.OperatorBox.Default)

	if len(crd.OperatorBox.Finalizers) > 0 {
		fmt.Println("    Finalizers:")
		for _, f := range crd.OperatorBox.Finalizers {
			fmt.Printf("      - %s\n", f)
		}
	}

	fmt.Println()
}

func printGraph(crds map[string]orktypes.CRDEntry) {
	fmt.Println("Dependency Graph:")
	for name, crd := range crds {
		fmt.Printf("%s\n", name)
		for _, dep := range crd.DependsOn.Names() {
			fmt.Printf("  └─ %s\n", dep)
		}
	}
}
