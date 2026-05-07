//go:build !runtime

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/orkspace/orkestra/pkg/doctor"
	"github.com/orkspace/orkestra/pkg/tunnel"
	"github.com/spf13/cobra"
)

// controlCenterTunnelName is the canonical key used in the tunnel state map.
const controlCenterTunnelName = "controlcenter"

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Manage Orkestra tunnel daemons",
	Long: `Manage the public HTTPS tunnels started by ork deploy --expose.

  ork tunnel expose <app|controlcenter>   Start a tunnel for an app or Control Center
  ork tunnel status                        Show all running tunnels
  ork tunnel stop [name]                   Stop all tunnels, or just the named one
  ork tunnel restart <name>                Stop and start a fresh tunnel (new URL)`,
}

var tunnelExposeCmd = &cobra.Command{
	Use:   "expose <name>",
	Short: "Start a public HTTPS tunnel for an app or the Control Center",
	Long: `Start a background cloudflared (or ngrok) tunnel and print the public URL.

  ork tunnel expose my-app           Expose a deployed app
  ork tunnel expose controlcenter    Expose the Orkestra Control Center

Note: exposing orkestra-runtime is not supported.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.ToLower(strings.TrimSpace(args[0]))
		provider, _ := cmd.Flags().GetString("provider")
		token, _ := cmd.Flags().GetString("token")

		if name == "orkestra-runtime" || name == "runtime" {
			fmt.Fprintln(os.Stderr, "  ✗ Exposing orkestra-runtime is not supported.")
			fmt.Fprintln(os.Stderr, "    Use: ork tunnel expose controlcenter")
			return fmt.Errorf("unsupported target")
		}

		var opts tunnel.ExposeOptions
		if name == controlCenterTunnelName || name == "cc" || name == "control-center" {
			opts = tunnel.ExposeOptions{
				Name:        controlCenterTunnelName,
				Provider:    provider,
				Token:       token,
				ServiceName: doctor.OrkestraControlCenter,
				Namespace:   doctor.OrkestraNamespace,
				ServicePort: doctor.OrkestraControlCenterPort,
				PortForward: true,
			}
		} else {
			ns, err := resolveAppNamespace(name)
			if err != nil {
				return err
			}
			opts = tunnel.ExposeOptions{
				Name:        name,
				Provider:    provider,
				Token:       token,
				ServiceName: name + "-orkestra-svc",
				ServicePort: "8080",
				Namespace:   ns,
			}
		}

		fmt.Printf("  → Starting tunnel for %s...\n", opts.Name)
		url, err := tunnel.Expose(cmd.Context(), opts)
		if err != nil {
			return fmt.Errorf("tunnel: %w", err)
		}
		fmt.Printf("  ✓ %s → %s\n", opts.Name, url)
		return nil
	},
}

var tunnelStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show all running tunnels",
	RunE: func(cmd *cobra.Command, args []string) error {
		states, err := tunnel.LoadAllStates()
		if err != nil {
			return fmt.Errorf("reading tunnel state: %w", err)
		}
		if len(states) == 0 {
			fmt.Println("  No tunnels running")
			fmt.Println("  Start one with: ork deploy --expose  or  ork tunnel expose <name>")
			return nil
		}

		fmt.Println()
		alive := 0
		for name, s := range states {
			s.Name = name
			if !s.IsAlive() {
				fmt.Printf("  %-20s  stale (process gone)\n", name)
				_ = tunnel.RemoveTunnelState(name)
				continue
			}
			alive++
			fmt.Printf("  %-20s  %s\n", name, s.URL)
			fmt.Printf("    Provider:  %s\n", s.Provider)
			fmt.Printf("    Local:     http://localhost:%d\n", s.LocalPort)
			fmt.Printf("    Uptime:    %s\n", s.Uptime())
		}
		if alive == 0 {
			fmt.Println("  All entries were stale — run ork tunnel expose <name> to start fresh")
		}
		fmt.Println()
		return nil
	},
}

var tunnelStopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop one or all tunnel daemons",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		states, err := tunnel.LoadAllStates()
		if err != nil {
			return fmt.Errorf("reading tunnel state: %w", err)
		}
		if len(states) == 0 {
			fmt.Println("  No tunnels running")
			return nil
		}

		if len(args) == 1 {
			name := args[0]
			s, ok := states[name]
			if !ok {
				return fmt.Errorf("no tunnel named %q", name)
			}
			s.Name = name
			if err := s.Stop(); err != nil {
				return fmt.Errorf("stopping tunnel: %w", err)
			}
			fmt.Printf("  ✓ Stopped %s\n", name)
			return nil
		}

		for name, s := range states {
			s.Name = name
			_ = s.Stop()
		}
		fmt.Println("  ✓ All tunnels stopped")
		return nil
	},
}

var tunnelRestartCmd = &cobra.Command{
	Use:   "restart <name>",
	Short: "Stop the named tunnel and start a fresh one (new URL)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.ToLower(args[0])
		provider, _ := cmd.Flags().GetString("provider")
		token, _ := cmd.Flags().GetString("token")

		states, _ := tunnel.LoadAllStates()
		if s, ok := states[name]; ok {
			s.Name = name
			_ = s.Stop()
			fmt.Printf("  ✓ Stopped %s\n", name)
		}

		fmt.Printf("  → Starting fresh tunnel for %s...\n", name)
		exposeArgs := []string{name}
		_ = tunnelExposeCmd.Flags().Set("provider", provider)
		_ = tunnelExposeCmd.Flags().Set("token", token)
		return tunnelExposeCmd.RunE(cmd, exposeArgs)
	},
}

// resolveAppNamespace looks up the namespace for an app from deploy state,
// falling back to the conventional <app>-orkestra-ns pattern.
func resolveAppNamespace(appName string) (string, error) {
	state, err := doctor.LoadState()
	if err == nil && state != nil {
		if p, ok := state.Projects[appName]; ok && p.Namespace != "" {
			return p.Namespace, nil
		}
	}
	return appName + "-orkestra-ns", nil
}

func init() {
	tunnelExposeCmd.Flags().String("provider", "", "Tunnel provider: cloudflared (default) or ngrok")
	tunnelExposeCmd.Flags().String("token", "", "Auth token for ngrok")

	tunnelRestartCmd.Flags().String("provider", "", "Tunnel provider: cloudflared (default) or ngrok")
	tunnelRestartCmd.Flags().String("token", "", "Auth token for ngrok")

	tunnelCmd.AddCommand(tunnelExposeCmd)
	tunnelCmd.AddCommand(tunnelStatusCmd)
	tunnelCmd.AddCommand(tunnelStopCmd)
	tunnelCmd.AddCommand(tunnelRestartCmd)
	rootCmd.AddCommand(tunnelCmd)

	// Shadow global flags so they don't appear under `ork init`
	tunnelCmd.Flags().Bool("debug", false, "")
	tunnelCmd.Flags().String("kubeconfig", "", "")
	tunnelCmd.Flags().StringSlice("katalog", nil, "")
	tunnelCmd.Flags().Bool("verbose", false, "")

	// Hide them from help output
	tunnelCmd.Flags().MarkHidden("debug")
	tunnelCmd.Flags().MarkHidden("kubeconfig")
	tunnelCmd.Flags().MarkHidden("katalog")
	tunnelCmd.Flags().MarkHidden("verbose")
}
