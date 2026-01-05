package semver

import (
	"fmt"
	"strconv"
	"strings"
)

type SemVer struct {
	Major int
	Minor int
	Patch int
	Hash  string
}

// VersionInfo holds build-time version information.
// These can be set at build time using -ldflags:
//
//	go build -ldflags "-X github.com/hlfshell/gotonomy/utils/semver.VersionInfo.Major=1 -X github.com/hlfshell/gotonomy/utils/semver.VersionInfo.Minor=2 -X github.com/hlfshell/gotonomy/utils/semver.VersionInfo.Patch=3 -X github.com/hlfshell/gotonomy/utils/semver.VersionInfo.Hash=abc123"
var VersionInfo struct {
	Major string
	Minor string
	Patch string
	Hash  string
}

// GetBuildVersion returns the SemVer from build-time injected VersionInfo.
// Returns an error if VersionInfo is not set or if the values are invalid.
// This function is intended to be used in tool cards to embed version information at build time.
func GetBuildVersion() (SemVer, error) {
	// Check if any version info is set (if all are empty, version wasn't injected)
	if VersionInfo.Major == "" && VersionInfo.Minor == "" && VersionInfo.Patch == "" {
		return SemVer{}, fmt.Errorf("build-time version not set (use -ldflags to inject version)")
	}

	// Default to "0" if not set
	majorStr := VersionInfo.Major
	if majorStr == "" {
		majorStr = "0"
	}
	minorStr := VersionInfo.Minor
	if minorStr == "" {
		minorStr = "0"
	}
	patchStr := VersionInfo.Patch
	if patchStr == "" {
		patchStr = "0"
	}

	major, err := strconv.Atoi(majorStr)
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid build-time major version %q: %w", majorStr, err)
	}

	minor, err := strconv.Atoi(minorStr)
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid build-time minor version %q: %w", minorStr, err)
	}

	patch, err := strconv.Atoi(patchStr)
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid build-time patch version %q: %w", patchStr, err)
	}

	return SemVer{
		Major: major,
		Minor: minor,
		Patch: patch,
		Hash:  VersionInfo.Hash,
	}, nil
}

// NewSemVer parses a version string and returns a SemVer struct.
// Supports formats:
//   - "v1.2.3" or "1.2.3"
//   - "v1.2.3-abc123" or "1.2.3-abc123" (with hash)
//   - "1.2.3:abc123" (alternative hash separator)
func NewSemVer(v string) (SemVer, error) {
	if v == "" {
		return SemVer{}, fmt.Errorf("version string cannot be empty")
	}

	// Remove leading 'v' if present
	v = strings.TrimPrefix(v, "v")

	var hash string
	var versionPart string

	// Check for hash separator (either '-' or ':')
	if idx := strings.LastIndex(v, "-"); idx != -1 {
		versionPart = v[:idx]
		hash = v[idx+1:]
	} else if idx := strings.LastIndex(v, ":"); idx != -1 {
		versionPart = v[:idx]
		hash = v[idx+1:]
	} else {
		versionPart = v
	}

	// Split version into parts
	parts := strings.Split(versionPart, ".")
	if len(parts) != 3 {
		return SemVer{}, fmt.Errorf("version must have exactly 3 parts (major.minor.patch), got %d parts", len(parts))
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid major version: %w", err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid minor version: %w", err)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid patch version: %w", err)
	}

	return SemVer{
		Major: major,
		Minor: minor,
		Patch: patch,
		Hash:  hash,
	}, nil
}

// String returns the string representation of the version.
// Format: "v1.2.3" or "v1.2.3-abc123" if hash is present.
func (s SemVer) String() string {
	version := fmt.Sprintf("v%d.%d.%d", s.Major, s.Minor, s.Patch)
	if s.Hash != "" {
		version += "-" + s.Hash
	}
	return version
}

// Compare compares two SemVer values.
// Returns:
//   - -1 if s < other
//   - 0 if s == other
//   - 1 if s > other
//
// Hash is not considered in comparison, only major.minor.patch.
func (s SemVer) Compare(other SemVer) int {
	if s.Major < other.Major {
		return -1
	}
	if s.Major > other.Major {
		return 1
	}

	if s.Minor < other.Minor {
		return -1
	}
	if s.Minor > other.Minor {
		return 1
	}

	if s.Patch < other.Patch {
		return -1
	}
	if s.Patch > other.Patch {
		return 1
	}

	return 0
}

// LT returns true if s < other.
func (s SemVer) LT(other SemVer) bool {
	return s.Compare(other) < 0
}

// LTE returns true if s <= other.
func (s SemVer) LTE(other SemVer) bool {
	return s.Compare(other) <= 0
}

// GT returns true if s > other.
func (s SemVer) GT(other SemVer) bool {
	return s.Compare(other) > 0
}

// GTE returns true if s >= other.
func (s SemVer) GTE(other SemVer) bool {
	return s.Compare(other) >= 0
}

// EQ returns true if s == other (ignoring hash).
func (s SemVer) EQ(other SemVer) bool {
	return s.Compare(other) == 0
}
