// pkg/utils/cron.go
package utils

import (
	"strconv"
	"strings"
)

// CronFieldRanges define the allowed ranges for each cron field.
const (
	CronMinuteMin     = 0
	CronMinuteMax     = 59
	CronHourMin       = 0
	CronHourMax       = 23
	CronDayOfMonthMin = 1
	CronDayOfMonthMax = 31
	CronMonthMin      = 1
	CronMonthMax      = 12
	CronDayOfWeekMin  = 0
	CronDayOfWeekMax  = 6
)

// ValidateCronField validates a single cron field value against its allowed range.
//
// Supports standard cron syntax:
//   - "*"               — any value
//   - "*/5"             — step values (every 5 units)
//   - "1-5"             — inclusive ranges
//   - "1,2,3"           — comma-separated lists
//   - "5"               — single value
//
// Returns false for:
//   - Empty or malformed fields
//   - Values outside [min, max]
//   - Invalid range syntax (e.g., "5-3", "1-5-9")
//   - Invalid step syntax (e.g., "*/")
//   - Invalid list syntax (e.g., "1,,2")
func ValidateCronField(field string, min, max int) bool {
	if field == "" {
		return false
	}

	// Handle step values: "*/5"
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(strings.TrimPrefix(field, "*/"))
		if err != nil {
			return false
		}
		return step > 0
	}

	// Handle comma-separated values: "1,2,3"
	if strings.Contains(field, ",") {
		for _, part := range strings.Split(field, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				return false
			}
			if !ValidateCronField(part, min, max) {
				return false
			}
		}
		return true
	}

	// Handle ranges: "1-5"
	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		if len(parts) != 2 {
			return false
		}
		start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return false
		}
		end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return false
		}
		return start >= min && end <= max && start <= end
	}

	// Single value or wildcard
	if field == "*" {
		return true
	}

	val, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return val >= min && val <= max
}

// ExpandCronMacro converts @-style macros to five-field expressions.
func ExpandCronMacro(expr string) string {
	switch strings.TrimSpace(strings.ToLower(expr)) {
	case "@yearly", "@annually":
		return "0 0 1 1 *"
	case "@monthly":
		return "0 0 1 * *"
	case "@weekly":
		return "0 0 * * 0"
	case "@daily", "@midnight":
		return "0 0 * * *"
	case "@hourly":
		return "0 * * * *"
	default:
		return expr
	}
}

// IsValidCronExpr reports whether expr is a valid cron expression.
//
// Validates field count and that each field's values are within the correct range.
// Supports macros, steps, ranges, and lists.
func IsValidCronExpr(expr string) bool {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return false
	}

	fields := strings.Fields(ExpandCronMacro(trimmed))
	if len(fields) != 5 {
		return false
	}

	// Validate each field
	if !ValidateCronField(fields[0], CronMinuteMin, CronMinuteMax) {
		return false
	}
	if !ValidateCronField(fields[1], CronHourMin, CronHourMax) {
		return false
	}
	if !ValidateCronField(fields[2], CronDayOfMonthMin, CronDayOfMonthMax) {
		return false
	}
	if !ValidateCronField(fields[3], CronMonthMin, CronMonthMax) {
		return false
	}
	if !ValidateCronField(fields[4], CronDayOfWeekMin, CronDayOfWeekMax) {
		return false
	}
	return true
}
