package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/utils"
)

var (
	kfg *konfig.Konfig
	ctx context.Context
)

var rootCmd = &cobra.Command{
	Use:   "ork",
	Short: "Orkestra — The Universal CRD Runtime",
	Long: fmt.Sprintf(`
%s
Orkestra — The Universal CRD Runtime
Kompose. Konduct. OrKestrate.
`, utils.OrkestraLogoCLI),
}

func Execute(k *konfig.Konfig, c context.Context) {
	kfg = k
	ctx = c

	if err := rootCmd.Execute(); err != nil {
		utils.Exit(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags — highest‑priority overrides for config resolution
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().String("kubeconfig", "", "Path to kubeconfig file")
	rootCmd.PersistentFlags().StringSliceP("katalog", "k", nil, "Path(s) or URL(s) to crd-katalog.yaml (repeatable)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Show full context")

	// Mark katalog as required for all commands
	for _, cmd := range []*cobra.Command{
		validateCmd,
		templateCmd,
		generateRuntimeCmd,
		generateDocsCmd,
		generateDashboardsCmd,
		generateExamplesCmd,
		generateTestsCmd,
		generateAllCmd,
	} {
		cobra.MarkFlagRequired(cmd.Flags(), "katalog")
	}
}

func initConfig() {
	// Resolve log level (flag > env > default) and initialize logger
	level := resolveLogLevel(rootCmd)
	logger.Init(level)

	// Resolve kubeconfig path (flag > env > ~/.kube/config > in‑cluster)
	kubeconfig := resolveKubeconfig(rootCmd)

	// Persist resolved values into global Konfig
	if kfg != nil {
		kfg.Cluster().KubekonfigPath = kubeconfig
	}
}

// resolveKubeconfig determines which kubeconfig to use.
// Priority: CLI flag → $KUBECONFIG → ~/.kube/config → in‑cluster.
func resolveKubeconfig(cmd *cobra.Command) string {
	if flagVal, _ := cmd.Flags().GetString("kubeconfig"); flagVal != "" {
		return flagVal
	}
	if envVal := os.Getenv("KUBECONFIG"); envVal != "" {
		return envVal
	}
	if home, err := os.UserHomeDir(); err == nil {
		defaultPath := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(defaultPath); err == nil {
			return defaultPath
		}
	}
	return ""
}

// resolveLogLevel determines the effective log level.
// Priority: --debug flag → LOG_LEVEL env → "info".
func resolveLogLevel(cmd *cobra.Command) string {
	debug, _ := cmd.Flags().GetBool("debug")
	if debug {
		return "debug"
	}
	if env := os.Getenv("LOG_LEVEL"); env != "" {
		return env
	}
	return "info"
}
