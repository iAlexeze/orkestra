// Package e2e implements the orchestration loop for `ork e2e`.
// It runs a declarative end-to-end test defined in an E2E spec file:
// cluster creation → CRD apply → bundle generate+apply → Orkestra install →
// CR apply → expectation checking → cleanup → context restore.
package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/doctor"
	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"gopkg.in/yaml.v3"
)

const (
	defaultClusterName = "ork-e2e"
	defaultProvider    = "kind"
	defaultTimeout     = "60s"
)

// Runner executes a single E2E spec end-to-end.
type Runner struct {
	e2e         orktypes.E2E
	e2eDir      string // directory of the e2e.yaml file — resolves relative paths
	keepCluster bool
	clusterCtx  string // non-empty means use this context, skip cluster creation

	katalogFile string
	crFile      string
}

// New loads an E2E spec from a YAML file and constructs a Runner.
func New(e2eFile, clusterCtx string, keepCluster bool) (*Runner, error) {
	data, err := os.ReadFile(e2eFile)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", e2eFile, err)
	}

	var e2e orktypes.E2E
	if err := yaml.Unmarshal(data, &e2e); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", e2eFile, err)
	}
	if e2e.Kind != "E2E" {
		return nil, fmt.Errorf("%s: expected kind E2E, got %q", e2eFile, e2e.Kind)
	}

	r := &Runner{
		e2e:         e2e,
		e2eDir:      filepath.Dir(e2eFile),
		keepCluster: keepCluster,
		clusterCtx:  clusterCtx,
	}

	if err := r.resolveSource(); err != nil {
		return nil, err
	}

	return r, nil
}

// resolveSource resolves the katalog and CR file paths from the spec.
func (r *Runner) resolveSource() error {
	spec := r.e2e.Spec

	switch {
	case spec.Init != nil:
		// Example pack — resolve from examples/ relative to e2e file or cwd.
		base := r.e2eDir
		candidate := filepath.Join(base, "examples", spec.Init.Pack, spec.Init.Example)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			// Try from cwd
			cwd, _ := os.Getwd()
			candidate = filepath.Join(cwd, "examples", spec.Init.Pack, spec.Init.Example)
		}
		r.katalogFile = filepath.Join(candidate, "katalog.yaml")
		r.crFile = filepath.Join(candidate, "cr.yaml")

	case spec.Katalog != "" && spec.CR != "":
		r.katalogFile = r.abs(spec.Katalog)
		r.crFile = r.abs(spec.CR)

	default:
		return fmt.Errorf("e2e spec must declare either (katalog + cr) or init")
	}

	if _, err := os.Stat(r.katalogFile); err != nil {
		return fmt.Errorf("katalog file not found: %s", r.katalogFile)
	}
	if _, err := os.Stat(r.crFile); err != nil {
		return fmt.Errorf("CR file not found: %s", r.crFile)
	}
	return nil
}

// Run executes the full E2E test pipeline.
func (r *Runner) Run(ctx context.Context) error {
	name := r.e2e.Metadata.Name
	if desc := r.e2e.Metadata.Description; desc != "" {
		fmt.Printf("\nRunning E2E: %s — %s\n\n", name, desc)
	} else {
		fmt.Printf("\nRunning E2E: %s\n\n", name)
	}

	// Capture the original kubectl context so we can restore it after the
	// test completes (the cluster creation or --cluster flag switches context).
	if origCtx, err := currentKubectlContext(); err == nil && origCtx != "" {
		defer func() {
			if out, err := exec.Command("kubectl", "config", "use-context", origCtx).CombinedOutput(); err != nil {
				fmt.Printf("  ! Could not restore kubectl context to %q: %v\n%s\n", origCtx, err, out)
			} else {
				fmt.Printf("\n→ kubectl context restored to %q\n", origCtx)
			}
		}()
	}

	// ── 1. Cluster ───────────────────────────────────────────────────────
	if err := r.ensureCluster(ctx); err != nil {
		return fmt.Errorf("cluster: %w", err)
	}

	// ── 2. Dependencies ──────────────────────────────────────────────────
	fmt.Println("→ Ensuring dependencies...")
	if err := doctor.EnsureDependencies(); err != nil {
		return fmt.Errorf("dependencies: %w", err)
	}

	// ── 3. Apply operator CRD ────────────────────────────────────────────
	if err := r.applyCRD(ctx); err != nil {
		return fmt.Errorf("applying CRD: %w", err)
	}

	// ── 4. Generate and apply bundle ─────────────────────────────────────
	bundleFile, err := r.generateBundle(ctx)
	if err != nil {
		return fmt.Errorf("generate bundle: %w", err)
	}
	defer os.Remove(bundleFile)

	fmt.Printf("→ Applying bundle...\n")
	if out, err := kubectl(ctx, "apply", "-f", bundleFile); err != nil {
		return fmt.Errorf("apply bundle: %w\n%s", err, out)
	}
	fmt.Printf("  ✓ Bundle applied\n")

	// ── 5. Setup — namespaces, secrets, extra CRDs, other dependencies ──
	if err := r.applySetup(ctx); err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	// ── 6. Install Orkestra ──────────────────────────────────────────────
	if !doctor.OrkestraInstalled() {
		fmt.Printf("→ Installing Orkestra...\n")
		if err := doctor.InstallOrUpgradeOrkestra("", nil, false); err != nil {
			return fmt.Errorf("helm install: %w", err)
		}
		fmt.Printf("  ✓ Orkestra installed\n")
	} else {
		fmt.Printf("  ✓ Orkestra already installed\n")
	}

	// ── 7. Wait for Orkestra ready ───────────────────────────────────────
	fmt.Printf("→ Waiting for Orkestra to be ready...\n")
	status := doctor.CheckRuntimeHealth()
	if !status.Running {
		return fmt.Errorf("Orkestra not ready: %s", status.Reason)
	}
	fmt.Printf("  ✓ Orkestra runtime ready\n\n")

	// ── 7. Run expectations ──────────────────────────────────────────────
	type result struct {
		name    string
		elapsed time.Duration
		err     error
	}
	var results []result

	crApplied := false
	crDeleted := false

	for _, exp := range r.e2e.Spec.Expect {
		switch exp.After {
		case "cr-applied":
			if !crApplied {
				fmt.Printf("→ Applying CR...\n")
				if out, err := kubectl(ctx, "apply", "-f", r.crFile); err != nil {
					return fmt.Errorf("apply CR: %w\n%s", err, out)
				}
				fmt.Printf("  ✓ CR applied\n\n")
				crApplied = true
			}

		case "cr-deleted":
			if !crDeleted {
				fmt.Printf("→ Deleting CR...\n")
				if out, err := kubectl(ctx, "delete", "-f", r.crFile, "--ignore-not-found"); err != nil {
					return fmt.Errorf("delete CR: %w\n%s", err, out)
				}
				fmt.Printf("  ✓ CR deleted\n\n")
				crDeleted = true
			}

		default:
			return fmt.Errorf("unknown after: %q (must be cr-applied or cr-deleted)", exp.After)
		}

		to := exp.Timeout
		if to == "" {
			to = defaultTimeout
		}
		fmt.Printf("  Waiting for %q (timeout: %s)...\n", exp.Name, to)
		start := time.Now()
		verifyErr := verifyExpectation(ctx, exp, r.e2eDir)
		elapsed := time.Since(start)

		results = append(results, result{name: exp.Name, elapsed: elapsed, err: verifyErr})
		if verifyErr != nil {
			fmt.Printf("  ✗ %s (%s): %v\n", exp.Name, elapsed.Round(time.Millisecond), verifyErr)
		} else {
			fmt.Printf("  ✓ %s (%s)\n", exp.Name, elapsed.Round(time.Millisecond))
		}
	}

	// ── 8. Report ────────────────────────────────────────────────────────
	fmt.Printf("\nE2E Results: %s\n\n", name)
	passed := 0
	for _, res := range results {
		if res.err == nil {
			fmt.Printf("  ✓ %-40s (%s)\n", res.name, res.elapsed.Round(time.Millisecond))
			passed++
		} else {
			fmt.Printf("  ✗ %-40s (%s)\n", res.name, res.elapsed.Round(time.Millisecond))
		}
	}
	clusterInfo := r.clusterName()
	fmt.Printf("\n  %d of %d passed\n", passed, len(results))
	if clusterInfo != "" {
		fmt.Printf("  Cluster: %s (%s)\n", clusterInfo, r.provider())
	}

	// ── 9. Cleanup ───────────────────────────────────────────────────────
	if !r.keepCluster && r.clusterCtx == "" {
		fmt.Printf("\n→ Deleting cluster '%s'...\n", r.clusterName())
		if err := r.deleteCluster(ctx); err != nil {
			fmt.Printf("  ! Could not delete cluster: %v\n", err)
		} else {
			fmt.Printf("  ✓ Cluster deleted\n")
		}
	}

	// Return error if any expectation failed
	for _, res := range results {
		if res.err != nil {
			return fmt.Errorf("%d of %d expectations failed", len(results)-passed, len(results))
		}
	}
	return nil
}

// ensureCluster sets up the cluster according to the spec.
func (r *Runner) ensureCluster(ctx context.Context) error {
	if r.clusterCtx != "" {
		fmt.Printf("→ Using existing cluster context %q...\n", r.clusterCtx)
		if out, err := exec.CommandContext(ctx, "kubectl", "config", "use-context", r.clusterCtx).CombinedOutput(); err != nil {
			return fmt.Errorf("switching context: %w\n%s", err, out)
		}
		fmt.Printf("  ✓ Using context %s\n", r.clusterCtx)
		return nil
	}

	provider := r.provider()
	if provider != "kind" {
		return fmt.Errorf("provider %q not supported — only 'kind' is available", provider)
	}

	name := r.clusterName()
	spec := r.e2e.Spec.Cluster

	if !spec.Reuse {
		// Delete if already exists for a clean state
		if clusterExists(name) {
			fmt.Printf("→ Recreating cluster '%s' (reuse: false)...\n", name)
			if err := deleteKindCluster(ctx, name); err != nil {
				return fmt.Errorf("deleting old cluster: %w", err)
			}
		}
	}

	fmt.Printf("→ Ensuring cluster '%s'...\n", name)
	return doctor.EnsureKindCluster(name)
}

// applyCRD applies the operator's CRD to the cluster.
// Uses spec.crd if declared; falls back to crdFile entries in the katalog.
func (r *Runner) applyCRD(ctx context.Context) error {
	if crd := r.e2e.Spec.CRD; crd != "" {
		path := r.abs(crd)
		fmt.Printf("→ Applying CRD from %s...\n", crd)
		if out, err := kubectl(ctx, "apply", "-f", path); err != nil {
			return fmt.Errorf("applying CRD %s: %w\n%s", crd, err, out)
		}
		fmt.Printf("  ✓ CRD applied\n")
		return nil
	}

	// Fallback: read crdFile references from the katalog.
	var raw struct {
		Spec struct {
			CRDs map[string]struct {
				CRDFile string `yaml:"crdFile"`
			} `yaml:"crds"`
		} `yaml:"spec"`
	}
	data, err := os.ReadFile(r.katalogFile)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}
	katalogDir := filepath.Dir(r.katalogFile)
	for name, entry := range raw.Spec.CRDs {
		if entry.CRDFile == "" {
			continue
		}
		path := entry.CRDFile
		if !filepath.IsAbs(path) && !strings.HasPrefix(path, "http") {
			path = filepath.Join(katalogDir, path)
		}
		fmt.Printf("→ Applying CRD '%s' from %s...\n", name, entry.CRDFile)
		if out, err := kubectl(ctx, "apply", "-f", path); err != nil {
			fmt.Printf("  ! CRD apply failed (continuing): %v\n%s\n", err, out)
		} else {
			fmt.Printf("  ✓ CRD applied\n")
		}
	}
	return nil
}

func (r *Runner) generateBundle(ctx context.Context) (string, error) {
	// Resolve any crdFile references to inline apiTypes before bundling.
	// The Orkestra runtime runs inside a container and cannot read local files —
	// all type information must be embedded in the ConfigMap.
	resolved, err := katalog.ResolveCRDFiles(r.katalogFile)
	if err != nil {
		return "", fmt.Errorf("resolving crdFile references: %w", err)
	}

	resolvedKatalog, err := os.CreateTemp("", "ork-e2e-katalog-*.yaml")
	if err != nil {
		return "", err
	}
	if _, err := resolvedKatalog.Write(resolved); err != nil {
		resolvedKatalog.Close()
		os.Remove(resolvedKatalog.Name())
		return "", err
	}
	resolvedKatalog.Close()
	defer os.Remove(resolvedKatalog.Name())

	bundleFile, err := os.CreateTemp("", "ork-e2e-bundle-*.yaml")
	if err != nil {
		return "", err
	}
	bundleFile.Close()

	fmt.Printf("→ Generating bundle from %s...\n", r.katalogFile)
	orkBin, err := os.Executable()
	if err != nil {
		orkBin = "ork"
	}
	cmd := exec.CommandContext(ctx, orkBin, "generate", "bundle",
		"-f", resolvedKatalog.Name(),
		"-o", bundleFile.Name(),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Remove(bundleFile.Name())
		return "", fmt.Errorf("ork generate bundle: %w", err)
	}
	fmt.Printf("  ✓ Bundle generated\n")
	return bundleFile.Name(), nil
}

func (r *Runner) applySetup(ctx context.Context) error {
	for _, path := range r.e2e.Spec.Setup {
		abs := r.abs(path)
		fmt.Printf("→ Applying setup file %s...\n", path)
		if out, err := kubectl(ctx, "apply", "-f", abs); err != nil {
			return fmt.Errorf("applying setup %s: %w\n%s", path, err, out)
		}
		fmt.Printf("  ✓ Applied\n")
	}
	return nil
}

func (r *Runner) deleteCluster(ctx context.Context) error {
	return deleteKindCluster(ctx, r.clusterName())
}

func (r *Runner) clusterName() string {
	if r.e2e.Spec.Cluster.Name != "" {
		return r.e2e.Spec.Cluster.Name
	}
	return defaultClusterName
}

func (r *Runner) provider() string {
	if r.e2e.Spec.Cluster.Provider != "" {
		return r.e2e.Spec.Cluster.Provider
	}
	return defaultProvider
}

func (r *Runner) abs(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(r.e2eDir, path)
}

func currentKubectlContext() (string, error) {
	out, err := exec.Command("kubectl", "config", "current-context").Output()
	return strings.TrimSpace(string(out)), err
}

// kubectl runs a kubectl command and returns combined output.
func kubectl(ctx context.Context, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, "kubectl", args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func clusterExists(name string) bool {
	out, _ := exec.Command("kind", "get", "clusters").Output()
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) == name {
			return true
		}
	}
	return false
}

func deleteKindCluster(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "kind", "delete", "cluster", "--name", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
