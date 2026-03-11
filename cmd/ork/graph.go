package ork

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/spf13/cobra"
)

var (
	flagAll  bool
	flagJSON bool
	flagDOT  bool
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Visualize Orkestra dependency graphs",
}

var graphDepsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Show the CRD dependency graph",
	RunE: func(cmd *cobra.Command, args []string) error {
		k := katalog.NewKatalog(kfg.Katalog().Mode, kfg.Katalog().Path)

		graph := k.Graph()

		switch {
		case flagJSON:
			return json.NewEncoder(os.Stdout).Encode(graph)
		case flagDOT:
			printDOT(graph)
		default:
			printGraph(graph)
		}

		return nil
	},
}

var graphOrderCmd = &cobra.Command{
	Use:   "order",
	Short: "Show CRDs in dependency-safe order",
	RunE: func(cmd *cobra.Command, args []string) error {
		k := katalog.NewKatalog(kfg.Katalog().Mode, kfg.Katalog().Path)

		for i, name := range k.Order() {
			fmt.Printf("%d. %s\n", i+1, name)
		}

		return nil
	},
}

var graphTreeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Show a tree view of CRD dependencies",
	RunE: func(cmd *cobra.Command, args []string) error {
		k := katalog.NewKatalog(kfg.Katalog().Mode, kfg.Katalog().Path)

		printTree(k)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(graphCmd)

	graphCmd.AddCommand(graphDepsCmd)
	graphCmd.AddCommand(graphOrderCmd)
	graphCmd.AddCommand(graphTreeCmd)

	graphDepsCmd.Flags().BoolVar(&flagAll, "all", false, "Include disabled CRDs")
	graphDepsCmd.Flags().BoolVar(&flagJSON, "json", false, "Output graph as JSON")
	graphDepsCmd.Flags().BoolVar(&flagDOT, "dot", false, "Output graph in Graphviz DOT format")
}

// Helpers
// Pretty dependency graph
func printGraph(graph map[string][]string) {
	for crd, deps := range graph {
		if len(deps) == 0 {
			fmt.Printf("%s\n", crd)
			continue
		}
		fmt.Printf("%s -> %v\n", crd, deps)
	}
}

// DOT dependency graph
func printDOT(graph map[string][]string) {
	fmt.Println("digraph CRDs {")
	for crd, deps := range graph {
		for _, dep := range deps {
			fmt.Printf("  \"%s\" -> \"%s\";\n", crd, dep)
		}
	}
	fmt.Println("}")
}

// Tree dependency graph
func printTree(k *katalog.Katalog) {
	graph := k.Graph()
	rev := reverseGraph(graph)

	// Find roots (CRDs with no dependencies)
	roots := []string{}
	for name, deps := range graph {
		if len(deps) == 0 {
			roots = append(roots, name)
		}
	}

	for _, root := range roots {
		printTreeNode(rev, root, "")
	}
}

func printTreeNode(graph map[string][]string, node, indent string) {
	fmt.Println(indent + node)
	for _, dep := range graph[node] {
		printTreeNode(graph, dep, indent+"  ")
	}
}

func reverseGraph(graph map[string][]string) map[string][]string {
	rev := make(map[string][]string)
	for parent := range graph {
		rev[parent] = []string{}
	}
	for child, deps := range graph {
		for _, dep := range deps {
			rev[dep] = append(rev[dep], child)
		}
	}
	return rev
}
