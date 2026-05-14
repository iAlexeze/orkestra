// Tests for CRDEntry methods declared in methods.go.
// All methods follow a nil-pointer-safe / default-value pattern.
package types_test

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func boolp(v bool) *bool { return &v }

func emptyCRD() orktypes.CRDEntry {
	return orktypes.CRDEntry{}
}

// ── SetMaxQueueDepth ──────────────────────────────────────────────────────────

func TestSetMaxQueueDepth_UsesDefault(t *testing.T) {
	c := emptyCRD()
	assert.Equal(t, 10, c.SetMaxQueueDepth(10))
}

func TestSetMaxQueueDepth_UsesPerCRDValue(t *testing.T) {
	c := emptyCRD()
	c.Queue.MaxQueueDepth = 25
	assert.Equal(t, 25, c.SetMaxQueueDepth(10))
}

// ── SetWorkers ────────────────────────────────────────────────────────────────

func TestSetWorkers_UsesDefault(t *testing.T) {
	c := emptyCRD()
	assert.Equal(t, 4, c.SetWorkers(4))
}

func TestSetWorkers_UsesPerCRDValue(t *testing.T) {
	c := emptyCRD()
	c.Workers = 8
	assert.Equal(t, 8, c.SetWorkers(4))
}

// ── IsDynamic ─────────────────────────────────────────────────────────────────

func TestIsDynamic_ExplicitDynamic(t *testing.T) {
	c := emptyCRD()
	c.Mode = orktypes.CRDModeDynamic
	assert.True(t, c.IsDynamic())
}

func TestIsDynamic_ExplicitTyped(t *testing.T) {
	c := emptyCRD()
	c.Mode = orktypes.CRDModeTyped
	assert.False(t, c.IsDynamic())
}

func TestIsDynamic_NoLocationDefaultsDynamic(t *testing.T) {
	c := emptyCRD()
	// No mode, no location — defaults to dynamic
	assert.True(t, c.IsDynamic())
}

func TestIsDynamic_WithLocationDefaultsTyped(t *testing.T) {
	c := emptyCRD()
	c.APITypes.Location = "pkg/crds"
	assert.False(t, c.IsDynamic())
}

// ── IsEnabled ─────────────────────────────────────────────────────────────────

func TestIsEnabled_NilDefaultsTrue(t *testing.T) {
	c := emptyCRD()
	assert.True(t, c.IsEnabled())
}

func TestIsEnabled_ExplicitTrue(t *testing.T) {
	c := emptyCRD()
	c.Enabled = boolp(true)
	assert.True(t, c.IsEnabled())
}

func TestIsEnabled_ExplicitFalse(t *testing.T) {
	c := emptyCRD()
	c.Enabled = boolp(false)
	assert.False(t, c.IsEnabled())
}

// ── IsNamespaced ──────────────────────────────────────────────────────────────

func TestIsNamespaced_NilDefaultsTrue(t *testing.T) {
	c := emptyCRD()
	assert.True(t, c.IsNamespaced())
}

func TestIsNamespaced_ExplicitFalse(t *testing.T) {
	c := emptyCRD()
	c.Namespaced = boolp(false)
	assert.False(t, c.IsNamespaced())
}

// ── DefaultReconcile ──────────────────────────────────────────────────────────

func TestDefaultReconcile_NilDefaultsTrue(t *testing.T) {
	c := emptyCRD()
	assert.True(t, c.DefaultReconcile())
}

func TestDefaultReconcile_ExplicitFalse(t *testing.T) {
	c := emptyCRD()
	c.OperatorBox.Default = boolp(false)
	assert.False(t, c.DefaultReconcile())
}

// ── DefaultQueue ──────────────────────────────────────────────────────────────

func TestDefaultQueue_NilDefaultsFalse(t *testing.T) {
	c := emptyCRD()
	assert.False(t, c.DefaultQueue())
}

func TestDefaultQueue_ExplicitTrue(t *testing.T) {
	c := emptyCRD()
	c.Queue.Default = boolp(true)
	assert.True(t, c.DefaultQueue())
}

// ── IsHealthEnabled / IsInfoEnabled / IsEnabledAllEndpoints ──────────────────

func TestIsHealthEnabled_NilDefaultsTrue(t *testing.T) {
	c := emptyCRD()
	assert.True(t, c.IsHealthEnabled())
}

func TestIsHealthEnabled_ExplicitFalse(t *testing.T) {
	c := emptyCRD()
	c.Endpoints.Health = boolp(false)
	assert.False(t, c.IsHealthEnabled())
}

func TestIsInfoEnabled_NilDefaultsTrue(t *testing.T) {
	c := emptyCRD()
	assert.True(t, c.IsInfoEnabled())
}

func TestIsInfoEnabled_ExplicitFalse(t *testing.T) {
	c := emptyCRD()
	c.Endpoints.Info = boolp(false)
	assert.False(t, c.IsInfoEnabled())
}

func TestIsEnabledAllEndpoints_NilDefaultsTrue(t *testing.T) {
	c := emptyCRD()
	assert.True(t, c.IsEnabledAllEndpoints())
}

func TestIsEnabledAllEndpoints_ExplicitFalse(t *testing.T) {
	c := emptyCRD()
	c.Endpoints.Enabled = boolp(false)
	assert.False(t, c.IsEnabledAllEndpoints())
}

// ── GVK / GVR strings ────────────────────────────────────────────────────────

func TestGVKString(t *testing.T) {
	c := emptyCRD()
	c.GroupVersionKind = schema.GroupVersionKind{Group: "demo.io", Version: "v1", Kind: "Website"}
	assert.Equal(t, "demo.io/v1, Kind=Website", c.GVKString())
}

func TestGVRString(t *testing.T) {
	c := emptyCRD()
	c.GroupVersionResource = schema.GroupVersionResource{Group: "demo.io", Version: "v1", Resource: "websites"}
	assert.Equal(t, "demo.io/v1, Resource=websites", c.GVRString())
}

// ── HasValidationOrMutationRules ─────────────────────────────────────────────

func TestHasValidationOrMutationRules_Empty(t *testing.T) {
	c := emptyCRD()
	assert.False(t, c.HasValidationOrMutationRules())
}

func TestHasValidationRules_WithRules(t *testing.T) {
	c := emptyCRD()
	c.Validation = &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{{Field: "spec.image"}},
	}
	assert.True(t, c.HasValidationRules())
	assert.True(t, c.HasValidationOrMutationRules())
	assert.False(t, c.HasMutationRules())
}

func TestHasMutationRules_WithRules(t *testing.T) {
	c := emptyCRD()
	c.Mutation = &orktypes.MutationConfig{
		Rules: []orktypes.MutationRule{{Field: "spec.replicas", Default: "1"}},
	}
	assert.True(t, c.HasMutationRules())
	assert.True(t, c.HasValidationOrMutationRules())
	assert.False(t, c.HasValidationRules())
}

func TestHasMutationRules_NilMutation(t *testing.T) {
	c := emptyCRD()
	assert.False(t, c.HasMutationRules())
}

func TestHasValidationRules_NilValidation(t *testing.T) {
	c := emptyCRD()
	assert.False(t, c.HasValidationRules())
}

// ── HasOnCreate / HasOnReconcile / HasOnDelete / HasAnyHooks ─────────────────

func TestHasOnCreate_False(t *testing.T) {
	c := emptyCRD()
	assert.False(t, c.HasOnCreate())
}

func TestHasOnCreate_True(t *testing.T) {
	c := emptyCRD()
	c.OperatorBox.OnCreate = &orktypes.HookTemplates{}
	assert.True(t, c.HasOnCreate())
}

func TestHasOnReconcile_True(t *testing.T) {
	c := emptyCRD()
	c.OperatorBox.OnReconcile = &orktypes.HookTemplates{}
	assert.True(t, c.HasOnReconcile())
}

func TestHasOnDelete_True(t *testing.T) {
	c := emptyCRD()
	c.OperatorBox.OnDelete = &orktypes.HookTemplates{}
	assert.True(t, c.HasOnDelete())
}

func TestHasAnyHooks_None(t *testing.T) {
	c := emptyCRD()
	assert.False(t, c.HasAnyHookTemplates())
}

func TestHasAnyHooks_OnCreateOnly(t *testing.T) {
	c := emptyCRD()
	c.OperatorBox.OnCreate = &orktypes.HookTemplates{}
	assert.True(t, c.HasAnyHookTemplates())
}

// ── HasTemplates ─────────────────────────────────────────────────────────────

func TestHasTemplates_None(t *testing.T) {
	c := emptyCRD()
	assert.False(t, c.HasTemplates())
}

func TestHasTemplates_OnReconcile(t *testing.T) {
	c := emptyCRD()
	c.OperatorBox.OnReconcile = &orktypes.HookTemplates{}
	assert.True(t, c.HasTemplates())
}

// ── HasRollbackRules ──────────────────────────────────────────────────────────

func TestHasRollbackRules_None(t *testing.T) {
	c := emptyCRD()
	assert.False(t, c.HasRollbackRules())
}

func TestHasRollbackRules_ViaShorthand(t *testing.T) {
	c := emptyCRD()
	c.OperatorBox.RollBackOnError = true
	assert.True(t, c.HasRollbackRules())
}

func TestHasRollbackRules_ViaBlock(t *testing.T) {
	c := emptyCRD()
	c.OperatorBox.Rollback = &orktypes.RollbackBlock{}
	assert.True(t, c.HasRollbackRules())
}

// ── IsNotificationEnabled ────────────────────────────────────────────────────

func TestIsNotificationEnabled_NilDefaultsTrue(t *testing.T) {
	c := emptyCRD()
	assert.True(t, c.IsNotificationEnabled())
}

func TestIsNotificationEnabled_ExplicitFalse(t *testing.T) {
	c := emptyCRD()
	c.NotificationEnabled = boolp(false)
	assert.False(t, c.IsNotificationEnabled())
}

// ── ValidateMetricField ───────────────────────────────────────────────────────

func TestValidateMetricField_KnownFields(t *testing.T) {
	c := emptyCRD()
	known := []string{
		"metrics.workersBusyPercent",
		"metrics.workersIdlePercent",
		"metrics.queueDepth",
		"metrics.reconcileDurationP95Ms",
		"metrics.errorRatePercent",
	}
	for _, f := range known {
		assert.NoError(t, c.ValidateMetricField(f), "field %q should be valid", f)
	}
}

func TestValidateMetricField_Unknown(t *testing.T) {
	c := emptyCRD()
	err := c.ValidateMetricField("metrics.unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown autoscale metric field")
}

// ── InvolvedInConversion ─────────────────────────────────────────────────────

func TestInvolvedInConversion_Nil(t *testing.T) {
	c := emptyCRD()
	assert.False(t, c.InvolvedInConversion())
}

func TestInvolvedInConversion_WithPaths(t *testing.T) {
	c := emptyCRD()
	c.Conversion = &orktypes.CRDConversion{
		Paths: []orktypes.ConversionPath{{From: "v1", To: "v2"}},
	}
	assert.True(t, c.InvolvedInConversion())
}

func TestInvolvedInConversion_Participant(t *testing.T) {
	c := emptyCRD()
	c.Conversion = &orktypes.CRDConversion{Participant: true}
	assert.True(t, c.InvolvedInConversion())
}

// ── UpdateCRDCaBundle ─────────────────────────────────────────────────────────

func TestUpdateCRDCaBundle_Nil(t *testing.T) {
	c := emptyCRD()
	assert.False(t, c.UpdateCRDCaBundle())
}

func TestUpdateCRDCaBundle_True(t *testing.T) {
	c := emptyCRD()
	c.Conversion = &orktypes.CRDConversion{UpdateCRD: true}
	assert.True(t, c.UpdateCRDCaBundle())
}

// ── HasNamespaceRules ─────────────────────────────────────────────────────────

func TestHasNamespaceRules_Empty(t *testing.T) {
	c := emptyCRD()
	assert.False(t, c.HasNamespaceRules())
}

func TestHasNamespaceRules_WithAllowed(t *testing.T) {
	c := emptyCRD()
	c.AllowedNamespaces = orktypes.AllowedNamespaces{"apps"}
	assert.True(t, c.HasNamespaceRules())
}

func TestHasNamespaceRules_WithRestricted(t *testing.T) {
	c := emptyCRD()
	c.RestrictedNamespaces = orktypes.RestrictedNamespaces{"kube-system"}
	assert.True(t, c.HasNamespaceRules())
}

// ── AllowedNamespacesOnly / RestrictedNamespacesOnly ─────────────────────────

func TestAllowedNamespacesOnly_OnlyAllowed(t *testing.T) {
	c := emptyCRD()
	c.AllowedNamespaces = orktypes.AllowedNamespaces{"apps"}
	assert.True(t, c.AllowedNamespacesOnly())
	assert.False(t, c.RestrictedNamespacesOnly())
}

func TestRestrictedNamespacesOnly_OnlyRestricted(t *testing.T) {
	c := emptyCRD()
	c.RestrictedNamespaces = orktypes.RestrictedNamespaces{"kube-system"}
	assert.True(t, c.RestrictedNamespacesOnly())
	assert.False(t, c.AllowedNamespacesOnly())
}

func TestAllowedAndRestrictedBoth_NeitherOnly(t *testing.T) {
	c := emptyCRD()
	c.AllowedNamespaces = orktypes.AllowedNamespaces{"apps"}
	c.RestrictedNamespaces = orktypes.RestrictedNamespaces{"kube-system"}
	assert.False(t, c.AllowedNamespacesOnly())
	assert.False(t, c.RestrictedNamespacesOnly())
}

// ── AutoscaleEnabled / HasAutoscaleProfile ───────────────────────────────────

func TestAutoscaleEnabled_Nil(t *testing.T) {
	c := emptyCRD()
	assert.False(t, c.AutoscaleEnabled())
}

func TestAutoscaleEnabled_Set(t *testing.T) {
	c := emptyCRD()
	c.OperatorBox.Autoscale = &orktypes.AutoscaleSpec{}
	assert.True(t, c.AutoscaleEnabled())
}

func TestHasAutoscaleProfile_NoProfile(t *testing.T) {
	c := emptyCRD()
	c.OperatorBox.Autoscale = &orktypes.AutoscaleSpec{}
	assert.False(t, c.HasAutoscaleProfile())
}

func TestHasAutoscaleProfile_WithProfile(t *testing.T) {
	c := emptyCRD()
	c.OperatorBox.Autoscale = &orktypes.AutoscaleSpec{Profile: "burst"}
	assert.True(t, c.HasAutoscaleProfile())
	assert.Equal(t, "burst", c.AutoScaleProfile())
}

// ── WithHooksDecl / WithConstructorDecl ───────────────────────────────────────

func TestWithHooksDecl_Nil(t *testing.T) {
	c := emptyCRD()
	assert.False(t, c.WithHooksDecl())
}

func TestWithHooksDecl_WithLocation(t *testing.T) {
	c := emptyCRD()
	c.OperatorBox.Hooks = &orktypes.HookDeclaration{Location: "hooks/"}
	assert.True(t, c.WithHooksDecl())
}

func TestWithConstructorDecl_Nil(t *testing.T) {
	c := emptyCRD()
	assert.False(t, c.WithConstructorDecl())
}

func TestWithConstructorDecl_WithLocation(t *testing.T) {
	c := emptyCRD()
	c.OperatorBox.ConstructorDecl = &orktypes.ConstructorDeclaration{Location: "cmd/"}
	assert.True(t, c.WithConstructorDecl())
}
