package cache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewDiskCache(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	c, err := NewDiskCache(cacheDir, 24*time.Hour, 1024*1024)
	if err != nil {
		t.Fatalf("unexpected error creating DiskCache: %v", err)
	}

	imagesDir := filepath.Join(cacheDir, "images")
	if info, err := os.Stat(imagesDir); err != nil || !info.IsDir() {
		t.Fatalf("expected images directory %s to exist", imagesDir)
	}

	urlStr := "https://example.com/test.jpg"
	expectedHash := sha256.Sum256([]byte(urlStr))
	expectedHex := hex.EncodeToString(expectedHash[:])

	if key := c.Key(urlStr); key != expectedHex {
		t.Fatalf("expected key %s, got %s", expectedHex, key)
	}

	shardDir, imgPath, jsonPath := c.paths(expectedHex)
	expectedShard := filepath.Join(cacheDir, "images", expectedHex[:2], expectedHex[2:4])
	if shardDir != expectedShard {
		t.Fatalf("expected shardDir %s, got %s", expectedShard, shardDir)
	}
	if imgPath != filepath.Join(expectedShard, expectedHex+".img") {
		t.Fatalf("expected imgPath %s, got %s", filepath.Join(expectedShard, expectedHex+".img"), imgPath)
	}
	if jsonPath != filepath.Join(expectedShard, expectedHex+".json") {
		t.Fatalf("expected jsonPath %s, got %s", filepath.Join(expectedShard, expectedHex+".json"), jsonPath)
	}
}

func TestMetadataSidecarFormat(t *testing.T) {
	tempDir := t.TempDir()
	c, err := NewDiskCache(tempDir, 1*time.Hour, 1024*1024)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	urlStr := "https://example.com/sidecar.jpg"
	data := []byte("sidecar-image-bytes")

	rc, _, err := c.GetOrFetch(context.Background(), urlStr, func(ctx context.Context) (io.ReadCloser, Meta, error) {
		return io.NopCloser(bytes.NewReader(data)), Meta{
			ContentType:   "image/jpeg",
			ContentLength: int64(len(data)),
		}, nil
	})
	if err != nil {
		t.Fatalf("GetOrFetch failed: %v", err)
	}
	_ = rc.Close()

	hash := c.Key(urlStr)
	_, _, jsonPath := c.paths(hash)

	jsonBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read json sidecar file: %v", err)
	}

	var parsed Meta
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal sidecar json: %v", err)
	}

	if parsed.ContentType != "image/jpeg" {
		t.Fatalf("expected ContentType image/jpeg, got %s", parsed.ContentType)
	}
	if parsed.ContentLength != int64(len(data)) {
		t.Fatalf("expected ContentLength %d, got %d", len(data), parsed.ContentLength)
	}
	if parsed.CreatedAt.IsZero() || parsed.LastAccessed.IsZero() {
		t.Fatalf("expected non-zero CreatedAt and LastAccessed timestamps")
	}
}

func TestGetOrFetchAndCacheHit(t *testing.T) {
	tempDir := t.TempDir()
	c, err := NewDiskCache(tempDir, 1*time.Hour, 10*1024*1024)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	urlStr := "https://example.com/image1.png"
	imageData := []byte("dummy-png-image-binary-data")
	fetchCount := 0

	fetcher := func(ctx context.Context) (io.ReadCloser, Meta, error) {
		fetchCount++
		return io.NopCloser(bytes.NewReader(imageData)), Meta{
			ContentType:   "image/png",
			ContentLength: int64(len(imageData)),
		}, nil
	}

	// 1. Initial Cache Miss
	rc, meta, err := c.GetOrFetch(context.Background(), urlStr, fetcher)
	if err != nil {
		t.Fatalf("GetOrFetch failed: %v", err)
	}
	defer rc.Close()

	if fetchCount != 1 {
		t.Fatalf("expected fetchCount=1, got %d", fetchCount)
	}
	if meta.ContentType != "image/png" {
		t.Fatalf("expected ContentType image/png, got %s", meta.ContentType)
	}
	if meta.ContentLength != int64(len(imageData)) {
		t.Fatalf("expected ContentLength %d, got %d", len(imageData), meta.ContentLength)
	}

	readBytes, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read from returned rc: %v", err)
	}
	if !bytes.Equal(readBytes, imageData) {
		t.Fatalf("expected data %s, got %s", string(imageData), string(readBytes))
	}

	// 2. Cache Hit via GetPath
	imgPath, hitMeta, hit := c.GetPath(urlStr)
	if !hit {
		t.Fatal("expected GetPath to return cache hit")
	}
	if imgPath == "" {
		t.Fatal("expected non-empty imgPath")
	}
	if hitMeta.ContentType != "image/png" {
		t.Fatalf("expected ContentType image/png, got %s", hitMeta.ContentType)
	}
	if hitMeta.LastAccessed.IsZero() {
		t.Fatal("expected non-zero LastAccessed timestamp")
	}

	// 3. Cache Hit via GetOrFetch (fetcher should NOT be called again)
	rc2, meta2, err := c.GetOrFetch(context.Background(), urlStr, fetcher)
	if err != nil {
		t.Fatalf("GetOrFetch second call failed: %v", err)
	}
	defer rc2.Close()

	if fetchCount != 1 {
		t.Fatalf("expected fetchCount to still be 1, got %d", fetchCount)
	}
	if meta2.ContentType != "image/png" {
		t.Fatalf("expected ContentType image/png, got %s", meta2.ContentType)
	}

	readBytes2, err := io.ReadAll(rc2)
	if err != nil {
		t.Fatalf("failed to read from rc2: %v", err)
	}
	if !bytes.Equal(readBytes2, imageData) {
		t.Fatalf("expected data %s, got %s", string(imageData), string(readBytes2))
	}
}

func TestSingleflightDeduplication(t *testing.T) {
	tempDir := t.TempDir()
	c, err := NewDiskCache(tempDir, 1*time.Hour, 10*1024*1024)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	urlStr := "https://example.com/concurrent.jpg"
	imageData := []byte("concurrent-fetch-test-payload")

	var fetchCalls int32
	fetcher := func(ctx context.Context) (io.ReadCloser, Meta, error) {
		atomic.AddInt32(&fetchCalls, 1)
		// Small delay to ensure concurrent requests overlap
		time.Sleep(50 * time.Millisecond)
		return io.NopCloser(bytes.NewReader(imageData)), Meta{
			ContentType: "image/jpeg",
		}, nil
	}

	concurrency := 10
	var wg sync.WaitGroup
	wg.Add(concurrency)

	errs := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			rc, meta, err := c.GetOrFetch(context.Background(), urlStr, fetcher)
			if err != nil {
				errs <- err
				return
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				errs <- err
				return
			}
			if !bytes.Equal(data, imageData) {
				errs <- errors.New("read data mismatch")
				return
			}
			if meta.ContentType != "image/jpeg" {
				errs <- errors.New("unexpected content type")
				return
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent worker error: %v", err)
		}
	}

	if calls := atomic.LoadInt32(&fetchCalls); calls != 1 {
		t.Fatalf("expected exactly 1 fetch call due to singleflight, got %d", calls)
	}
}

func TestCopyTo(t *testing.T) {
	tempDir := t.TempDir()
	c, err := NewDiskCache(filepath.Join(tempDir, "cache"), 1*time.Hour, 10*1024*1024)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	urlStr := "https://example.com/chapter1/001.jpg"
	imageData := []byte("manga-page-binary-stream-12345")

	fetcher := func(ctx context.Context) (io.ReadCloser, Meta, error) {
		return io.NopCloser(bytes.NewReader(imageData)), Meta{
			ContentType: "image/jpeg",
		}, nil
	}

	destPath := filepath.Join(tempDir, "dest", "manga", "001.jpg")

	// 1. CopyTo on cache miss (populates cache and copies/links)
	err = c.CopyTo(context.Background(), urlStr, destPath, fetcher)
	if err != nil {
		t.Fatalf("CopyTo failed: %v", err)
	}

	destData, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read destPath: %v", err)
	}
	if !bytes.Equal(destData, imageData) {
		t.Fatalf("dest data mismatch: got %s, want %s", string(destData), string(imageData))
	}

	// Verify cached file exists
	imgPath, _, hit := c.GetPath(urlStr)
	if !hit {
		t.Fatal("expected cached image to exist")
	}

	// 2. CopyTo on cache hit to a second destination
	destPath2 := filepath.Join(tempDir, "dest", "manga", "001_copy.jpg")
	err = c.CopyTo(context.Background(), urlStr, destPath2, nil)
	if err != nil {
		t.Fatalf("CopyTo second destination failed: %v", err)
	}

	destData2, err := os.ReadFile(destPath2)
	if err != nil {
		t.Fatalf("failed to read destPath2: %v", err)
	}
	if !bytes.Equal(destData2, imageData) {
		t.Fatalf("dest2 data mismatch: got %s, want %s", string(destData2), string(imageData))
	}

	// Verify hard link share if same filesystem
	cachedStat, _ := os.Stat(imgPath)
	destStat, _ := os.Stat(destPath)
	if cachedStat != nil && destStat != nil {
		// Both files should have identical sizes
		if cachedStat.Size() != destStat.Size() {
			t.Fatalf("size mismatch between cached %d and dest %d", cachedStat.Size(), destStat.Size())
		}
	}
}

func TestTTLExpirationAndPurge(t *testing.T) {
	tempDir := t.TempDir()
	// TTL = 100 milliseconds
	c, err := NewDiskCache(tempDir, 100*time.Millisecond, 100*1024*1024)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	urlStr := "https://example.com/ttl-test.png"
	imageData := []byte("ttl-image-data")

	fetcher := func(ctx context.Context) (io.ReadCloser, Meta, error) {
		return io.NopCloser(bytes.NewReader(imageData)), Meta{
			ContentType: "image/png",
		}, nil
	}

	rc, _, err := c.GetOrFetch(context.Background(), urlStr, fetcher)
	if err != nil {
		t.Fatalf("GetOrFetch failed: %v", err)
	}
	_ = rc.Close()

	// Immediately should hit
	if _, _, hit := c.GetPath(urlStr); !hit {
		t.Fatal("expected cache hit before TTL expiry")
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// GetPath should report cache miss due to TTL expiry
	if _, _, hit := c.GetPath(urlStr); hit {
		t.Fatal("expected cache miss after TTL expiry")
	}

	// Purge should remove the expired file
	purgedBytes, err := c.PurgeExpiredAndLRU()
	if err != nil {
		t.Fatalf("PurgeExpiredAndLRU failed: %v", err)
	}
	if purgedBytes < int64(len(imageData)) {
		t.Fatalf("expected purgedBytes >= %d, got %d", len(imageData), purgedBytes)
	}
}

func TestLRUEviction(t *testing.T) {
	tempDir := t.TempDir()
	// MaxBytes = 250 bytes (can hold ~2 items of 100 bytes each)
	c, err := NewDiskCache(tempDir, 24*time.Hour, 250)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	payload1 := bytes.Repeat([]byte("1"), 100)
	payload2 := bytes.Repeat([]byte("2"), 100)
	payload3 := bytes.Repeat([]byte("3"), 100)

	url1 := "https://example.com/item1.jpg"
	url2 := "https://example.com/item2.jpg"
	url3 := "https://example.com/item3.jpg"

	// Fetch item 1
	rc1, _, _ := c.GetOrFetch(context.Background(), url1, func(ctx context.Context) (io.ReadCloser, Meta, error) {
		return io.NopCloser(bytes.NewReader(payload1)), Meta{}, nil
	})
	_ = rc1.Close()

	time.Sleep(10 * time.Millisecond)

	// Fetch item 2
	rc2, _, _ := c.GetOrFetch(context.Background(), url2, func(ctx context.Context) (io.ReadCloser, Meta, error) {
		return io.NopCloser(bytes.NewReader(payload2)), Meta{}, nil
	})
	_ = rc2.Close()

	time.Sleep(10 * time.Millisecond)

	// Touch item 1 via GetPath so item 1 is accessed more recently than item 2
	_, _, hit1 := c.GetPath(url1)
	if !hit1 {
		t.Fatal("expected hit for item1")
	}

	time.Sleep(10 * time.Millisecond)

	// Fetch item 3 (total size is now 300 bytes > 250 maxBytes)
	rc3, _, _ := c.GetOrFetch(context.Background(), url3, func(ctx context.Context) (io.ReadCloser, Meta, error) {
		return io.NopCloser(bytes.NewReader(payload3)), Meta{}, nil
	})
	_ = rc3.Close()

	// Run LRU purge
	purged, err := c.PurgeExpiredAndLRU()
	if err != nil {
		t.Fatalf("PurgeExpiredAndLRU failed: %v", err)
	}
	if purged < 100 {
		t.Fatalf("expected purged >= 100, got %d", purged)
	}

	// Item 2 was oldest accessed, so item 2 should have been evicted
	if _, _, hit2 := c.GetPath(url2); hit2 {
		t.Fatal("expected item2 to be evicted by LRU")
	}

	// Item 1 and Item 3 should still be in cache
	if _, _, hit1 := c.GetPath(url1); !hit1 {
		t.Fatal("expected item1 to remain in cache")
	}
	if _, _, hit3 := c.GetPath(url3); !hit3 {
		t.Fatal("expected item3 to remain in cache")
	}
}

func TestStartCleanupWorker(t *testing.T) {
	tempDir := t.TempDir()
	c, err := NewDiskCache(tempDir, 50*time.Millisecond, 1024*1024)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	urlStr := "https://example.com/cleanup-worker.jpg"
	imageData := []byte("worker-image-data")

	rc, _, err := c.GetOrFetch(context.Background(), urlStr, func(ctx context.Context) (io.ReadCloser, Meta, error) {
		return io.NopCloser(bytes.NewReader(imageData)), Meta{}, nil
	})
	if err != nil {
		t.Fatalf("GetOrFetch failed: %v", err)
	}
	_ = rc.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c.StartCleanupWorker(ctx, 30*time.Millisecond)

	// Wait for TTL (50ms) + at least two worker ticks
	time.Sleep(150 * time.Millisecond)

	// Verify file is gone from cache
	if _, _, hit := c.GetPath(urlStr); hit {
		t.Fatal("expected cleanup worker to have purged expired item")
	}
}

func TestFetchErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	c, err := NewDiskCache(tempDir, 1*time.Hour, 1024*1024)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	expectedErr := errors.New("upstream fetch error")
	urlStr := "https://example.com/fail.jpg"

	_, _, err = c.GetOrFetch(context.Background(), urlStr, func(ctx context.Context) (io.ReadCloser, Meta, error) {
		return nil, Meta{}, expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	// Verify no stray files were created
	if _, _, hit := c.GetPath(urlStr); hit {
		t.Fatal("cache should not have entry after failed fetch")
	}
}

func TestDiskCache_ParallelPurgeRace(t *testing.T) {
	tempDir := t.TempDir()
	// TTL of 50ms and small maxBytes (8KB) to trigger frequent TTL expiration and LRU eviction
	c, err := NewDiskCache(tempDir, 50*time.Millisecond, 8*1024)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	const numURLs = 30
	urls := make([]string, numURLs)
	payloads := make([][]byte, numURLs)
	for i := 0; i < numURLs; i++ {
		urls[i] = fmt.Sprintf("https://example.com/stress_%d.jpg", i)
		payloads[i] = bytes.Repeat([]byte{byte('a' + (i % 26))}, 512)
	}

	var wg sync.WaitGroup

	// Concurrently perform GetOrFetch across multiple goroutines
	for i := 0; i < 8; i++ {
		workerID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					idx := (workerID + int(time.Now().UnixNano())) % numURLs
					urlStr := urls[idx]
					expected := payloads[idx]

					rc, meta, err := c.GetOrFetch(ctx, urlStr, func(fetchCtx context.Context) (io.ReadCloser, Meta, error) {
						return io.NopCloser(bytes.NewReader(expected)), Meta{
							ContentType:   "image/jpeg",
							ContentLength: int64(len(expected)),
						}, nil
					})
					if err != nil {
						continue
					}

					data, err := io.ReadAll(rc)
					_ = rc.Close()
					if err == nil && len(data) > 0 {
						if !bytes.Equal(data, expected) {
							t.Errorf("data corruption detected for %s: got %d bytes, want %d", urlStr, len(data), len(expected))
							return
						}
					}
					if meta.ContentType != "" && meta.ContentType != "image/jpeg" {
						t.Errorf("unexpected meta content type: %s", meta.ContentType)
						return
					}
				}
			}
		}()
	}

	// Concurrently perform GetPath across multiple goroutines
	for i := 0; i < 6; i++ {
		workerID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					idx := (workerID + int(time.Now().UnixNano())) % numURLs
					urlStr := urls[idx]
					imgPath, meta, hit := c.GetPath(urlStr)
					if hit && imgPath != "" {
						if meta.ContentType != "" && meta.ContentType != "image/jpeg" {
							t.Errorf("unexpected meta content type: %s", meta.ContentType)
							return
						}
						if data, err := os.ReadFile(imgPath); err == nil {
							if len(data) > 0 && !bytes.Equal(data, payloads[idx]) {
								t.Errorf("data mismatch from GetPath for %s", urlStr)
								return
							}
						}
					}
				}
			}
		}()
	}

	// Concurrently perform PurgeExpiredAndLRU in background goroutines
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, _ = c.PurgeExpiredAndLRU()
					time.Sleep(5 * time.Millisecond)
				}
			}
		}()
	}

	wg.Wait()
}

func TestStatsAndClear(t *testing.T) {
	tempDir := t.TempDir()
	c, err := NewDiskCache(tempDir, 24*time.Hour, 10*1024*1024)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	// 1. Initially empty stats
	size, count, err := c.Stats()
	if err != nil {
		t.Fatalf("Stats failed on empty cache: %v", err)
	}
	if size != 0 || count != 0 {
		t.Fatalf("expected 0 bytes and 0 count, got %d bytes and %d count", size, count)
	}

	// 2. Add 2 cached items
	data1 := []byte("image-data-one-12345")
	data2 := []byte("image-data-two-67890")

	rc1, _, err := c.GetOrFetch(context.Background(), "https://example.com/img1.jpg", func(ctx context.Context) (io.ReadCloser, Meta, error) {
		return io.NopCloser(bytes.NewReader(data1)), Meta{ContentType: "image/jpeg"}, nil
	})
	if err != nil {
		t.Fatalf("GetOrFetch 1 failed: %v", err)
	}
	_ = rc1.Close()

	rc2, _, err := c.GetOrFetch(context.Background(), "https://example.com/img2.jpg", func(ctx context.Context) (io.ReadCloser, Meta, error) {
		return io.NopCloser(bytes.NewReader(data2)), Meta{ContentType: "image/jpeg"}, nil
	})
	if err != nil {
		t.Fatalf("GetOrFetch 2 failed: %v", err)
	}
	_ = rc2.Close()

	// 3. Stats after adding items
	expectedSize := int64(len(data1) + len(data2))
	size, count, err = c.Stats()
	if err != nil {
		t.Fatalf("Stats failed after fetching images: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
	if size != expectedSize {
		t.Fatalf("expected size %d, got %d", expectedSize, size)
	}

	// 4. Clear cache
	if err := c.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// 5. Stats after clear
	size, count, err = c.Stats()
	if err != nil {
		t.Fatalf("Stats failed after clear: %v", err)
	}
	if size != 0 || count != 0 {
		t.Fatalf("expected 0 bytes and 0 count after clear, got %d bytes and %d count", size, count)
	}

	// 6. GetPath returns false
	if _, _, hit := c.GetPath("https://example.com/img1.jpg"); hit {
		t.Fatal("expected cache miss for img1 after clear")
	}
	if _, _, hit := c.GetPath("https://example.com/img2.jpg"); hit {
		t.Fatal("expected cache miss for img2 after clear")
	}
}

