package applyapi

import (
	"fmt"
	"net/http"
	"strings"

	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// resolvePollURL builds the poll URL for the Apply API response.
//
// It follows this resolution order:
//  1. Start with the default resource path: /api/v1/resources/{kind}/{namespace}/{name}
//  2. If poll.url is configured and resolves to a non-empty value, replace the default
//  3. If poll.field is configured and resolves to a non-empty value, append ?field=<value>
//  4. Return the final URL
//
// The resolver is used to evaluate templates in poll.url and poll.field.
func resolvePollURL(
	kind, namespace, name string,
	config *orktypes.IPDPollingConfig,
	resolver *orktmpl.Resolver,
) string {
	// 1. Start with default
	pollURL := resourcePath(kind, namespace, name)

	// 2. Override with custom URL if configured
	if config != nil && config.URL != "" {
		if resolved, err := resolver.Resolve(config.URL); err == nil && resolved != "" {
			pollURL = resolved
		}
	}

	// 3. Append field if configured
	if config != nil && config.Field != "" {
		if resolved, err := resolver.Resolve(config.Field); err == nil && resolved != "" {
			pollURL = fmt.Sprintf("%s?field=%s", pollURL, resolved)
		}
	}

	return pollURL
}

// resourcePath builds the canonical /api/v1/resources/{kind}/{namespace}/{name}
// path for a CR. namespace is "" for cluster-scoped kinds.
func resourcePath(kind, namespace, name string) string {
	if namespace == "" {
		return fmt.Sprintf("/api/v1/resources/%s//%s", kind, name)
	}
	return fmt.Sprintf("/api/v1/resources/%s/%s/%s", kind, namespace, name)
}

// parsePath extracts kind, namespace, and optional name from
// /api/v1/resources/{kind}/{namespace}[/{name}]
func parsePath(path string) (kind, ns, name string, err error) {
	// Strip prefix — the handler is registered at /api/v1/resources/
	path = strings.TrimPrefix(path, "/api/v1/resources/")
	path = strings.TrimPrefix(path, "/") // normalise double-slash

	parts := strings.SplitN(path, "/", 3)
	switch len(parts) {
	case 2:
		return parts[0], parts[1], "", nil
	case 3:
		return parts[0], parts[1], parts[2], nil
	default:
		return "", "", "", fmt.Errorf("path must be /{kind}/{namespace}[/{name}], got %q", path)
	}
}

// writeKubeError maps common Kubernetes API errors to HTTP status codes.
func writeKubeError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found") || strings.Contains(msg, "NotFound"):
		http.Error(w, msg, http.StatusNotFound)
	case strings.Contains(msg, "forbidden") || strings.Contains(msg, "Forbidden"):
		http.Error(w, msg, http.StatusForbidden)
	default:
		http.Error(w, msg, http.StatusInternalServerError)
	}
}
