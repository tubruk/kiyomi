package host_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tubruk/kiyomi/internal/plugin/host"
)

func TestParseSemVer(t *testing.T) {
	tests := []struct {
		input    string
		expected host.SemVer
		err      bool
	}{
		{
			input: "0.1.0",
			expected: host.SemVer{
				Major: 0,
				Minor: 1,
				Patch: 0,
				Raw:   "0.1.0",
			},
		},
		{
			input: "v1.2.3",
			expected: host.SemVer{
				Major: 1,
				Minor: 2,
				Patch: 3,
				Raw:   "v1.2.3",
			},
		},
		{
			input: "2.0.0-beta.1",
			expected: host.SemVer{
				Major:      2,
				Minor:      0,
				Patch:      0,
				Prerelease: "beta.1",
				Raw:        "2.0.0-beta.1",
			},
		},
		{
			input: "invalid",
			err:   true,
		},
		{
			input: "",
			err:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			sv, err := host.ParseSemVer(tt.input)
			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.Major, sv.Major)
				assert.Equal(t, tt.expected.Minor, sv.Minor)
				assert.Equal(t, tt.expected.Patch, sv.Patch)
				assert.Equal(t, tt.expected.Prerelease, sv.Prerelease)
			}
		})
	}
}

func TestCompareSemVer(t *testing.T) {
	assert.Equal(t, 1, host.CompareSemVer("1.2.0", "1.1.9"))
	assert.Equal(t, -1, host.CompareSemVer("1.1.0", "1.2.0"))
	assert.Equal(t, 0, host.CompareSemVer("1.2.3", "1.2.3"))
	assert.Equal(t, 1, host.CompareSemVer("2.0.0", "1.99.99"))
	assert.Equal(t, 1, host.CompareSemVer("1.0.0", "1.0.0-alpha"))
	assert.Equal(t, -1, host.CompareSemVer("1.0.0-alpha", "1.0.0"))
}

func TestCheckSDKCompatibility(t *testing.T) {
	// Pre-v1 tests: exact minor match required
	require.NoError(t, host.CheckSDKCompatibility("0.1.0", "0.1.0"))
	require.NoError(t, host.CheckSDKCompatibility("0.1.2", "0.1.0"))
	require.NoError(t, host.CheckSDKCompatibility("0.1.0", "0.1.9"))
	require.Error(t, host.CheckSDKCompatibility("0.1.0", "0.2.0"))
	require.Error(t, host.CheckSDKCompatibility("0.1.0", "0.0.9"))
	require.Error(t, host.CheckSDKCompatibility("0.1.0", "1.0.0"))

	// v1+ tests: major must match, host minor >= plugin minor
	require.NoError(t, host.CheckSDKCompatibility("1.2.0", "1.2.0"))
	require.NoError(t, host.CheckSDKCompatibility("1.5.0", "1.2.0"))
	require.Error(t, host.CheckSDKCompatibility("1.2.0", "1.5.0"))
	require.Error(t, host.CheckSDKCompatibility("2.0.0", "1.5.0"))
}
