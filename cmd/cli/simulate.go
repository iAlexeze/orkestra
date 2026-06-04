//go:build !runtime && !gateway

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	orke2e "github.com/orkspace/orkestra/pkg/e2e"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/simulate"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
		crdName, _ := cmd.Flags().GetString("crd")
		maxCycles, _ := cmd.Flags().GetInt("cycles")

		skipExternal, _ := cmd.Flags().GetBool("skip-external")
		opts := simulate.RunOptions{SkipExternal: skipExternal}

		// Discovery mode: ork simulate ./...
		if len(args) > 0 && args[0] == "./..." {
			skipRaw, _ := cmd.Flags().GetStringSlice("skip")
			root := "."
			return runSimulateDiscovery(cmd.Context(), root, crdName, maxCycles, skipRaw, opts)
		}

		katalogFile, _ := cmd.Flags().GetString("file")
		if katalogFile == "" {
			if d := defaultFilePaths(); len(d) > 0 {
				katalogFile = d[0]
			}
		}
		if katalogFile == "" {
			return fmt.Errorf(errNoKatalog)
		}

		// E2E mode: input is an e2e.yaml
		if isE2EDoc(katalogFile) {
			return runSimulateFromE2E(cmd.Context(), katalogFile, crdName, maxCycles, opts)
		}

		crFile, _ := cmd.Flags().GetString("cr")
		if crFile == "" {
			crFile = fileCr
		}
		if crFile == "" {
			return fmt.Errorf("--cr is required")
		}

		return runSimulate(cmd.Context(), katalogFile, crFile, crdName, maxCycles, opts)
	},
}

func runSimulate(ctx context.Context, katalogFile, crFile, crdName string, maxCycles int, opts simulate.RunOptions) error {
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

	// Parse all documents; key by lowercase kind so each CRD gets its own CR.
	crs := parseMultiDocCRs(crData)
	if len(crs) == 0 {
		return fmt.Errorf("reading CR: no valid documents found in %s", crFile)
	}

	// If --crd is given, simulate that CRD only. Otherwise simulate all.
	var targets []string
	if crdName != "" {
		targets = []string{crdName}
	} else {
		targets = kat.CRDNames()
	}

	for _, name := range targets {
		crdEntry, ok := kat.CRDEntry(name)
		if !ok {
			continue
		}
		cr, ok := crs[strings.ToLower(crdEntry.APITypes.Kind)]
		if !ok {
			if len(targets) > 1 {
				fmt.Printf("  %s no CR found for %s — skipped\n\n", dim("note:"), crdEntry.APITypes.Kind)
				continue
			}
			return fmt.Errorf("no CR found for CRD %q (kind: %s) in %s", name, crdEntry.APITypes.Kind, crFile)
		}
		if err := simulateOne(ctx, kat, name, cr, maxCycles, opts); err != nil {
			return err
		}
	}
	return nil
}

func simulateOne(ctx context.Context, kat *katalog.Katalog, crdName string, cr *unstructured.Unstructured, maxCycles int, opts simulate.RunOptions) error {
	fmt.Printf("Simulating %s/%s\n", crdName, cr.GetName())

	// Emit notes for operatorBox blocks that cannot execute in the fake cluster.
	crdEntry, _ := kat.CRDEntry(crdName)
	if crdEntry.OperatorBox.OnReconcile != nil && len(crdEntry.OperatorBox.OnReconcile.External) > 0 {
		if opts.SkipExternal {
			fmt.Printf("  %s external: calls stubbed — result fields will be empty\n", dim("note:"))
		} else {
			fmt.Printf("  %s external: calls will hit the real network (pass --skip-external to stub)\n", dim("note:"))
		}
	}
	if len(crdEntry.OperatorBox.Cross) > 0 {
		fmt.Printf("  %s cross: observation not executed — cross.* fields will be empty\n", dim("note:"))
	}
	fmt.Println()

	start := time.Now()
	result, err := simulate.Run(ctx, kat, crdName, cr, maxCycles, opts)
	if err != nil {
		return err
	}
	elapsed := time.Since(start)

	for _, note := range result.Notes {
		fmt.Printf("  %s %s\n", dim("note:"), note)
	}
	if len(result.Notes) > 0 {
		fmt.Println()
	}

	var prevKey string
	var repeatStart int
	flush := func(upTo int) {
		if repeatStart > 0 && upTo > repeatStart {
			fmt.Printf("  %s\n", gray(fmt.Sprintf("(cycles %d–%d: identical)", repeatStart, upTo)))
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
			fmt.Printf("    %s %v\n", failureMark(), cycle.Error)
		}
	}
	flush(result.Cycles[len(result.Cycles)-1].Cycle)

	if result.Steady {
		fmt.Printf("\n  %s Steady state at cycle %d in %s\n\n", successMark(), result.SteadyAt, elapsed.Round(time.Millisecond))
	} else {
		fmt.Printf("\n  ~ Max cycles reached (%d) in %s\n\n", maxCycles, elapsed.Round(time.Millisecond))
	}

	for _, c := range result.Cycles {
		if c.Error != nil {
			return fmt.Errorf("simulation completed with cycle errors")
		}
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

// parseMultiDocCRs splits a YAML file on document separators and returns all
// valid CR documents keyed by lowercase kind. Supports single- and multi-doc
// CR files (multiple CRs separated by ---).
func parseMultiDocCRs(data []byte) map[string]*unstructured.Unstructured {
	crs := map[string]*unstructured.Unstructured{}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var node yaml.Node
		if err := dec.Decode(&node); err != nil {
			break // EOF or parse error
		}
		b, err := yaml.Marshal(&node)
		if err != nil {
			continue
		}
		j, err := sigsyaml.YAMLToJSON(b)
		if err != nil {
			continue
		}
		var cr unstructured.Unstructured
		if err := json.Unmarshal(j, &cr.Object); err != nil {
			continue
		}
		if kind := cr.GetKind(); kind != "" {
			crs[strings.ToLower(kind)] = &cr
		}
	}
	return crs
}

// ── E2E-aware entry points ─────────────────────────────────────────────────────

// isE2EDoc returns true when the file's kind is "E2E".
func isE2EDoc(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var head struct {
		Kind string `yaml:"kind"`
	}
	_ = yaml.Unmarshal(data, &head)
	return head.Kind == "E2E"
}

// runSimulateFromE2E loads an e2e.yaml, applies skip/note rules, and runs
// simulate against the katalog and CR it declares.
// Returns (skipped, skipReason, error).
func runSimulateFromE2E(ctx context.Context, path, crdName string, maxCycles int, opts simulate.RunOptions) error {
	// Convert to absolute so all downstream path joins (crdFile, crFiles, katalog
	// imports) use the file's directory as the base — same pattern as files.go.
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	var doc orktypes.E2E
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	if doc.Spec.CustomOperator {
		fmt.Printf("  %s %s — skipped (customOperator)\n", dim("○"), path)
		return nil
	}

	dir := filepath.Dir(path)

	// Aggregator: has imports but no direct cr/katalog — loop through imports.
	if doc.Spec.CR == "" && len(doc.Imports) > 0 {
		for _, imp := range doc.Imports {
			impPath := imp.Path
			if !filepath.IsAbs(impPath) {
				impPath = filepath.Join(dir, impPath)
			}
			if err := runSimulateFromE2E(ctx, impPath, crdName, maxCycles, opts); err != nil {
				return err
			}
		}
		return nil
	}

	katalogPath := filepath.Join(dir, doc.Spec.Katalog)
	crPath := filepath.Join(dir, doc.Spec.CR)
	return runSimulate(ctx, katalogPath, crPath, crdName, maxCycles, opts)
}

// ── Discovery mode ─────────────────────────────────────────────────────────────

type simulateFileResult struct {
	path      string
	skipped   bool
	skipMsg   string
	steady    bool
	cycle     int
	elapsed   time.Duration
	cycleErrs bool
}

// runSimulateDiscovery finds all e2e.yaml files under root, simulates each,
// and prints an aggregate summary.
func runSimulateDiscovery(ctx context.Context, root, crdName string, maxCycles int, skip []string, opts simulate.RunOptions) error {
	var patterns []string
	for _, s := range skip {
		patterns = append(patterns, s)
	}
	paths, err := orke2e.DiscoverE2EFiles(root, patterns)
	if err != nil {
		return fmt.Errorf("discovering e2e files: %w", err)
	}
	if len(paths) == 0 {
		fmt.Printf("no e2e.yaml files found under %s\n", root)
		return nil
	}

	fmt.Printf("Simulating %d e2e file(s) under %s\n\n", len(paths), root)

	// DiscoverE2EFiles returns absolute paths; Rel needs an absolute base too.
	absRoot, _ := filepath.Abs(root)

	var results []simulateFileResult
	for _, p := range paths {
		rel, _ := filepath.Rel(absRoot, p)

		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var doc orktypes.E2E
		if err := yaml.Unmarshal(data, &doc); err != nil {
			continue
		}
		if doc.Spec.CustomOperator {
			fmt.Printf("  %-55s %s\n", rel, dim("○ skipped (customOperator)"))
			results = append(results, simulateFileResult{path: rel, skipped: true, skipMsg: "customOperator"})
			continue
		}
		if doc.Spec.CR == "" {
			continue // pure aggregator — skip silently
		}

		dir := filepath.Dir(p)
		katalogPath := filepath.Join(dir, doc.Spec.Katalog)
		crPath := filepath.Join(dir, doc.Spec.CR)

		start := time.Now()
		// Capture simulate output by running inline via a minimal path.
		kat, err := katalog.ParseFile(katalogPath)
		if err != nil {
			fmt.Printf("  %-55s %s\n", rel, red("✗ "+err.Error()))
			continue
		}
		crData, err := os.ReadFile(crPath)
		if err != nil {
			fmt.Printf("  %-55s %s\n", rel, red("✗ "+err.Error()))
			continue
		}

		crs := parseMultiDocCRs(crData)
		if len(crs) == 0 {
			fmt.Printf("  %-55s %s\n", rel, red("✗ no valid CR documents"))
			continue
		}

		targets := kat.CRDNames()
		if crdName != "" {
			targets = []string{crdName}
		}

		var res simulateFileResult
		res.path = rel

		var hasCycleErrors bool
		for _, name := range targets {
			crdEntry, ok := kat.CRDEntry(name)
			if !ok {
				continue
			}
			cr, ok := crs[strings.ToLower(crdEntry.APITypes.Kind)]
			if !ok {
				continue // no CR for this CRD — skip silently in discovery
			}
			r, err := simulate.Run(ctx, kat, name, cr, maxCycles, opts)
			if err != nil {
				fmt.Printf("  %-55s %s\n", rel, red("✗ "+err.Error()))
				break
			}
			res.steady = r.Steady
			res.cycle = r.SteadyAt
			for _, c := range r.Cycles {
				if c.Error != nil {
					hasCycleErrors = true
					break
				}
			}
		}
		res.elapsed = time.Since(start)

		var suffix string
		if res.steady {
			suffix = green(fmt.Sprintf("✓ steady at cycle %d (%s)", res.cycle, res.elapsed.Round(time.Millisecond)))
		} else {
			suffix = yellow(fmt.Sprintf("~ max cycles (%s)", res.elapsed.Round(time.Millisecond)))
		}

		// Append inactive-block and cycle-error tags
		crdEntry, _ := kat.CRDEntry(targets[0])
		var tags []string
		if crdEntry.OperatorBox.OnReconcile != nil && len(crdEntry.OperatorBox.OnReconcile.External) > 0 {
			tags = append(tags, "external: inactive")
		}
		if len(crdEntry.OperatorBox.Cross) > 0 {
			tags = append(tags, "cross: inactive")
		}
		if hasCycleErrors {
			tags = append(tags, "cycle errors")
			res.cycleErrs = true
		}
		if len(tags) > 0 {
			suffix += "  " + dim("["+strings.Join(tags, ", ")+"]")
		}

		fmt.Printf("  %-55s %s\n", rel, suffix)
		results = append(results, res)
	}

	// Aggregate summary
	var simulated, skipped int
	var slowest simulateFileResult
	for _, r := range results {
		if r.skipped {
			skipped++
		} else {
			simulated++
			if r.elapsed > slowest.elapsed {
				slowest = r
			}
		}
	}
	fmt.Printf("\n  %d file(s) — %d simulated, %d skipped\n", len(results), simulated, skipped)
	if slowest.path != "" {
		fmt.Printf("  Slowest: %s (cycle %d, %s)\n", slowest.path, slowest.cycle, slowest.elapsed.Round(time.Millisecond))
	}

	var errFiles []simulateFileResult
	for _, r := range results {
		if r.cycleErrs {
			errFiles = append(errFiles, r)
		}
	}
	if len(errFiles) > 0 {
		fmt.Printf("\n  %s — run directly for full output:\n", yellow("Files with cycle errors"))
		for _, r := range errFiles {
			fmt.Printf("    ork simulate -f %s\n", r.path)
		}
		return fmt.Errorf("simulation completed with cycle errors in %d file(s)", len(errFiles))
	}

	return nil
}

func init() {
	rootCmd.AddCommand(simulateCmd)

	simulateCmd.Flags().StringP("file", "f", "", "Path to katalog.yaml")
	simulateCmd.Flags().String("cr", "", "Path to the CR YAML file to simulate")
	simulateCmd.Flags().String("crd", "", "CRD name to simulate (default: all CRDs in Katalog)")
	simulateCmd.Flags().Int("cycles", 10, "Maximum number of reconcile cycles")
	simulateCmd.Flags().StringSlice("skip", []string{}, "Comma-separated path patterns to skip during ./... discovery (e.g. vendor,cr-e2e.yaml)")
	simulateCmd.Flags().Bool("skip-external", false, "Stub external: HTTP calls with empty 200 responses instead of hitting the real network")

	// Shadow global flags
	simulateCmd.Flags().Bool("debug", false, "")
	simulateCmd.Flags().String("kubeconfig", "", "")
	simulateCmd.Flags().Bool("verbose", false, "")
	simulateCmd.Flags().MarkHidden("debug")
	simulateCmd.Flags().MarkHidden("kubeconfig")
	simulateCmd.Flags().MarkHidden("verbose")
}
