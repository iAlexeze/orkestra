package intake

import (
	"github.com/gobwas/glob"
)

// MatchedWatchFiles returns the subset of changed that matches at least one
// declared watch pattern. An empty watch list matches every changed file —
// no patterns configured means every push on the watched branch is processed.
//
// A pattern that fails to compile is skipped rather than erroring the request
// — ork validate is where a malformed pattern should be caught, not a live
// webhook delivery.
func MatchedWatchFiles(watch []string, changed []string) []string {
	if len(watch) == 0 {
		return changed
	}

	patterns := make([]glob.Glob, 0, len(watch))
	for _, pattern := range watch {
		g, err := glob.Compile(pattern, '/')
		if err != nil {
			continue
		}
		patterns = append(patterns, g)
	}

	var matched []string
	for _, path := range changed {
		for _, g := range patterns {
			if g.Match(path) {
				matched = append(matched, path)
				break
			}
		}
	}
	return matched
}

// CollectChangedFiles unions path lists across a push's commits — typically
// one group per commit's "added" and "modified" lists — deduplicating while
// preserving first-seen order. Removed files are the caller's job to
// exclude: there's no content left to fetch for a deleted path.
func CollectChangedFiles(groups ...[]string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, group := range groups {
		for _, path := range group {
			if seen[path] {
				continue
			}
			seen[path] = true
			out = append(out, path)
		}
	}
	return out
}
