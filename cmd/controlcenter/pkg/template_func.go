package controlcenter

import (
	"html/template"
	"strings"
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
}
