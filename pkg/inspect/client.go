// pkg/inspect/client.go
package inspect

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Clients holds the Kubernetes clients needed by all inspect commands.
// Built once, shared across all operations within a single command invocation.
type Clients struct {
	// Dynamic — for reading any CRD as unstructured.Unstructured.
	// Used by get, describe, and reconcile commands.
	Dynamic dynamic.Interface

	// Discovery — for listing CRDs and resolving GVR from a name.
	// Used by all commands to discover what the user is referring to.
	Discovery discovery.DiscoveryInterface

	// Core — for reading Kubernetes events.
	// Used by describe and events commands.
	Core kubernetes.Interface

	// RestConfig — raw REST config, kept for any future direct REST calls.
	RestConfig *rest.Config
}

// NewClients builds Kubernetes clients from the provided kubeconfig path.
//
// Resolution order:
//  1. Explicit kubeconfigPath argument (from --kubeconfig flag)
//  2. KUBECONFIG environment variable
//  3. ~/.kube/config (default kubeconfig location)
//  4. In-cluster config (when running inside Kubernetes)
//
// Returns a clear error if none of these paths produce a valid config.
func NewClients(kubeconfigPath string) (*Clients, error) {
	cfg, err := buildRestConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("building kubeconfig: %w", err)
	}

	// Increase QPS and burst for list operations — inspect commands may
	// list many CRs at once and should not be throttled unnecessarily.
	cfg.QPS = 50
	cfg.Burst = 100

	dynamic, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("building dynamic client: %w", err)
	}

	discovery, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("building discovery client: %w", err)
	}

	core, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("building core client: %w", err)
	}

	return &Clients{
		Dynamic:    dynamic,
		Discovery:  discovery,
		Core:       core,
		RestConfig: cfg,
	}, nil
}

// buildRestConfig resolves the kubeconfig and returns a *rest.Config.
func buildRestConfig(explicit string) (*rest.Config, error) {
	// Priority 1 — explicit --kubeconfig flag
	if explicit != "" {
		return clientcmd.BuildConfigFromFlags("", explicit)
	}

	// Priority 2 — KUBECONFIG environment variable
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return clientcmd.BuildConfigFromFlags("", env)
	}

	// Priority 3 — default ~/.kube/config
	home, err := os.UserHomeDir()
	if err == nil {
		defaultPath := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(defaultPath); err == nil {
			return clientcmd.BuildConfigFromFlags("", defaultPath)
		}
	}

	// Priority 4 — in-cluster config (running inside a pod)
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf(
			"no kubeconfig found — set KUBECONFIG, pass --kubeconfig, or run inside a cluster",
		)
	}

	return cfg, nil
}
