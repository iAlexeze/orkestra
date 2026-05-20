//go:build !runtime && !gateway

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/simulate"
	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	sigsyaml "sigs.k8s.io/yaml"
)

var simulateCmd = &cobra.Command{
	Use:   "simulate",
	Short: "Simulate operator reconciliation in memory — no cluster required",
	Long: `Runs the operator reconcile loop against a fake in-memory cluster.
Shows resource creation and state transitions across reconcile cycles.

  ork simulate -f katalog.yaml --cr cr.yaml
  ork simulate -f katalog.yaml --cr cr.yaml --crd website --cycles 5`,
	RunE: func(cmd *cobra.Command, args []string) error {
		katalogFile, _ := cmd.Flags().GetString("file")
		crFile, _ := cmd.Flags().GetString("cr")
		crdName, _ := cmd.Flags().GetString("crd")
		maxCycles, _ := cmd.Flags().GetInt("cycles")
		return runSimulate(cmd.Context(), katalogFile, crFile, crdName, maxCycles)
	},
}

func runSimulate(ctx context.Context, katalogFile, crFile, crdName string, maxCycles int) error {
	if maxCycles <= 0 {
		maxCycles = 10
	}

	kat, err := katalog.ParseFile(katalogFile)
	if err != nil {
		return fmt.Errorf("parsing Katalog: %w", err)
	}

	crData, err := os.ReadFile(crFile)
	if err != nil {
		return fmt.Errorf("reading CR: %w", err)
	}
	// Convert YAML→JSON so numbers become float64, not int.
	// k8s DeepCopyJSON only handles float64 for numeric values.
	jsonData, err := sigsyaml.YAMLToJSON(crData)
	if err != nil {
		return fmt.Errorf("parsing CR: %w", err)
	}
	var cr unstructured.Unstructured
	if err := json.Unmarshal(jsonData, &cr.Object); err != nil {
		return fmt.Errorf("parsing CR: %w", err)
	}

	// If --crd is given, simulate that CRD only. Otherwise simulate all.
	var targets []string
	if crdName != "" {
		targets = []string{crdName}
	} else {
		targets = kat.CRDNames()
	}

	for _, name := range targets {
		if err := simulateOne(ctx, kat, name, &cr, maxCycles); err != nil {
			return err
		}
	}
	return nil
}

func simulateOne(ctx context.Context, kat *katalog.Katalog, crdName string, cr *unstructured.Unstructured, maxCycles int) error {
	fmt.Printf("Simulating %s/%s\n\n", crdName, cr.GetName())

	start := time.Now()
	result, err := simulate.Run(ctx, kat, crdName, cr, maxCycles)
	if err != nil {
		return err
	}
	elapsed := time.Since(start)

	var prevKey string
	var repeatStart int
	flush := func(upTo int) {
		if repeatStart > 0 && upTo > repeatStart {
			fmt.Printf("  %s\n", utils.Gray(fmt.Sprintf("(cycles %d–%d: identical)", repeatStart, upTo)))
			repeatStart = 0
		}
	}

	for _, cycle := range result.Cycles {
		meaningful := filterOps(cycle.Ops, "create", "update", "delete", "patch")
		if len(meaningful) == 0 && cycle.Error == nil {
			continue
		}

		key := opsKey(meaningful)
		if key == prevKey && cycle.Error == nil {
			if repeatStart == 0 {
				repeatStart = cycle.Cycle
			}
			continue
		}
		flush(cycle.Cycle - 1)
		prevKey = key

		fmt.Printf("  Cycle %d:\n", cycle.Cycle)
		printCycleOps(meaningful)
		if cycle.Error != nil {
			fmt.Printf("    %s %v\n", utils.FailureMark(), cycle.Error)
		}
	}
	flush(result.Cycles[len(result.Cycles)-1].Cycle)

	if result.Steady {
		fmt.Printf("\n  %s Steady state at cycle %d in %s\n\n", utils.SuccessMark(), result.SteadyAt, elapsed.Round(time.Millisecond))
	} else {
		fmt.Printf("\n  ~ Max cycles reached (%d) in %s\n\n", maxCycles, elapsed.Round(time.Millisecond))
	}

	return nil
}

// printCycleOps prints one cycle's ops, coalescing duplicate resource+name entries.
// If a resource is both created and updated in the same cycle (e.g. reconcile: true),
// only the create icon (+) is shown.
func printCycleOps(ops []simulate.Op) {
	type entry struct {
		resource  string
		name      string
		hasCreate bool
		hasDelete bool
	}
	seen := map[string]*entry{}
	var order []string
	for _, op := range ops {
		key := op.Resource + "/" + op.Name
		if _, ok := seen[key]; !ok {
			seen[key] = &entry{resource: op.Resource, name: op.Name}
			order = append(order, key)
		}
		switch op.Verb {
		case "create":
			seen[key].hasCreate = true
		case "delete":
			seen[key].hasDelete = true
		}
	}
	for _, key := range order {
		e := seen[key]
		var icon string
		switch {
		case e.hasCreate:
			icon = iconAdded()
		case e.hasDelete:
			icon = iconRemoved()
		default:
			icon = iconChanged()
		}
		fmt.Printf("    %s %s/%s\n", icon, e.resource, e.name)
	}
}

// opsKey returns a stable string key for a slice of ops, used to detect identical cycles.
func opsKey(ops []simulate.Op) string {
	var b strings.Builder
	for _, op := range ops {
		b.WriteString(op.Verb)
		b.WriteByte('/')
		b.WriteString(op.Resource)
		b.WriteByte('/')
		b.WriteString(op.Name)
		b.WriteByte('|')
	}
	return b.String()
}

func filterOps(ops []simulate.Op, verbs ...string) []simulate.Op {
	verbSet := make(map[string]bool)
	for _, v := range verbs {
		verbSet[v] = true
	}
	var result []simulate.Op
	for _, op := range ops {
		if verbSet[op.Verb] {
			result = append(result, op)
		}
	}
	return result
}

func init() {
	rootCmd.AddCommand(simulateCmd)

	simulateCmd.Flags().StringP("file", "f", "", "Path to katalog.yaml")
	simulateCmd.Flags().String("cr", "", "Path to the CR YAML file to simulate")
	simulateCmd.Flags().String("crd", "", "CRD name to simulate (default: all CRDs in Katalog)")
	simulateCmd.Flags().Int("cycles", 10, "Maximum number of reconcile cycles")

	// Shadow global flags
	simulateCmd.Flags().Bool("debug", false, "")
	simulateCmd.Flags().String("kubeconfig", "", "")
	simulateCmd.Flags().Bool("verbose", false, "")
	simulateCmd.Flags().MarkHidden("debug")
	simulateCmd.Flags().MarkHidden("kubeconfig")
	simulateCmd.Flags().MarkHidden("verbose")
}
