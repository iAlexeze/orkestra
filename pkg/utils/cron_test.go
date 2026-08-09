// pkg/utils/cron_test.go
package utils

import (
	"testing"
)

func TestValidateCronField(t *testing.T) {
	tests := []struct {
		name  string
		field string
		min   int
		max   int
		want  bool
	}{
		// Wildcards
		{"wildcard", "*", 0, 59, true},
		{"wildcard any range", "*", 1, 31, true},

		// Single values
		{"single value valid", "5", 0, 59, true},
		{"single value min", "0", 0, 59, true},
		{"single value max", "59", 0, 59, true},
		{"single value below min", "-1", 0, 59, false},
		{"single value above max", "60", 0, 59, false},
		{"single value outside hour range", "25", 0, 23, false},
		{"single value outside day range", "0", 1, 31, false},
		{"single value invalid", "abc", 0, 59, false},

		// Steps
		{"step valid", "*/5", 0, 59, true},
		{"step valid zero", "*/0", 0, 59, false},
		{"step valid negative", "*/-1", 0, 59, false},
		{"step invalid", "*/abc", 0, 59, false},

		// Ranges
		{"range valid", "1-5", 0, 59, true},
		{"range min max", "0-59", 0, 59, true},
		{"range start below min", "-1-5", 0, 59, false},
		{"range end above max", "1-60", 0, 59, false},
		{"range start end reversed", "5-1", 0, 59, false},
		{"range too many parts", "1-5-9", 0, 59, false},
		{"range invalid start", "abc-5", 0, 59, false},
		{"range invalid end", "1-xyz", 0, 59, false},
		{"range with spaces", "1 - 5", 0, 59, true},

		// Lists
		{"list valid", "1,2,3", 0, 59, true},
		{"list with spaces", "1, 2, 3", 0, 59, true},
		{"list contains invalid", "1,2,60", 0, 59, false},
		{"list empty", "1,,2", 0, 59, false},
		{"list trailing comma", "1,2,", 0, 59, false},
		{"list leading comma", ",1,2", 0, 59, false},
		{"list single", "5", 0, 59, true},
		{"list with range", "1-3,5,7-9", 0, 59, true},

		// Empty
		{"empty string", "", 0, 59, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateCronField(tt.field, tt.min, tt.max)
			if got != tt.want {
				t.Errorf("ValidateCronField(%q, %d, %d) = %v, want %v", tt.field, tt.min, tt.max, got, tt.want)
			}
		})
	}
}

func TestExpandCronMacro(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want string
	}{
		{"yearly", "@yearly", "0 0 1 1 *"},
		{"annually", "@annually", "0 0 1 1 *"},
		{"monthly", "@monthly", "0 0 1 * *"},
		{"weekly", "@weekly", "0 0 * * 0"},
		{"daily", "@daily", "0 0 * * *"},
		{"midnight", "@midnight", "0 0 * * *"},
		{"hourly", "@hourly", "0 * * * *"},
		{"yearly uppercase", "@YEARLY", "0 0 1 1 *"},
		{"yearly mixed", "@Yearly", "0 0 1 1 *"},
		{"daily with space", " @daily ", "0 0 * * *"},
		{"not a macro", "0 2 * * 1-5", "0 2 * * 1-5"},
		{"empty", "", ""},
		{"unknown macro", "@unknown", "@unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandCronMacro(tt.expr)
			if got != tt.want {
				t.Errorf("ExpandCronMacro(%q) = %q, want %q", tt.expr, got, tt.want)
			}
		})
	}
}

func TestCronValid(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want bool
	}{
		// Valid expressions
		{"valid standard", "0 2 * * 1-5", true},
		{"valid all stars", "* * * * *", true},
		{"valid steps", "*/5 * * * *", true},
		{"valid ranges", "0 9-17 * * 1-5", true},
		{"valid lists", "1,15,30 * * * *", true},
		{"valid macro hourly", "@hourly", true},
		{"valid macro daily", "@daily", true},
		{"valid macro weekly", "@weekly", true},
		{"valid macro monthly", "@monthly", true},
		{"valid macro yearly", "@yearly", true},
		{"valid with leading zeros", "05 02 * * 1-5", true},

		// Invalid expressions
		{"invalid hour 25", "25 * * 1-5", false},
		{"invalid minute 60", "60 * * * *", false},
		{"invalid day 32", "0 2 32 * *", false},
		{"invalid month 13", "0 2 * 13 *", false},
		{"invalid day of week 7", "0 2 * * 7", false},
		{"too few fields", "0 2 * *", false},
		{"too many fields", "0 2 * * 1-5 extra", false},
		{"empty", "", false},
		{"only spaces", "   ", false},
		{"invalid range", "0 2 * * 1-5-9", false},
		{"invalid step", "0 */ * * *", false},
		{"invalid list", "0 2 * * 1,,5", false},
		{"invalid value", "0 2 * * abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidCronExpr(tt.expr)
			if got != tt.want {
				t.Errorf("CronValid(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

// TestCronValid_IntegrationWithCronNotes
func TestCronValid_IntegrationWithCronNotes(t *testing.T) {
	validExpressions := []string{
		"0 2 * * 1-5",
		"*/5 * * * *",
		"0 9-17 * * 1-5",
		"1,15,30 * * * *",
		"@hourly",
		"@daily",
		"@weekly",
		"@monthly",
		"@yearly",
		"25 * * * *", // minute 25 is valid
	}

	for _, expr := range validExpressions {
		if !IsValidCronExpr(expr) {
			t.Errorf("CronValid(%q) returned false, but it should be valid", expr)
		}
	}

	invalidExpressions := []string{
		"0 25 * * *",  // hour 25 invalid
		"60 * * * *",  // minute 60 invalid
		"0 2 32 * *",  // day 32 invalid
		"0 2 * 13 *",  // month 13 invalid
		"0 2 * * 7",   // day of week 7 invalid
		"0 2 * *",     // too few fields
		"0 2 * * * *", // too many fields
		"",            // empty
		"   ",         // spaces
	}

	for _, expr := range invalidExpressions {
		if IsValidCronExpr(expr) {
			t.Errorf("CronValid(%q) returned true, but it should be invalid", expr)
		}
	}
}
