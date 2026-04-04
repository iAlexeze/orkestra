package note

import (
	"fmt"
	"strings"
	"text/template"
)

func cronNotes() template.FuncMap {
	return template.FuncMap{
		"cronMinute": cronMinute,
		"cronHour":   cronHour,
		"cronDom":    cronDom,
		"cronMonth":  cronMonth,
		"cronDow":    cronDow,
		"cronField":  cronField,
		"cronExpr":   cronExpr,
		"cronValid":  cronValid,
	}
}

// cronMinute extracts the minute field (position 0) from a cron expression.
// Handles @-macros. Returns "*" for empty input.
//
//	{{ cronMinute "*/5 * * * *" }}   →  "*/5"
//	{{ cronMinute "@hourly" }}        →  "0"
//	{{ cronMinute .spec.schedule }}
func cronMinute(expr string) (string, error) { return cronField(expr, 0) }

// cronHour extracts the hour field (position 1) from a cron expression.
//
//	{{ cronHour "0 2 * * *" }}   →  "2"
//	{{ cronHour "@daily" }}       →  "0"
func cronHour(expr string) (string, error) { return cronField(expr, 1) }

// cronDom extracts the day-of-month field (position 2).
//
//	{{ cronDom "0 0 15 * *" }}   →  "15"
func cronDom(expr string) (string, error) { return cronField(expr, 2) }

// cronMonth extracts the month field (position 3).
//
//	{{ cronMonth "0 0 1 6 *" }}  →  "6"
func cronMonth(expr string) (string, error) { return cronField(expr, 3) }

// cronDow extracts the day-of-week field (position 4).
//
//	{{ cronDow "0 0 * * 1" }}    →  "1"
//	{{ cronDow "@weekly" }}       →  "0"
func cronDow(expr string) (string, error) { return cronField(expr, 4) }

// cronField extracts a single field from a cron expression by position (0-4).
// Expands @-macros before splitting. Returns "*" for empty input.
//
//	{{ cronField .spec.schedule 0 }}   →  minute field
func cronField(expr string, pos int) (string, error) {
	if strings.TrimSpace(expr) == "" {
		return "*", nil
	}
	if pos < 0 || pos > 4 {
		return "*", fmt.Errorf("cronField: position %d out of range [0-4]", pos)
	}
	expanded := expandCronMacro(expr)
	parts := strings.Fields(expanded)
	if len(parts) != 5 {
		return "*", fmt.Errorf(
			"cronField: expected 5 fields in %q, got %d",
			expr, len(parts),
		)
	}
	return parts[pos], nil
}

// cronExpr reconstructs a standard cron expression from five named parts.
// Empty parts default to "*". Used in v2→v1 conversion paths.
//
//	{{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}
//	→  "*/1 * * * *"
func cronExpr(minute, hour, dom, month, dow string) string {
	return fmt.Sprintf("%s %s %s %s %s",
		starIfEmpty(minute), starIfEmpty(hour),
		starIfEmpty(dom), starIfEmpty(month), starIfEmpty(dow),
	)
}

// cronValid reports whether expr is a structurally valid cron expression.
// Does not validate field ranges — only that five fields are present.
//
//	{{ cronValid .spec.schedule }}
func cronValid(expr string) bool {
	if strings.TrimSpace(expr) == "" {
		return false
	}
	return len(strings.Fields(expandCronMacro(expr))) == 5
}

// expandCronMacro converts @-style macros to five-field expressions.
func expandCronMacro(expr string) string {
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

func starIfEmpty(s string) string {
	if strings.TrimSpace(s) == "" {
		return "*"
	}
	return s
}
