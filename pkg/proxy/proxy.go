//go:build !runtime && !gateway

package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/orkspace/orkestra/pkg/utils"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// ForwardTarget describes one component to port-forward.
type ForwardTarget struct {
	Label     string // display name, e.g. "Runtime"
	Komponent string // KomponentRuntime | KomponentCC | KomponentGateway
	Namespace string
	LocalPort int
	Scheme    string // "http" or "https"
	ViaLease  bool   // true for Runtime: resolve pod from Lease
}

// RunAll discovers and forwards all targets, printing status to out.
// It blocks until ctx is cancelled, then closes all forwards cleanly.
func RunAll(ctx context.Context, cfg *rest.Config, cs kubernetes.Interface, targets []ForwardTarget, out io.Writer) {
	for _, t := range targets {
		go func(t ForwardTarget) {
			forwardWithReconnect(ctx, cfg, cs, t, out)
		}(t)
	}
	<-ctx.Done()
}

func forwardWithReconnect(ctx context.Context, cfg *rest.Config, cs kubernetes.Interface, t ForwardTarget, out io.Writer) {
	first := true
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		pod, ns, err := resolveTarget(ctx, cs, t)
		if err != nil {
			if first {
				fmt.Fprintf(out, "  %s %-14s %v\n", utils.Yellow("✗"), t.Label, err)
				first = false
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}

		if first {
			fmt.Fprintf(out, "  %s %-14s %s://localhost:%d   (%s)\n", utils.Green("✓"), t.Label, t.Scheme, t.LocalPort, pod)
			first = false
		} else {
			fmt.Fprintf(out, "  %s %-14s %s://localhost:%d   (%s)  [reconnected]\n", utils.Green("✓"), t.Label, t.Scheme, t.LocalPort, pod)
		}

		err = forward(ctx, cfg, ns, pod, t.LocalPort, int(resolveRemotePort(t)))
		if err != nil && ctx.Err() == nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}
}

func resolveTarget(ctx context.Context, cs kubernetes.Interface, t ForwardTarget) (pod, ns string, err error) {
	svc, err := FindService(ctx, cs, t.Namespace, t.Komponent)
	if err != nil {
		return "", "", err
	}
	if svc == nil {
		return "", "", fmt.Errorf("not deployed in %s", t.Namespace)
	}

	if t.ViaLease {
		pod, err = ResolveRuntimePod(ctx, cs, t.Namespace)
		if err != nil {
			return "", "", err
		}
	} else {
		pod, err = ResolvePod(ctx, cs, t.Namespace, svc.Name)
		if err != nil {
			return "", "", err
		}
	}
	return pod, t.Namespace, nil
}

// resolveRemotePort returns the container port for the target.
// For Gateway we forward to the http port by default; https is on a different local port.
func resolveRemotePort(t ForwardTarget) int32 {
	// Service port == container port for all Orkestra components (see charts).
	// The caller sets LocalPort to the desired local port; remote is always the service port.
	// We rediscover via FindService to get the actual port — but for simplicity we use
	// the local port as the remote port since they are the same by default.
	// Custom local ports (--runtime-port etc.) do not change the remote port.
	switch t.Komponent {
	case KomponentRuntime:
		return 8080
	case KomponentCC:
		return 8081
	case KomponentGateway:
		return 8080
	}
	return int32(t.LocalPort)
}

func forward(ctx context.Context, cfg *rest.Config, ns, pod string, localPort, remotePort int) error {
	url := cfg.Host
	// Build the portforward URL: /api/v1/namespaces/{ns}/pods/{pod}/portforward
	// We use the REST client URL builder via a round-about but stable approach.
	// restclient is used only to form the URL; the SPDY transport is separate.
	pfURL := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s/portforward", url, ns, pod)
	parsedURL, err := http.NewRequest(http.MethodPost, pfURL, nil)
	if err != nil {
		return fmt.Errorf("build portforward url: %w", err)
	}

	transport, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return fmt.Errorf("spdy transport: %w", err)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, parsedURL.URL)

	stopChan := make(chan struct{})
	readyChan := make(chan struct{}, 1)

	go func() {
		<-ctx.Done()
		close(stopChan)
	}()

	fw, err := portforward.New(dialer,
		[]string{fmt.Sprintf("%d:%d", localPort, remotePort)},
		stopChan, readyChan,
		io.Discard, io.Discard,
	)
	if err != nil {
		return fmt.Errorf("portforward: %w", err)
	}
	return fw.ForwardPorts()
}
