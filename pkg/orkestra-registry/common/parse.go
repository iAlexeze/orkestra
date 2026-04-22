package common

import "fmt"

// ParseBool interprets common boolean representations from template expressions.
func ParseBool(s string) bool {
	switch s {
	case "true", "True", "TRUE", "1", "yes", "YES":
		return true
	default:
		return false
	}
}

// ParsePort interprets common port representations from template expressions.
func ParsePort(s string) int {
	var p int
	fmt.Sscanf(s, "%d", &p)
	return p
}
