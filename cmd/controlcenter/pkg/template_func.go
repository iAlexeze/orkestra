package controlcenter

import (
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
