package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Version holds the parsed components of a semver string.
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string // e.g. "alpha", "beta", "rc1", ""
}

// Parse parses a version string of the form [v]MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD].
// The leading "v" and build metadata are stripped before parsing.
func Parse(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")
	if idx := strings.IndexByte(s, '+'); idx >= 0 {
		s = s[:idx]
	}
	var pre string
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		pre = s[idx+1:]
		s = s[:idx]
	}
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version %q: expected MAJOR.MINOR.PATCH", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major in %q: %w", s, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor in %q: %w", s, err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch in %q: %w", s, err)
	}
	return Version{Major: major, Minor: minor, Patch: patch, Prerelease: pre}, nil
}

// CheckCompatibility returns an error when binaryVersion is incompatible with
// configVersion under the phase-specific breaking-change rules:
//
//   - binary prerelease "alpha":   any of major/minor/patch differing is breaking
//   - binary prerelease "beta":    major or minor differing is breaking
//   - anything else (GA, RC, …):   only major differing is breaking
//
// Returns nil when the versions are compatible. Skips the check (returns nil)
// when binaryVersion is "dev" or either string cannot be parsed.
func CheckCompatibility(binaryVersion, configVersion string) error {
	if binaryVersion == "dev" || binaryVersion == "" {
		return nil
	}
	bin, err := Parse(binaryVersion)
	if err != nil {
		return nil
	}
	cfg, err := Parse(configVersion)
	if err != nil {
		return nil
	}

	var breaking bool
	switch bin.Prerelease {
	case "alpha":
		breaking = bin.Major != cfg.Major || bin.Minor != cfg.Minor || bin.Patch != cfg.Patch
	case "beta":
		breaking = bin.Major != cfg.Major || bin.Minor != cfg.Minor
	default:
		breaking = bin.Major != cfg.Major
	}

	if !breaking {
		return nil
	}
	return fmt.Errorf(
		"qode binary (%s) is incompatible with this project's config (%s)\nRun 'qode init' to refresh your configuration, prompts, and IDE commands",
		trimBuild(binaryVersion),
		trimBuild(configVersion),
	)
}

// trimBuild strips build metadata (+…) and the leading "v" for display.
func trimBuild(s string) string {
	s = strings.TrimPrefix(s, "v")
	if idx := strings.IndexByte(s, '+'); idx >= 0 {
		s = s[:idx]
	}
	return s
}
