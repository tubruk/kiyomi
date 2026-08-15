package host

import (
	"fmt"
	"strconv"
	"strings"
)

// SemVer represents a parsed semantic version.
type SemVer struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Raw        string
}

// String returns the normalized version string.
func (v SemVer) String() string {
	if v.Prerelease != "" {
		return fmt.Sprintf("%d.%d.%d-%s", v.Major, v.Minor, v.Patch, v.Prerelease)
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// ParseSemVer parses a SemVer string, ignoring leading 'v'.
func ParseSemVer(v string) (SemVer, error) {
	raw := strings.TrimSpace(v)
	s := strings.TrimPrefix(raw, "v")
	if s == "" {
		return SemVer{}, fmt.Errorf("empty version string")
	}

	var prerelease string
	if dashIdx := strings.Index(s, "-"); dashIdx >= 0 {
		prerelease = s[dashIdx+1:]
		s = s[:dashIdx]
	}

	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return SemVer{}, fmt.Errorf("invalid semver format %q", v)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid major version in %q: %w", v, err)
	}

	minor := 0
	if len(parts) > 1 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return SemVer{}, fmt.Errorf("invalid minor version in %q: %w", v, err)
		}
	}

	patch := 0
	if len(parts) > 2 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return SemVer{}, fmt.Errorf("invalid patch version in %q: %w", v, err)
		}
	}

	return SemVer{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
		Raw:        raw,
	}, nil
}

// CompareSemVer compares two semantic version strings.
// Returns +1 if v1 > v2, -1 if v1 < v2, and 0 if v1 == v2.
func CompareSemVer(v1, v2 string) int {
	sv1, err1 := ParseSemVer(v1)
	sv2, err2 := ParseSemVer(v2)
	if err1 != nil && err2 != nil {
		return strings.Compare(v1, v2)
	}
	if err1 != nil {
		return -1
	}
	if err2 != nil {
		return 1
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

	// Non-prerelease is greater than prerelease
	if sv1.Prerelease == "" && sv2.Prerelease != "" {
		return 1
	}
	if sv1.Prerelease != "" && sv2.Prerelease == "" {
		return -1
	}
	return strings.Compare(sv1.Prerelease, sv2.Prerelease)
}

// CheckSDKCompatibility verifies that the plugin's SDK version is compatible with the host's SDK version.
// - For pre-v1 (0.x.x) releases: Exact minor version match required (e.g. host v0.1.x accepts plugin v0.1.y, rejects v0.2.y).
// - For v1+ releases: Major version must match, and host minor must be >= plugin minor.
func CheckSDKCompatibility(hostSDKVersion, pluginSDKVersion string) error {
	if hostSDKVersion == "" {
		return fmt.Errorf("host SDK version is not set")
	}
	if pluginSDKVersion == "" {
		return fmt.Errorf("plugin SDK version is empty")
	}

	hostVer, err := ParseSemVer(hostSDKVersion)
	if err != nil {
		return fmt.Errorf("invalid host SDK version %q: %w", hostSDKVersion, err)
	}

	plugVer, err := ParseSemVer(pluginSDKVersion)
	if err != nil {
		return fmt.Errorf("invalid plugin SDK version %q: %w", pluginSDKVersion, err)
	}

	if hostVer.Major == 0 {
		// Pre-v1 rules: Exact minor version match required
		if plugVer.Major != 0 || plugVer.Minor != hostVer.Minor {
			return fmt.Errorf("incompatible SDK version: host is %s, plugin is %s (pre-v1 requires exact minor version match 0.%d.x)", hostSDKVersion, pluginSDKVersion, hostVer.Minor)
		}
		return nil
	}

	// v1+ rules
	if plugVer.Major != hostVer.Major {
		return fmt.Errorf("incompatible SDK major version: host is %s, plugin is %s", hostSDKVersion, pluginSDKVersion)
	}

	if hostVer.Minor < plugVer.Minor {
		return fmt.Errorf("plugin requires newer SDK minor version (%s) than host provides (%s)", pluginSDKVersion, hostSDKVersion)
	}

	return nil
}
