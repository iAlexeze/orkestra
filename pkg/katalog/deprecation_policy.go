package katalog

import (
	"fmt"
	"time"
)

// CheckDeprecationPolicy enforces runtime deprecation gates for ork run and
// ork gate. It is called after ValidateConfig — not inside it.
//
// A deprecated Katalog always blocks when run directly. To run a deprecated
// Katalog, import it via a Komposer and add it to lifecycle.accept.patterns.
//
// Rules:
//   - state "warning" (deprecated, before EOL): blocks startup
//   - state "eol" (past end-of-life date):      blocks startup
//   - state "none": no-op
func (k *Katalog) CheckDeprecationPolicy() error {
	d := k.Deprecation()
	if d == nil {
		return nil
	}
	state := d.DeprecationState(time.Now())
	switch state {
	case "none":
		return nil

	case "warning":
		msg := d.MigrationMessage()
		target := d.MigrationTarget()
		eolLine := ""
		if days := d.DaysUntilEOL(time.Now()); days >= 0 {
			eolLine = fmt.Sprintf("\n  End of life in %d days (%s).", days, d.TimelineTo())
		}
		migrLine := ""
		if target != "" {
			migrLine = fmt.Sprintf("\n  Migrate to: %s", target)
		}
		return fmt.Errorf(
			"%s This katalog is deprecated and cannot start without explicit acknowledgement.%s\n  %s%s\n\n"+
				"  To proceed — in your Komposer:\n\n"+
				"    lifecycle:\n"+
				"      accept:\n"+
				"        patterns:\n"+
				"          - name: <pattern-name>\n",
			warningMark(), eolLine, msg, migrLine,
		)

	case "eol":
		msg := d.MigrationMessage()
		target := d.MigrationTarget()
		migrLine := ""
		if target != "" {
			migrLine = fmt.Sprintf("\n  Migrate to: %s", target)
		}
		return fmt.Errorf(
			"%s This katalog is past its end-of-life date (%s) and cannot start.\n  %s%s\n\n"+
				"  To proceed — in your Komposer:\n\n"+
				"    lifecycle:\n"+
				"      accept:\n"+
				"        patterns:\n"+
				"          - name: <pattern-name>\n",
			failureMark(), d.TimelineTo(), msg, migrLine,
		)
	}
	return nil
}
