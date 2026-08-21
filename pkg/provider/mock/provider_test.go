//go:build e2e

package mock

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

func getFixturesDir() string {
	_, filename, _, _ := runtime.Caller(0)
	// mock is at pkg/provider/mock, fixtures is at docs/e2e/fixtures
	root := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	return filepath.Join(root, "docs", "e2e", "fixtures")
}

func TestMockProviderAvailability(t *testing.T) {
	p := New(getFixturesDir())

	results, err := p.Search(context.Background(), "", sdk.SearchOptions{})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) < 4 {
		t.Fatalf("expected at least 4 catalog results, got %d", len(results))
	}

	foundUnavailable := false
	for _, r := range results {
		if r.RemoteID == "unavailable-manga" {
			foundUnavailable = true
			if r.Availability != sdk.AvailabilityUnavailable {
				t.Errorf("expected unavailable-manga availability to be %s, got %s", sdk.AvailabilityUnavailable, r.Availability)
			}
		}
		if r.RemoteID == "alpha" {
			if r.Availability != sdk.AvailabilityAvailable {
				t.Errorf("expected alpha availability to be %s, got %s", sdk.AvailabilityAvailable, r.Availability)
			}
		}
	}
	if !foundUnavailable {
		t.Errorf("unavailable-manga not found in search results")
	}

	// Details test for unavailable-manga
	meta, err := p.Details(context.Background(), "unavailable-manga")
	if err != nil {
		t.Fatalf("Details failed: %v", err)
	}
	if meta.Availability != sdk.AvailabilityUnavailable {
		t.Errorf("expected meta availability %s, got %s", sdk.AvailabilityUnavailable, meta.Availability)
	}
	if meta.Title != "Unavailable Manga" {
		t.Errorf("expected title 'Unavailable Manga', got %s", meta.Title)
	}
}

func TestMockProviderWithCustomID(t *testing.T) {
	p := NewWithID("mock-a", "Mock A", getFixturesDir())
	if p.ID() != "mock-a" {
		t.Errorf("expected ID mock-a, got %s", p.ID())
	}
	if p.Name() != "Mock A" {
		t.Errorf("expected name Mock A, got %s", p.Name())
	}
	// Verify it still works as a provider
	results, err := p.Search(context.Background(), "", sdk.SearchOptions{})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected catalog results")
	}
}

func TestMockProviderDistinctInstances(t *testing.T) {
	a := NewWithID("mock-a", "Mock A", getFixturesDir())
	b := NewWithID("mock-b", "Mock B", getFixturesDir())
	if a.ID() == b.ID() {
		t.Error("instances should have distinct IDs")
	}
}
