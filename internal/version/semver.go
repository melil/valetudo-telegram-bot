package version

import (
	"strconv"
	"strings"
)

// SemVer represents parsed semantic version components (Major.Minor.Patch-Pre).
type SemVer struct {
	Major int
	Minor int
	Patch int
	Pre   string
}

// Parse parses a semantic version string (e.g. "1.0.0", "v1.2.3", "2.0.0-beta.1").
func Parse(v string) (SemVer, bool) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")

	if v == "" {
		return SemVer{}, false
	}

	parts := strings.SplitN(v, "-", 2)
	pre := ""
	if len(parts) == 2 {
		pre = parts[1]
	}

	nums := strings.Split(parts[0], ".")
	if len(nums) < 1 {
		return SemVer{}, false
	}

	major, err := strconv.Atoi(nums[0])
	if err != nil {
		return SemVer{}, false
	}

	minor := 0
	if len(nums) >= 2 {
		minor, err = strconv.Atoi(nums[1])
		if err != nil {
			return SemVer{}, false
		}
	}

	patch := 0
	if len(nums) >= 3 {
		patch, err = strconv.Atoi(nums[2])
		if err != nil {
			return SemVer{}, false
		}
	}

	return SemVer{
		Major: major,
		Minor: minor,
		Patch: patch,
		Pre:   pre,
	}, true
}

// Compare compares two semantic versions:
// returns 1 if v1 > v2, -1 if v1 < v2, and 0 if v1 == v2.
func Compare(v1, v2 string) int {
	sv1, ok1 := Parse(v1)
	sv2, ok2 := Parse(v2)

	if !ok1 || !ok2 {
		// Fallback to normalized string comparison if semver parsing fails
		s1 := strings.TrimPrefix(strings.TrimSpace(v1), "v")
		s2 := strings.TrimPrefix(strings.TrimSpace(v2), "v")
		return strings.Compare(s1, s2)
	}

	if sv1.Major != sv2.Major {
		if sv1.Major > sv2.Major {
			return 1
		}
		return -1
	}

	if sv1.Minor != sv2.Minor {
		if sv1.Minor > sv2.Minor {
			return 1
		}
		return -1
	}

	if sv1.Patch != sv2.Patch {
		if sv1.Patch > sv2.Patch {
			return 1
		}
		return -1
	}

	// Normal release is higher than pre-release of the same version
	if sv1.Pre == "" && sv2.Pre != "" {
		return 1
	}
	if sv1.Pre != "" && sv2.Pre == "" {
		return -1
	}

	return strings.Compare(sv1.Pre, sv2.Pre)
}

// IsNewer returns true if candidate version is strictly newer than current version.
func IsNewer(candidate, current string) bool {
	return Compare(candidate, current) > 0
}

// Normalize removes leading 'v' / 'V' and trims spaces.
func Normalize(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	return v
}
