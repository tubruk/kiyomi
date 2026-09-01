package main
import (
    "encoding/json"
    "fmt"
    "strings"
)
type StringOrSlice []string

// UnmarshalJSON accepts either a JSON string or an array of strings.
func (s *StringOrSlice) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as an array of strings.
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*s = arr
		return nil
	}
	// Try to unmarshal as a single string.
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*s = []string{single}
		return nil
	}
	var err error
	return fmt.Errorf("StringOrSlice: %w", err)
}

// StringObjectOrSlice can hold a JSON value that may be a string, an object, or an array of strings.
// Use it when a field's type is ambiguous in the Atsumaru API.
// After unmarshalling, only one of the fields will be non‑nil/non‑empty.

type StringObjectOrSlice struct {
    // When the JSON value is a string.
    Str *string
    // When the JSON value is an object (generic map).
    Obj map[string]interface{}
    // When the JSON value is an array of strings.
    Slice []string
}

func (v *StringObjectOrSlice) UnmarshalJSON(data []byte) error {
    // Trim whitespace.
    d := strings.TrimSpace(string(data))
    if len(d) == 0 {
        return nil
    }
    switch d[0] {
    case '{':
        // Object.
        var m map[string]interface{}
        if err := json.Unmarshal(data, &m); err != nil {
            return fmt.Errorf("StringObjectOrSlice (object): %w", err)
        }
        v.Obj = m
        return nil
    case '[':
        // Slice of strings.
        var s []string
        if err := json.Unmarshal(data, &s); err != nil {
            return fmt.Errorf("StringObjectOrSlice (slice): %w", err)
        }
        v.Slice = s
        return nil
    case '"':
        // Simple string.
        var str string
        if err := json.Unmarshal(data, &str); err != nil {
            return fmt.Errorf("StringObjectOrSlice (string): %w", err)
        }
        v.Str = &str
        return nil
    default:
        // Unexpected type.
        return fmt.Errorf("StringObjectOrSlice: unexpected JSON token %c", d[0])
    }
}

// Helper getters.
func (v *StringObjectOrSlice) GetString() (string, bool) {
    if v.Str != nil {
        return *v.Str, true
    }
    return "", false
}

func (v *StringObjectOrSlice) GetObject() (map[string]interface{}, bool) {
    if v.Obj != nil {
        return v.Obj, true
    }
    return nil, false
}

// GetStringField returns the value of a named field when the underlying JSON is an object.
func (v *StringObjectOrSlice) GetStringField(key string) (string, bool) {
    if v == nil {
        return "", false
    }
    if v.Obj != nil {
        if raw, ok := v.Obj[key]; ok {
            if s, ok2 := raw.(string); ok2 {
                return s, true
            }
        }
    }
    return "", false
}

func (v *StringObjectOrSlice) GetSlice() ([]string, bool) {
    if v == nil {
        return nil, false
    }
    if v.Slice != nil {
        return v.Slice, true
    }
    return nil, false
}

// atsumaruSearchHit represents a single search document match.
type atsumaruSearchHit struct {
	Document atsumaruMangaDto `json:"document"`
}

// atsumaruSearchResultsDto represents the Typesense search results payload.
type atsumaruSearchResultsDto struct {
	Page          int                 `json:"page"`
	Found         int                 `json:"found"`
	Hits          []atsumaruSearchHit `json:"hits"`
	RequestParams struct {
		PerPage int `json:"per_page"`
	} `json:"request_params"`
}

// atsumaruBrowseDto represents the popular / recentlyUpdated response payload.
type atsumaruBrowseDto struct {
	Items []atsumaruMangaDto `json:"items"`
}

// atsumaruMangaObjectDto represents the /api/manga/page response wrapper.
type atsumaruMangaObjectDto struct {
	MangaPage atsumaruMangaDto `json:"mangaPage"`
}

// atsumaruAuthorDto represents an author/artist with optional role type.
type atsumaruAuthorDto struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Type string `json:"type"` // "Author", "Artist", etc.
}

// UnmarshalJSON handles cases where the API returns an author as an object or a plain string.
func (a *atsumaruAuthorDto) UnmarshalJSON(data []byte) error {
	// Attempt to unmarshal as the expected struct.
	type Alias atsumaruAuthorDto
	var aux Alias
	err := json.Unmarshal(data, &aux)
	if err == nil {
		*a = atsumaruAuthorDto(aux)
		return nil
	}
	// If that fails, try unmarshalling a simple string (e.g., "") and ignore.
	var s string
	if err2 := json.Unmarshal(data, &s); err2 == nil {
		// No further data to set; keep zero values.
		return nil
	}
	// Return the original error if both attempts fail.
	return fmt.Errorf("atsumaruAuthorDto: %w", err)
}

// atsumaruScanlatorDto represents a scanlation group mapping.
type atsumaruScanlatorDto struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

// atsumaruPosterDto represents multi-resolution poster image URLs.
type atsumaruPosterDto struct {
	ID          string `json:"id"`
	Image       string `json:"image"`
	SmallImage  string `json:"smallImage"`
	MediumImage string `json:"mediumImage"`
	LargeImage  string `json:"largeImage"`
}

// UnmarshalJSON tolerates the poster field being a JSON object or a plain string (e.g., empty).
func (p *atsumaruPosterDto) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as the expected struct.
	type Alias atsumaruPosterDto
	var aux Alias
	err := json.Unmarshal(data, &aux)
	if err == nil {
		*p = atsumaruPosterDto(aux)
		return nil
	}
	// If it is a string, ignore the value.
	var s string
	if err2 := json.Unmarshal(data, &s); err2 == nil {
		return nil
	}
	return fmt.Errorf("atsumaruPosterDto: %w", err)
}

// atsumaruMangaDto represents manga metadata details from Atsumaru.
type atsumaruMangaDto struct {
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	EnglishTitle   string                 `json:"englishTitle"`
	OtherNames     []string               `json:"otherNames"`
	Image          string                 `json:"image"`
	SmallImage     string                 `json:"smallImage"`
	MediumImage    string                 `json:"mediumImage"`
	LargeImage     string                 `json:"largeImage"`
	Poster         *StringObjectOrSlice     `json:"poster"`
	Synopsis       string                 `json:"synopsis"`
	Status         string                 `json:"status"`
	Type           string                 `json:"type"`
	Medium         string                 `json:"medium"`
	Released       int64                  `json:"released"` // Epoch ms
	AvgRating      float32                `json:"avgRating"`
	MBRating       float32                `json:"mbRating"`
	Views          json.RawMessage        `json:"views"`
	Genres         []atsumaruGenreItem    `json:"genres"`
	Tags           []atsumaruTagItem      `json:"tags"`
	Authors        []atsumaruAuthorDto    `json:"authors"`
	Scanlators     []atsumaruScanlatorDto `json:"scanlators"`
	TotalChapters  float64                `json:"totalChapterCount"`
	IsAdult        bool                   `json:"isAdult"`
	MALID          string                 `json:"malId"`
	AniListID      string                 `json:"anilistId"`
	KitsuID        string                 `json:"kitsuId"`
	MangaUpdatesID string                 `json:"mangaUpdatesId"`
}

// atsumaruGenreItem represents genre item.
type atsumaruGenreItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UnmarshalJSON handles cases where the genre is returned as an object or a plain string.
func (g *atsumaruGenreItem) UnmarshalJSON(data []byte) error {
	type Alias atsumaruGenreItem
	var aux Alias
	if err := json.Unmarshal(data, &aux); err == nil {
		*g = atsumaruGenreItem(aux)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*g = atsumaruGenreItem{Name: s}
		return nil
	}
	return nil
}

// atsumaruTagItem represents tag item.
type atsumaruTagItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UnmarshalJSON handles cases where the tag is returned as an object or a plain string.
func (t *atsumaruTagItem) UnmarshalJSON(data []byte) error {
	type Alias atsumaruTagItem
	var aux Alias
	if err := json.Unmarshal(data, &aux); err == nil {
		*t = atsumaruTagItem(aux)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*t = atsumaruTagItem{Name: s}
		return nil
	}
	return nil
}

// atsumaruAllChaptersDto represents the /api/manga/allChapters response payload.
type atsumaruAllChaptersDto struct {
	Chapters []atsumaruChapterDto `json:"chapters"`
}

// atsumaruChapterDto represents a single chapter in Atsumaru.
type atsumaruChapterDto struct {
	ID                 string  `json:"id"`
	Title              string  `json:"title"`
	Number             float32 `json:"number"`
	ScanlationMangaID  string  `json:"scanlationMangaId"`
	CreatedAt          int64   `json:"createdAt"` // Epoch ms
	Index              int     `json:"index"`
	PageCount          int     `json:"pageCount"`
}

// atsumaruPageObjectDto represents the /api/read/chapter response wrapper.
type atsumaruPageObjectDto struct {
	ReadChapter atsumaruPageDataDto `json:"readChapter"`
}

// atsumaruPageDataDto represents the chapter pages array.
type atsumaruPageDataDto struct {
	ID                string             `json:"id"`
	Title             string             `json:"title"`
	ScanlationMangaID string             `json:"scanlationMangaId"`
	Pages             []atsumaruImageDto `json:"pages"`
}

// atsumaruImageDto represents a single page image reference.
type atsumaruImageDto struct {
	ID          string  `json:"id"`
	Image       string  `json:"image"`
	Number      int     `json:"number"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	AspectRatio float64 `json:"aspectRatio"`
}
