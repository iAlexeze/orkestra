package katalog

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckDeprecationPolicy_NilBlock(t *testing.T) {
	k := katalogWithDeprecation(nil)
	assert.NoError(t, k.CheckDeprecationPolicy())
}

func TestCheckDeprecationPolicy_NoTimeline_NoAccept(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
	})
	err := k.CheckDeprecationPolicy()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "beforeEol: true")
	assert.Contains(t, err.Error(), "deprecated")
}

func TestCheckDeprecationPolicy_NoTimeline_Accepted(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Accept:  &orktypes.DeprecationAccept{BeforeEol: true},
	})
	assert.NoError(t, k.CheckDeprecationPolicy())
}

func TestCheckDeprecationPolicy_WarningWindow_NoAccept(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Timeline: &orktypes.DeprecationTimeline{
			From: "2020-01-01", // in the past — window is open
			To:   "2099-01-01",
		},
	})
	err := k.CheckDeprecationPolicy()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "beforeEol: true")
}

func TestCheckDeprecationPolicy_WarningWindow_Accepted(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Timeline: &orktypes.DeprecationTimeline{
			From: "2020-01-01",
			To:   "2099-01-01",
		},
		Accept: &orktypes.DeprecationAccept{BeforeEol: true},
	})
	assert.NoError(t, k.CheckDeprecationPolicy())
}

func TestCheckDeprecationPolicy_BeforeFrom_NoAcceptNeeded(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Timeline: &orktypes.DeprecationTimeline{
			From: "2099-01-01", // not yet reached
			To:   "2099-06-01",
		},
	})
	assert.NoError(t, k.CheckDeprecationPolicy())
}

func TestCheckDeprecationPolicy_EOL_NeitherAccepted(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Timeline: &orktypes.DeprecationTimeline{
			From: "2020-01-01",
			To:   "2020-06-01", // past EOL
		},
	})
	err := k.CheckDeprecationPolicy()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "end-of-life")
	assert.Contains(t, err.Error(), "beforeEol: true")
	assert.Contains(t, err.Error(), "eol: true")
}

func TestCheckDeprecationPolicy_EOL_OnlyBeforeEol(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Timeline: &orktypes.DeprecationTimeline{
			From: "2020-01-01",
			To:   "2020-06-01",
		},
		Accept: &orktypes.DeprecationAccept{BeforeEol: true},
	})
	err := k.CheckDeprecationPolicy()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "eol: true")
}

func TestCheckDeprecationPolicy_EOL_BothAccepted(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Timeline: &orktypes.DeprecationTimeline{
			From: "2020-01-01",
			To:   "2020-06-01",
		},
		Accept: &orktypes.DeprecationAccept{BeforeEol: true, Eol: true},
	})
	assert.NoError(t, k.CheckDeprecationPolicy())
}

func TestCheckDeprecationPolicy_EOL_OnlyEolWithoutBeforeEol(t *testing.T) {
	// eol: true alone is insufficient — beforeEol must also be true
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Timeline: &orktypes.DeprecationTimeline{
			From: "2020-01-01",
			To:   "2020-06-01",
		},
		Accept: &orktypes.DeprecationAccept{Eol: true},
	})
	err := k.CheckDeprecationPolicy()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "beforeEol: true")
}

func TestCheckDeprecationPolicy_MigrationTargetInError(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message:    "use v2",
		MigratedTo: "my-pattern:v2.0.0",
	})
	err := k.CheckDeprecationPolicy()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "my-pattern:v2.0.0")
}
