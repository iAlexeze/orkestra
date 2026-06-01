//go:build !runtime && !gateway

package cli

import (
	"github.com/orkspace/orkestra/pkg/e2e"
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
  ork e2e -f e2e.yaml --version v1.2.3 --values values.yaml`,
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

		runner, err := e2e.New(file, clusterCtx, useCurrentCtx, keepCluster, devServer, version, valuesFiles, helmArgs...)
		if err != nil {
			return err
		}
		_, err = runner.Run(cmd.Context())
		return err
	},
}

func init() {
	rootCmd.AddCommand(e2eCmd)

	e2eCmd.Flags().StringP("file", "f", "e2e.yaml", "Path to the E2E spec file")
	e2eCmd.Flags().Bool("keep-cluster", false, "Keep the kind cluster after the test completes")
	e2eCmd.Flags().Bool("use-current", false, "Use the current kubectl context, skip cluster creation")
	e2eCmd.Flags().String("cluster", "", "Use an existing kubectl context instead of creating a cluster")
	e2eCmd.Flags().String("version", "", "Orkestra version to install (e.g., v1.2.3)")
	e2eCmd.Flags().StringSlice("values", []string{}, "Helm values files to pass to Orkestra installation")
	e2eCmd.Flags().StringSlice("helm-arg", []string{}, "Additional Helm --set arguments (e.g., key=value)")
	e2eCmd.Flags().Bool("dev-server", false, "Deploy the mock dev server into the cluster for external: examples")

	// Shadow global flags
	e2eCmd.Flags().Bool("debug", false, "")
	e2eCmd.Flags().String("kubeconfig", "", "")
	e2eCmd.Flags().Bool("verbose", false, "")
	e2eCmd.Flags().MarkHidden("debug")
	e2eCmd.Flags().MarkHidden("kubeconfig")
	e2eCmd.Flags().MarkHidden("verbose")
}
