package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	sdk "github.com/tubruk/kiyomi/plugin-sdk"
	"github.com/tubruk/kiyomi/plugin-sdk/utils"
)

// JSONSource provides helper methods for calling REST JSON APIs and extracting nested responses.
type JSONSource struct {
	BaseURL string
	Client  HTTPDoer
}

// NewJSONSource creates a new JSONSource. If client is nil, http.DefaultClient is used.
func NewJSONSource(baseURL string, client HTTPDoer) *JSONSource {
	if client == nil {
		client = http.DefaultClient
	}
	return &JSONSource{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client:  client,
	}
}

// ResolveURL resolves a relative or absolute URL against the source's BaseURL.
func (j *JSONSource) ResolveURL(targetURL string) string {
	resolved, err := utils.ResolveURL(j.BaseURL, targetURL)
	if err != nil {
		return targetURL
	}
	return resolved
}

// FetchJSON sends a GET request and unmarshals JSON response directly into target.
func (j *JSONSource) FetchJSON(ctx context.Context, pathOrURL string, target any) error {
	fullURL := j.ResolveURL(pathOrURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := j.Client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed for %s: %w", fullURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("http %d for %s: %s", resp.StatusCode, fullURL, strings.TrimSpace(string(snippet)))
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("failed to decode JSON from %s: %w", fullURL, err)
		}
	}
	return nil
}

// FetchExtractor sends a GET request and returns a JSONExtractor on the response body.
func (j *JSONSource) FetchExtractor(ctx context.Context, pathOrURL string) (*JSONExtractor, error) {
	return j.FetchExtractorWithHeaders(ctx, pathOrURL, nil)
}

// FetchExtractorWithHeaders sends a GET request with headers and returns a JSONExtractor.
func (j *JSONSource) FetchExtractorWithHeaders(ctx context.Context, pathOrURL string, headers map[string]string) (*JSONExtractor, error) {
	fullURL := j.ResolveURL(pathOrURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := j.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed for %s: %w", fullURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("http %d for %s: %s", resp.StatusCode, fullURL, strings.TrimSpace(string(snippet)))
	}

	return NewJSONExtractorFromReader(resp.Body)
}

// PostJSON sends a POST request with a JSON body and unmarshals the response into target.
func (j *JSONSource) PostJSON(ctx context.Context, pathOrURL string, body any, target any) error {
	fullURL := j.ResolveURL(pathOrURL)
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := j.Client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed for %s: %w", fullURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("http %d for %s: %s", resp.StatusCode, fullURL, strings.TrimSpace(string(snippet)))
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("failed to decode JSON from %s: %w", fullURL, err)
		}
	}
	return nil
}

// PostExtractor sends a POST request with a JSON body and returns a JSONExtractor.
func (j *JSONSource) PostExtractor(ctx context.Context, pathOrURL string, body any) (*JSONExtractor, error) {
	return j.PostExtractorWithHeaders(ctx, pathOrURL, body, nil)
}

// PostExtractorWithHeaders sends a POST request with body and headers and returns a JSONExtractor.
func (j *JSONSource) PostExtractorWithHeaders(ctx context.Context, pathOrURL string, body any, headers map[string]string) (*JSONExtractor, error) {
	fullURL := j.ResolveURL(pathOrURL)
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := j.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed for %s: %w", fullURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("http %d for %s: %s", resp.StatusCode, fullURL, strings.TrimSpace(string(snippet)))
	}

	return NewJSONExtractorFromReader(resp.Body)
}

// JSONExtractor wraps parsed JSON data and provides dot-path navigation and type conversions.
type JSONExtractor struct {
	val any
}

// NewJSONExtractor parses raw JSON bytes into a JSONExtractor.
func NewJSONExtractor(data []byte) (*JSONExtractor, error) {
	var val any
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return &JSONExtractor{val: val}, nil
}

// NewJSONExtractorFromReader reads and parses JSON from an io.Reader.
func NewJSONExtractorFromReader(r io.Reader) (*JSONExtractor, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON: %w", err)
	}
	return NewJSONExtractor(data)
}

// NewJSONExtractorFromAny creates a JSONExtractor wrapping any Go value.
func NewJSONExtractorFromAny(val any) *JSONExtractor {
	return &JSONExtractor{val: val}
}

// parsePath splits a path string like "data.items[0].name" or "items.0.title" into discrete segment tokens.
func parsePath(path string) []string {
	if strings.TrimSpace(path) == "" {
		return nil
	}

	var tokens []string
	var current strings.Builder

	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}

	for i := 0; i < len(path); i++ {
		ch := path[i]
		switch ch {
		case '.':
			flush()
		case '[':
			flush()
		case ']':
			flush()
		default:
			current.WriteByte(ch)
		}
	}
	flush()

	return tokens
}

// Get traverses a dot-separated / index-bracketed path and returns a new JSONExtractor.
func (e *JSONExtractor) Get(path string) *JSONExtractor {
	if e == nil || e.val == nil {
		return &JSONExtractor{val: nil}
	}

	tokens := parsePath(path)
	current := e.val

	for _, token := range tokens {
		if current == nil {
			return &JSONExtractor{val: nil}
		}

		switch v := current.(type) {
		case map[string]any:
			current = v[token]
		case []any:
			idx, err := strconv.Atoi(token)
			if err != nil || idx < 0 || idx >= len(v) {
				return &JSONExtractor{val: nil}
			}
			current = v[idx]
		default:
			return &JSONExtractor{val: nil}
		}
	}

	return &JSONExtractor{val: current}
}

// targetExtractor resolves an optional path parameter.
func (e *JSONExtractor) targetExtractor(path ...string) *JSONExtractor {
	if len(path) > 0 && path[0] != "" {
		return e.Get(path[0])
	}
	return e
}

// Exists reports whether the extractor holds a non-nil value.
func (e *JSONExtractor) Exists(path ...string) bool {
	target := e.targetExtractor(path...)
	return target != nil && target.val != nil
}

// IsEmpty reports whether the extractor is nil, empty string, empty map, or empty slice.
func (e *JSONExtractor) IsEmpty(path ...string) bool {
	target := e.targetExtractor(path...)
	if target == nil || target.val == nil {
		return true
	}
	switch v := target.val.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	}
	return false
}

// Raw returns the underlying generic value (e.g. map[string]any, []any, float64, string, bool).
func (e *JSONExtractor) Raw() any {
	if e == nil {
		return nil
	}
	return e.val
}

// RawBytes marshals the wrapped value back to JSON bytes.
func (e *JSONExtractor) RawBytes() ([]byte, error) {
	if e == nil || e.val == nil {
		return []byte("null"), nil
	}
	return json.Marshal(e.val)
}

// Unmarshal unmarshals the current node into target struct.
func (e *JSONExtractor) Unmarshal(target any) error {
	data, err := e.RawBytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// String extracts string value, converting numbers/bools to string representation if needed.
func (e *JSONExtractor) String(path ...string) string {
	return e.StringOr("", path...)
}

// StringOr extracts string value or returns defaultValue.
func (e *JSONExtractor) StringOr(defaultValue string, path ...string) string {
	target := e.targetExtractor(path...)
	if target == nil || target.val == nil {
		return defaultValue
	}

	switch v := target.val.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Int extracts int value.
func (e *JSONExtractor) Int(path ...string) int {
	return e.IntOr(0, path...)
}

// IntOr extracts int value or returns defaultValue.
func (e *JSONExtractor) IntOr(defaultValue int, path ...string) int {
	target := e.targetExtractor(path...)
	if target == nil || target.val == nil {
		return defaultValue
	}

	switch v := target.val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return i
		}
	}
	return defaultValue
}

// Int64 extracts int64 value.
func (e *JSONExtractor) Int64(path ...string) int64 {
	return e.Int64Or(0, path...)
}

// Int64Or extracts int64 value or returns defaultValue.
func (e *JSONExtractor) Int64Or(defaultValue int64, path ...string) int64 {
	target := e.targetExtractor(path...)
	if target == nil || target.val == nil {
		return defaultValue
	}

	switch v := target.val.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		if i, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			return i
		}
	}
	return defaultValue
}

// Float extracts float64 value.
func (e *JSONExtractor) Float(path ...string) float64 {
	return e.FloatOr(0, path...)
}

// FloatOr extracts float64 value or returns defaultValue.
func (e *JSONExtractor) FloatOr(defaultValue float64, path ...string) float64 {
	target := e.targetExtractor(path...)
	if target == nil || target.val == nil {
		return defaultValue
	}

	switch v := target.val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f
		}
	}
	return defaultValue
}

// Float32 extracts float32 value.
func (e *JSONExtractor) Float32(path ...string) float32 {
	return float32(e.Float(path...))
}

// Bool extracts bool value.
func (e *JSONExtractor) Bool(path ...string) bool {
	return e.BoolOr(false, path...)
}

// BoolOr extracts bool value or returns defaultValue.
func (e *JSONExtractor) BoolOr(defaultValue bool, path ...string) bool {
	target := e.targetExtractor(path...)
	if target == nil || target.val == nil {
		return defaultValue
	}

	switch v := target.val.(type) {
	case bool:
		return v
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		return lower == "true" || lower == "1" || lower == "yes"
	case float64:
		return v != 0
	case int:
		return v != 0
	}
	return defaultValue
}

// Array returns a slice of JSONExtractors if the node is a JSON array.
func (e *JSONExtractor) Array(path ...string) []*JSONExtractor {
	target := e.targetExtractor(path...)
	if target == nil || target.val == nil {
		return nil
	}

	arr, ok := target.val.([]any)
	if !ok {
		return nil
	}

	result := make([]*JSONExtractor, len(arr))
	for i, item := range arr {
		result[i] = &JSONExtractor{val: item}
	}
	return result
}

// StringArray extracts a slice of strings from an array.
func (e *JSONExtractor) StringArray(path ...string) []string {
	target := e.targetExtractor(path...)
	if target == nil || target.val == nil {
		return nil
	}

	arr, ok := target.val.([]any)
	if !ok {
		return nil
	}

	var result []string
	for _, item := range arr {
		if str, ok := item.(string); ok {
			result = append(result, strings.TrimSpace(str))
		} else if item != nil {
			result = append(result, fmt.Sprintf("%v", item))
		}
	}
	return result
}

// Map returns a map of child extractors if the node is a JSON object.
func (e *JSONExtractor) Map(path ...string) map[string]*JSONExtractor {
	target := e.targetExtractor(path...)
	if target == nil || target.val == nil {
		return nil
	}

	obj, ok := target.val.(map[string]any)
	if !ok {
		return nil
	}

	result := make(map[string]*JSONExtractor, len(obj))
	for k, v := range obj {
		result[k] = &JSONExtractor{val: v}
	}
	return result
}

// Time parses an RFC3339, custom layout, or standard date format string.
func (e *JSONExtractor) Time(layout string, path ...string) (time.Time, error) {
	str := e.String(path...)
	if str == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}
	if layout != "" {
		return time.Parse(layout, str)
	}
	return utils.ParseDate(str)
}

// UnixTime parses a unix timestamp (in seconds or milliseconds).
func (e *JSONExtractor) UnixTime(path ...string) time.Time {
	target := e.targetExtractor(path...)
	if target == nil || target.val == nil {
		return time.Time{}
	}

	ts := target.Int64()
	if ts <= 0 {
		return time.Time{}
	}

	// If timestamp > 1e11, it's likely milliseconds (e.g. 1700000000000)
	if ts > 100_000_000_000 {
		return time.UnixMilli(ts)
	}
	return time.Unix(ts, 0)
}

// MapArray maps each element of the array at path using fn.
func (e *JSONExtractor) MapArray(path string, fn func(item *JSONExtractor) error) error {
	items := e.Array(path)
	for _, item := range items {
		if err := fn(item); err != nil {
			return err
		}
	}
	return nil
}

// ExtractSearchResults maps an array of JSON objects to sdk.SearchResult.
func (e *JSONExtractor) ExtractSearchResults(path string, fn func(item *JSONExtractor) (*sdk.SearchResult, error)) ([]sdk.SearchResult, error) {
	items := e.Array(path)
	var results []sdk.SearchResult
	for _, item := range items {
		res, err := fn(item)
		if err != nil {
			return nil, err
		}
		if res != nil {
			results = append(results, *res)
		}
	}
	return results, nil
}

// ExtractChapters maps an array of JSON objects to sdk.Chapter.
func (e *JSONExtractor) ExtractChapters(path string, fn func(item *JSONExtractor) (*sdk.Chapter, error)) ([]sdk.Chapter, error) {
	items := e.Array(path)
	var chapters []sdk.Chapter
	for _, item := range items {
		ch, err := fn(item)
		if err != nil {
			return nil, err
		}
		if ch != nil {
			chapters = append(chapters, *ch)
		}
	}
	return chapters, nil
}

// ExtractPages maps an array of JSON objects or strings to sdk.Page.
func (e *JSONExtractor) ExtractPages(path string, fn func(item *JSONExtractor) (*sdk.Page, error)) ([]sdk.Page, error) {
	items := e.Array(path)
	var pages []sdk.Page
	for _, item := range items {
		page, err := fn(item)
		if err != nil {
			return nil, err
		}
		if page != nil {
			pages = append(pages, *page)
		}
	}
	return pages, nil
}
