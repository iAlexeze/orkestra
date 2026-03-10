// pkg/registry/dependencies.go
package registry

import (
	"fmt"

	"github.com/ialexeze/multi-crd-controller/pkg/config/initialize"
)

type DependencyGraph struct {
	nodes    map[string]*Node
	edges    map[string][]string
	registry *CRDRegistry
}

type Node struct {
	Name      string
	CRD       initialize.CRDEntry
	InDegree  int
	OutDegree int
}

func NewDependencyGraph(registry *CRDRegistry) *DependencyGraph {
	crds := registry.CRDs
	g := &DependencyGraph{
		nodes:    make(map[string]*Node),
		edges:    make(map[string][]string),
		registry: registry,
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
				panic(fmt.Sprintf("dependency %s not found for %s", dep, crd.Name))
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
		panic("circular dependency detected")
	}

	return order
}

// Reverse topological sort for shutdown order
func (g *DependencyGraph) ShutdownOrder() []string {
	order := g.StartupOrder()
	// Reverse it
	for i := len(order)/2 - 1; i >= 0; i-- {
		opp := len(order) - 1 - i
		order[i], order[opp] = order[opp], order[i]
	}
	return order
}

// Constructors
func (g *DependencyGraph) GetMode() string {
	if g.registry.Mode.Go {
		return "Go"
	} else if g.registry.Mode.Yaml {
		return "YAML"
	} else {
		return ""
	}
}
func (g *DependencyGraph) GetNode(name string) *Node {
	return g.nodes[name]
}

func (g *DependencyGraph) GetEdge(name string) []string {
	return g.edges[name]
}

func (g *DependencyGraph) GetInDegree(name string) int {
	return g.nodes[name].InDegree
}

func (g *DependencyGraph) GetOutDegree(name string) int {
	return g.nodes[name].OutDegree
}

func (g *DependencyGraph) GetCRD(name string) initialize.CRDEntry {
	return g.nodes[name].CRD
}

func (g *DependencyGraph) GetName(name string) string {
	return g.nodes[name].Name
}

func (g *DependencyGraph) GetNodes() map[string]*Node {
	return g.nodes
}

func (g *DependencyGraph) GetEdges() map[string][]string {
	return g.edges
}
