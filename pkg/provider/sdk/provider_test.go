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
		Authors:      []string{"Tatsuki Fujimoto"},
		Artists:      []string{"Tatsuki Fujimoto"},
		Tags:         []string{"Action", "Supernatural"},
		Publishers:   []string{"Shueisha"},
		ReleaseYear:  2018,
		StartDate:    "2018-12-03",
		EndDate:      "2020-12-14",
		Country:      "JP",
		ReadingMode:  sdk.ReadingModeRTL,
		Availability: sdk.AvailabilityAvailable,
	}

	if meta.RemoteID != "116778" || meta.Title != "Chainsaw Man" {
		t.Errorf("unexpected metadata: %+v", meta)
	}
	if len(meta.Aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(meta.Aliases))
	}
	if len(meta.Authors) != 1 || meta.Authors[0] != "Tatsuki Fujimoto" {
		t.Errorf("expected authors [Tatsuki Fujimoto], got %v", meta.Authors)
	}
	if len(meta.Artists) != 1 || meta.Artists[0] != "Tatsuki Fujimoto" {
		t.Errorf("expected artists [Tatsuki Fujimoto], got %v", meta.Artists)
	}
	if len(meta.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(meta.Tags))
	}
	if len(meta.Publishers) != 1 || meta.Publishers[0] != "Shueisha" {
		t.Errorf("expected publishers [Shueisha], got %v", meta.Publishers)
	}
	if meta.ReleaseYear != 2018 || meta.StartDate != "2018-12-03" || meta.EndDate != "2020-12-14" || meta.Country != "JP" {
		t.Errorf("unexpected date/country metadata: %+v", meta)
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
