// Package e2e implements the orchestration loop for `ork e2e`.
//
// A Runner executes a declarative E2E spec through its full lifecycle:
//
//  1. Cluster provisioning (kind) — skipped when --use-current or --cluster is set
//  2. CRD apply
//  3. Optional setup manifests
//  4. Bundle generate + apply
//  5. Orkestra helm install
//  6. CR apply
//  7. Expectation polling
//  8. Teardown — always runs for non-owned clusters (--use-current, --cluster);
//     for owned clusters only when --keep-cluster is absent
//
// Teardown reverses every applied resource in the correct order:
// CR delete → helm uninstall → bundle delete → setup (reverse) → CRDs.
// This keeps borrowed clusters clean regardless of pass/fail.
//
// Run returns a *Result with per-case timings that callers (e.g. registry push)
// embed as OCI annotations.
package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/motif"
	"github.com/orkspace/orkestra/pkg/ork"
	"github.com/orkspace/orkestra/pkg/registry"
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
	e2e           orktypes.E2E
	e2eDir        string // directory of the e2e.yaml file — resolves relative paths
	keepCluster   bool
	useCurrentCtx bool   // Default (false) - means whether to use the current context, skip cluster creation
	clusterCtx    string // non-empty means use this context, skip cluster creation

	katalogFile string
	crFile      string
}

// New loads an E2E spec from a YAML file and constructs a Runner.
func New(e2eFile, clusterCtx string, useCurrentCtx, keepCluster bool) (*Runner, error) {
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
		e2e:           e2e,
		e2eDir:        filepath.Dir(e2eFile),
		keepCluster:   keepCluster,
		clusterCtx:    clusterCtx,
		useCurrentCtx: useCurrentCtx,
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

// Run executes the full E2E test pipeline and returns a structured Result.
func (r *Runner) Run(ctx context.Context) (*Result, error) {
	name := r.e2e.Metadata.Name
	start := time.Now()
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

	// When e2e runs against an existing cluster (--cluster flag, --current-context or --keep-cluster),
	// it owns the resources it applied but not the cluster itself. Track everything
	// applied so it can be torn down in reverse order when the test completes.
	//
	// When e2e creates and deletes its own ephemeral cluster, teardown is handled
	// by the cluster deletion — no per-resource cleanup is needed.
	ownsCluster := r.clusterCtx == "" && !r.keepCluster && !r.useCurrentCtx
	var (
		appliedCRDPaths   []string
		appliedBundlePath string
		appliedSetupPaths []string
		installedOrkestra bool
	)
	if !ownsCluster {
		defer func() {
			r.teardown(context.Background(),
				appliedCRDPaths, appliedBundlePath, appliedSetupPaths, installedOrkestra)
		}()
	}

	// ── 1. Cluster ───────────────────────────────────────────────────────
	if err := r.ensureCluster(ctx); err != nil {
		return nil, fmt.Errorf("cluster: %w", err)
	}

	// ── 2. Dependencies ──────────────────────────────────────────────────
	fmt.Println("→ Ensuring dependencies...")
	if err := ork.EnsureDependencies(); err != nil {
		return nil, fmt.Errorf("dependencies: %w", err)
	}

	// ── 3. Apply operator CRD ────────────────────────────────────────────
	crdPaths, err := r.applyCRD(ctx)
	if err != nil {
		return nil, fmt.Errorf("applying CRD: %w", err)
	}
	appliedCRDPaths = crdPaths

	// ── 4. Pre-pull OCI imports so bundle generation works without credentials ──
	if err := r.pullOCIImports(ctx); err != nil {
		return nil, fmt.Errorf("pulling OCI imports: %w", err)
	}

	// ── 5. Generate and apply bundle ─────────────────────────────────────
	bundleFile, err := r.generateBundle(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate bundle: %w", err)
	}
	// Bundle temp file is removed after teardown uses it (or immediately if
	// cluster is ephemeral and teardown won't need it).
	if ownsCluster {
		defer os.Remove(bundleFile)
	} else {
		appliedBundlePath = bundleFile
	}

	fmt.Printf("→ Applying bundle...\n")
	if out, err := kubectl(ctx, "apply", "-f", bundleFile); err != nil {
		return nil, fmt.Errorf("apply bundle: %w\n%s", err, out)
	}
	fmt.Printf("  ✓ Bundle applied\n")

	// ── 5. Setup — namespaces, secrets, extra CRDs, other dependencies ──
	setupPaths, err := r.applySetup(ctx)
	if err != nil {
		return nil, fmt.Errorf("setup: %w", err)
	}
	appliedSetupPaths = setupPaths

	// ── 6. Install Orkestra ──────────────────────────────────────────────
	args := []string{}
	text := "..."

	gatewayEnabled, err := resolveGatewayEnabled(r.katalogFile)
	if err != nil {
		return nil, err
	}
	if gatewayEnabled {
		args = append(args, "--set", "gateway.enabled=true")
		text = " with gateway..."
	}

	if !ork.OrkestraInstalled() {
		fmt.Printf("→ Installing Orkestra%s\n", text)
		if err := ork.InstallOrUpgradeOrkestra("", nil, args...); err != nil {
			return nil, fmt.Errorf("helm install: %w", err)
		}
		installedOrkestra = true
		fmt.Printf("  ✓ Orkestra installed\n")
	} else if ork.RuntimeDeployed() {
		// Orkestra is installed and the runtime deployment exists — the bundle
		// applied above updated the orkestra-katalog ConfigMap, so the runtime
		// must reload to pick up the new Katalog.
		fmt.Printf("→ Updating Orkestra with current bundle...\n")
		if err := ork.SyncRuntime(); err != nil {
			return nil, fmt.Errorf("syncing Orkestra runtime: %w", err)
		}
		fmt.Printf("  ✓ Orkestra updated\n")
	} else {
		fmt.Printf("  ✓ Orkestra already installed\n")
	}

	// ── 7. Wait for Orkestra ready ───────────────────────────────────────
	fmt.Printf("→ Waiting for Orkestra to be ready...\n")
	status := ork.CheckRuntimeHealth()
	if !status.Running {
		return nil, fmt.Errorf("Orkestra not ready: %s", status.Reason)
	}
	fmt.Printf("  ✓ Orkestra runtime ready\n\n")

	// ── 8. Run expectations ──────────────────────────────────────────────
	var cases []CaseResult
	crApplied := false
	crDeleted := false

	for _, exp := range r.e2e.Spec.Expect {
		switch exp.After {
		case "cr-applied":
			if !crApplied {
				fmt.Printf("→ Applying CR...\n")
				if out, err := kubectl(ctx, "apply", "-f", r.crFile); err != nil {
					return nil, fmt.Errorf("apply CR: %w\n%s", err, out)
				}
				fmt.Printf("  ✓ CR applied\n\n")
				crApplied = true
			}

		case "cr-deleted":
			if !crDeleted {
				fmt.Printf("→ Deleting CR...\n")
				if out, err := kubectl(ctx, "delete", "-f", r.crFile, "--ignore-not-found"); err != nil {
					return nil, fmt.Errorf("delete CR: %w\n%s", err, out)
				}
				fmt.Printf("  ✓ CR deleted\n\n")
				crDeleted = true
			}

		default:
			return nil, fmt.Errorf("unknown after: %q (must be cr-applied or cr-deleted)", exp.After)
		}

		to := exp.Timeout
		if to == "" {
			to = defaultTimeout
		}
		fmt.Printf("  Waiting for %q (timeout: %s)...\n", exp.Name, to)
		caseStart := time.Now()
		verifyErr := verifyExpectation(ctx, exp, r.e2eDir)
		caseElapsed := time.Since(caseStart)

		cases = append(cases, CaseResult{
			Name:    exp.Name,
			Passed:  verifyErr == nil,
			Elapsed: caseElapsed,
			Err:     verifyErr,
		})
		if verifyErr != nil {
			fmt.Printf("  ✗ %s (%s): %v\n", exp.Name, caseElapsed.Round(time.Millisecond), verifyErr)
		} else {
			fmt.Printf("  ✓ %s (%s)\n", exp.Name, caseElapsed.Round(time.Millisecond))
		}
	}

	result := &Result{
		Name:    name,
		Cases:   cases,
		Elapsed: time.Since(start),
	}

	// ── 9. Report ────────────────────────────────────────────────────────
	fmt.Printf("\nE2E Results: %s\n\n", name)
	for _, c := range cases {
		if c.Passed {
			fmt.Printf("  ✓ %-40s (%s)\n", c.Name, c.Elapsed.Round(time.Millisecond))
		} else {
			fmt.Printf("  ✗ %-40s (%s)\n", c.Name, c.Elapsed.Round(time.Millisecond))
		}
	}
	clusterInfo := r.clusterName()
	fmt.Printf("\n  %s\n", result.Summary())
	if clusterInfo != "" {
		fmt.Printf("  Cluster: %s (%s)\n", clusterInfo, r.provider())
	}

	// ── 10. Cleanup ──────────────────────────────────────────────────────
	if !r.useCurrentCtx && !r.keepCluster && r.clusterCtx == "" {
		fmt.Printf("\n→ Deleting cluster '%s'...\n", r.clusterName())
		if err := r.deleteCluster(ctx); err != nil {
			fmt.Printf("  ! Could not delete cluster: %v\n", err)
		} else {
			fmt.Printf("  ✓ Cluster deleted\n")
		}
	}

	if !result.AllPassed() {
		return result, fmt.Errorf("%d of %d expectations failed", result.Total()-result.Passed(), result.Total())
	}
	return result, nil
}

// resolveGatewayEnabled inspects the katalog file and returns true if the
// gateway block is present. This allows Helm installation to automatically
// enable the gateway chart when required by the katalog.
func resolveGatewayEnabled(katalogFile string) (bool, error) {
	var raw struct {
		Gateway *orktypes.GatewayConfig `yaml:"gateway,omitempty"`
	}

	data, err := os.ReadFile(katalogFile)
	if err != nil {
		return false, err
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false, err
	}
	return raw.Gateway != nil, nil
}

// ensureCluster sets up the cluster according to the spec.
func (r *Runner) ensureCluster(ctx context.Context) error {
	if r.useCurrentCtx {
		fmt.Printf("→ Using current cluster context...\n")
		return nil
	}

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
	return ork.EnsureKindCluster(name)
}

// applyCRD applies the operator's CRD to the cluster and returns the paths applied.
// Uses spec.crd if declared; falls back to crdFile entries in the katalog.
func (r *Runner) applyCRD(ctx context.Context) ([]string, error) {
	if crd := r.e2e.Spec.CRD; crd != "" {
		path := r.abs(crd)
		fmt.Printf("→ Applying CRD from %s...\n", crd)
		if out, err := kubectl(ctx, "apply", "-f", path); err != nil {
			return nil, fmt.Errorf("applying CRD %s: %w\n%s", crd, err, out)
		}
		fmt.Printf("  ✓ CRD applied\n")
		return []string{path}, nil
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
		return nil, err
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	katalogDir := filepath.Dir(r.katalogFile)
	var applied []string
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
			applied = append(applied, path)
		}
	}
	return applied, nil
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

// pullOCIImports pre-pulls all OCI motif and registry imports referenced in
// the katalog file so that bundle generation never needs to do OCI calls.
// Uses the same resolution logic as the merge path, including bare-name shorthands.
func (r *Runner) pullOCIImports(_ context.Context) error {
	imports, err := registry.ExtractOCIImports(r.katalogFile)
	if err != nil {
		return fmt.Errorf("extracting OCI imports from %s: %w", r.katalogFile, err)
	}
	if imports.Empty() {
		return nil
	}

	fmt.Printf("→ Pulling OCI imports...\n")

	for _, imp := range imports.MotifImports {
		fmt.Printf("  → motif %s\n", imp.Motif)
		if err := motif.PullImport(&imp); err != nil {
			return fmt.Errorf("pulling motif %q: %w", imp.Motif, err)
		}
		fmt.Printf("  ✓ %s\n", imp.Motif)
	}

	client, err := registry.NewClient()
	if err != nil {
		return fmt.Errorf("initializing registry client: %w", err)
	}
	_ = client // used for registry source pulls below when Komposer support is needed

	return nil
}

func (r *Runner) applySetup(ctx context.Context) ([]string, error) {
	var applied []string
	for _, path := range r.e2e.Spec.Setup {
		abs := r.abs(path)
		fmt.Printf("→ Applying setup file %s...\n", path)
		if out, err := kubectl(ctx, "apply", "-f", abs); err != nil {
			return applied, fmt.Errorf("applying setup %s: %w\n%s", path, err, out)
		}
		fmt.Printf("  ✓ Applied\n")
		applied = append(applied, abs)
	}
	return applied, nil
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

// teardown cleans up every resource applied to an existing cluster.
// Called via defer when e2e runs against --current-context, --cluster or --keep-cluster, where
// deleting the cluster itself is not an option.
// Teardown order is the reverse of apply order: CR → Orkestra → bundle → setup → CRDs.
// The CR is expected to already be deleted by the cr-deleted expectation block;
// this handles everything else.
func (r *Runner) teardown(ctx context.Context, crdPaths []string, bundlePath string, setupPaths []string, uninstallOrkestra bool) {
	fmt.Printf("\n→ Cleaning up resources...\n")

	// Helm uninstall orkestra — must happen before bundle delete so the
	// runtime is stopped before its RBAC and ConfigMap are removed.
	if uninstallOrkestra {
		fmt.Printf("  → Uninstalling Orkestra...\n")
		cmd := exec.CommandContext(ctx, "helm", "uninstall", ork.Orkestra,
			"--namespace", ork.OrkestraNamespace, "--ignore-not-found")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("  ! helm uninstall failed: %v\n", err)
		} else {
			fmt.Printf("  ✓ Orkestra uninstalled\n")
		}
	}

	// Bundle (RBAC, ConfigMap, Namespace created by ork generate bundle).
	if bundlePath != "" {
		fmt.Printf("  → Deleting bundle resources...\n")
		if out, err := kubectl(ctx, "delete", "-f", bundlePath, "--ignore-not-found"); err != nil {
			fmt.Printf("  ! bundle delete failed: %v\n%s\n", err, out)
		} else {
			fmt.Printf("  ✓ Bundle resources deleted\n")
		}
		os.Remove(bundlePath)
	}

	// Setup files in reverse order.
	for i := len(setupPaths) - 1; i >= 0; i-- {
		path := setupPaths[i]
		fmt.Printf("  → Deleting setup %s...\n", filepath.Base(path))
		if out, err := kubectl(ctx, "delete", "-f", path, "--ignore-not-found"); err != nil {
			fmt.Printf("  ! setup delete failed (%s): %v\n%s\n", path, err, out)
		}
	}

	// CRDs last — deleting a CRD cascades to all CRs of that type.
	for _, path := range crdPaths {
		fmt.Printf("  → Deleting CRD %s...\n", filepath.Base(path))
		if out, err := kubectl(ctx, "delete", "-f", path, "--ignore-not-found"); err != nil {
			fmt.Printf("  ! CRD delete failed (%s): %v\n%s\n", path, err, out)
		} else {
			fmt.Printf("  ✓ CRD deleted\n")
		}
	}

	fmt.Printf("  ✓ Cleanup complete\n")
}

func deleteKindCluster(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "kind", "delete", "cluster", "--name", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
