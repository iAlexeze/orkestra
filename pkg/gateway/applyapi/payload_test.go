package applyapi

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// EvaluatePayload
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestEvaluatePayload(t *testing.T) {
	t.Run("nil config returns nil", func(t *testing.T) {
		crd := appCRD()
		crd.IDP.Config = nil
		result := EvaluatePayload(map[string]interface{}{}, crd, noopNotes())
		assert.Nil(t, result)
	})

	t.Run("no payload declared returns nil", func(t *testing.T) {
		crd := appCRD()
		crd.IDP.Config = &orktypes.IDPConfig_Config{
			Response: &orktypes.IDPResponseConfig{},
		}
		result := EvaluatePayload(map[string]interface{}{}, crd, noopNotes())
		assert.Nil(t, result)
	})

	t.Run("plain string passes through", func(t *testing.T) {
		crd := appCRD()
		crd.IDP.Config = &orktypes.IDPConfig_Config{
			Response: &orktypes.IDPResponseConfig{
				Payload: map[string]string{
					"supportChannel": "#platform",
				},
			},
		}
		result := EvaluatePayload(map[string]interface{}{}, crd, noopNotes())
		require.NotNil(t, result)
		assert.Equal(t, "#platform", result["supportChannel"])
	})

	t.Run("template expression resolved against CR", func(t *testing.T) {
		crd := appCRD()
		crd.IDP.Config = &orktypes.IDPConfig_Config{
			Response: &orktypes.IDPResponseConfig{
				Payload: map[string]string{
					"phase": `{{ .status.phase }}`,
				},
			},
		}
		obj := map[string]interface{}{
			"status": map[string]interface{}{"phase": "Ready"},
		}
		result := EvaluatePayload(obj, crd, noopNotes())
		require.NotNil(t, result)
		assert.Equal(t, "Ready", result["phase"])
	})

	t.Run("unresolvable expression returns empty string not error", func(t *testing.T) {
		crd := appCRD()
		crd.IDP.Config = &orktypes.IDPConfig_Config{
			Response: &orktypes.IDPResponseConfig{
				Payload: map[string]string{
					"phase": `{{ .status.phase }}`,
				},
			},
		}
		// status not yet present — at apply time
		result := EvaluatePayload(map[string]interface{}{}, crd, noopNotes())
		require.NotNil(t, result)
		assert.Equal(t, "", result["phase"])
	})

	t.Run("default true includes full CR plus payload", func(t *testing.T) {
		crd := appCRD()
		tr := true
		crd.IDP.Config = &orktypes.IDPConfig_Config{
			Response: &orktypes.IDPResponseConfig{
				Default: &tr,
				Payload: map[string]string{"extra": "value"},
			},
		}
		obj := map[string]interface{}{
			"spec": map[string]interface{}{"image": "myimage"},
		}
		result := EvaluatePayload(obj, crd, noopNotes())
		require.NotNil(t, result)
		assert.Equal(t, "value", result["extra"])
		// default true — spec is preserved
		_, hasSpec := result["spec"]
		assert.True(t, hasSpec)
	})

	t.Run("default false returns only payload fields", func(t *testing.T) {
		crd := appCRD()
		f := false
		crd.IDP.Config = &orktypes.IDPConfig_Config{
			Response: &orktypes.IDPResponseConfig{
				Default: &f,
				Payload: map[string]string{"extra": "value"},
			},
		}
		obj := map[string]interface{}{
			"spec": map[string]interface{}{"image": "myimage"},
		}
		result := EvaluatePayload(obj, crd, noopNotes())
		require.NotNil(t, result)
		assert.Equal(t, "value", result["extra"])
		_, hasSpec := result["spec"]
		assert.False(t, hasSpec, "spec must not appear when default: false")
	})

	t.Run("exclude strips declared paths", func(t *testing.T) {
		crd := appCRD()
		crd.IDP.Config = &orktypes.IDPConfig_Config{
			Response: &orktypes.IDPResponseConfig{
				Exclude: []string{"metadata.managedFields", "status.observedGeneration"},
			},
		}
		obj := map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":          "my-app",
				"managedFields": []interface{}{"something"},
			},
			"status": map[string]interface{}{
				"phase":              "Ready",
				"observedGeneration": 1,
			},
		}
		result := EvaluatePayload(obj, crd, noopNotes())
		require.NotNil(t, result)
		meta := result["metadata"].(map[string]interface{})
		_, hasMF := meta["managedFields"]
		assert.False(t, hasMF, "managedFields must be excluded")
		assert.Equal(t, "my-app", meta["name"])
		status := result["status"].(map[string]interface{})
		_, hasOG := status["observedGeneration"]
		assert.False(t, hasOG, "observedGeneration must be excluded")
		assert.Equal(t, "Ready", status["phase"])
	})
}
