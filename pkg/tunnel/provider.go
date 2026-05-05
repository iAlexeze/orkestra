// pkg/tunnel/provider.go
//
// Provider interface and auto-selection logic for tunnel providers.
// Cloudflared is the default (no account required); ngrok is the fallback.
package tunnel

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Provider is implemented by every supported tunnel backend.
type Provider interface {
	// Name returns the provider identifier used in state files and output.
	Name() string

	// Available reports whether the provider binary is in PATH or ~/.orkestra/bin.
	Available() bool

	// Install downloads and installs the provider binary when not available.
	Install(ctx context.Context) error

	// Authenticate stores credentials (no-op for cloudflared).
	Authenticate(ctx context.Context, token string) error

	// Start launches a background tunnel to localPort and returns the public URL
	// once it is available. The daemon outlives the process that called Start.
	Start(ctx context.Context, localPort int) (url string, pid int, err error)

	// Stop terminates the daemon identified by pid.
	Stop(pid int) error
}

// Select returns the first available provider in priority order.
// cloudflared comes first (no account). ngrok is second.
func Select() (Provider, error) {
	providers := []Provider{
		&CloudflaredProvider{},
		&NgrokProvider{},
	}

	// Prefer one that is already installed
	for _, p := range providers {
		if p.Available() {
			return p, nil
		}
	}

	// Fall back to cloudflared — it can install itself
	return &CloudflaredProvider{}, nil
}

// SelectByName returns a provider by explicit name.
func SelectByName(name string) (Provider, error) {
	switch name {
	case "cloudflared", "cloudflare":
		return &CloudflaredProvider{}, nil
	case "ngrok":
		return &NgrokProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown tunnel provider %q — use cloudflared or ngrok", name)
	}
}

// orkestaBinDir returns ~/.orkestra/bin — the shared directory for tools
// downloaded by Orkestra (kind, cloudflared, etc.).
func orkestaBinDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orkestra", "bin")
}

// binaryInPath returns the full path to a binary if it exists in PATH,
// empty string otherwise.
func binaryInPath(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}
