//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orkspace/orkestra/pkg/e2e"
	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
)

var e2eCmd = &cobra.Command{
	Use:   "e2e",
	Short: "Run declarative end-to-end tests against a real cluster",
	Long: `Runs an E2E test defined in a YAML spec file.

Orchestrates the full lifecycle: cluster creation → dependency installation →
CRD apply → bundle apply → Orkestra install → CR apply → expectation checking → cleanup.

The same command runs locally and in CI. The e2e.yaml file is the source of truth.

  ork e2e
  ork e2e -f e2e.yaml
  ork e2e -f e2e.yaml --keep-cluster
  ork e2e -f e2e.yaml --cluster my-existing-context
  ork e2e -f e2e.yaml --version v1.2.3 --values values.yaml

Discovery mode — runs all *e2e.yaml files found recursively (skips pure aggregators):

  ork e2e ./...
  ork e2e ./examples/beginner/...
  ork e2e ./... --wait 2s
  ork e2e ./... --skip vendor,testdata,external/07-vault`,
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")
		keepCluster, _ := cmd.Flags().GetBool("keep-cluster")
		useCurrentCtx, _ := cmd.Flags().GetBool("use-current")
		clusterCtx, _ := cmd.Flags().GetString("cluster")
		version, _ := cmd.Flags().GetString("version")
		valuesFiles, _ := cmd.Flags().GetStringSlice("values")
		helmArgRaw, _ := cmd.Flags().GetStringSlice("helm-arg")
		var helmArgs []string
		for _, arg := range helmArgRaw {
			helmArgs = append(helmArgs, "--set", arg)
		}
		devServer, _ := cmd.Flags().GetBool("dev-server")
		wait, _ := cmd.Flags().GetString("wait")
		skipRaw, _ := cmd.Flags().GetStringSlice("skip")

		// Discovery mode: -f ./... or -f ./some/path/...
		if strings.HasSuffix(file, "/...") || file == "./..." || file == "..." {
			root := strings.TrimSuffix(file, "/...")
			if root == "." || root == "" {
				root = "."
			}
			return runDiscovery(cmd, root, wait, skipRaw, clusterCtx, useCurrentCtx, keepCluster, devServer, version, valuesFiles, helmArgs)
		}

		runner, err := e2e.New(file, clusterCtx, useCurrentCtx, keepCluster, devServer, version, valuesFiles, helmArgs...)
		if err != nil {
			return err
		}
		_, err = runner.Run(cmd.Context())
		return err
	},
}

// runDiscovery finds all *e2e.yaml leaf files under root, builds a temp
// aggregator, and runs it as a normal suite.
func runDiscovery(cmd *cobra.Command, root, wait string, skip []string, clusterCtx string, useCurrentCtx, keepCluster, devServer bool, version string, valuesFiles, helmArgs []string) error {
	// Expand comma-separated skip entries
	var patterns []string
	for _, s := range skip {
		for _, p := range strings.Split(s, ",") {
			if p = strings.TrimSpace(p); p != "" {
				patterns = append(patterns, p)
			}
		}
	}

	paths, err := e2e.DiscoverE2EFiles(root, patterns)
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}
	if len(paths) == 0 {
		fmt.Printf("No e2e files found under %s\n", root)
		return nil
	}
	absRoot, _ := filepath.Abs(root)
	fmt.Printf("→ Discovered %d e2e file(s) under %s\n", len(paths), root)
	for _, p := range paths {
		rel, _ := filepath.Rel(absRoot, p)
		fmt.Printf("    %s\n", rel)
	}
	fmt.Println()

	suite := e2e.BuildDiscoveryE2E(paths, wait)

	// Write to a temp file so the runner resolves relative paths correctly.
	tmp, err := os.CreateTemp("", "ork-e2e-discovery-*.yaml")
	if err != nil {
		return fmt.Errorf("creating temp suite file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if err := yaml.NewEncoder(tmp).Encode(suite); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp suite: %w", err)
	}
	tmp.Close()

	runner, err := e2e.New(tmp.Name(), clusterCtx, useCurrentCtx, keepCluster, devServer, version, valuesFiles, helmArgs...)
	if err != nil {
		return err
	}
	_, err = runner.Run(cmd.Context())
	return err
}

func init() {
	rootCmd.AddCommand(e2eCmd)

	e2eCmd.Flags().StringP("file", "f", "e2e.yaml", "Path to the E2E spec file, or ./... for discovery")
	e2eCmd.Flags().Bool("keep-cluster", false, "Keep the kind cluster after the test completes")
	e2eCmd.Flags().Bool("use-current", false, "Use the current kubectl context, skip cluster creation")
	e2eCmd.Flags().String("cluster", "", "Use an existing kubectl context instead of creating a cluster")
	e2eCmd.Flags().String("version", "", "Orkestra version to install (e.g., v1.2.3)")
	e2eCmd.Flags().StringSlice("values", []string{}, "Helm values files to pass to Orkestra installation")
	e2eCmd.Flags().StringSlice("helm-arg", []string{}, "Additional Helm --set arguments (e.g., key=value)")
	e2eCmd.Flags().Bool("dev-server", false, "Deploy the mock dev server into the cluster for external: examples")
	e2eCmd.Flags().String("wait", "", "Duration to wait between discovered tests (e.g. 2s). Only applies in ./... discovery mode.")
	e2eCmd.Flags().StringSlice("skip", []string{}, "Comma-separated path patterns to skip during ./... discovery (e.g. vendor,testdata)")

	// Shadow global flags
	e2eCmd.Flags().Bool("debug", false, "")
	e2eCmd.Flags().String("kubeconfig", "", "")
	e2eCmd.Flags().Bool("verbose", false, "")
	e2eCmd.Flags().MarkHidden("debug")
	e2eCmd.Flags().MarkHidden("kubeconfig")
	e2eCmd.Flags().MarkHidden("verbose")
}
