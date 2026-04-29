// Tests for ParseRotationDuration and NeedsRotation (secret_rotation.go).
package types_test

import (
	"testing"
	"time"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── ParseRotationDuration ─────────────────────────────────────────────────────

func TestParseRotationDuration_Empty(t *testing.T) {
	_, err := orktypes.ParseRotationDuration("")
	assert.Error(t, err)
}

func TestParseRotationDuration_Seconds(t *testing.T) {
	d, err := orktypes.ParseRotationDuration("30s")
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, d)
}

func TestParseRotationDuration_Minutes(t *testing.T) {
	d, err := orktypes.ParseRotationDuration("5m")
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, d)
}

func TestParseRotationDuration_Hours(t *testing.T) {
	d, err := orktypes.ParseRotationDuration("12h")
	require.NoError(t, err)
	assert.Equal(t, 12*time.Hour, d)
}

func TestParseRotationDuration_Days(t *testing.T) {
	d, err := orktypes.ParseRotationDuration("10d")
	require.NoError(t, err)
	assert.Equal(t, 10*24*time.Hour, d)
}

func TestParseRotationDuration_Weeks(t *testing.T) {
	d, err := orktypes.ParseRotationDuration("2w")
	require.NoError(t, err)
	assert.Equal(t, 14*24*time.Hour, d)
}

func TestParseRotationDuration_Months(t *testing.T) {
	d, err := orktypes.ParseRotationDuration("3mo")
	require.NoError(t, err)
	assert.Equal(t, 90*24*time.Hour, d)
}

func TestParseRotationDuration_Years(t *testing.T) {
	d, err := orktypes.ParseRotationDuration("1y")
	require.NoError(t, err)
	assert.Equal(t, 365*24*time.Hour, d)
}

func TestParseRotationDuration_FractionalYear(t *testing.T) {
	d, err := orktypes.ParseRotationDuration("0.5y")
	require.NoError(t, err)
	// 0.5 * 365 days
	assert.Equal(t, time.Duration(0.5*float64(365*24*time.Hour)), d)
}

func TestParseRotationDuration_FractionalMonths(t *testing.T) {
	d, err := orktypes.ParseRotationDuration("1.5mo")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(1.5*float64(30*24*time.Hour)), d)
}

func TestParseRotationDuration_InvalidYear(t *testing.T) {
	_, err := orktypes.ParseRotationDuration("xy")
	assert.Error(t, err)
}

func TestParseRotationDuration_InvalidDay(t *testing.T) {
	_, err := orktypes.ParseRotationDuration("xd")
	assert.Error(t, err)
}

func TestParseRotationDuration_InvalidGoSyntax(t *testing.T) {
	_, err := orktypes.ParseRotationDuration("not-a-duration")
	assert.Error(t, err)
}

// ── NeedsRotation ─────────────────────────────────────────────────────────────

func TestNeedsRotation_NoRotateAfter(t *testing.T) {
	past := time.Now().Add(-400 * 24 * time.Hour).Format(time.RFC3339)
	assert.False(t, orktypes.NeedsRotation(past, ""))
}

func TestNeedsRotation_NotYetExpired(t *testing.T) {
	recent := time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	assert.False(t, orktypes.NeedsRotation(recent, "90d"))
}

func TestNeedsRotation_Expired(t *testing.T) {
	old := time.Now().Add(-91 * 24 * time.Hour).Format(time.RFC3339)
	assert.True(t, orktypes.NeedsRotation(old, "90d"))
}

func TestNeedsRotation_ExactThreshold(t *testing.T) {
	// Just at the boundary: age == threshold means rotate (>=)
	exact := time.Now().Add(-365 * 24 * time.Hour).Format(time.RFC3339)
	assert.True(t, orktypes.NeedsRotation(exact, "1y"))
}

func TestNeedsRotation_InvalidTimestamp(t *testing.T) {
	// Unparseable annotation → rotate to be safe
	assert.True(t, orktypes.NeedsRotation("not-a-timestamp", "30d"))
}

func TestNeedsRotation_InvalidDuration(t *testing.T) {
	// Invalid duration → do not rotate unexpectedly
	recent := time.Now().Add(-5 * 24 * time.Hour).Format(time.RFC3339)
	assert.False(t, orktypes.NeedsRotation(recent, "invalid"))
}

func TestNeedsRotation_WeekRotation(t *testing.T) {
	old := time.Now().Add(-8 * 24 * time.Hour).Format(time.RFC3339)
	assert.True(t, orktypes.NeedsRotation(old, "1w"))
}

func TestNeedsRotation_MonthRotation_NotYet(t *testing.T) {
	recent := time.Now().Add(-15 * 24 * time.Hour).Format(time.RFC3339)
	assert.False(t, orktypes.NeedsRotation(recent, "1mo"))
}
