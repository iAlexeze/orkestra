// pkg/tunnel/expose.go
//
// High-level Expose function used by `ork deploy --expose`.
// Detects the local port, starts the tunnel, and persists state.
package tunnel

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ExposeOptions controls how the tunnel is started.
type ExposeOptions struct {
	// Provider is the explicit provider name ("cloudflared" or "ngrok").
	// Empty means auto-select.
	Provider string

	// Token is passed to Provider.Authenticate when non-empty.
	Token string

	// LocalPort overrides auto-detection when non-zero.
	LocalPort int

	// ServiceName is the bare app name (e.g. "my-app") used for
	// direct service port-forward fallback.
	ServiceName string

	// Namespace is the target namespace for service port-forward fallback.
	Namespace string

	// ServicePort is the app's container port for direct forwarding fallback.
	ServicePort string
}

// Expose starts or reuses a tunnel and returns the public URL.
// It persists the daemon state to ~/.orkestra/tunnel-state.json.
func Expose(ctx context.Context, opts ExposeOptions) (string, error) {
	// Reuse an existing alive tunnel on the same port
	if existing, err := LoadState(); err == nil && existing != nil {
		port := opts.LocalPort
		if port == 0 {
			port = 80
		}
		if existing.IsAlive() && existing.LocalPort == port {
			return existing.URL, nil
		}
		// Stale — clean up
		existing.Stop() //nolint:errcheck
	}

	var p Provider
	var err error
	if opts.Provider != "" {
		p, err = SelectByName(opts.Provider)
	} else {
		p, err = Select()
	}
	if err != nil {
		return "", err
	}

	// Install if needed
	if !p.Available() {
		fmt.Printf("  → Installing %s...\n", p.Name())
		if err := p.Install(ctx); err != nil {
			return "", fmt.Errorf("tunnel: install %s: %w", p.Name(), err)
		}
	}

	// Authenticate
	if opts.Token != "" {
		if err := p.Authenticate(ctx, opts.Token); err != nil {
			return "", fmt.Errorf("tunnel: auth: %w", err)
		}
	}

	localPort := opts.LocalPort
	if localPort == 0 {
		localPort, err = detectLocalPort(opts.ServiceName, opts.Namespace, opts.ServicePort)
		if err != nil {
			return "", fmt.Errorf("tunnel: port detection: %w", err)
		}
	}

	url, pid, err := p.Start(ctx, localPort)
	if err != nil {
		return "", fmt.Errorf("tunnel: start: %w", err)
	}

	state := State{
		Provider:  p.Name(),
		PID:       pid,
		URL:       url,
		LocalPort: localPort,
		StartedAt: time.Now(),
	}
	if err := SaveState(state); err != nil {
		fmt.Printf("  ~ could not save tunnel state: %v\n", err)
	}

	return url, nil
}

// detectLocalPort finds the local port to tunnel to in order of preference:
//  1. Ingress controller NodePort (for kind clusters)
//  2. Port 80 (standard ingress)
//  3. Direct service port-forward as last resort
func detectLocalPort(serviceName, namespace, servicePort string) (int, error) {
	// 1. Try ingress controller NodePort
	if port := ingressNodePort(); port > 0 {
		return port, nil
	}

	// 2. Standard port 80 — always reachable on kind with port-mapped ingress
	if isPortListening(80) {
		return 80, nil
	}

	// 3. Direct port-forward to the app's service
	if serviceName != "" && namespace != "" {
		port, err := startPortForward(serviceName, namespace, servicePort)
		if err == nil {
			return port, nil
		}
	}

	// Fall back to 80 and let cloudflared surface the connection error
	return 80, nil
}

// ingressNodePort queries the ingress-nginx service for its HTTP NodePort.
func ingressNodePort() int {
	out, err := exec.Command(
		"kubectl", "get", "svc",
		"-n", "ingress-nginx", "ingress-nginx-controller",
		"-o", `jsonpath={.spec.ports[?(@.name=="http")].nodePort}`,
	).Output()
	if err != nil {
		return 0
	}
	port, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return port
}

// isPortListening checks if something is already bound to the port on localhost.
func isPortListening(port int) bool {
	out, err := exec.Command("sh", "-c",
		fmt.Sprintf("curl -s --connect-timeout 1 http://localhost:%d >/dev/null 2>&1; echo $?", port),
	).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "0"
}

// startPortForward starts a kubectl port-forward in the background for apps
// without an ingress controller, returning the local port chosen.
func startPortForward(serviceName, namespace, servicePort string) (int, error) {
	if servicePort == "" {
		servicePort = "80"
	}
	localPort := 18080 // arbitrary high port unlikely to be in use

	cmd := exec.Command(
		"kubectl", "port-forward",
		"-n", namespace,
		"svc/"+serviceName+"-svc",
		fmt.Sprintf("%d:%s", localPort, servicePort),
	)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	// Give the port-forward a moment to bind
	time.Sleep(500 * time.Millisecond)
	return localPort, nil
}
