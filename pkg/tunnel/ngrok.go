// pkg/tunnel/ngrok.go
//
// ngrok provider — free-tier tunnels, requires an account and auth token.
package tunnel

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// NgrokProvider implements Provider for ngrok.
type NgrokProvider struct{}

func (n *NgrokProvider) Name() string { return "ngrok" }

func (n *NgrokProvider) Available() bool {
	return binaryInPath("ngrok") != ""
}

// Install prints guidance — ngrok is not auto-downloaded because it requires
// an account and the installer is platform-specific.
func (n *NgrokProvider) Install(_ context.Context) error {
	return fmt.Errorf(
		"ngrok is not installed\n" +
			"  Install: https://ngrok.com/download\n" +
			"  Then run: ork deploy --expose --tunnel-provider ngrok",
	)
}

// Authenticate configures the ngrok auth token.
func (n *NgrokProvider) Authenticate(_ context.Context, token string) error {
	if token == "" {
		return fmt.Errorf(
			"ngrok auth token required — get yours at https://dashboard.ngrok.com/get-started/your-authtoken\n" +
				"  Then run: ork deploy --expose --tunnel-provider ngrok --tunnel-token <token>",
		)
	}
	cmd := exec.Command("ngrok", "config", "add-authtoken", token)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Start launches ngrok as a background process.
func (n *NgrokProvider) Start(ctx context.Context, localPort int) (string, int, error) {
	cmd := exec.Command(
		"ngrok", "http", fmt.Sprintf("%d", localPort),
		"--log=stdout", "--log-format=json",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", 0, fmt.Errorf("ngrok: pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", 0, fmt.Errorf("ngrok: start: %w", err)
	}

	urlCh := make(chan string, 1)
	go scanNgrokURL(stdout, urlCh)

	timeout := time.NewTimer(20 * time.Second)
	defer timeout.Stop()

	select {
	case url := <-urlCh:
		return url, cmd.Process.Pid, nil
	case <-timeout.C:
		cmd.Process.Kill()
		return "", 0, fmt.Errorf("ngrok: timed out waiting for tunnel URL")
	case <-ctx.Done():
		cmd.Process.Kill()
		return "", 0, ctx.Err()
	}
}

func (n *NgrokProvider) Stop(pid int) error {
	if pid <= 0 {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	return proc.Kill()
}

// ngrokLogLine is the JSON format ngrok emits with --log-format=json.
type ngrokLogLine struct {
	URL string `json:"url"`
	Msg string `json:"msg"`
}

func scanNgrokURL(r io.Reader, urlCh chan<- string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var line ngrokLogLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.URL != "" {
			urlCh <- line.URL
			go io.Copy(io.Discard, r)
			return
		}
	}
}
