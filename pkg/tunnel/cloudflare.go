// pkg/tunnel/cloudflare.go
//
// Cloudflared provider — anonymous tunnel, no account required.
// Uses trycloudflare.com (Cloudflare Quick Tunnels).
package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	cloudflaredVersion = "2025.4.2"
)

// cloudflaredURL is matched against cloudflared output to find the tunnel URL.
var cloudflaredURL = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// CloudflaredProvider implements Provider for Cloudflare Quick Tunnels.
type CloudflaredProvider struct{}

func (c *CloudflaredProvider) Name() string { return "cloudflared" }

func (c *CloudflaredProvider) Available() bool {
	return cloudflaredBin() != ""
}

func (c *CloudflaredProvider) Install(ctx context.Context) error {
	dest := filepath.Join(orkestaBinDir(), "cloudflared")
	_, err := downloadCloudflared(dest)
	return err
}

// Authenticate is a no-op — trycloudflare.com requires no account or token.
func (c *CloudflaredProvider) Authenticate(_ context.Context, _ string) error {
	return nil
}

// Start launches cloudflared as a detached background process and returns
// the public URL once it appears in cloudflared's output.
func (c *CloudflaredProvider) Start(ctx context.Context, localPort int) (string, int, error) {
	bin, err := ensureCloudflared()
	if err != nil {
		return "", 0, err
	}

	localURL := fmt.Sprintf("http://localhost:%d", localPort)

	cmd := exec.Command(bin, "tunnel", "--url", localURL, "--no-autoupdate")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")

	// Cloudflared writes the URL to stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", 0, fmt.Errorf("cloudflared: pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", 0, fmt.Errorf("cloudflared: pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", 0, fmt.Errorf("cloudflared: start: %w", err)
	}

	// Read combined output looking for the URL
	urlCh := make(chan string, 1)
	go scanForURL(stderr, urlCh)
	go io.Copy(io.Discard, stdout)

	// Wait up to 30 seconds for the URL to appear
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	select {
	case url := <-urlCh:
		return url, cmd.Process.Pid, nil
	case <-timeout.C:
		cmd.Process.Kill()
		return "", 0, fmt.Errorf("cloudflared: timed out waiting for tunnel URL (30s)")
	case <-ctx.Done():
		cmd.Process.Kill()
		return "", 0, ctx.Err()
	}
}

func (c *CloudflaredProvider) Stop(pid int) error {
	if pid <= 0 {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	return proc.Kill()
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func cloudflaredBin() string {
	if p := binaryInPath("cloudflared"); p != "" {
		return p
	}
	p := filepath.Join(orkestaBinDir(), "cloudflared")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

func ensureCloudflared() (string, error) {
	if b := cloudflaredBin(); b != "" {
		return b, nil
	}
	dest := filepath.Join(orkestaBinDir(), "cloudflared")
	return downloadCloudflared(dest)
}

func downloadCloudflared(dest string) (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var filename string
	switch goos {
	case "linux":
		switch goarch {
		case "amd64":
			filename = "cloudflared-linux-amd64"
		case "arm64":
			filename = "cloudflared-linux-arm64"
		default:
			return "", fmt.Errorf("cloudflared: unsupported arch %s", goarch)
		}
	case "darwin":
		switch goarch {
		case "amd64":
			filename = "cloudflared-darwin-amd64.tgz"
		case "arm64":
			filename = "cloudflared-darwin-arm64.tgz"
		default:
			return "", fmt.Errorf("cloudflared: unsupported arch %s", goarch)
		}
	default:
		return "", fmt.Errorf("cloudflared: unsupported OS %s", goos)
	}

	url := fmt.Sprintf(
		"https://github.com/cloudflare/cloudflared/releases/download/%s/%s",
		cloudflaredVersion, filename,
	)

	fmt.Printf("  → Downloading cloudflared %s...\n", cloudflaredVersion)

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("cloudflared: mkdir: %w", err)
	}

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return "", fmt.Errorf("cloudflared: download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cloudflared: download returned %s", resp.Status)
	}

	// macOS releases are .tgz — extract the binary
	if strings.HasSuffix(filename, ".tgz") {
		return extractCloudflaredTGZ(resp.Body, dest)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", fmt.Errorf("cloudflared: write: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(dest)
		return "", fmt.Errorf("cloudflared: write: %w", err)
	}
	f.Close()

	fmt.Printf("  ✓ cloudflared %s installed (%s)\n", cloudflaredVersion, dest)
	return dest, nil
}

func extractCloudflaredTGZ(r io.Reader, dest string) (string, error) {
	// Write tgz to a temp file, then extract
	tmp, err := os.CreateTemp("", "cloudflared-*.tgz")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return "", err
	}
	tmp.Close()

	// Extract using tar
	cmd := exec.Command("tar", "-xzf", tmp.Name(), "-C", filepath.Dir(dest), "cloudflared")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("cloudflared: extract: %w", err)
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		return "", err
	}
	fmt.Printf("  ✓ cloudflared %s installed (%s)\n", cloudflaredVersion, dest)
	return dest, nil
}

func scanForURL(r io.Reader, urlCh chan<- string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if m := cloudflaredURL.FindString(line); m != "" {
			urlCh <- m
			// Drain the rest so the process doesn't block on pipe writes
			go io.Copy(io.Discard, r)
			return
		}
	}
}

// orkestaBinDir is defined in kind.go — reused here.
