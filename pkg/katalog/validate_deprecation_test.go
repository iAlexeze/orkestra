package katalog

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func katalogWithDeprecation(d *orktypes.KatalogDeprecation) *Katalog {
	return &Katalog{
		metadata: orktypes.KatalogMeta{
			Deprecation: d,
		},
	}
}

func TestValidateDeprecation_Nil(t *testing.T) {
	k := katalogWithDeprecation(nil)
	assert.NoError(t, k.validateDeprecation())
}

func TestValidateDeprecation_MessageOnly(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
	})
	assert.NoError(t, k.validateDeprecation())
}

func TestValidateDeprecation_MissingMessage(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		MigratedTo: "mypattern:v2",
	})
	err := k.validateDeprecation()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "message is required")
}

func TestValidateDeprecation_ValidTimeline(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Timeline: &orktypes.DeprecationTimeline{
			From: "2026-01-01",
			To:   "2027-01-01",
		},
	})
	assert.NoError(t, k.validateDeprecation())
}

func TestValidateDeprecation_BadFromDate(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Timeline: &orktypes.DeprecationTimeline{
			From: "not-a-date",
			To:   "2027-01-01",
		},
	})
	err := k.validateDeprecation()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeline.from")
	assert.Contains(t, err.Error(), "YYYY-MM-DD")
}

func TestValidateDeprecation_BadToDate(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Timeline: &orktypes.DeprecationTimeline{
			From: "2026-01-01",
			To:   "01/01/2027",
		},
	})
	err := k.validateDeprecation()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeline.to")
	assert.Contains(t, err.Error(), "YYYY-MM-DD")
}

func TestValidateDeprecation_FromAfterTo(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Timeline: &orktypes.DeprecationTimeline{
			From: "2027-06-01",
			To:   "2027-01-01",
		},
	})
	err := k.validateDeprecation()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "from")
	assert.Contains(t, err.Error(), "before")
	assert.Contains(t, err.Error(), "to")
}

func TestValidateDeprecation_FromEqualTo(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Timeline: &orktypes.DeprecationTimeline{
			From: "2027-01-01",
			To:   "2027-01-01",
		},
	})
	err := k.validateDeprecation()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "before")
}

func TestValidateDeprecation_TimelineFromOnly(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Timeline: &orktypes.DeprecationTimeline{
			From: "2026-01-01",
		},
	})
	assert.NoError(t, k.validateDeprecation())
}

func TestValidateDeprecation_TimelineToOnly(t *testing.T) {
	k := katalogWithDeprecation(&orktypes.KatalogDeprecation{
		Message: "use v2",
		Timeline: &orktypes.DeprecationTimeline{
			To: "2027-01-01",
		},
	})
	assert.NoError(t, k.validateDeprecation())
}
