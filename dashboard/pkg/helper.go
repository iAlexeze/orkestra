package dashboard

import (
	"fmt"
	"time"
)

func timeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	seconds := int(diff.Seconds())

	if seconds < 10 {
		return "just now"
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds ago", seconds)
	}

	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%dm ago", minutes)
	}

	hours := minutes / 60
	if hours < 24 {
		return fmt.Sprintf("%dh %dm ago", hours, minutes%60)
	}

	days := hours / 24
	return fmt.Sprintf("%dd ago", days)
}

func parseTime(value string) time.Time {
    t, err := time.Parse(time.RFC3339, value)
    if err == nil {
        return t
    }

    // fallback for your current format
    t, err = time.Parse("2006-01-02 15:04:05 -0700 MST", value)
    if err == nil {
        return t
    }

    return time.Time{}
}