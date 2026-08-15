package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstantsAndTypes(t *testing.T) {
	assert.Equal(t, "0.1.0", Version)
	assert.Equal(t, ContentAvailability("available"), AvailabilityAvailable)
	assert.Equal(t, ContentAvailability("unavailable"), AvailabilityUnavailable)
	assert.Equal(t, ContentAvailability("unknown"), AvailabilityUnknown)

	assert.Equal(t, ReadingMode(""), ReadingModeUnspecified)
	assert.Equal(t, ReadingMode("ltr"), ReadingModeLTR)
	assert.Equal(t, ReadingMode("rtl"), ReadingModeRTL)
	assert.Equal(t, ReadingMode("vertical"), ReadingModeVertical)
	assert.Equal(t, ReadingMode("longstrip"), ReadingModeLongstrip)

	assert.Equal(t, ImageSize("small"), ImageSizeSmall)
	assert.Equal(t, ImageSize("medium"), ImageSizeMedium)
	assert.Equal(t, ImageSize("large"), ImageSizeLarge)
}

func TestPluginMap(t *testing.T) {
	mockPlug := &mockPluginImpl{}
	mockProv := &mockProviderImpl{id: "mock"}

	pset := PluginMap(mockPlug,
		map[string]MetadataProvider{"mock": mockProv},
		map[string]ContentProvider{"mock": mockProv},
		map[string]Tracker{"mock": mockProv},
	)

	assert.Contains(t, pset, PluginServiceName)
	assert.Contains(t, pset, MetadataProviderPluginName)
	assert.Contains(t, pset, ContentProviderPluginName)
	assert.Contains(t, pset, TrackerPluginName)
}
