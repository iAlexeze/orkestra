package note

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/semver/v3"
)

func semverNotes() template.FuncMap {
	return template.FuncMap{
		"semverMajor":      noteSemverMajor,
		"semverMinor":      noteSemverMinor,
		"semverPatch":      noteSemverPatch,
		"semverValid":      noteSemverValid,
		"semverCompare":    noteSemverCompare,
		"semverBump":       noteSemverBump,
		"semverConstraint": noteSemverConstraint,
	}
}

// noteSemverMajor returns the major version component of a semver string.
// Returns "" for invalid input.
//
//	{{ semverMajor "1.2.3" }}   → "1"
//	{{ semverMajor "v2.0.0" }}  → "2"
func noteSemverMajor(v string) string {
	sv, err := semver.NewVersion(v)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d", sv.Major())
}

// noteSemverMinor returns the minor version component.
//
//	{{ semverMinor "1.2.3" }}  → "2"
func noteSemverMinor(v string) string {
	sv, err := semver.NewVersion(v)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d", sv.Minor())
}

// noteSemverPatch returns the patch version component.
//
//	{{ semverPatch "1.2.3" }}  → "3"
func noteSemverPatch(v string) string {
	sv, err := semver.NewVersion(v)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d", sv.Patch())
}

// noteSemverValid reports whether the string is a valid semver version.
//
//	{{ semverValid "1.2.3" }}   → true
//	{{ semverValid "latest" }}  → false
func noteSemverValid(v string) bool {
	_, err := semver.NewVersion(v)
	return err == nil
}

// noteSemverCompare compares two semver strings.
// Returns -1 when a < b, 0 when a == b, 1 when a > b.
// Returns 0 for invalid input (safe zero value).
//
//	{{ semverCompare "1.2.3" "1.3.0" }}  → -1
//	{{ semverCompare "2.0.0" "1.9.9" }}  → 1
func noteSemverCompare(a, b string) int {
	sa, err := semver.NewVersion(a)
	if err != nil {
		return 0
	}
	sb, err := semver.NewVersion(b)
	if err != nil {
		return 0
	}
	return sa.Compare(sb)
}

// The issue is that semver.IncPatch() does not increment the patch number
// when there is a prerelease; it only drops the prerelease.
// To conform to industry standard (bump patch from 1.2.3-alpha to 1.2.4),
// we need to manually increment the component and discard any prerelease/metadata.
func noteSemverBump(v, component string) string {
	// Strip leading 'v' if present
	strippedV := strings.TrimPrefix(v, "v")
	sv, err := semver.NewVersion(strippedV)
	if err != nil {
		return v
	}
	major, minor, patch := sv.Major(), sv.Minor(), sv.Patch()
	switch strings.ToLower(component) {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "patch":
		patch++
	default:
		return v
	}
	// Create a new version without prerelease or metadata
	newVer := semver.New(major, minor, patch, "", "")
	return newVer.String()
}

// noteSemverConstraint reports whether version satisfies the constraint expression.
// Uses Masterminds semver constraint syntax: ">=1.0.0", "^2.0", "~1.2", "1.x".
// Returns false for invalid input (safe zero value).
//
//	{{ semverConstraint "1.2.3" ">=1.0.0,<2.0.0" }}  → true
//	{{ semverConstraint "2.1.0" "^1.0" }}             → false
func noteSemverConstraint(v, constraint string) bool {
	sv, err := semver.NewVersion(v)
	if err != nil {
		return false
	}
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false
	}
	return c.Check(sv)
}
