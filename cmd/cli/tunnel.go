//go:build !runtime

package cli

import (
	"fmt"
	"os"

	"github.com/orkspace/orkestra/pkg/tunnel"
	"github.com/spf13/cobra"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Manage the Orkestra tunnel daemon",
	Long: `Manage the public HTTPS tunnel started by ork deploy --expose.

  ork tunnel status    Show URL, provider, and uptime
  ork tunnel stop      Stop the tunnel daemon
  ork tunnel restart   Stop and start a fresh tunnel (new URL)`,
}

var tunnelStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the current tunnel URL and status",
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := tunnel.LoadState()
		if err != nil {
			return fmt.Errorf("reading tunnel state: %w", err)
		}
		if state == nil {
			fmt.Println("  No tunnel running")
			fmt.Println("  Start one with: ork deploy --expose")
			return nil
		}
		if !state.IsAlive() {
			fmt.Println("  Tunnel process is no longer running (stale state)")
			fmt.Println("  Start a fresh tunnel with: ork deploy --expose")
			_ = tunnel.RemoveState()
			return nil
		}

		fmt.Printf("\n  Provider:   %s\n", state.Provider)
		fmt.Printf("  URL:        %s\n", state.URL)
		fmt.Printf("  Local:      http://localhost:%d\n", state.LocalPort)
		fmt.Printf("  Uptime:     %s\n", state.Uptime())
		fmt.Printf("  Status:     running\n\n")
		return nil
	},
}

var tunnelStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the tunnel daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := tunnel.LoadState()
		if err != nil {
			return fmt.Errorf("reading tunnel state: %w", err)
		}
		if state == nil {
			fmt.Println("  No tunnel running")
			return nil
		}
		if err := state.Stop(); err != nil {
			return fmt.Errorf("stopping tunnel: %w", err)
		}
		fmt.Println("  ✓ Tunnel stopped")
		return nil
	},
}

var tunnelRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Stop the current tunnel and start a fresh one (new URL)",
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		token, _ := cmd.Flags().GetString("token")

		// Stop existing tunnel if any
		if state, err := tunnel.LoadState(); err == nil && state != nil {
			_ = state.Stop()
			fmt.Println("  ✓ Stopped previous tunnel")
		}

		p, err := selectTunnelProvider(provider)
		if err != nil {
			return err
		}

		if token != "" {
			if err := p.Authenticate(cmd.Context(), token); err != nil {
				return err
			}
		}

		fmt.Printf("  → Starting %s tunnel...\n", p.Name())
		url, pid, err := p.Start(cmd.Context(), 80)
		if err != nil {
			return fmt.Errorf("starting tunnel: %w", err)
		}

		state := tunnel.State{
			Provider:  p.Name(),
			PID:       pid,
			URL:       url,
			LocalPort: 80,
		}
		if err := tunnel.SaveState(state); err != nil {
			fmt.Fprintf(os.Stderr, "  ~ could not save tunnel state: %v\n", err)
		}

		fmt.Printf("  ✓ New URL: %s\n", url)
		return nil
	},
}

// selectTunnelProvider returns a Provider by name or auto-selects.
func selectTunnelProvider(name string) (tunnel.Provider, error) {
	if name != "" {
		return tunnel.SelectByName(name)
	}
	return tunnel.Select()
}

func init() {
	tunnelRestartCmd.Flags().String("provider", "", "Tunnel provider: cloudflared (default) or ngrok")
	tunnelRestartCmd.Flags().String("token", "", "Auth token for ngrok")

	tunnelCmd.AddCommand(tunnelStatusCmd)
	tunnelCmd.AddCommand(tunnelStopCmd)
	tunnelCmd.AddCommand(tunnelRestartCmd)
	rootCmd.AddCommand(tunnelCmd)
}
