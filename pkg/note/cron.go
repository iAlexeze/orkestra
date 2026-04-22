package note

import (
	"fmt"
	"strings"
	"text/template"
)

func cronNotes() template.FuncMap {
	return template.FuncMap{
		"cronMinute":    cronMinute,
		"cronHour":      cronHour,
		"cronDom":       cronDom,
		"cronMonth":     cronMonth,
		"cronDow":       cronDow,
		"cronField":     cronField,
		"cronExpr":      cronExpr,
		"cronValid":     cronValid,
		"cronFromMap":   cronFromMap,
		"cronToMap":     cronToMap,
		"cronNormalize": cronNormalize,
		"cronDescribe":  cronDescribe,
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

func cronFromMap(m map[string]interface{}) string {
	get := func(key string) string {
		if v, ok := m[key]; ok {
			s := fmt.Sprintf("%v", v)
			if strings.TrimSpace(s) != "" {
				return s
			}
		}
		return "*"
	}

	return cronExpr(
		get("minute"),
		get("hour"),
		get("dayOfMonth"),
		get("month"),
		get("dayOfWeek"),
	)
}

// This takes a cron string (including @‑macros) and returns a map:
func cronToMap(expr string) map[string]interface{} {
	out := map[string]interface{}{
		"minute":     "*",
		"hour":       "*",
		"dayOfMonth": "*",
		"month":      "*",
		"dayOfWeek":  "*",
	}

	if strings.TrimSpace(expr) == "" {
		return out
	}

	expanded := expandCronMacro(expr)
	parts := strings.Fields(expanded)
	if len(parts) != 5 {
		return out
	}

	out["minute"] = parts[0]
	out["hour"] = parts[1]
	out["dayOfMonth"] = parts[2]
	out["month"] = parts[3]
	out["dayOfWeek"] = parts[4]

	return out
}

// cronNormalize — canonical formatting
// This ensures:
// macros expanded
// whitespace trimmed
// missing fields → *
// output always 5 fields
// no surprises
func cronNormalize(expr string) string {
	if strings.TrimSpace(expr) == "" {
		return "* * * * *"
	}

	expanded := expandCronMacro(expr)
	parts := strings.Fields(expanded)

	// If invalid, normalize to all "*"
	if len(parts) != 5 {
		return "* * * * *"
	}

	// Trim each field
	for i := range parts {
		if strings.TrimSpace(parts[i]) == "" {
			parts[i] = "*"
		}
	}

	return strings.Join(parts, " ")
}

// cronDescribe — human‑readable explanation
// This is the fun one.
// It turns:
//
//	*/5 * * * *
//		into:
//	Every 5 minutes
//
// And:
//
//	0 2 * * 1
//		into:
//	At 02:00 on Mondays
func cronDescribe(expr string) string {
	norm := cronNormalize(expr)
	parts := strings.Fields(norm)
	if len(parts) != 5 {
		return "Invalid cron expression"
	}

	min, hr, dom, mon, dow := parts[0], parts[1], parts[2], parts[3], parts[4]

	// Minute-only interval
	if strings.HasPrefix(min, "*/") && hr == "*" && dom == "*" && mon == "*" && dow == "*" {
		return fmt.Sprintf("Every %s minutes", strings.TrimPrefix(min, "*/"))
	}

	// Hourly
	if min == "0" && hr == "*" && dom == "*" && mon == "*" && dow == "*" {
		return "Every hour"
	}

	// Daily at specific time
	if dom == "*" && mon == "*" && dow == "*" && min != "*" && hr != "*" {
		return fmt.Sprintf("At %s:%s every day", hr, min)
	}

	// Weekly
	if dom == "*" && mon == "*" && dow != "*" {
		return fmt.Sprintf("At %s:%s on day-of-week %s", hr, min, dow)
	}

	// Monthly
	if dom != "*" && mon == "*" && dow == "*" {
		return fmt.Sprintf("At %s:%s on day %s of every month", hr, min, dom)
	}

	// Fallback
	return fmt.Sprintf("Cron: %s", norm)
}
