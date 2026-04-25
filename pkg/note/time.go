package note

import (
	"fmt"
	"math"
	"text/template"
	"time"
)

// ime notes
// - timeAgo
// - timeSince
// - isExpired
// - timeFormat

func timeNotes() template.FuncMap {
	return template.FuncMap{
		"timeAgo":    noteTimeAgo,
		"timeSince":  noteTimeSince,
		"isExpired":  noteIsExpired,
		"timeFormat": noteTimeFormat,
	}
}

// ── Time notes ────────────────────────────────────────────────────────────────

// parseTime attempts to parse a timestamp string in RFC3339 or common formats.
func parseTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// noteTimeAgo returns a human-readable "X ago" string from a timestamp.
//
//	{{ timeAgo .children.cronjob.status.lastScheduleTime }}
//	→ "5m ago" or "2h ago" or "3d ago"
func noteTimeAgo(s string) string {
	t, ok := parseTime(fmt.Sprint(s))
	if !ok {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// noteTimeSince returns the number of seconds elapsed since the timestamp.
//
//	{{ timeSince .metadata.creationTimestamp }}  → 3600
func noteTimeSince(s string) int64 {
	t, ok := parseTime(fmt.Sprint(s))
	if !ok {
		return 0
	}
	return int64(math.Round(time.Since(t).Seconds()))
}

// noteIsExpired returns true when the timestamp plus the given duration is in the past.
// The duration string follows Go's time.ParseDuration format ("30m", "24h", "7d" not supported —
// use "168h" for 7 days).
//
// Primary use: declarative rotation checks in when: conditions.
//
//	{{ isExpired (index .metadata.annotations "orkestra.orkspace.io/generated-at") "720h" }}
//	→ true when the annotation timestamp is more than 30 days old
func noteIsExpired(timestamp interface{}, duration string) bool {
	t, ok := parseTime(fmt.Sprint(timestamp))
	if !ok {
		return false
	}
	d, err := time.ParseDuration(duration)
	if err != nil {
		return false
	}
	return time.Now().After(t.Add(d))
}

// noteTimeFormat reformats a timestamp string into a human-readable form.
// layout follows Go's time format convention ("Jan 2, 2006", "15:04", etc.)
//
//	{{ timeFormat .metadata.creationTimestamp "Jan 2, 2006" }}
//	→ "Apr 13, 2026"
func noteTimeFormat(timestamp interface{}, layout string) string {
	t, ok := parseTime(fmt.Sprint(timestamp))
	if !ok {
		return ""
	}
	return t.UTC().Format(layout)
}
