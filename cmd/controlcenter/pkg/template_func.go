package controlcenter

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

// templateFuncs provides helper functions for HTML templates
var templateFuncs = template.FuncMap{
	"mul": func(a, b int) int {
		return a * b
	},
	"div": func(a, b int) int {
		if b == 0 {
			return 0
		}
		return a / b
	},
	"sub": func(a, b int) int {
		return a - b
	},
	"title": strings.Title,
	"truncate": func(s string, n int) string {
		if len(s) <= n {
			return s
		}
		return s[:n-3] + "..."
	},
	// formatNumber formats large numbers with K/M/B suffixes
	"formatNumber": func(n int) string {
		if n >= 1_000_000_000 {
			return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
		}
		if n >= 1_000_000 {
			return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
		}
		if n >= 10_000 {
			return fmt.Sprintf("%.1fK", float64(n)/1_000)
		}
		return fmt.Sprintf("%d", n)
	},
	"add": func(a, b int) int {
		return a + b
	},
	"min": func(a, b int) int {
		if a < b {
			return a
		}
		return b
	},
	// format a time string to a readable format
	"formatTime": func(timeStr string) string {
		if timeStr == "" {
			return "never"
		}
		t, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return timeStr
		}
		return t.Format("2006-01-02 15:04:05")
	},
	// join a slice of strings with a separator
	"join": func(sep string, items []string) string {
		if len(items) == 0 {
			return ""
		}
		return strings.Join(items, sep)
	},
	// Helper to check if a value is in a slice (for dependency states)
	"contains": func(slice []string, item string) bool {
		for _, s := range slice {
			if s == item {
				return true
			}
		}
		return false
	},
}
