package e2e

import (
	"fmt"
	"time"
)

// Result holds the outcome of a complete E2E run.
// It is returned by Run and consumed by ork registry push to embed
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
