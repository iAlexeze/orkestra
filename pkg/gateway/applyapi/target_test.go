package applyapi

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestIsTargetRequest(t *testing.T) {
	assert.True(t, IsTargetRequest(map[string]interface{}{
		"target": "app",
	}))
	// target wins even when apiVersion is also present (gradual migration path)
	assert.True(t, IsTargetRequest(map[string]interface{}{
		"target":     "app",
		"apiVersion": "v1",
	}))
	assert.False(t, IsTargetRequest(map[string]interface{}{
		"apiVersion": "platform.myorg.io/v1",
		"kind":       "App",
	}))
	assert.False(t, IsTargetRequest(map[string]interface{}{}))
}

func TestBuildCRFromTarget(t *testing.T) {
	appCRD := &orktypes.CRDEntry{
		APITypes: orktypes.APITypes{
			Group:   "platform.myorg.io",
			Version: "v1",
			Kind:    "App",
			Plural:  "apps",
		},
		GroupVersionKind: schema.GroupVersionKind{
			Group: "platform.myorg.io", Version: "v1", Kind: "App",
		},
		IDP: &orktypes.IDPConfig{
			Target:    "app",
			Name:      `{{ .repository | repoSlug }}`,
			Namespace: `{{ .team }}-{{ .environment }}`,
			Fields: map[string]orktypes.IDPFieldConfig{
				"repository":  {},
				"image":       {},
				"environment": {},
				"replicas":    {},
			},
			AdditionalFields: &orktypes.AdditionalIDPFields{
				Labels: map[string]orktypes.IDPFieldConfig{
					"team": {},
				},
				Annotations: map[string]orktypes.IDPFieldConfig{
					"jira-ticket": {},
				},
			},
		},
	}

	t.Run("spec fields routed correctly", func(t *testing.T) {
		raw := map[string]interface{}{
			"target":      "app",
			"repository":  "myorg/payments-api",
			"image":       "ghcr.io/myorg/payments-api:v1",
			"environment": "staging",
			"replicas":    float64(2),
			"team":        "payments",
			"jira-ticket": "PLAT-1234",
		}

		obj, err := BuildCRFromTarget(raw, appCRD, orktypes.NoteRegistry{})
		require.NoError(t, err)

		spec := obj.Object["spec"].(map[string]interface{})
		assert.Equal(t, "myorg/payments-api", spec["repository"])
		assert.Equal(t, "ghcr.io/myorg/payments-api:v1", spec["image"])
		assert.Equal(t, "staging", spec["environment"])
		assert.Equal(t, float64(2), spec["replicas"])

		labels := obj.Object["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
		assert.Equal(t, "payments", labels["team"])

		annotations := obj.Object["metadata"].(map[string]interface{})["annotations"].(map[string]interface{})
		assert.Equal(t, "PLAT-1234", annotations["jira-ticket"])

		// team and jira-ticket must NOT be in spec.
		assert.Nil(t, spec["team"])
		assert.Nil(t, spec["jira-ticket"])
	})

	t.Run("unknown fields ignored", func(t *testing.T) {
		raw := map[string]interface{}{
			"target":        "app",
			"repository":    "myorg/payments-api",
			"team":          "payments",
			"environment":   "staging",
			"unknown-field": "should be ignored",
		}
		obj, err := BuildCRFromTarget(raw, appCRD, orktypes.NoteRegistry{})
		require.NoError(t, err)

		spec := obj.Object["spec"].(map[string]interface{})
		_, exists := spec["unknown-field"]
		assert.False(t, exists)
	})

	t.Run("apiVersion and kind set from CRD entry", func(t *testing.T) {
		raw := map[string]interface{}{
			"target":      "app",
			"repository":  "myorg/payments-api",
			"team":        "payments",
			"environment": "staging",
		}
		obj, err := BuildCRFromTarget(raw, appCRD, orktypes.NoteRegistry{})
		require.NoError(t, err)
		assert.Equal(t, "platform.myorg.io/v1", obj.GetAPIVersion())
		assert.Equal(t, "App", obj.GetKind())
	})
}
