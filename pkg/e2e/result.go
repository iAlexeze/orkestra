package e2e

import (
	"fmt"
	"strings"
	"time"
)

// Result holds the outcome of a complete E2E run.
// It is returned by Run and consumed by ork push to embed
// verification metadata into OCI annotations.
type Result struct {
	Name    string
	Cases   []CaseResult
	Elapsed time.Duration
}

// CaseResult holds the outcome of a single expectation.
type CaseResult struct {
	Name    string
	Passed  bool
	Elapsed time.Duration
	Err     error
}

// AllPassed returns true when every expectation passed.
func (r *Result) AllPassed() bool {
	for _, c := range r.Cases {
		if !c.Passed {
			return false
		}
	}
	return true
}

// Passed returns the number of passing expectations.
func (r *Result) Passed() int {
	n := 0
	for _, c := range r.Cases {
		if c.Passed {
			n++
		}
	}
	return n
}

// Total returns the total number of expectations.
func (r *Result) Total() int { return len(r.Cases) }

// Duration returns the total elapsed time as a human-readable string (e.g. "45s").
func (r *Result) Duration() string {
	return r.Elapsed.Round(time.Second).String()
}

// Summary returns a one-line result string, e.g. "3 of 3 passed (12s)".
func (r *Result) Summary() string {
	return fmt.Sprintf("%d of %d passed (%s)", r.Passed(), r.Total(), r.Duration())
}

// ImportResult holds the outcome of one imported E2E file in a suite run.
type ImportResult struct {
	// Path is the import path as declared in imports: (e.g. "./01-basic/e2e.yaml").
	Path string
	// Result is the structured outcome. Nil when the import failed to load or
	// the runner itself errored before producing any expectations.
	Result *Result
	// Err is non-nil when the import failed (load error, Orkestra not ready,
	// or one or more expectations failed).
	Err error
}

// Name returns the display name — the E2E metadata.name when available,
// otherwise the import path.
func (i ImportResult) Name() string {
	if i.Result != nil && i.Result.Name != "" {
		return i.Result.Name
	}
	return i.Path
}

// Passed reports whether this import succeeded.
func (i ImportResult) Passed() bool { return i.Err == nil }

// printImportSummary prints the suite results table and returns a non-nil
// error when any import failed.
func printImportSummary(suiteName string, results []ImportResult) error {
	passed := 0
	for _, ir := range results {
		if ir.Passed() {
			passed++
		}
	}

	var total time.Duration
	for _, ir := range results {
		if ir.Result != nil {
			total += ir.Result.Elapsed
		}
	}

	fmt.Printf("\n─── Suite Results: %s ───\n\n", suiteName)
	for _, ir := range results {
		icon := "✓"
		if !ir.Passed() {
			icon = "✗"
		}

		score := "—"
		elapsed := ""
		if ir.Result != nil {
			score = fmt.Sprintf("%d / %d", ir.Result.Passed(), ir.Result.Total())
			elapsed = ir.Result.Duration()
		}

		errSuffix := ""
		if ir.Err != nil {
			msg := ir.Err.Error()
			// Trim the "import ./path: " prefix if present so the table stays narrow.
			if idx := strings.Index(msg, ": "); idx >= 0 {
				msg = msg[idx+2:]
			}
			if len(msg) > 60 {
				msg = msg[:57] + "..."
			}
			errSuffix = "  → " + msg
		}

		fmt.Printf("  %s  %-44s  %-6s  %-6s%s\n", icon, ir.Name(), score, elapsed, errSuffix)
	}

	elapsed := total.Round(time.Second).String()
	fmt.Printf("\n  %d of %d imports passed  (%s)\n", passed, len(results), elapsed)

	if passed < len(results) {
		return fmt.Errorf("%d of %d imports failed", len(results)-passed, len(results))
	}
	return nil
}
