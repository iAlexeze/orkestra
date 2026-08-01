package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseTimeDuration parses a human‑friendly duration string.
//
// It extends Go's time.ParseDuration by adding long‑term units:
//
//	d   = days (24h)
//	w   = weeks (7d)
//	mo  = months (30d)
//	y   = years (365d)
//
// Examples:
//
//	"30s"     → 30 seconds
//	"5m"      → 5 minutes (Go duration)
//	"12h"     → 12 hours
//	"10d"     → 10 days
//	"2w"      → 14 days
//	"3mo"     → 90 days
//	"1y"      → 365 days
//
// Notes:
//   - Only "mo" is accepted for months to avoid collision with Go's "m" (minutes).
//   - Fractional values are supported (e.g., "1.5mo").
//   - "never" returns 0, nil — callers for which a zero duration is not a
//     valid outcome (e.g. an HTTP timeout) should reject "never" explicitly.
//   - Falls back to time.ParseDuration for standard units.
func ParseTimeDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	if s == "never" {
		return 0, nil
	}

	s = strings.TrimSpace(s)

	// Years (365 days)
	if strings.HasSuffix(s, "y") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "y"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid year duration %q: %w", s, err)
		}
		return time.Duration(n * float64(365*24*time.Hour)), nil
	}

	// Months (30 days) — explicit "mo" only
	if strings.HasSuffix(s, "mo") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "mo"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid month duration %q: %w", s, err)
		}
		return time.Duration(n * float64(30*24*time.Hour)), nil
	}

	// Weeks (7 days)
	if strings.HasSuffix(s, "w") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "w"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid week duration %q: %w", s, err)
		}
		return time.Duration(n * float64(7*24*time.Hour)), nil
	}

	// Days (24 hours)
	if strings.HasSuffix(s, "d") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "d"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid day duration %q: %w", s, err)
		}
		return time.Duration(n * float64(24*time.Hour)), nil
	}

	// Fall back to Go's duration parser (s, m, h)
	return time.ParseDuration(s)
}
