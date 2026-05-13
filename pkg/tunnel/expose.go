// pkg/tunnel/expose.go
//
// High-level Expose function used by `ork doctor deploy --expose` and
// `ork tunnel expose`. Detects the local port, starts the tunnel,
// and persists state under a named key.
package tunnel

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ExposeOptions controls how the tunnel is started.
type ExposeOptions struct {
	// Name identifies the tunnel in the state map (e.g. "my-app", "controlcenter").
	// Falls back to ServiceName when empty.
	Name string

	// Provider is the explicit provider name ("cloudflared" or "ngrok").
	// Empty means auto-select.
	Provider string

	// Token is passed to Provider.Authenticate when non-empty.
	Token string

	// LocalPort overrides auto-detection when non-zero.
	LocalPort int

	// ServiceName is the Kubernetes service name to port-forward to as a
	// fallback (e.g. "my-app-orkestra-svc", "orkestra-cc").
	ServiceName string

	// Namespace is the target namespace for the service port-forward.
	Namespace string

	// ServicePort is the container port on the service (default "80").
	ServicePort string

	// PortForward is kept for compatibility but has no effect when ServiceName
	// and Namespace are set — port-forward is always preferred in that case.
	// Left here so callers that explicitly set it true do not break.
	PortForward bool
}

// Expose starts or reuses a named tunnel and returns the public URL.
// State is persisted to ~/.orkestra/tunnel-state.json under opts.Name.
func Expose(ctx context.Context, opts ExposeOptions) (string, error) {
	name := opts.Name
	if name == "" {
		name = opts.ServiceName
	}

	// Reuse only when the cloudflared process is alive AND the local port it is
	// forwarding from is still accepting connections. A live PID that has lost
	// its port-forward target is a zombie — kill it and start fresh.
	if existing, err := LoadTunnelState(name); err == nil && existing != nil {
		if existing.IsAlive() && isTCPListening(existing.LocalPort) {
			return existing.URL, nil
		}
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

	if !p.Available() {
		fmt.Printf("  → Installing %s...\n", p.Name())
		if err := p.Install(ctx); err != nil {
			return "", fmt.Errorf("tunnel: install %s: %w", p.Name(), err)
		}
	}

	if opts.Token != "" {
		if err := p.Authenticate(ctx, opts.Token); err != nil {
			return "", fmt.Errorf("tunnel: auth: %w", err)
		}
	}

	localPort := opts.LocalPort
	var pfPID int

	if localPort == 0 {
		localPort, pfPID, err = resolveLocalPort(name, opts)
		if err != nil {
			return "", fmt.Errorf("tunnel: port resolution: %w", err)
		}
	}

	url, pid, err := p.Start(ctx, localPort)
	if err != nil {
		return "", fmt.Errorf("tunnel: start: %w", err)
	}

	state := State{
		Provider:       p.Name(),
		PID:            pid,
		PortForwardPID: pfPID,
		URL:            url,
		LocalPort:      localPort,
		StartedAt:      time.Now(),
	}
	if err := SaveTunnelState(name, state); err != nil {
		fmt.Printf("  ~ could not save tunnel state: %v\n", err)
	}

	return url, nil
}

// resolveLocalPort picks the local port cloudflared should point at.
//
// Priority order:
//  1. kubectl port-forward to the service — used whenever ServiceName is
//     provided. Port-forward connects directly to the K8s service, bypassing
//     the ingress. This is resilient to pod restarts and ingress disruptions,
//     and allows the reuse-guard (isTCPListening) to detect when the
//     port-forward has died so a fresh tunnel can be started.
//  2. Port 80 on localhost — fallback when no ServiceName is given. Works
//     when the kind ingress controller maps host port 80, but is unreliable:
//     if the pod or ingress backend becomes unavailable, port 80 stays
//     "listening" (the ingress controller itself is up) so cloudflared
//     cannot detect the broken origin and keeps serving 502s.
//
// The ingress NodePort is intentionally not used: on kind clusters it is only
// reachable inside the Docker network, not from localhost on the host machine.
func resolveLocalPort(name string, opts ExposeOptions) (localPort int, pfPID int, err error) {
	if opts.ServiceName != "" && opts.Namespace != "" {
		return startPortForward(name, opts.ServiceName, opts.Namespace, opts.ServicePort)
	}

	// No service info — fall back to port 80 (host-mapped ingress).
	if isTCPListening(80) {
		return 80, 0, nil
	}

	return 0, 0, fmt.Errorf(
		"no service reachable on port 80 — set ServiceName and Namespace for port-forward",
	)
}

// startPortForward starts a detached kubectl port-forward and returns
// the local port and the process PID once the port is actually listening.
//
// The process is placed in its own session (Setsid) so it survives after
// the parent CLI command exits.
//
// The local port is derived deterministically from tunnelName (range 19000–19999)
// so concurrent tunnels never conflict.
func startPortForward(tunnelName, serviceName, namespace, servicePort string) (int, int, error) {
	if servicePort == "" {
		servicePort = "80"
	}
	localPort := portForName(tunnelName)

	cmd := exec.Command(
		"kubectl", "port-forward",
		"-n", namespace,
		"svc/"+serviceName,
		fmt.Sprintf("%d:%s", localPort, servicePort),
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return 0, 0, fmt.Errorf("kubectl port-forward: %w", err)
	}
	pid := cmd.Process.Pid

	addrStr := fmt.Sprintf("127.0.0.1:%d", localPort)
	portAlive := func() bool {
		return cmd.Process.Signal(syscall.Signal(0)) == nil
	}

	// Phase 1: wait for the port-forward to bind the local port (up to 10s).
	deadline := time.Now().Add(10 * time.Second)
	bound := false
	for time.Now().Before(deadline) {
		if !portAlive() {
			return 0, 0, fmt.Errorf(
				"kubectl port-forward for svc/%s in %s exited — is the pod running?\n"+
					"  Check: kubectl get pods -n %s",
				serviceName, namespace, namespace,
			)
		}
		if isTCPListening(localPort) {
			bound = true
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if !bound {
		if !portAlive() {
			return 0, 0, fmt.Errorf(
				"kubectl port-forward for svc/%s in %s exited — is the pod running?\n"+
					"  Check: kubectl get pods -n %s",
				serviceName, namespace, namespace,
			)
		}
		// Port-forward is alive but port still not bound — continue anyway.
		return localPort, pid, nil
	}

	// Phase 2: warm-probe — verify the port-forward path reaches a live backend.
	// kubectl port-forward accepts TCP connections immediately after binding, but
	// closes them right away when the pod is not yet ready. A read timeout means
	// the connection stayed open → backend is live and ready for cloudflared.
	warmDeadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(warmDeadline) {
		if !portAlive() {
			break
		}
		conn, err := net.DialTimeout("tcp", addrStr, 500*time.Millisecond)
		if err != nil {
			time.Sleep(300 * time.Millisecond)
			continue
		}
		conn.SetDeadline(time.Now().Add(300 * time.Millisecond))
		buf := make([]byte, 1)
		_, readErr := conn.Read(buf)
		conn.Close()
		// A timeout means the connection stayed open — backend is live.
		if netErr, ok := readErr.(net.Error); ok && netErr.Timeout() {
			return localPort, pid, nil
		}
		// Connection was closed by kubectl — backend not ready yet. Retry.
		time.Sleep(300 * time.Millisecond)
	}

	return localPort, pid, nil
}

// isTCPListening reports whether localhost:port accepts TCP connections.
func isTCPListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// portForName maps a tunnel name to a stable local port in the range 19000–19999.
func portForName(name string) int {
	h := 0
	for _, c := range name {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return 19000 + (h % 1000)
}

// ingressNodePort is kept for completeness but is no longer used for port
// selection: NodePorts are only accessible inside the Docker/cluster network
// and cannot be reached from localhost on the host machine in kind setups.
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
