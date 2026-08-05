package applyapi

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewCRSkeleton(t *testing.T) {
	crd := &orktypes.CRDEntry{
		APITypes: orktypes.APITypes{
			Group:   "platform.myorg.io",
			Version: "v1",
			Kind:    "AppRequest",
		},
		GroupVersionKind: schema.GroupVersionKind{
			Group: "platform.myorg.io", Version: "v1", Kind: "AppRequest",
		},
	}

	obj := newCRSkeleton(crd)

	expectedAPIVersion := crd.APITypes.Group + "/" + crd.APITypes.Version
	assert.Equal(t, expectedAPIVersion, obj.Object["apiVersion"])
	assert.Equal(t, crd.APITypes.Kind, obj.Object["kind"])

	// Check metadata exists
	meta, ok := obj.Object["metadata"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotNil(t, meta["labels"])
	assert.NotNil(t, meta["annotations"])

	// Check spec exists
	spec, ok := obj.Object["spec"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotNil(t, spec)
}

func TestRouteFields(t *testing.T) {
	crd := &orktypes.CRDEntry{
		IDP: &orktypes.IDPConfig{
			Fields: map[string]orktypes.IDPFieldConfig{
				"repository": {},
				"image":      {},
				"replicas":   {},
			},
			AdditionalFields: &orktypes.AdditionalIDPFields{
				Labels: map[string]orktypes.IDPFieldConfig{
					"team":        {},
					"environment": {},
				},
				Annotations: map[string]orktypes.IDPFieldConfig{
					"jira-ticket": {},
					"expose":      {},
				},
			},
		},
	}

	raw := map[string]interface{}{
		"target":      "smartapp",
		"repository":  "myorg/payments-api",
		"image":       "ghcr.io/myorg/payments-api:v1.0.0",
		"replicas":    2,
		"team":        "team-payments",
		"environment": "staging",
		"jira-ticket": "PLAT-1234",
		"expose":      "true",
		"unknown":     "ignored",
	}

	obj := newCRSkeleton(crd)
	routeFields(raw, crd, obj)

	// Check spec fields
	spec := obj.Object["spec"].(map[string]interface{})
	assert.Equal(t, "myorg/payments-api", spec["repository"])
	assert.Equal(t, "ghcr.io/myorg/payments-api:v1.0.0", spec["image"])
	assert.Equal(t, 2, spec["replicas"])

	// Check labels
	meta := obj.Object["metadata"].(map[string]interface{})
	labels := meta["labels"].(map[string]interface{})
	assert.Equal(t, "team-payments", labels["team"])
	assert.Equal(t, "staging", labels["environment"])

	// Check annotations
	annotations := meta["annotations"].(map[string]interface{})
	assert.Equal(t, "PLAT-1234", annotations["jira-ticket"])
	assert.Equal(t, "true", annotations["expose"])

	// Check unknown field was ignored
	assert.NotContains(t, spec, "unknown")
	assert.NotContains(t, labels, "unknown")
	assert.NotContains(t, annotations, "unknown")
}

func TestRouteFields_MissingFields(t *testing.T) {
	crd := &orktypes.CRDEntry{
		IDP: &orktypes.IDPConfig{
			Fields: map[string]orktypes.IDPFieldConfig{
				"repository": {},
				"image":      {},
			},
			AdditionalFields: &orktypes.AdditionalIDPFields{
				Labels: map[string]orktypes.IDPFieldConfig{
					"team": {},
				},
			},
		},
	}

	raw := map[string]interface{}{
		"target": "smartapp",
		// Missing "repository" field
		"image": "ghcr.io/myorg/app:v1.0.0",
		"team":  "team-payments",
	}

	obj := newCRSkeleton(crd)
	routeFields(raw, crd, obj)

	spec := obj.Object["spec"].(map[string]interface{})
	assert.NotContains(t, spec, "repository") // Should be absent
	assert.Equal(t, "ghcr.io/myorg/app:v1.0.0", spec["image"])

	labels := obj.Object["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
	assert.Equal(t, "team-payments", labels["team"])
}

func TestRouteFields_NonStringLabels(t *testing.T) {
	crd := &orktypes.CRDEntry{
		IDP: &orktypes.IDPConfig{
			AdditionalFields: &orktypes.AdditionalIDPFields{
				Labels: map[string]orktypes.IDPFieldConfig{
					"count": {},
				},
			},
		},
	}

	raw := map[string]interface{}{
		"target": "smartapp",
		"count":  42, // Integer value
	}

	obj := newCRSkeleton(crd)
	routeFields(raw, crd, obj)

	labels := obj.Object["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
	assert.Equal(t, "42", labels["count"]) // Should be converted to string
}

func TestResolveIDPIdentity(t *testing.T) {
	// Use built-in notes (repoSlug is built-in)
	notes := orktypes.NoteRegistry{}

	namespaced := true
	crd := &orktypes.CRDEntry{
		Namespaced: &namespaced,
		APITypes: orktypes.APITypes{
			Kind: "AppRequest",
		},
		IDP: &orktypes.IDPConfig{
			Name:      "{{ repoSlug .repository }}",
			Namespace: "{{ .team }}-{{ .environment }}",
		},
	}

	raw := map[string]interface{}{
		"target":      "smartapp",
		"repository":  "myorg/payments-api",
		"team":        "team-payments",
		"environment": "staging",
	}

	obj := newCRSkeleton(crd)

	err := resolveIDPIdentity(raw, crd, notes, obj)
	require.NoError(t, err)

	assert.Equal(t, "payments-api", obj.GetName())
	assert.Equal(t, "team-payments-staging", obj.GetNamespace())
}

func TestResolveIDPIdentity_WithRepoSlugNote(t *testing.T) {
	// Use built-in notes (repoSlug is built-in)
	notes := orktypes.NoteRegistry{}

	namespaced := true
	crd := &orktypes.CRDEntry{
		Namespaced: &namespaced,
		APITypes: orktypes.APITypes{
			Kind: "AppRequest",
		},
		IDP: &orktypes.IDPConfig{
			Name:      "{{ repoSlug .repository }}",
			Namespace: "{{ .team }}-{{ .environment }}",
		},
	}

	raw := map[string]interface{}{
		"target":      "smartapp",
		"repository":  "myorg/payments-api",
		"team":        "team-payments",
		"environment": "staging",
	}

	obj := newCRSkeleton(crd)

	err := resolveIDPIdentity(raw, crd, notes, obj)
	require.NoError(t, err)

	assert.Equal(t, "payments-api", obj.GetName())
	assert.Equal(t, "team-payments-staging", obj.GetNamespace())
}

func TestResolveIDPIdentity_OnlyName(t *testing.T) {
	notes := orktypes.NoteRegistry{}

	crd := &orktypes.CRDEntry{
		APITypes: orktypes.APITypes{
			Kind: "AppRequest",
		},
		IDP: &orktypes.IDPConfig{
			Name: "{{ repoSlug .repository }}",
			// Namespace not set — should be empty
		},
	}

	raw := map[string]interface{}{
		"target":     "smartapp",
		"repository": "myorg/orders-api",
	}

	obj := newCRSkeleton(crd)

	err := resolveIDPIdentity(raw, crd, notes, obj)
	require.NoError(t, err)

	assert.Equal(t, "orders-api", obj.GetName())
	assert.Empty(t, obj.GetNamespace())
}

func TestResolveIDPIdentity_MissingName(t *testing.T) {
	notes := orktypes.NoteRegistry{}

	namespaced := true
	crd := &orktypes.CRDEntry{
		Namespaced: &namespaced,
		APITypes: orktypes.APITypes{
			Kind: "AppRequest",
		},
		IDP: &orktypes.IDPConfig{
			Name: "{{ repoSlug .repository }}",
		},
	}

	raw := map[string]interface{}{
		"target": "smartapp",
		// Missing "repository" field
		"image": "ghcr.io/myorg/app:v1.0.0",
	}

	obj := newCRSkeleton(crd)

	err := resolveIDPIdentity(raw, crd, notes, obj)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "idp.name expression")
	assert.Contains(t, err.Error(), "could not be resolved")
}

func TestResolveIDPIdentity_MissingNamespace(t *testing.T) {
	notes := orktypes.NoteRegistry{}

	namespaced := true
	crd := &orktypes.CRDEntry{
		Namespaced: &namespaced,
		APITypes: orktypes.APITypes{
			Kind: "AppRequest",
		},
		IDP: &orktypes.IDPConfig{
			Namespace: "{{ .team }}-{{ .environment }}",
		},
	}

	raw := map[string]interface{}{
		"target": "smartapp",
		// Missing "team" and "environment"
	}

	obj := newCRSkeleton(crd)

	err := resolveIDPIdentity(raw, crd, notes, obj)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "idp.namespace expression")
	assert.Contains(t, err.Error(), "could not be resolved")
}

func TestResolveIDPIdentity_EmptyName(t *testing.T) {
	notes := orktypes.NoteRegistry{}

	crd := &orktypes.CRDEntry{
		APITypes: orktypes.APITypes{
			Kind: "AppRequest",
		},
		IDP: &orktypes.IDPConfig{
			Name: "{{ .missingField }}",
		},
	}

	raw := map[string]interface{}{
		"target": "smartapp",
	}

	obj := newCRSkeleton(crd)

	err := resolveIDPIdentity(raw, crd, notes, obj)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "idp.name expression")
	assert.Contains(t, err.Error(), "could not be resolved")
}

func TestNewBuildCRFromTarget(t *testing.T) {
	// Use built-in notes (repoSlug is built-in)
	notes := orktypes.NoteRegistry{}

	namespaced := true
	crd := &orktypes.CRDEntry{
		Namespaced: &namespaced,
		APITypes: orktypes.APITypes{
			Group:   "platform.myorg.io",
			Version: "v1",
			Kind:    "AppRequest",
		},
		GroupVersionKind: schema.GroupVersionKind{
			Group: "platform.myorg.io", Version: "v1", Kind: "AppRequest",
		},
		IDP: &orktypes.IDPConfig{
			Name:      "{{ repoSlug .repository }}",
			Namespace: "{{ .team }}-{{ .environment }}",
			Fields: map[string]orktypes.IDPFieldConfig{
				"repository": {},
				"image":      {},
				"replicas":   {},
			},
			AdditionalFields: &orktypes.AdditionalIDPFields{
				Labels: map[string]orktypes.IDPFieldConfig{
					"team":        {},
					"environment": {},
				},
				Annotations: map[string]orktypes.IDPFieldConfig{
					"jira-ticket": {},
				},
			},
		},
	}

	raw := map[string]interface{}{
		"target":      "smartapp",
		"repository":  "myorg/payments-api",
		"image":       "ghcr.io/myorg/payments-api:v1.0.0",
		"replicas":    2,
		"team":        "team-payments",
		"environment": "staging",
		"jira-ticket": "PLAT-1234",
	}

	obj, err := BuildCRFromTarget(raw, crd, notes)
	require.NoError(t, err)

	expectedAPIVersion := crd.APITypes.Group + "/" + crd.APITypes.Version
	assert.Equal(t, expectedAPIVersion, obj.Object["apiVersion"])
	assert.Equal(t, crd.APITypes.Kind, obj.Object["kind"])

	// Check spec
	spec := obj.Object["spec"].(map[string]interface{})
	assert.Equal(t, "myorg/payments-api", spec["repository"])
	assert.Equal(t, "ghcr.io/myorg/payments-api:v1.0.0", spec["image"])
	assert.Equal(t, 2, spec["replicas"])

	// Check labels
	meta := obj.Object["metadata"].(map[string]interface{})
	labels := meta["labels"].(map[string]interface{})
	assert.Equal(t, "team-payments", labels["team"])
	assert.Equal(t, "staging", labels["environment"])

	// Check annotations
	annotations := meta["annotations"].(map[string]interface{})
	assert.Equal(t, "PLAT-1234", annotations["jira-ticket"])

	// Check name and namespace
	assert.Equal(t, "payments-api", obj.GetName())
	assert.Equal(t, "team-payments-staging", obj.GetNamespace())
}
