// pkg/katalog/dependencies.go
package katalog

import (
	"fmt"
	"sort"
	"sync"

	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"
)

type DependencyGraph struct {
	nodes        map[string]*Node
	edges        map[string][]string
	katalog      *Katalog
	startupOrder []string // cache
	once         sync.Once
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
	for name, crd := range crds {
		g.nodes[name] = &Node{
			Name: name,
			CRD:  crd,
		}
	}

	// Create edges
	for name, crd := range crds {
		for dep := range crd.DependsOn {
			if _, exists := g.nodes[dep]; !exists {
				utils.Exit(fmt.Errorf("dependency %s not found for %s", dep, name))
			}
			g.edges[dep] = append(g.edges[dep], name)
			g.nodes[name].InDegree++
			g.nodes[dep].OutDegree++
		}
	}

	return g
}

// Topological sort for startup order
func (g *DependencyGraph) StartupOrder() []string {
	g.once.Do(func() {
		// Copy indegree (DO NOT mutate original)
		indegree := make(map[string]int, len(g.nodes))
		for name, node := range g.nodes {
			indegree[name] = node.InDegree
		}

		// Sort node names for deterministic processing
		var names []string
		for name := range g.nodes {
			names = append(names, name)
		}
		sort.Strings(names)

		// Initialize queue (sorted)
		var queue []string
		for _, name := range names {
			if indegree[name] == 0 {
				queue = append(queue, name)
			}
		}

		var order []string

		for len(queue) > 0 {
			// Always pick the first (queue is kept sorted)
			current := queue[0]
			queue = queue[1:]

			order = append(order, current)

			// Process neighbors in sorted order
			neighbors := append([]string{}, g.edges[current]...)
			sort.Strings(neighbors)

			for _, neighbor := range neighbors {
				indegree[neighbor]--
				if indegree[neighbor] == 0 {
					queue = append(queue, neighbor)
				}
			}

			// Keep queue sorted for determinism
			sort.Strings(queue)
		}

		if len(order) != len(g.nodes) {
			utils.Exit(fmt.Errorf("circular dependency detected"))
		}

		g.startupOrder = order
	})

	return g.startupOrder
}

// Reverse topological sort for shutdown order
func (g *DependencyGraph) ShutdownOrder() []string {
	// Reverse it
	return utils.Reversed(g.StartupOrder())
}

// Constructors
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

func (g *DependencyGraph) Validate() error {
	for name, node := range g.nodes {
		if node.InDegree < 0 {
			return fmt.Errorf("negative in-degree for %s", name)
		}
		if node.OutDegree < 0 {
			return fmt.Errorf("negative out-degree for %s", name)
		}
	}
	return nil
}
