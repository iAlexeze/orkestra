package katalog

import (
	"fmt"
	"time"
)

// validateDeprecation checks the metadata.deprecation block when present.
//
// Rules:
//  1. message is required when deprecation: is declared.
//  2. timeline.from and timeline.to must parse as YYYY-MM-DD when set.
//  3. timeline.from must be before timeline.to when both are set.
func (k *Katalog) validateDeprecation() error {
	d := k.metadata.Deprecation
	if d == nil {
		return nil
	}

	if d.Message == "" {
		return fmt.Errorf(
			"%s metadata.deprecation: message is required when deprecation is declared",
			failureMark(),
		)
	}

	const layout = "2006-01-02"

	var from, to time.Time
	var hasFrom, hasTo bool

	if f := d.TimelineFrom(); f != "" {
		t, err := time.Parse(layout, f)
		if err != nil {
			return fmt.Errorf(
				"%s metadata.deprecation.timeline.from: %q is not a valid date (expected YYYY-MM-DD)",
				failureMark(), f,
			)
		}
		from = t
		hasFrom = true
	}

	if s := d.TimelineTo(); s != "" {
		t, err := time.Parse(layout, s)
		if err != nil {
			return fmt.Errorf(
				"%s metadata.deprecation.timeline.to: %q is not a valid date (expected YYYY-MM-DD)",
				failureMark(), s,
			)
		}
		to = t
		hasTo = true
	}

	if hasFrom && hasTo && !from.Before(to) {
		return fmt.Errorf(
			"%s metadata.deprecation.timeline: from (%s) must be before to (%s)",
			failureMark(), d.TimelineFrom(), d.TimelineTo(),
		)
	}

	return nil
}
