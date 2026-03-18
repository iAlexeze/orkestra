// pkg/katalog/dependencies.go
package katalog

import (
	"fmt"

	"github.com/ialexeze/orkestra/pkg/konfig"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"
)

type DependencyGraph struct {
	nodes   map[string]*Node
	edges   map[string][]string
	katalog *Katalog
}

type Node struct {
	Name      string
	CRD       orktypes.CRDEntry
	InDegree  int
	OutDegree int
}

func NewDependencyGraph(katalog *Katalog) *DependencyGraph {
	crds := katalog.enabledCRDs
	g := &DependencyGraph{
		nodes:   make(map[string]*Node),
		edges:   make(map[string][]string),
		katalog: katalog,
	}

	// Create nodes
	for _, crd := range crds {
		g.nodes[crd.Name] = &Node{
			Name: crd.Name,
			CRD:  crd,
		}
	}

	// Create edges
	for _, crd := range crds {
		for _, dep := range crd.DependsOn {
			if _, exists := g.nodes[dep]; !exists {
				utils.Exit(fmt.Errorf("dependency %s not found for %s", dep, crd.Name))
			}
			g.edges[dep] = append(g.edges[dep], crd.Name)
			g.nodes[crd.Name].InDegree++
			g.nodes[dep].OutDegree++
		}
	}

	return g
}

// Topological sort for startup order
func (g *DependencyGraph) StartupOrder() []string {
	var order []string
	queue := []string{}

	// Find nodes with no dependencies (in-degree 0)
	for name, node := range g.nodes {
		if node.InDegree == 0 {
			queue = append(queue, name)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)

		for _, neighbor := range g.edges[current] {
			g.nodes[neighbor].InDegree--
			if g.nodes[neighbor].InDegree == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(order) != len(g.nodes) {
		utils.Exit(fmt.Errorf("circular dependency detected"))
	}

	return order
}

// Reverse topological sort for shutdown order
func (g *DependencyGraph) ShutdownOrder() []string {
	// Reverse it
	return utils.Reversed(g.StartupOrder())
}

// Constructors
func (g *DependencyGraph) GetMode() string {
	switch {
	case g.katalog.mode.Dynamic:
		return konfig.DynamicMode
	case g.katalog.mode.Typed:
		return konfig.TypedMode
	}
	return ""
}

func (g *DependencyGraph) GetNode(name string) *Node {
	return g.nodes[name]
}

func (g *DependencyGraph) GetNodes() map[string]*Node {
	return g.nodes
}

func (g *DependencyGraph) GetEdges() map[string][]string {
	return g.edges
}

func (g *DependencyGraph) GetInDegree(name string) int {
	return g.nodes[name].InDegree
}

func (g *DependencyGraph) GetOutDegree(name string) int {
	return g.nodes[name].OutDegree
}

func (g *DependencyGraph) GetDependents(name string) []string {
	var dependents []string
	for dep, deps := range g.edges {
		for _, d := range deps {
			if d == name {
				dependents = append(dependents, dep)
			}
		}
	}
	return dependents
}

func (g *DependencyGraph) GetDependencies(name string) []string {
	return g.edges[name]
}
