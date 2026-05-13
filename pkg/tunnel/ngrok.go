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
	"path/filepath"
	"syscall"
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
			"  Then run: ork doctor deploy --expose --tunnel-provider ngrok",
	)
}

// Authenticate configures the ngrok auth token.
func (n *NgrokProvider) Authenticate(_ context.Context, token string) error {
	if token == "" {
		return fmt.Errorf(
			"ngrok auth token required — get yours at https://dashboard.ngrok.com/get-started/your-authtoken\n" +
				"  Then run: ork doctor deploy --expose --tunnel-provider ngrok --tunnel-token <token>",
		)
	}
	cmd := exec.Command("ngrok", "config", "add-authtoken", token)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Start launches ngrok as a detached background process.
//
// ngrok's stdout is redirected to a log file (not a pipe) so that our process
// exiting doesn't deliver SIGPIPE to ngrok while it's still initializing.
func (n *NgrokProvider) Start(ctx context.Context, localPort int) (string, int, error) {
	logPath := ngrokLogPath(localPort)

	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return "", 0, fmt.Errorf("ngrok: log dir: %w", err)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", 0, fmt.Errorf("ngrok: log file: %w", err)
	}

	cmd := exec.Command(
		"ngrok", "http", fmt.Sprintf("%d", localPort),
		"--log=stdout", "--log-format=json",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = logFile
	// cmd.Stderr = nil → /dev/null

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return "", 0, fmt.Errorf("ngrok: start: %w", err)
	}
	logFile.Close()

	urlCh := make(chan string, 1)
	go tailNgrokLogForURL(logPath, urlCh)

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

func ngrokLogPath(localPort int) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orkestra", fmt.Sprintf("ngrok-%d.log", localPort))
}

// tailNgrokLogForURL polls the log file for JSON lines with a "url" field.
type ngrokLogLine struct {
	URL string `json:"url"`
}

func tailNgrokLogForURL(logPath string, urlCh chan<- string) {
	var offset int64
	for {
		f, err := os.Open(logPath)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		_, _ = f.Seek(offset, io.SeekStart)
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var line ngrokLogLine
			if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
				continue
			}
			if line.URL != "" {
				f.Close()
				urlCh <- line.URL
				return
			}
		}
		offset, _ = f.Seek(0, io.SeekCurrent)
		f.Close()
		time.Sleep(100 * time.Millisecond)
	}
}
