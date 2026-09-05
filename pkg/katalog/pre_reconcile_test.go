package katalog

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
)

func katalogWithPreReconcile(pr orktypes.PreReconcileConfig) *Katalog {
	return &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app": {
				APITypes: orktypes.APITypes{
					Kind:    "Application",
					Version: "v1",
					Group:   "test.orkestra.katalog",
				},
				OperatorBox: orktypes.OperatorBoxConfig{
					PreReconcile: &pr,
				},
			},
		},
	}
}

func TestIsEventAware(t *testing.T) {
	tests := []struct {
		name     string
		katalog  *Katalog
		gvk      string
		expected bool
	}{
		{
			name: "event aware gate",
			katalog: katalogWithPreReconcile(orktypes.PreReconcileConfig{
				ReconcileGate: &orktypes.GateConditions{
					EventAware: true,
				},
			}),
			gvk:      "app",
			expected: true,
		},
		{
			name: "event aware disabled",
			katalog: katalogWithPreReconcile(orktypes.PreReconcileConfig{
				ReconcileGate: &orktypes.GateConditions{
					EventAware: false,
				},
			}),
			gvk:      "app",
			expected: false,
		},
		{
			name: "reconcile gate without event awareness",
			katalog: katalogWithPreReconcile(orktypes.PreReconcileConfig{
				ReconcileGate: &orktypes.GateConditions{
					When: []orktypes.Condition{
						{
							Field:  "{{ .metadata.name }}",
							Equals: "app",
						},
					},
				},
			}),
			gvk:      "app",
			expected: false,
		},
		{
			name:     "unknown gvk",
			katalog:  katalogWithPreReconcile(orktypes.PreReconcileConfig{}),
			gvk:      "does-not-exist",
			expected: false,
		},
		{
			name:     "nil katalog",
			katalog:  nil,
			gvk:      "app",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got bool

			if tt.katalog == nil {
				got = (*Katalog)(nil).IsEventAware(tt.gvk)
			} else {
				got = tt.katalog.IsEventAware(tt.gvk)
			}

			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGetPreReconcileSentinels_ReturnsDeclared(t *testing.T) {
	k := katalogWithPreReconcile(orktypes.PreReconcileConfig{
		Sentinels: []string{
			"generationChanged",
			"labelsChanged",
		},
	})

	sentinels := k.GetPreReconcileSentinels("app")

	assert.Equal(t, []string{"generationChanged", "labelsChanged"}, sentinels)
}

func TestGetPreReconcileSentinels_NoSentinels(t *testing.T) {
	k := katalogWithPreReconcile(orktypes.PreReconcileConfig{})

	sentinels := k.GetPreReconcileSentinels("app")

	assert.Empty(t, sentinels)
}

func TestGetPreReconcileSentinels_Unknown(t *testing.T) {
	k := katalogWithPreReconcile(orktypes.PreReconcileConfig{
		Sentinels: []string{"generationChanged"},
	})

	sentinels := k.GetPreReconcileSentinels("unknown")

	assert.Nil(t, sentinels)
}

func TestGetPreReconcileSentinels_NilKatalog(t *testing.T) {
	var k *Katalog

	sentinels := k.GetPreReconcileSentinels("app")

	assert.Nil(t, sentinels)
}
