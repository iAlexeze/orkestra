// pkg/inspect/discovery.go
package inspect

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

// CRDInfo holds the resolved GVR and display metadata for one CRD.
// Discovered from the cluster via the API server resource list.
type CRDInfo struct {
	// Name — CRD name e.g. "website"
	Name string

	// Group — API group e.g. "demo.orkestra.io"
	Group string

	// Version — API version e.g. "v1alpha1"
	Version string

	// Kind — singular Kind e.g. "Website"
	Kind string

	// Plural — plural resource name e.g. "websites"
	Plural string

	// Namespaced — true if the CRD is namespace-scoped
	Namespaced bool

	// GVR — the GroupVersionResource used by the dynamic client
	GVR schema.GroupVersionResource
}

// DiscoverCRD finds a CRD by name in the cluster.
//
// The input may be any of:
//   - plural resource name:  "websites"
//   - singular resource name: "website"
//   - Kind (case-insensitive): "Website" or "website"
//
// Returns an error if zero or more than one CRD matches.
// When multiple matches are found, all candidates are listed in the error
// so the user can be more specific.
func DiscoverCRD(disc discovery.DiscoveryInterface, name string) (*CRDInfo, error) {
	groups, err := disc.ServerPreferredResources()
	if err != nil {
		// ServerPreferredResources returns partial results alongside errors
		// when some API groups are unavailable. We proceed with what we have.
		if groups == nil {
			return nil, fmt.Errorf("listing cluster resources: %w", err)
		}
	}

	needle := strings.ToLower(name)
	var matches []CRDInfo

	for _, group := range groups {
		gv, err := schema.ParseGroupVersion(group.GroupVersion)
		if err != nil {
			continue
		}

		for _, r := range group.APIResources {
			// Skip subresources (e.g. pods/log, deployments/scale)
			if strings.Contains(r.Name, "/") {
				continue
			}

			plural := strings.ToLower(r.Name)
			singular := strings.ToLower(r.SingularName)
			kind := strings.ToLower(r.Kind)

			if plural == needle || singular == needle || kind == needle {
				matches = append(matches, CRDInfo{
					Group:      gv.Group,
					Version:    gv.Version,
					Kind:       r.Kind,
					Plural:     r.Name,
					Namespaced: r.Namespaced,
					GVR: schema.GroupVersionResource{
						Group:    gv.Group,
						Version:  gv.Version,
						Resource: r.Name,
					},
				})
			}
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf(
			"CRD %q not found in cluster — "+
				"check the name is correct and the CRD is installed\n"+
				"Hint: kubectl get crds | grep %s",
			name, needle,
		)
	case 1:
		return &matches[0], nil
	default:
		// Multiple matches — list them all so the user can disambiguate
		var candidates []string
		for _, m := range matches {
			candidates = append(candidates, fmt.Sprintf("  %s.%s (%s)", m.Plural, m.Group, m.Version))
		}
		return nil, fmt.Errorf(
			"%q matches multiple CRDs — be more specific:\n%s",
			name, strings.Join(candidates, "\n"),
		)
	}
}

// DiscoverOrkestraCRDs returns all CRDs in the cluster that carry the
// managed-by=orkestra label. Used by `ork reconcile all` when no --katalog
// is provided — discovers what Orkestra is already managing from the cluster
// rather than requiring a Katalog file.
func DiscoverOrkestraCRDs(disc discovery.DiscoveryInterface) ([]CRDInfo, error) {
	groups, err := disc.ServerPreferredResources()
	if err != nil && groups == nil {
		return nil, fmt.Errorf("listing cluster resources: %w", err)
	}

	var crds []CRDInfo
	seen := map[string]bool{}

	for _, group := range groups {
		gv, err := schema.ParseGroupVersion(group.GroupVersion)
		if err != nil {
			continue
		}

		for _, r := range group.APIResources {
			if strings.Contains(r.Name, "/") {
				continue
			}

			// Include only resources from groups that end in known Orkestra domains.
			// This is a heuristic — in practice users pass --katalog for precision.
			// The reconcile all command with no katalog is a convenience, not a guarantee.
			if !isUserDefinedGroup(gv.Group) {
				continue
			}

			key := gv.Group + "/" + r.Name
			if seen[key] {
				continue
			}
			seen[key] = true

			crds = append(crds, CRDInfo{
				Group:      gv.Group,
				Version:    gv.Version,
				Kind:       r.Kind,
				Plural:     r.Name,
				Namespaced: r.Namespaced,
				GVR: schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: r.Name,
				},
			})
		}
	}

	return crds, nil
}

// isUserDefinedGroup returns true for API groups that are likely custom CRDs
// rather than built-in Kubernetes resources. Filters out core, apps, batch etc.
func isUserDefinedGroup(group string) bool {
	builtIn := []string{
		"", // core group
		"apps",
		"batch",
		"extensions",
		"networking.k8s.io",
		"storage.k8s.io",
		"rbac.authorization.k8s.io",
		"policy",
		"autoscaling",
		"coordination.k8s.io",
		"apiextensions.k8s.io",
		"admissionregistration.k8s.io",
		"events.k8s.io",
		"certificates.k8s.io",
		"node.k8s.io",
		"scheduling.k8s.io",
		"discovery.k8s.io",
		"flowcontrol.apiserver.k8s.io",
		"internal.apiserver.k8s.io",
		"metrics.k8s.io",
	}

	for _, b := range builtIn {
		if group == b {
			return false
		}
	}
	return true
}
