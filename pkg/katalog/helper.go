package katalog

import (
	"github.com/orkspace/orkestra/pkg/utils"
)

// ── utils aliases ────────────────────────────────────────────────────────────
// Import utils once here. All other files in this package use these names
// directly — no per-file utils import needed.

var (
	// colors / styles
	yellow = utils.Yellow
	red    = utils.Red

	// marks and icons
	failureMark = utils.FailureMark

	// duration parsing
	parseTimeDuration = utils.ParseTimeDuration

	// helpers
	toStringSet = utils.ToStringSet
)
