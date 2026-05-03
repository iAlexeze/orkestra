package doktor

import (
	"os/exec"
	"strings"
)

// IngressController is the ingress controller detected on the cluster.
type IngressController string

const (
	IngressNginx   IngressController = "nginx"
	IngressTraefik IngressController = "traefik"
	IngressNone    IngressController = ""
)

// DetectIngressController tries to identify which ingress controller is
// installed on the current cluster. Returns IngressNone when kubectl is not
// available or when no known controller is found.
func DetectIngressController() IngressController {
	out, err := exec.Command("kubectl", "get", "pods", "--all-namespaces",
		"-l", "app.kubernetes.io/name=ingress-nginx",
		"--no-headers", "-o", "name").Output()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		return IngressNginx
	}

	out, err = exec.Command("kubectl", "get", "pods", "--all-namespaces",
		"-l", "app.kubernetes.io/name=traefik",
		"--no-headers", "-o", "name").Output()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		return IngressTraefik
	}

	return IngressNone
}

// OrkestraInstalled reports whether the Orkestra runtime deployment exists
// and is available in the given namespace.
func OrkestraInstalled() bool {
	out, err := exec.Command("kubectl", "get", "deploy",
		OrkestraRuntime,
		"-n", OrkestraNamespace,
		"--no-headers",
		"-o", "name",
	).Output()

	if err != nil {
		return false
	}

	return strings.TrimSpace(string(out)) != ""
}

// OrkestraInstalled reports whether the Orkestra runtime deployment exists
// and is available and running in the given namespace.
func OrkestraInstalledRunning() bool {
	out, err := exec.Command("kubectl", "get", "deploy",
		OrkestraRuntime,
		"-n", OrkestraNamespace,
		"-o", "jsonpath={.status.conditions[?(@.type=='Available')].status}",
	).Output()

	if err != nil {
		return false
	}

	return strings.TrimSpace(string(out)) == "True"
}

// KubectlAvailable reports whether kubectl is present in PATH.
func KubectlAvailable() bool {
	_, err := exec.LookPath("kubectl")
	return err == nil
}
