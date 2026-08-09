package katalog

import (
	"fmt"
	"time"
)

// CheckDeprecationPolicy enforces runtime deprecation gates for ork run and
// ork gate. It is called after ValidateConfig — not inside it — because
// validation is a pre-flight tool and must not block; enforcement belongs to
// the runtime startup path.
//
// Rules:
//   - state "warning" (deprecated, before EOL): requires accept.beforeEol: true
//   - state "eol" (past end-of-life date):      requires accept.beforeEol: true AND accept.eol: true
//   - state "none": no-op
func (k *Katalog) CheckDeprecationPolicy() error {
	d := k.metadata.Deprecation
	if d == nil {
		return nil
	}
	state := d.DeprecationState(time.Now())
	switch state {
	case "none":
		return nil

	case "warning":
		if d.AcceptsBeforeEol() {
			return nil
		}
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
				"  To proceed, add to your katalog:\n\n"+
				"    deprecation:\n"+
				"      accept:\n"+
				"        beforeEol: true\n",
			warningMark(), eolLine, msg, migrLine,
		)

	case "eol":
		if d.AcceptsEol() {
			return nil
		}
		msg := d.MigrationMessage()
		target := d.MigrationTarget()
		migrLine := ""
		if target != "" {
			migrLine = fmt.Sprintf("\n  Migrate to: %s", target)
		}
		if !d.AcceptsBeforeEol() {
			return fmt.Errorf(
				"%s This katalog is past its end-of-life date (%s) and cannot start.\n  %s%s\n\n"+
					"  To proceed, add to your katalog:\n\n"+
					"    deprecation:\n"+
					"      accept:\n"+
					"        beforeEol: true\n"+
					"        eol: true\n",
				failureMark(), d.TimelineTo(), msg, migrLine,
			)
		}
		return fmt.Errorf(
			"%s This katalog is past its end-of-life date (%s) and cannot start.\n  %s%s\n\n"+
				"  accept.beforeEol is set but accept.eol is required to run past EOL.\n\n"+
				"  Add to your katalog:\n\n"+
				"    deprecation:\n"+
				"      accept:\n"+
				"        beforeEol: true\n"+
				"        eol: true\n",
			failureMark(), d.TimelineTo(), msg, migrLine,
		)
	}
	return nil
}
