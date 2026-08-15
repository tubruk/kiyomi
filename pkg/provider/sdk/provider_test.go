package sdk_test

import (
	"testing"

	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

func TestMangaMetadata(t *testing.T) {
	meta := &sdk.MangaMetadata{
		RemoteID:     "116778",
		Title:        "Chainsaw Man",
		Aliases:      []string{"CSM", "チェンソーマン"},
		ReadingMode:  sdk.ReadingModeRTL,
		Availability: sdk.AvailabilityAvailable,
	}

	if meta.RemoteID != "116778" || meta.Title != "Chainsaw Man" {
		t.Errorf("unexpected metadata: %+v", meta)
	}
	if len(meta.Aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(meta.Aliases))
	}
	if meta.ReadingMode != sdk.ReadingModeRTL {
		t.Errorf("expected reading mode %s, got %s", sdk.ReadingModeRTL, meta.ReadingMode)
	}
	if meta.Availability != sdk.AvailabilityAvailable {
		t.Errorf("expected availability %s, got %s", sdk.AvailabilityAvailable, meta.Availability)
	}
}

func TestContentAvailabilityConstants(t *testing.T) {
	if sdk.AvailabilityAvailable != "available" {
		t.Errorf("expected available, got %s", sdk.AvailabilityAvailable)
	}
	if sdk.AvailabilityUnavailable != "unavailable" {
		t.Errorf("expected unavailable, got %s", sdk.AvailabilityUnavailable)
	}
	if sdk.AvailabilityUnknown != "unknown" {
		t.Errorf("expected unknown, got %s", sdk.AvailabilityUnknown)
	}

	sr := sdk.SearchResult{
		RemoteID:     "sr-1",
		Title:        "Sample",
		Availability: sdk.AvailabilityUnavailable,
	}
	if sr.Availability != sdk.AvailabilityUnavailable {
		t.Errorf("expected %s, got %s", sdk.AvailabilityUnavailable, sr.Availability)
	}
}

func TestReadingModeConstants(t *testing.T) {
	if sdk.ReadingModeUnspecified != "" {
		t.Errorf("expected empty string, got %q", sdk.ReadingModeUnspecified)
	}
	if sdk.ReadingModeLTR != "ltr" {
		t.Errorf("expected ltr, got %s", sdk.ReadingModeLTR)
	}
	if sdk.ReadingModeRTL != "rtl" {
		t.Errorf("expected rtl, got %s", sdk.ReadingModeRTL)
	}
	if sdk.ReadingModeVertical != "vertical" {
		t.Errorf("expected vertical, got %s", sdk.ReadingModeVertical)
	}
	if sdk.ReadingModeLongstrip != "longstrip" {
		t.Errorf("expected longstrip, got %s", sdk.ReadingModeLongstrip)
	}
}
