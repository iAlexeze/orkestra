package controlcenter

import "strings"

// encodeInstance encodes an instance URL for use in URL paths.
// Strips http:// and replaces : with - for safe path embedding.
func encodeInstance(url string) string {
	s := strings.TrimPrefix(url, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.TrimSuffix(s, "/")
	return s
}
