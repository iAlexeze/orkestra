// Tests for EvaluateConditions, EvaluateOneCond, NavigateDotPath, NavigateRawPath,
// ResolveConditionOp (when.go).
package types_test

import (
	"errors"
	"testing"
	"time"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
)

// ── NavigateDotPath ───────────────────────────────────────────────────────────

func TestNavigateDotPath_EmptyPath(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	assert.Equal(t, "", orktypes.NavigateDotPath(data, ""))
}

func TestNavigateDotPath_TopLevel(t *testing.T) {
	data := map[string]interface{}{"phase": "Running"}
	assert.Equal(t, "Running", orktypes.NavigateDotPath(data, "phase"))
}

func TestNavigateDotPath_Nested(t *testing.T) {
	data := map[string]interface{}{
		"status": map[string]interface{}{"phase": "Ready"},
	}
	assert.Equal(t, "Ready", orktypes.NavigateDotPath(data, "status.phase"))
}

func TestNavigateDotPath_DeepNested(t *testing.T) {
	data := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": "deep",
			},
		},
	}
	assert.Equal(t, "deep", orktypes.NavigateDotPath(data, "a.b.c"))
}

func TestNavigateDotPath_MissingKey(t *testing.T) {
	data := map[string]interface{}{"phase": "Running"}
	assert.Equal(t, "", orktypes.NavigateDotPath(data, "status.phase"))
}

func TestNavigateDotPath_NonMapIntermediate(t *testing.T) {
	data := map[string]interface{}{"status": "running"}
	assert.Equal(t, "", orktypes.NavigateDotPath(data, "status.phase"))
}

func TestNavigateDotPath_IntValue(t *testing.T) {
	data := map[string]interface{}{"replicas": 3}
	assert.Equal(t, "3", orktypes.NavigateDotPath(data, "replicas"))
}

func TestNavigateDotPath_NilValue(t *testing.T) {
	data := map[string]interface{}{"key": nil}
	assert.Equal(t, "", orktypes.NavigateDotPath(data, "key"))
}

// ── NavigateRawPath ───────────────────────────────────────────────────────────

func TestNavigateRawPath_EmptyPath(t *testing.T) {
	data := map[string]interface{}{"key": "val"}
	assert.Nil(t, orktypes.NavigateRawPath(data, ""))
}

func TestNavigateRawPath_TopLevelString(t *testing.T) {
	data := map[string]interface{}{"phase": "Running"}
	assert.Equal(t, "Running", orktypes.NavigateRawPath(data, "phase"))
}

func TestNavigateRawPath_TopLevelMap(t *testing.T) {
	inner := map[string]interface{}{"x": 1}
	data := map[string]interface{}{"spec": inner}
	assert.Equal(t, inner, orktypes.NavigateRawPath(data, "spec"))
}

func TestNavigateRawPath_TopLevelSlice(t *testing.T) {
	slice := []interface{}{"a", "b"}
	data := map[string]interface{}{"items": slice}
	assert.Equal(t, slice, orktypes.NavigateRawPath(data, "items"))
}

func TestNavigateRawPath_MissingKey(t *testing.T) {
	data := map[string]interface{}{}
	assert.Nil(t, orktypes.NavigateRawPath(data, "missing"))
}

func TestNavigateRawPath_BoolValue(t *testing.T) {
	data := map[string]interface{}{"enabled": true}
	assert.Equal(t, true, orktypes.NavigateRawPath(data, "enabled"))
}

// ── ResolveConditionOp ────────────────────────────────────────────────────────

func TestResolveConditionOp_Equals(t *testing.T) {
	c := orktypes.Condition{Equals: "Ready"}
	op, val := orktypes.ResolveConditionOp(c)
	assert.Equal(t, orktypes.ConditionEquals, op)
	assert.Equal(t, "Ready", val)
}

func TestResolveConditionOp_NotEquals(t *testing.T) {
	c := orktypes.Condition{NotEquals: "Failed"}
	op, val := orktypes.ResolveConditionOp(c)
	assert.Equal(t, orktypes.ConditionNotEquals, op)
	assert.Equal(t, "Failed", val)
}

func TestResolveConditionOp_Prefix(t *testing.T) {
	c := orktypes.Condition{Prefix: "prod-"}
	op, val := orktypes.ResolveConditionOp(c)
	assert.Equal(t, orktypes.ConditionPrefix, op)
	assert.Equal(t, "prod-", val)
}

func TestResolveConditionOp_Suffix(t *testing.T) {
	c := orktypes.Condition{Suffix: "-v2"}
	op, val := orktypes.ResolveConditionOp(c)
	assert.Equal(t, orktypes.ConditionSuffix, op)
	assert.Equal(t, "-v2", val)
}

func TestResolveConditionOp_Contains(t *testing.T) {
	c := orktypes.Condition{Contains: "error"}
	op, val := orktypes.ResolveConditionOp(c)
	assert.Equal(t, orktypes.ConditionContains, op)
	assert.Equal(t, "error", val)
}

func TestResolveConditionOp_GreaterThan(t *testing.T) {
	c := orktypes.Condition{GreaterThan: "10"}
	op, val := orktypes.ResolveConditionOp(c)
	assert.Equal(t, orktypes.ConditionGt, op)
	assert.Equal(t, "10", val)
}

func TestResolveConditionOp_LessThan(t *testing.T) {
	c := orktypes.Condition{LessThan: "5"}
	op, val := orktypes.ResolveConditionOp(c)
	assert.Equal(t, orktypes.ConditionLt, op)
	assert.Equal(t, "5", val)
}

func TestResolveConditionOp_ExplicitOperator(t *testing.T) {
	c := orktypes.Condition{Operator: orktypes.ConditionIn, Value: "a,b,c"}
	op, val := orktypes.ResolveConditionOp(c)
	assert.Equal(t, orktypes.ConditionIn, op)
	assert.Equal(t, "a,b,c", val)
}

func TestResolveConditionOp_ValueShorthand(t *testing.T) {
	c := orktypes.Condition{Value: "Running"}
	op, val := orktypes.ResolveConditionOp(c)
	assert.Equal(t, orktypes.ConditionEquals, op)
	assert.Equal(t, "Running", val)
}

func TestResolveConditionOp_Default(t *testing.T) {
	c := orktypes.Condition{Field: "status.phase"}
	op, val := orktypes.ResolveConditionOp(c)
	assert.Equal(t, orktypes.ConditionExists, op)
	assert.Equal(t, "", val)
}

// ── EvaluateOneCond — field operators ────────────────────────────────────────

func data(kv ...interface{}) map[string]interface{} {
	m := make(map[string]interface{}, len(kv)/2)
	for i := 0; i < len(kv)-1; i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	return m
}

func TestEvaluateOneCond_Equals_Match(t *testing.T) {
	d := data("phase", "Running")
	c := orktypes.Condition{Field: "phase", Equals: "Running"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Equals_NoMatch(t *testing.T) {
	d := data("phase", "Pending")
	c := orktypes.Condition{Field: "phase", Equals: "Running"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_NotEquals(t *testing.T) {
	d := data("phase", "Pending")
	c := orktypes.Condition{Field: "phase", NotEquals: "Running"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Contains(t *testing.T) {
	d := data("message", "connection refused")
	c := orktypes.Condition{Field: "message", Contains: "refused"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Contains_NoMatch(t *testing.T) {
	d := data("message", "all good")
	c := orktypes.Condition{Field: "message", Contains: "error"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Prefix(t *testing.T) {
	d := data("name", "prod-app")
	c := orktypes.Condition{Field: "name", Prefix: "prod-"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Prefix_NoMatch(t *testing.T) {
	d := data("name", "dev-app")
	c := orktypes.Condition{Field: "name", Prefix: "prod-"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Suffix(t *testing.T) {
	d := data("name", "app-v2")
	c := orktypes.Condition{Field: "name", Suffix: "-v2"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Exists_Present(t *testing.T) {
	d := data("phase", "Running")
	c := orktypes.Condition{Field: "phase", Operator: orktypes.ConditionExists}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Exists_Missing(t *testing.T) {
	d := map[string]interface{}{}
	c := orktypes.Condition{Field: "phase", Operator: orktypes.ConditionExists}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_NotExists_Missing(t *testing.T) {
	d := map[string]interface{}{}
	c := orktypes.Condition{Field: "phase", Operator: orktypes.ConditionNotExists}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_NotExists_Present(t *testing.T) {
	d := data("phase", "Running")
	c := orktypes.Condition{Field: "phase", Operator: orktypes.ConditionNotExists}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Gt_Pass(t *testing.T) {
	d := data("replicas", "5")
	c := orktypes.Condition{Field: "replicas", GreaterThan: "3"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Gt_Fail(t *testing.T) {
	d := data("replicas", "2")
	c := orktypes.Condition{Field: "replicas", GreaterThan: "3"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Gt_AbsentFieldTreatedAsZero(t *testing.T) {
	d := map[string]interface{}{}
	c := orktypes.Condition{Field: "count", GreaterThan: "0"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Lt_Pass(t *testing.T) {
	d := data("cpu", "50")
	c := orktypes.Condition{Field: "cpu", LessThan: "80"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_In_Match(t *testing.T) {
	d := data("env", "prod")
	c := orktypes.Condition{Field: "env", Operator: orktypes.ConditionIn, Value: "dev,staging,prod"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_In_NoMatch(t *testing.T) {
	d := data("env", "test")
	c := orktypes.Condition{Field: "env", Operator: orktypes.ConditionIn, Value: "dev,staging,prod"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_NotIn_Match(t *testing.T) {
	d := data("env", "canary")
	c := orktypes.Condition{Field: "env", NotIn: "dev,staging,prod"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_NotIn_NoMatch(t *testing.T) {
	d := data("env", "prod")
	c := orktypes.Condition{Field: "env", NotIn: "dev,staging,prod"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Gte_EqualBound(t *testing.T) {
	d := data("replicas", "3")
	c := orktypes.Condition{Field: "replicas", GreaterThanOrEqual: "3"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Gte_BelowBound(t *testing.T) {
	d := data("replicas", "2")
	c := orktypes.Condition{Field: "replicas", GreaterThanOrEqual: "3"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Lte_EqualBound(t *testing.T) {
	d := data("cpu", "80")
	c := orktypes.Condition{Field: "cpu", LessThanOrEqual: "80"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Lte_AboveBound(t *testing.T) {
	d := data("cpu", "81")
	c := orktypes.Condition{Field: "cpu", LessThanOrEqual: "80"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestResolveConditionOp_Min(t *testing.T) {
	op, val := orktypes.ResolveConditionOp(orktypes.Condition{Min: "1"})
	assert.Equal(t, orktypes.ConditionGte, op)
	assert.Equal(t, "1", val)
}

func TestResolveConditionOp_Max(t *testing.T) {
	op, val := orktypes.ResolveConditionOp(orktypes.Condition{Max: "10"})
	assert.Equal(t, orktypes.ConditionLte, op)
	assert.Equal(t, "10", val)
}

func TestEvaluateOneCond_Min_EqualBound(t *testing.T) {
	d := data("replicas", "1")
	c := orktypes.Condition{Field: "replicas", Min: "1"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Min_BelowBound(t *testing.T) {
	d := data("replicas", "0")
	c := orktypes.Condition{Field: "replicas", Min: "1"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Max_EqualBound(t *testing.T) {
	d := data("cpu", "80")
	c := orktypes.Condition{Field: "cpu", Max: "80"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Max_AboveBound(t *testing.T) {
	d := data("cpu", "81")
	c := orktypes.Condition{Field: "cpu", Max: "80"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Between_Inside(t *testing.T) {
	d := data("replicas", "5")
	c := orktypes.Condition{Field: "replicas", Between: "1,10"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Between_AtBounds(t *testing.T) {
	c := orktypes.Condition{Field: "replicas", Between: "1,10"}
	assert.True(t, orktypes.EvaluateOneCond(data("replicas", "1"), c, nil))
	assert.True(t, orktypes.EvaluateOneCond(data("replicas", "10"), c, nil))
}

func TestEvaluateOneCond_Between_Outside(t *testing.T) {
	d := data("replicas", "11")
	c := orktypes.Condition{Field: "replicas", Between: "1,10"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Between_MalformedRange(t *testing.T) {
	d := data("replicas", "5")
	c := orktypes.Condition{Field: "replicas", Between: "not,numbers"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_NotBetween_Outside(t *testing.T) {
	d := data("replicas", "11")
	c := orktypes.Condition{Field: "replicas", NotBetween: "1,10"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_NotBetween_Inside(t *testing.T) {
	d := data("replicas", "5")
	c := orktypes.Condition{Field: "replicas", NotBetween: "1,10"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_NotContains_Match(t *testing.T) {
	d := data("image", "myorg/app:latest")
	c := orktypes.Condition{Field: "image", NotContains: "docker.io"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_NotContains_NoMatch(t *testing.T) {
	d := data("image", "docker.io/app:latest")
	c := orktypes.Condition{Field: "image", NotContains: "docker.io"}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Regex_Match(t *testing.T) {
	d := data("name", "app-prod-01")
	c := orktypes.Condition{Field: "name", Regex: `^app-\w+-\d+$`}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Regex_NoMatch(t *testing.T) {
	d := data("name", "APP")
	c := orktypes.Condition{Field: "name", Regex: `^app-\w+-\d+$`}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Regex_InvalidPattern(t *testing.T) {
	d := data("name", "app")
	c := orktypes.Condition{Field: "name", Regex: `(unclosed`}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_TypeOf_Map(t *testing.T) {
	d := map[string]interface{}{
		"spec": map[string]interface{}{"key": "val"},
	}
	c := orktypes.Condition{Field: "spec", Operator: orktypes.ConditionTypeMap}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_TypeOf_List(t *testing.T) {
	d := map[string]interface{}{
		"items": []interface{}{"a", "b"},
	}
	c := orktypes.Condition{Field: "items", Operator: orktypes.ConditionTypeList}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_TypeOf_String(t *testing.T) {
	d := data("phase", "Running")
	c := orktypes.Condition{Field: "phase", Operator: orktypes.ConditionTypeString}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_TypeOf_Bool(t *testing.T) {
	d := map[string]interface{}{"enabled": true}
	c := orktypes.Condition{Field: "enabled", Operator: orktypes.ConditionTypeBool}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_TypeOf_Number(t *testing.T) {
	d := map[string]interface{}{"replicas": 3}
	c := orktypes.Condition{Field: "replicas", Operator: orktypes.ConditionTypeNumber}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_TypeOf_Null(t *testing.T) {
	d := map[string]interface{}{"field": nil}
	c := orktypes.Condition{Field: "field", Operator: orktypes.ConditionTypeNull}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_TypeOf_Explicit(t *testing.T) {
	d := data("phase", "Running")
	c := orktypes.Condition{Field: "phase", Operator: orktypes.ConditionTypeOf, Value: "string"}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Unique_AlwaysTrue(t *testing.T) {
	d := data("name", "foo")
	c := orktypes.Condition{Field: "name", Operator: orktypes.ConditionUnique}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

// ── EvaluateConditions — allOf / anyOf ──────────────────────────────────────────────

func TestEvaluateConditions_EmptyBothPasses(t *testing.T) {
	assert.True(t, orktypes.EvaluateConditions(nil, nil, nil, nil))
}

func TestEvaluateConditions_AllOfAllPass(t *testing.T) {
	d := data("phase", "Running", "env", "prod")
	allOf := []orktypes.Condition{
		{Field: "phase", Equals: "Running"},
		{Field: "env", Equals: "prod"},
	}
	assert.True(t, orktypes.EvaluateConditions(d, allOf, nil, nil))
}

func TestEvaluateConditions_AllOfOneFails(t *testing.T) {
	d := data("phase", "Pending", "env", "prod")
	allOf := []orktypes.Condition{
		{Field: "phase", Equals: "Running"},
		{Field: "env", Equals: "prod"},
	}
	assert.False(t, orktypes.EvaluateConditions(d, allOf, nil, nil))
}

func TestEvaluateConditions_AnyOfOneMatches(t *testing.T) {
	d := data("phase", "Failed")
	anyOf := []orktypes.Condition{
		{Field: "phase", Equals: "Failed"},
		{Field: "phase", Equals: "Succeeded"},
	}
	assert.True(t, orktypes.EvaluateConditions(d, nil, anyOf, nil))
}

func TestEvaluateConditions_AnyOfNoneMatch(t *testing.T) {
	d := data("phase", "Running")
	anyOf := []orktypes.Condition{
		{Field: "phase", Equals: "Failed"},
		{Field: "phase", Equals: "Succeeded"},
	}
	assert.False(t, orktypes.EvaluateConditions(d, nil, anyOf, nil))
}

func TestEvaluateConditions_BothMustPass(t *testing.T) {
	d := data("env", "prod", "phase", "Failed")
	allOf := []orktypes.Condition{{Field: "env", Equals: "prod"}}
	anyOf := []orktypes.Condition{
		{Field: "phase", Equals: "Failed"},
		{Field: "phase", Equals: "Succeeded"},
	}
	assert.True(t, orktypes.EvaluateConditions(d, allOf, anyOf, nil))
}

func TestEvaluateConditions_AllOfPassAnyOfFails(t *testing.T) {
	d := data("env", "prod", "phase", "Running")
	allOf := []orktypes.Condition{{Field: "env", Equals: "prod"}}
	anyOf := []orktypes.Condition{
		{Field: "phase", Equals: "Failed"},
		{Field: "phase", Equals: "Succeeded"},
	}
	assert.False(t, orktypes.EvaluateConditions(d, allOf, anyOf, nil))
}

// ── EvaluateOneCond — cron window injection via _cronWindows ─────────────────

func TestEvaluateOneCond_CronWindowInjected_True(t *testing.T) {
	d := map[string]interface{}{
		"_cronWindows": map[string]interface{}{
			"0 9 * * 1-5": "true",
		},
	}
	c := orktypes.Condition{Cron: "0 9 * * 1-5", Duration: orktypes.Duration{Duration: time.Hour}}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_CronWindowInjected_False(t *testing.T) {
	d := map[string]interface{}{
		"_cronWindows": map[string]interface{}{
			"0 9 * * 1-5": "false",
		},
	}
	c := orktypes.Condition{Cron: "0 9 * * 1-5", Duration: orktypes.Duration{Duration: time.Hour}}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

// ── EvaluateOneCond — unique operator via _uniquenessChecker injection ───────
//
// The concrete live-list-backed checker lives in
// pkg/runtime/reconciler/uniqueness.go; here we inject a fake under the
// same "_uniquenessChecker" key template.Resolver.WithUniquenessChecker
// uses, matching the _cronWindows injection convention above.

type fakeUniqueChecker struct {
	unique bool
	err    error
}

func (f *fakeUniqueChecker) IsUnique(field, value, selfNamespace, selfName string) (bool, error) {
	return f.unique, f.err
}

func TestEvaluateOneCond_Unique_NoCheckerInjected_AlwaysPasses(t *testing.T) {
	d := data("domain", "a.example.com")
	c := orktypes.Condition{Field: "domain", Operator: orktypes.ConditionUnique}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Unique_CheckerReportsUnique(t *testing.T) {
	d := data("domain", "a.example.com")
	d["_uniquenessChecker"] = &fakeUniqueChecker{unique: true}
	c := orktypes.Condition{Field: "domain", Operator: orktypes.ConditionUnique}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Unique_CheckerReportsDuplicate(t *testing.T) {
	d := data("domain", "shared.example.com")
	d["_uniquenessChecker"] = &fakeUniqueChecker{unique: false}
	c := orktypes.Condition{Field: "domain", Operator: orktypes.ConditionUnique}
	assert.False(t, orktypes.EvaluateOneCond(d, c, nil))
}

func TestEvaluateOneCond_Unique_CheckerErrors_FailsOpen(t *testing.T) {
	d := data("domain", "a.example.com")
	d["_uniquenessChecker"] = &fakeUniqueChecker{err: errors.New("list failed")}
	c := orktypes.Condition{Field: "domain", Operator: orktypes.ConditionUnique}
	assert.True(t, orktypes.EvaluateOneCond(d, c, nil))
}

// ── EvaluateOneCond — time window (via Condition.Time) ───────────────────────

func TestEvaluateOneCond_TimeWindow_WithinRange(t *testing.T) {
	// Use a window that is definitely open right now: 00:00–23:59
	c := orktypes.Condition{
		Time: &orktypes.TimeWindow{After: "00:00", Before: "23:59"},
	}
	assert.True(t, orktypes.EvaluateOneCond(nil, c, nil))
}

func TestEvaluateOneCond_TimeWindow_OnlyAfter_Pass(t *testing.T) {
	// After 00:00 — always true
	c := orktypes.Condition{
		Time: &orktypes.TimeWindow{After: "00:00"},
	}
	assert.True(t, orktypes.EvaluateOneCond(nil, c, nil))
}

func TestEvaluateOneCond_TimeWindow_OnlyBefore_Pass(t *testing.T) {
	// Before 23:59 — always true
	c := orktypes.Condition{
		Time: &orktypes.TimeWindow{Before: "23:59"},
	}
	assert.True(t, orktypes.EvaluateOneCond(nil, c, nil))
}

func TestEvaluateOneCond_TimeWindow_InvalidAfter(t *testing.T) {
	c := orktypes.Condition{
		Time: &orktypes.TimeWindow{After: "not-a-time"},
	}
	assert.False(t, orktypes.EvaluateOneCond(nil, c, nil))
}

func TestEvaluateOneCond_TimeWindow_InvalidBefore(t *testing.T) {
	c := orktypes.Condition{
		Time: &orktypes.TimeWindow{Before: "not-a-time"},
	}
	assert.False(t, orktypes.EvaluateOneCond(nil, c, nil))
}

// ── EvaluateOneCond — day of week (via Condition.DayOfWeek) ──────────────────

func TestEvaluateOneCond_DayOfWeek_InMatchesAllDays(t *testing.T) {
	// In: all 7 days — always true regardless of current weekday
	c := orktypes.Condition{
		DayOfWeek: &orktypes.DayOfWeekCondition{
			In: []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
		},
	}
	assert.True(t, orktypes.EvaluateOneCond(nil, c, nil))
}

func TestEvaluateOneCond_DayOfWeek_NotInEmpty_ReturnsFalse(t *testing.T) {
	// Neither In nor NotIn set — evalDayOfWeek returns false
	c := orktypes.Condition{
		DayOfWeek: &orktypes.DayOfWeekCondition{},
	}
	assert.False(t, orktypes.EvaluateOneCond(nil, c, nil))
}

func TestEvaluateOneCond_DayOfWeek_NotInAllDays_ReturnsFalse(t *testing.T) {
	// Excluding all days — always false
	c := orktypes.Condition{
		DayOfWeek: &orktypes.DayOfWeekCondition{
			NotIn: []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
		},
	}
	assert.False(t, orktypes.EvaluateOneCond(nil, c, nil))
}

func TestEvaluateOneCond_DayOfWeek_NotInNoMatch_ReturnsTrue(t *testing.T) {
	// NotIn: ["Funday"] — "Funday" never matches any real weekday, so always true
	c := orktypes.Condition{
		DayOfWeek: &orktypes.DayOfWeekCondition{
			NotIn: []string{"Funday"},
		},
	}
	assert.True(t, orktypes.EvaluateOneCond(nil, c, nil))
}

func TestEvaluateOneCond_DayOfWeek_WeekdayTrue_OnWeekday(t *testing.T) {
	b := true
	c := orktypes.Condition{DayOfWeek: &orktypes.DayOfWeekCondition{Weekday: &b}}
	// Monday is a weekday — must pass
	got := orktypes.EvalDayOfWeekAt(c.DayOfWeek, mustParseTime("2026-07-13T12:00:00Z")) // Monday
	assert.True(t, got)
}

func TestEvaluateOneCond_DayOfWeek_WeekdayTrue_OnWeekend(t *testing.T) {
	b := true
	c := orktypes.Condition{DayOfWeek: &orktypes.DayOfWeekCondition{Weekday: &b}}
	// Saturday is not a weekday — must fail
	got := orktypes.EvalDayOfWeekAt(c.DayOfWeek, mustParseTime("2026-07-12T12:00:00Z")) // Saturday
	assert.False(t, got)
}

func TestEvaluateOneCond_DayOfWeek_WeekendTrue_OnWeekend(t *testing.T) {
	b := true
	c := orktypes.Condition{DayOfWeek: &orktypes.DayOfWeekCondition{Weekend: &b}}
	got := orktypes.EvalDayOfWeekAt(c.DayOfWeek, mustParseTime("2026-07-12T12:00:00Z")) // Saturday
	assert.True(t, got)
}

func TestEvaluateOneCond_DayOfWeek_WeekendTrue_OnWeekday(t *testing.T) {
	b := true
	c := orktypes.Condition{DayOfWeek: &orktypes.DayOfWeekCondition{Weekend: &b}}
	got := orktypes.EvalDayOfWeekAt(c.DayOfWeek, mustParseTime("2026-07-14T12:00:00Z")) // Monday
	assert.False(t, got)
}

func TestEvaluateOneCond_DayOfWeek_WeekdayAndWeekend_MutuallyExclusive(t *testing.T) {
	b := true
	// weekday: true on a weekday, weekend: true on a weekend — never both true simultaneously
	mon := mustParseTime("2026-07-13T12:00:00Z")
	sat := mustParseTime("2026-07-12T12:00:00Z")
	weekdayCond := &orktypes.DayOfWeekCondition{Weekday: &b}
	weekendCond := &orktypes.DayOfWeekCondition{Weekend: &b}
	assert.True(t, orktypes.EvalDayOfWeekAt(weekdayCond, mon))
	assert.False(t, orktypes.EvalDayOfWeekAt(weekendCond, mon))
	assert.False(t, orktypes.EvalDayOfWeekAt(weekdayCond, sat))
	assert.True(t, orktypes.EvalDayOfWeekAt(weekendCond, sat))
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// ── EvaluateOneCond — cron (stateless fallback) ───────────────────────────────

func TestEvaluateOneCond_CronStateless_InvalidExpr(t *testing.T) {
	// Invalid cron expression → false
	c := orktypes.Condition{Cron: "not-a-cron", Duration: orktypes.Duration{Duration: time.Hour}}
	assert.False(t, orktypes.EvaluateOneCond(nil, c, nil))
}
