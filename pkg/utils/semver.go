package utils

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// SemverValid reports whether v is a valid semver string.
func SemverValid(v string) bool {
	_, err := semver.NewVersion(v)
	return err == nil
}

// SemverMajor returns the major component of v.
func SemverMajor(v string) (uint64, error) {
	sv, err := semver.NewVersion(v)
	if err != nil {
		return 0, fmt.Errorf("invalid semver %q: %w", v, err)
	}
	return sv.Major(), nil
}

// SemverMinor returns the minor component of v.
func SemverMinor(v string) (uint64, error) {
	sv, err := semver.NewVersion(v)
	if err != nil {
		return 0, fmt.Errorf("invalid semver %q: %w", v, err)
	}
	return sv.Minor(), nil
}

// SemverPatch returns the patch component of v.
func SemverPatch(v string) (uint64, error) {
	sv, err := semver.NewVersion(v)
	if err != nil {
		return 0, fmt.Errorf("invalid semver %q: %w", v, err)
	}
	return sv.Patch(), nil
}

// SemverCompare compares two semver strings.
// Returns -1 when a < b, 0 when a == b, 1 when a > b.
func SemverCompare(a, b string) (int, error) {
	sa, err := semver.NewVersion(a)
	if err != nil {
		return 0, fmt.Errorf("invalid semver %q: %w", a, err)
	}
	sb, err := semver.NewVersion(b)
	if err != nil {
		return 0, fmt.Errorf("invalid semver %q: %w", b, err)
	}
	return sa.Compare(sb), nil
}

// SemverCheck reports whether version v satisfies the constraint expression.
// Uses Masterminds semver constraint syntax: ">=1.0.0", "^2.0", "~1.2", "1.x".
func SemverCheck(v, constraint string) (bool, error) {
	sv, err := semver.NewVersion(v)
	if err != nil {
		return false, fmt.Errorf("invalid semver %q: %w", v, err)
	}
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false, fmt.Errorf("invalid semver constraint %q: %w", constraint, err)
	}
	return c.Check(sv), nil
}

// SemverBump increments the given component ("major", "minor", or "patch") of v,
// dropping any prerelease or build metadata from the result.
func SemverBump(v, component string) (string, error) {
	sv, err := semver.NewVersion(strings.TrimPrefix(v, "v"))
	if err != nil {
		return "", fmt.Errorf("invalid semver %q: %w", v, err)
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
		return "", fmt.Errorf("unknown semver component %q: must be major, minor, or patch", component)
	}
	return semver.New(major, minor, patch, "", "").String(), nil
}
