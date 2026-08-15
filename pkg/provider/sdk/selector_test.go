package sdk_test

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

func TestParseChapterNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float32
	}{
		{"volume and chapter", "Vol.01 Ch.001 Romance Dawn", 1.0},
		{"volume and decimal chapter", "Vol.2 Ch.10.5 Extra", 10.5},
		{"simple chapter prefix", "Chapter 42", 42.0},
		{"short prefix", "c123", 123.0},
		{"floating number", "15.5", 15.5},
		{"no numbers", "Prologue", -1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sdk.ParseChapterNumber(tt.input)
			if got != tt.expected {
				t.Errorf("ParseChapterNumber(%q) = %f, want %f", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractHelpers(t *testing.T) {
	html := `
	<div class="item">
		<span class="title">  Item Title  </span>
		<a class="link" href="/item/123" data-src="https://example.com/image.jpg">Link</a>
	</div>
	`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
	}

	item := doc.Find(".item")

	if text := sdk.ExtractText(item, ".title"); text != "Item Title" {
		t.Errorf("ExtractText = %q, want %q", text, "Item Title")
	}

	if text := sdk.ExtractText(item, ""); !strings.Contains(text, "Item Title") {
		t.Errorf("ExtractText with empty selector failed: %q", text)
	}

	if attr := sdk.ExtractAttr(item, ".link", "href"); attr != "/item/123" {
		t.Errorf("ExtractAttr = %q, want %q", attr, "/item/123")
	}

	if img := sdk.ExtractImageURL(item, ".link"); img != "https://example.com/image.jpg" {
		t.Errorf("ExtractImageURL = %q, want %q", img, "https://example.com/image.jpg")
	}

	if img := sdk.ExtractImageURL(item, ".nonexistent"); img != "" {
		t.Errorf("ExtractImageURL nonexistent = %q, want empty", img)
	}
}
